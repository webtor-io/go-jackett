package jackett

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

const newznabFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:newznab="http://www.newznab.com/DTD/2010/feeds/attributes/">
<channel>
  <item>
    <title>The.Matrix.1999.1080p.BluRay.x264</title>
    <guid>https://tracker.example.com/t/1</guid>
    <link>magnet:?xt=urn:btih:aaaabbbbccccddddeeeeffff0000111122223333&amp;dn=x</link>
    <pubDate>Mon, 02 Jan 2006 15:04:05 GMT</pubDate>
    <enclosure url="magnet:?xt=urn:btih:aaaabbbbccccddddeeeeffff0000111122223333" type="application/x-bittorrent" length="734003200"/>
    <newznab:attr name="seeders" value="17"/>
    <newznab:attr name="peers" value="20"/>
    <newznab:attr name="infohash" value="aaaabbbbccccddddeeeeffff0000111122223333"/>
  </item>
</channel>
</rss>`

const endpointCaps = `<?xml version="1.0" encoding="UTF-8"?>
<caps>
  <server title="My Prowlarr"/>
  <limits max="100" default="50"/>
  <searching>
    <search available="yes" supportedParams="q"/>
    <tv-search available="yes" supportedParams="q,season,ep,imdbid"/>
    <movie-search available="yes" supportedParams="q,imdbid"/>
  </searching>
