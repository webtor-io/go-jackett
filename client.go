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
	Client *http.Client

	// DefaultTrackers is a list of tracker IDs to use if a FetchRequest does not specify any.
	// If empty and a FetchRequest does not specify trackers, "all" trackers will be used.
	DefaultTrackers []string
}

// Client is a Jackett API client.
type Client struct {
	apiURL *url.URL
	cfg    Settings
	tcache sync.Map
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
	j := Client{cfg: s}
	apiURLStr := valOrEnv(s.ApiURL, envAPIURL)
	apiURL, err := url.Parse(apiURLStr)
	if err != nil {
		return nil, fmt.Errorf("parse url: %s: %w", apiURLStr, err)
	}
	j.apiURL = apiURL

	if j.cfg.Client == nil {
		j.cfg.Client = http.DefaultClient
	}
	j.cfg.Client.Transport = wrapTransport(j.cfg.Client.Transport,
		j.apiURL,
		valOrEnv(s.ApiKey, envAPIKey))
	return &j, nil
}

type fetchOpts struct {
	MaxConcurrency     int
	ProgressReportFunc func(complete uint, total uint)
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
			tracker := extractTracker(u.Path)
			if tracker == "all" {
				return nil // can't check caps of meta indexers
			}
			caps, err := j.getIndexerCaps(ctx, tracker)
			if err != nil {
				return fmt.Errorf("get indexer caps: %s: %w", tracker, err)
			}
			if err := caps.Validate(u.Query()); err != nil {
				return fmt.Errorf("%s does not support this query: %w", tracker, err)
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
				return fmt.Errorf("fetch: %s: %w", url.String(), err)
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

func (j *Client) getIndexerCaps(ctx context.Context, id string) (IndexerCaps, error) {
	if v, ok := j.tcache.Load(id); ok {
		return v.(IndexerCaps), nil
	}
	u := *j.apiURL
	u.Path = fmt.Sprintf("/api/v2.0/indexers/%s/results/torznab", id)
	q := u.Query()
	q.Add("t", "caps")
	u.RawQuery = q.Encode()
	caps, err := getXML[IndexerCaps](ctx, j.cfg.Client, u.String())
	if err != nil {
		return caps, fmt.Errorf("list indexers: %w", err)
	}
	j.tcache.Store(id, caps)
	return caps, nil
}

func (j *Client) generateFetchURLs(fr *FetchRequest) ([]url.URL, error) {
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
