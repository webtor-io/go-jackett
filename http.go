package jackett

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	if err := xml.Unmarshal(b, &data); err != nil {
		return data, fmt.Errorf("unmarshal response data: %s: %w\n%s", url, err, string(b))
	}
	return data, nil
}

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
	b, err := io.ReadAll(res.Body)
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
	if m.UserAgent != "" {
		r.Header.Set("User-Agent", m.UserAgent)
	} else {
		r.Header.Set("User-Agent", ua())
	}

	if m.matchesTarget(r.URL) {
		q := r.URL.Query()
		q.Set("apikey", m.APIKey)
		r.URL.RawQuery = q.Encode()
	}

	return m.Transport.RoundTrip(r)
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