</caps>`

// TestNewznabAttrsAreParsed guards the namespace-agnostic attr binding. A
// feed served in the Newznab flavour used to lose every attribute, which
// meant no seeders and — fatally — no infohash.
func TestNewznabAttrsAreParsed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(newznabFeed))
	}))
	defer srv.Close()

	j, err := NewTorznab(Settings{ApiURL: srv.URL})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	results, err := j.Fetch(context.Background(), NewRawSearch().WithQuery("matrix").Build(), WithoutCapsValidation())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	r := results[0]
	if r.Seeders != 17 || r.Peers != 20 {
		t.Errorf("swarm = %d/%d, want 17/20", r.Seeders, r.Peers)
	}
	if r.InfoHash != "aaaabbbbccccddddeeeeffff0000111122223333" {
		t.Errorf("infohash = %q, want the newznab:attr value", r.InfoHash)
	}
	if r.MagnetURI == "" {
		t.Error("magnet was not picked up from the item link")
	}
	if r.Size != 734003200 {
		t.Errorf("size = %d, want the enclosure length", r.Size)
	}
	if r.PublishDate.IsZero() {
		t.Error("an RFC1123 (named zone) pubDate was not parsed")
	}
}

// TestTorznabEndpointURL checks that the endpoint URL is used as given —
// Prowlarr and bare tracker feeds have no Jackett indexer path to build.
func TestTorznabEndpointURL(t *testing.T) {
	var mu sync.Mutex
	var got []*url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		u := *r.URL
		got = append(got, &u)
		mu.Unlock()
		_, _ = w.Write([]byte(newznabFeed))
	}))
	defer srv.Close()

	j, err := NewTorznab(Settings{ApiURL: srv.URL + "/api/v1/indexer/3/newznab?extra=keep&q=stale", ApiKey: "secret"})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	if _, err := j.Fetch(context.Background(),
		NewTVSearch().WithIMDBID("1839578").WithSeason(5).WithEpisode(14).Build(),
		WithoutCapsValidation()); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("made %d requests, want 1", len(got))
	}
	if got[0].Path != "/api/v1/indexer/3/newznab" {
		t.Errorf("path = %q, want the endpoint path untouched", got[0].Path)
	}
	q := got[0].Query()
	if q.Get("t") != "tvsearch" || q.Get("season") != "5" || q.Get("ep") != "14" || q.Get("imdbid") != "1839578" {
		t.Errorf("query = %v, want the tvsearch parameters", q)
	}
	if q.Get("apikey") != "secret" {
		t.Errorf("apikey = %q, want it appended by the middleware", q.Get("apikey"))
	}
	// A parameter the pasted URL carried survives...
	if q.Get("extra") != "keep" {
		t.Errorf("extra = %q, want the endpoint's own parameter kept", q.Get("extra"))
	}
	// ...but one this query owns is replaced, not appended to.
	if vs := q["q"]; len(vs) > 1 || (len(vs) == 1 && vs[0] == "stale") {
		t.Errorf("q = %v, want the stale pasted value gone", vs)
	}
}

func TestTorznabEndpointCaps(t *testing.T) {
	var calls int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		if r.URL.Query().Get("t") != "caps" {
			t.Errorf("t = %q, want caps", r.URL.Query().Get("t"))
		}
		if r.URL.Path != "/torznab" {
			t.Errorf("path = %q, want the endpoint path untouched", r.URL.Path)
		}
		_, _ = w.Write([]byte(endpointCaps))
	}))
	defer srv.Close()

	j, err := NewTorznab(Settings{ApiURL: srv.URL + "/torznab"})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	caps, err := j.Caps(context.Background(), "")
	if err != nil {
		t.Fatalf("Caps() error = %v", err)
	}
	if caps.Server.Title != "My Prowlarr" {
		t.Errorf("server title = %q, want My Prowlarr", caps.Server.Title)
	}
	if _, err := j.Caps(context.Background(), "ignored-in-endpoint-mode"); err != nil {
		t.Fatalf("Caps() second call error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Errorf("made %d caps requests, want 1 — the result is cached", calls)
	}
}

func TestTorznabEndpointCapsValidation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("t") == "caps" {
			_, _ = w.Write([]byte(endpointCaps))
			return
		}
		_, _ = w.Write([]byte(newznabFeed))
	}))
	defer srv.Close()

	j, err := NewTorznab(Settings{ApiURL: srv.URL + "/torznab"})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	// tvsearch here advertises no tvdbid, so the pre-flight check has to
	// reject the query rather than let the indexer answer with nothing.
	_, err = j.Fetch(context.Background(), NewTVSearch().WithTVDBID(1234).Build())
	if err == nil {
		t.Fatal("Fetch() accepted a query the endpoint does not support")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("error = %v, want an unsupported-parameter error", err)
	}
}

// TestClientDoesNotMutateSharedHTTPClient is the regression guard for
// per-user clients sharing one *http.Client: each client must carry its own
// api key, and the caller's client must come back unmodified.
func TestClientDoesNotMutateSharedHTTPClient(t *testing.T) {
	var mu sync.Mutex
	keys := map[string]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		keys[r.URL.Query().Get("q")] = r.URL.Query().Get("apikey")
		mu.Unlock()
		_, _ = w.Write([]byte(newznabFeed))
	}))
	defer srv.Close()

	shared := &http.Client{}
	first, err := NewTorznab(Settings{ApiURL: srv.URL + "/torznab", ApiKey: "key-one", Client: shared})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	second, err := NewTorznab(Settings{ApiURL: srv.URL + "/torznab", ApiKey: "key-two", Client: shared})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	if shared.Transport != nil {
		t.Error("the caller's http.Client was modified")
	}

	if _, err := first.Fetch(context.Background(), NewRawSearch().WithQuery("one").Build(), WithoutCapsValidation()); err != nil {
		t.Fatalf("first Fetch() error = %v", err)
	}
	if _, err := second.Fetch(context.Background(), NewRawSearch().WithQuery("two").Build(), WithoutCapsValidation()); err != nil {
		t.Fatalf("second Fetch() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if keys["one"] != "key-one" {
		t.Errorf("first client sent apikey %q, want key-one", keys["one"])
	}
	if keys["two"] != "key-two" {
		t.Errorf("second client sent apikey %q, want key-two", keys["two"])
	}
}

func TestSettingsUserAgent(t *testing.T) {
	var mu sync.Mutex
	var agents []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		agents = append(agents, r.UserAgent())
		mu.Unlock()
		_, _ = w.Write([]byte(newznabFeed))
	}))
	defer srv.Close()

	// Trackers reject or throttle by user agent, so a caller that sets one
	// must not have the library's own identifier put back over it.
	j, err := NewTorznab(Settings{ApiURL: srv.URL, UserAgent: "webtor.io"})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	if _, err := j.Fetch(context.Background(), NewRawSearch().WithQuery("x").Build(), WithoutCapsValidation()); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	def, err := NewTorznab(Settings{ApiURL: srv.URL})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	if _, err := def.Fetch(context.Background(), NewRawSearch().WithQuery("y").Build(), WithoutCapsValidation()); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(agents) != 2 {
		t.Fatalf("got %d requests, want 2", len(agents))
	}
	if agents[0] != "webtor.io" {
		t.Errorf("user agent = %q, want the configured one", agents[0])
	}
	if !strings.HasPrefix(agents[1], "go-jackett/") {
		t.Errorf("default user agent = %q, want go-jackett/<version>", agents[1])
	}
}

func TestCapsSurfacesErrorDocument(t *testing.T) {
	// A wrong API key answers a caps request with HTTP 200 and an error
	// document. Unmarshalled into IndexerCaps that yields an empty struct,
	// so without the check the caller is told the indexer supports no
	// search modes — pointing the user at the wrong problem entirely.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><error code="100" description="Incorrect user credentials"/>`))
	}))
	defer srv.Close()

	j, err := NewTorznab(Settings{ApiURL: srv.URL, ApiKey: "wrong"})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	_, err = j.Caps(context.Background(), "")
	if err == nil {
		t.Fatal("Caps() accepted an error document")
	}
	if !strings.Contains(err.Error(), "Incorrect user credentials") {
		t.Errorf("error = %v, want the indexer's description", err)
	}
}

