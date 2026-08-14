package jackett

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"

	"golang.org/x/net/context"
)

func getXML[T any](ctx context.Context, client *http.Client, url string) (T, error) {
	var data T
	b, err := getBytes(ctx, client, url)
	if err != nil {
		return data, err
	}
	if err := checkErrorDoc(b); err != nil {
		return data, err
	}
	if err := xml.Unmarshal(b, &data); err != nil {
		// The response body is deliberately not included. Callers expose
		// these errors to their own users, and the body is whatever the
		// endpoint returned — which, after a redirect, may be a page from
		// somewhere the caller never intended to fetch.
		return data, fmt.Errorf("unmarshal response data: %s: %w", url, err)
	}
	return data, nil
}

// errorResp is Torznab's in-band error document. XMLName pins the root
// element, so unmarshalling any other document into it fails — which is
// exactly how we tell an error apart from a real response.
type errorResp struct {
	XMLName     xml.Name `xml:"error"`
	Code        string   `xml:"code,attr"`
	Description string   `xml:"description,attr"`
	// Some implementations put the message in the element body instead of
	// the description attribute.
	Message string `xml:",chardata"`
}

// IndexerError is an error the indexer itself reported, as opposed to a
// transport or parsing failure. Callers can use errors.As to tell a
// rejected API key apart from an unreachable host.
type IndexerError struct {
	Code        string
	Description string
}

func (e *IndexerError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("indexer returned an error: %s", e.Description)
	}
	return fmt.Sprintf("indexer returned an error: %s: %s", e.Code, e.Description)
}

// checkErrorDoc surfaces the error document indexers return with HTTP 200.
// A caps request is where this matters most: an unauthenticated caps
// response otherwise unmarshals into an empty, entirely plausible-looking
// capabilities struct, so a wrong API key reads as "this indexer supports
// nothing" instead of "your key is wrong".
func checkErrorDoc(b []byte) error {
	var e errorResp
	if err := xml.Unmarshal(b, &e); err != nil {
		return nil
	}
	desc := strings.TrimSpace(e.Description)
	if desc == "" {
		desc = strings.TrimSpace(e.Message)
	}
	if e.Code == "" && desc == "" {
		return nil
	}
	return &IndexerError{Code: e.Code, Description: desc}
}

// redactURL hides credential-carrying query parameters so a URL can appear
// in an error message. Endpoint URLs are pasted by end users and may embed
// the key, and callers log these errors.
func redactURL(raw string) string {
	return secretParams.ReplaceAllString(raw, "$1=***")
}

var secretParams = regexp.MustCompile(`(?i)\b(apikey|api_key|jackett_apikey|passkey|rss_?key|token)=([^&\s"']+)`)

// maxBodyBytes caps what a single response may contribute. A 100-item feed
// is well under a megabyte; without a cap an endpoint — which is user-
// supplied for most callers — can stream until the process dies.
const maxBodyBytes = 8 << 20

func getBytes(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("make fetch request: %w", err)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("invoke fetch request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		// Reported before parsing, so a login page or a 404 says what it
		// is instead of surfacing as malformed XML.
		return nil, fmt.Errorf("fetch %s: unexpected status %d", url, res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("read respose: %w", err)
	}
	return b, nil
}

var _ http.RoundTripper = (*middleware)(nil)

// wrapTransport wraps the given http.Transport with a middleware that adds the user agent to all outgoing requests. It also adds the api key to all requests matching BaseURL.
func wrapTransport(rt http.RoundTripper, base *url.URL, apiKey, userAgent string) http.RoundTripper {
	if rt == nil {
		rt = http.DefaultTransport
	}
	return &middleware{
		Transport: rt,
		BaseURL:   base,
		APIKey:    apiKey,
		UserAgent: userAgent,
	}
}

type middleware struct {
	Transport http.RoundTripper
	BaseURL   *url.URL
	APIKey    string
	// UserAgent overrides the library's own identifier. Trackers do reject
	// or throttle by user agent, so the calling service has to be able to
	// decide what it presents itself as.
	UserAgent string
}

func (m *middleware) RoundTrip(r *http.Request) (*http.Response, error) {
	// RoundTrip must not modify the request it is given, and here that rule
	// has teeth: writing the key into r.URL leaves it in the URL the
	// http.Client reports in every *url.Error, and in the Referer it sends
	// on a cross-host redirect. Callers then log or display an error that
	// carries the credential.
	r2 := r.Clone(r.Context())
	if m.UserAgent != "" {
		r2.Header.Set("User-Agent", m.UserAgent)
	} else {
		r2.Header.Set("User-Agent", ua())
	}

	if m.APIKey != "" && m.matchesTarget(r2.URL) {
		u := *r2.URL
		q := u.Query()
		q.Set("apikey", m.APIKey)
		u.RawQuery = q.Encode()
		r2.URL = &u
	}

	return m.Transport.RoundTrip(r2)
}

func (m *middleware) matchesTarget(reqURL *url.URL) bool {
	if m.BaseURL == nil {
		return false
	}

	if reqURL.Scheme != m.BaseURL.Scheme {
		return false
	}

	return normalizeHost(reqURL) == normalizeHost(m.BaseURL)
}

func normalizeHost(u *url.URL) string {
	host := u.Host
	if !strings.Contains(host, ":") {
		switch u.Scheme {
		case "http":
			host += ":80"
		case "https":
			host += ":443"
		}
	}
	return strings.ToLower(host)
}

var ua = sync.OnceValue(func() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "go-jackett/unknown"
	}
	version := buildInfo.Main.Version
	if version == "" || version == "(devel)" {
		version = "dev"
	}
	return fmt.Sprintf("go-jackett/%s", version)
})
