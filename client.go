package jackett

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
)

// Settings defines the configuration for the Jackett client.
type Settings struct {
	// ApiURL is the base URL for the Jackett API.
	// If empty, the value of the JACKETT_API_URL environment variable will be used.
	ApiURL string

	// ApiKey is the API key for accessing the Jackett API.
	// If empty, the value of the JACKETT_API_KEY environment variable will be used.
	ApiKey string

	// Client is the HTTP client to use for making requests.
	// If nil, http.DefaultClient will be used.
	//
	// A copy is taken at construction time and the api-key middleware is
	// installed on the copy, so the caller's client is left untouched and
	// several clients may share one. Two consequences: changes made to the
	// client after New returns are not picked up, and requests the caller
	// makes through its own client do not get the api key.
	Client *http.Client

	// DefaultTrackers is a list of tracker IDs to use if a FetchRequest does not specify any.
	// If empty and a FetchRequest does not specify trackers, "all" trackers will be used.
	DefaultTrackers []string

	// UserAgent is sent with every request. If empty, the library
	// identifies itself as go-jackett/<version>.
	UserAgent string
}

// Client is a Jackett API client.
type Client struct {
	apiURL *url.URL
	cfg    Settings
	tcache sync.Map
	// endpoint is set by NewTorznab: apiURL is then a complete Torznab
	// feed URL rather than a Jackett API root, and no indexer path is
	// constructed from it.
	endpoint bool
}

const (
	envAPIURL = "JACKETT_API_URL"
	envAPIKey = "JACKETT_API_KEY"
)

// New creates a new Jackett client with the given settings.
// It will return an error if the API URL cannot be parsed.
// Environment variables JACKETT_API_URL and JACKETT_API_KEY can be used
// as fallbacks if ApiURL or ApiKey are not provided in Settings.
func New(s Settings) (*Client, error) {
	return newClient(s, false)
}

// NewTorznab creates a client for a single Torznab endpoint. ApiURL is used
// as given — it is the feed URL itself, not a Jackett API root — so the
// client also speaks to Prowlarr, NZBHydra2 and a tracker's own feed, none
// of which expose Jackett's /api/v2.0/indexers/<id>/results/torznab layout.
//
// Trackers, DefaultTrackers and ListIndexers are Jackett-specific and do
// not apply: the endpoint is whatever the URL points at, which may itself
// be an aggregate over many trackers.
func NewTorznab(s Settings) (*Client, error) {
	return newClient(s, true)
}

func newClient(s Settings, endpoint bool) (*Client, error) {
	j := Client{cfg: s, endpoint: endpoint}
	apiURLStr := valOrEnv(s.ApiURL, envAPIURL)
	apiURL, err := url.Parse(apiURLStr)
	if err != nil {
		return nil, fmt.Errorf("parse url: %s: %w", apiURLStr, err)
	}
	j.apiURL = apiURL

	// The api-key middleware is per-client, so it must go on a copy of the
	// caller's http.Client rather than on the client itself. Mutating the
	// original would install one client's key on every other client sharing
	// it — including http.DefaultClient, process-wide — and with several
	// clients built from one http.Client the wrapped transports stack, so
	// requests to a host two of them share end up carrying whichever key
	// was installed last.
	base := s.Client
	if base == nil {
		base = http.DefaultClient
	}
	copied := *base
	copied.Transport = wrapTransport(base.Transport,
		j.apiURL,
		valOrEnv(s.ApiKey, envAPIKey),
		s.UserAgent)
	j.cfg.Client = &copied
	return &j, nil
}

type fetchOpts struct {
	MaxConcurrency     int
	ProgressReportFunc func(complete uint, total uint)
	SkipCapsValidation bool
}

// FetchOption is a function that configures a Fetch operation.
type FetchOption func(*fetchOpts)

// WithMaxConcurrency sets the maximum number of concurrent requests for a Fetch operation.
// If n is less than 1, it defaults to runtime.NumCPU().
func WithMaxConcurrency(n uint) FetchOption {
	return func(o *fetchOpts) {
		o.MaxConcurrency = int(n)
	}
}

// WithoutCapsValidation skips the pre-flight capability check.
//
// Fetch normally asks every target for its caps before querying it, so an
// unsupported query fails as an *UnsupportedError instead of as an empty
// result. Callers that already know the capabilities — because they keep
// their own snapshot of them — can skip that round-trip.
func WithoutCapsValidation() FetchOption {
	return func(o *fetchOpts) {
		o.SkipCapsValidation = true
	}
}

// WithProgressReportFunc sets a callback function to report progress during a Fetch operation.
// The callback receives the number of completed requests and the total number of requests.
func WithProgressReportFunc(f func(complete uint, total uint)) FetchOption {
	return func(o *fetchOpts) {
		o.ProgressReportFunc = f
	}
}