// TestAPIKeyNeverLeavesTheRequest is the regression guard for the api key
// showing up where callers can log or display it. Writing the key into the
// request's own URL puts it into every *url.Error the http.Client produces
// and into the Referer sent on a cross-host redirect.
func TestAPIKeyNeverLeavesTheRequest(t *testing.T) {
	var mu sync.Mutex
	var referers []string
	var second *httptest.Server

	second = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		referers = append(referers, r.Referer())
		mu.Unlock()
		_, _ = w.Write([]byte(newznabFeed))
	}))
	defer second.Close()

	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, second.URL+"/elsewhere", http.StatusFound)
	}))
	defer first.Close()

	j, err := NewTorznab(Settings{ApiURL: first.URL, ApiKey: "SUPERSECRET"})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	if _, err := j.Fetch(context.Background(), NewRawSearch().WithQuery("x").Build(), WithoutCapsValidation()); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	for _, ref := range referers {
		if strings.Contains(ref, "SUPERSECRET") {
			t.Errorf("Referer sent to another host carries the api key: %s", ref)
		}
	}
}

func TestTransportErrorDoesNotCarryAPIKey(t *testing.T) {
	// Nothing listens here, so the client returns a *url.Error naming the
	// URL it dialled — the single most common way this credential escapes.
	j, err := NewTorznab(Settings{ApiURL: "http://127.0.0.1:1/torznab", ApiKey: "SUPERSECRET"})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	_, err = j.Fetch(context.Background(), NewRawSearch().WithQuery("x").Build(), WithoutCapsValidation())
	if err == nil {
		t.Fatal("Fetch() succeeded against a closed port")
	}
	if strings.Contains(err.Error(), "SUPERSECRET") {
		t.Errorf("transport error carries the api key: %v", err)
	}
}

func TestResponseBodyIsNotReflectedInErrors(t *testing.T) {
	// An endpoint that redirects to something that is not a feed must not
	// hand the caller that page's contents: callers surface these errors to
	// their own users, which would turn the client into a read primitive.
	secret := "INTERNAL-SERVICE-CONTENTS"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>" + secret + "</body></html>"))
	}))
	defer srv.Close()

	j, err := NewTorznab(Settings{ApiURL: srv.URL})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	_, err = j.Fetch(context.Background(), NewRawSearch().WithQuery("x").Build(), WithoutCapsValidation())
	if err == nil {
		t.Fatal("Fetch() accepted an HTML page as a feed")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("error reflects the response body: %v", err)
	}
}

func TestNonSuccessStatusIsReportedAsSuch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("<html>login</html>"))
	}))
	defer srv.Close()

	j, err := NewTorznab(Settings{ApiURL: srv.URL})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	_, err = j.Fetch(context.Background(), NewRawSearch().WithQuery("x").Build(), WithoutCapsValidation())
	if err == nil {
		t.Fatal("Fetch() accepted a 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %v, want the status code", err)
	}
}

// TestEndpointCapsValidationIgnoresInheritedParams covers the case the
// feature exists for: a pasted URL that carries its own parameters. Those
// are not part of the query the caller asked for, so validating them
// against supportedParams rejects every such endpoint.
func TestEndpointCapsValidationIgnoresInheritedParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("t") == "caps" {
			_, _ = w.Write([]byte(endpointCaps))
			return
		}
		_, _ = w.Write([]byte(newznabFeed))
	}))
	defer srv.Close()

	j, err := NewTorznab(Settings{ApiURL: srv.URL + "/torznab?apikey=USERKEY&extra=keep"})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	// Caps validation ON — this is the default path.
	results, err := j.Fetch(context.Background(), NewRawSearch().WithQuery("x").Build())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

// TestPastedAPIKeyIsNotWiped: the key lives in the URL for Prowlarr and
// NZBHydra2 copy-paste URLs, and Settings.ApiKey is then empty. Overwriting
// it with "" sends every request unauthenticated.
func TestPastedAPIKeyIsNotWiped(t *testing.T) {
	var mu sync.Mutex
	var key string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		key = r.URL.Query().Get("apikey")
		mu.Unlock()
		_, _ = w.Write([]byte(newznabFeed))
	}))
	defer srv.Close()

	j, err := NewTorznab(Settings{ApiURL: srv.URL + "/torznab?apikey=USERKEY"})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	if _, err := j.Fetch(context.Background(), NewRawSearch().WithQuery("x").Build(), WithoutCapsValidation()); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if key != "USERKEY" {
		t.Errorf("apikey sent = %q, want the one from the pasted URL", key)
	}
}

// TestPastedLimitSurvives: limit has no builder, so owning it would strip
// the pasted value and replace it with nothing.
func TestPastedLimitSurvives(t *testing.T) {
	var mu sync.Mutex
	var limit string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		limit = r.URL.Query().Get("limit")
		mu.Unlock()
		_, _ = w.Write([]byte(newznabFeed))
	}))
	defer srv.Close()

	j, err := NewTorznab(Settings{ApiURL: srv.URL + "/torznab?limit=100"})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	if _, err := j.Fetch(context.Background(), NewRawSearch().WithQuery("x").Build(), WithoutCapsValidation()); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if limit != "100" {
		t.Errorf("limit sent = %q, want the pasted 100", limit)
	}
}

// TestEndpointModeChecksCapsForAnAllIndexersURL: in Jackett mode the "all"
// meta indexer has no caps to check and is skipped. A pasted Jackett
// all-indexers feed URL hits the same path name in endpoint mode, where the
// endpoint does answer t=caps — skipping it there would silently disable
// validation for the most common Jackett paste.
func TestEndpointModeChecksCapsForAnAllIndexersURL(t *testing.T) {
	var mu sync.Mutex
	var capsCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("t") == "caps" {
			mu.Lock()
			capsCalls++
			mu.Unlock()
			_, _ = w.Write([]byte(endpointCaps))
			return
		}
		_, _ = w.Write([]byte(newznabFeed))
	}))
	defer srv.Close()

	j, err := NewTorznab(Settings{ApiURL: srv.URL + "/api/v2.0/indexers/all/results/torznab"})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	if _, err := j.Fetch(context.Background(), NewRawSearch().WithQuery("x").Build()); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if capsCalls != 1 {
		t.Errorf("caps requests = %d, want 1 — validation must not be skipped in endpoint mode", capsCalls)
	}
}

func TestIndexerErrorIsTyped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Message in the element body rather than the description
		// attribute — some implementations do it this way.
		_, _ = w.Write([]byte(`<error code="100">Incorrect user credentials</error>`))
	}))
	defer srv.Close()

	j, err := NewTorznab(Settings{ApiURL: srv.URL})
	if err != nil {
		t.Fatalf("NewTorznab() error = %v", err)
	}
	_, err = j.Caps(context.Background(), "")
	if err == nil {
		t.Fatal("Caps() accepted a chardata-only error document")
	}
	var ie *IndexerError
	if !errors.As(err, &ie) {
		t.Fatalf("error = %v, want an *IndexerError callers can match on", err)
	}
	if ie.Code != "100" || !strings.Contains(ie.Description, "Incorrect user credentials") {
		t.Errorf("IndexerError = %+v, want code 100 and the message", ie)
	}
}