// Fetch executes a fetch request against the Jackett API.
// It returns a slice containing the combined results of the fetch, or an error.
// Fetch will concurrently request data from multiple trackers when possible.
// Error type will be *UnsupportedError when the query is not supported by the tracker.
func (j *Client) Fetch(ctx context.Context, fr *FetchRequest, opts ...FetchOption) ([]Result, error) {
	var o fetchOpts
	for _, f := range opts {
		f(&o)
	}
	if o.MaxConcurrency < 1 {
		o.MaxConcurrency = runtime.NumCPU()
	}
	if o.ProgressReportFunc == nil {
		o.ProgressReportFunc = func(_, _ uint) {}
	}
	o.ProgressReportFunc(0, math.MaxInt32)

	urls, err := j.generateFetchURLs(fr)
	if err != nil {
		return nil, fmt.Errorf("generate urls: %w", err)
	}

	var wg errgroup.Group
	wg.SetLimit(o.MaxConcurrency)

	// Ensure that all selected trackers support this query
	for _, u := range urls {
		wg.Go(func() error {
			if o.SkipCapsValidation {
				return nil
			}
			tracker := extractTracker(u.Path)
			if !j.endpoint && tracker == "all" {
				return nil // can't check caps of meta indexers
			}
			caps, err := j.getIndexerCaps(ctx, tracker)
			if err != nil {
				return fmt.Errorf("get indexer caps: %s: %w", tracker, err)
			}
			// Validate what the request asks for, not the whole URL: in
			// endpoint mode the URL also carries whatever the pasted feed
			// URL brought along (an apikey, a tracker's own parameter),
			// and no indexer advertises those in supportedParams.
			own, err := fr.Values()
			if err != nil {
				return fmt.Errorf("marshal url: %w", err)
			}
			if err := caps.Validate(own); err != nil {
				return fmt.Errorf("%s does not support this query: %w", targetName(tracker, u), err)
			}
			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		return nil, fmt.Errorf("at least one query was invalid: %w", err)
	}

	total := uint(len(urls))
	var complete atomic.Uint32

	o.ProgressReportFunc(0, total)
	results := make([][]Result, len(urls))
	for i, url := range urls {
		wg.Go(func() error {
			defer func() {
				o.ProgressReportFunc(uint(complete.Add(1)), total)
			}()
			resp, err := getXML[searchResp](ctx, j.cfg.Client, url.String())
			if err != nil {
				return fmt.Errorf("fetch: %s: %w", redactURL(url.String()), err)
			}
			results[i], err = resp.Unmarshal()
			return err
		})
	}

	err = wg.Wait()
	resp := slices.Concat(results...)
	o.ProgressReportFunc(total, total)
	return resp, err
}

// ListIndexers returns a slice of all configured indexers on this Jackett instance.
func (j *Client) ListIndexers(ctx context.Context) ([]IndexerDetails, error) {
	u := *j.apiURL
	u.Path = "/api/v2.0/indexers/all/results/torznab"
	q := u.Query()
	q.Add("t", "indexers")
	q.Add("configured", "true")
	u.RawQuery = q.Encode()
	idxs, err := getXML[indexersResp](ctx, j.cfg.Client, u.String())
	if err != nil {
		return nil, fmt.Errorf("list indexers: %w", err)
	}
	slices.SortFunc(idxs.Indexers, func(a, b IndexerDetails) int {
		return strings.Compare(a.ID, b.ID)
	})
	return idxs.Indexers, err
}

// Caps returns the capabilities of an indexer, cached per client.
//
// In Torznab-endpoint mode the id is ignored — the endpoint reports its own
// capabilities. In Jackett mode it is the indexer id ("all" for the meta
// indexer).
func (j *Client) Caps(ctx context.Context, id string) (IndexerCaps, error) {
	return j.getIndexerCaps(ctx, id)
}

func (j *Client) getIndexerCaps(ctx context.Context, id string) (IndexerCaps, error) {
	if j.endpoint {
		id = ""
	}
	if v, ok := j.tcache.Load(id); ok {
		return v.(IndexerCaps), nil
	}
	u := *j.apiURL
	if !j.endpoint {
		u.Path = fmt.Sprintf("/api/v2.0/indexers/%s/results/torznab", id)
	}
	q := u.Query()
	q.Set("t", "caps")
	u.RawQuery = q.Encode()
	caps, err := getXML[IndexerCaps](ctx, j.cfg.Client, u.String())
	if err != nil {
		return caps, fmt.Errorf("get caps: %w", err)
	}
	j.tcache.Store(id, caps)
	return caps, nil
}

func (j *Client) generateFetchURLs(fr *FetchRequest) ([]url.URL, error) {
	if j.endpoint {
		u := *j.apiURL
		q, err := fr.Values()
		if err != nil {
			return nil, fmt.Errorf("marshal url: %w", err)
		}
		// Whatever else the pasted feed URL carried stays as a default —
		// but never a search parameter. A URL copied out of a browser
		// address bar brings the last query along with it, and inheriting
		// its q= would and-search every later request against a stale
		// term. Categories are the one search parameter left inheritable,
		// so a user can paste a category-filtered feed and have it apply
		// to requests that set no categories of their own.
		for k, vs := range u.Query() {
			if isOwnedParam(k) {
				continue
			}
			if _, ok := q[k]; !ok {
				q[k] = vs
			}
		}
		u.RawQuery = q.Encode()
		return []url.URL{u}, nil
	}
	trackers := fr.Trackers()
	if len(trackers) == 0 {
		trackers = j.cfg.DefaultTrackers
	}
	if len(trackers) == 0 {
		trackers = []string{"all"} // meta tracker
	}

	var urls []url.URL
	for _, tracker := range trackers {
		u := *j.apiURL
		u.Path = fmt.Sprintf("/api/v2.0/indexers/%s/results/torznab", tracker)
		q, err := fr.Values()
		if err != nil {
			return nil, fmt.Errorf("marshal url: %w", err)
		}
		u.RawQuery = q.Encode()
		urls = append(urls, u)
	}
	return urls, nil
}

// targetName labels a target in error messages. Endpoint mode has no
// tracker id to report, so the host stands in for it.
func targetName(tracker string, u url.URL) string {
	if tracker != "" {
		return tracker
	}
	return u.Host
}

func valOrEnv(v, env string) string {
	if v != "" {
		return v
	}
	return os.Getenv(env)
}

func extractTracker(path string) string {
	_, tracker, ok := strings.Cut(path, "api/v2.0/indexers/")
	if !ok {
		return ""
	}
	return strings.TrimSuffix(tracker, "/results/torznab")
}
