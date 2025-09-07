package jackett

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const testAPIKey string = "abracadabra"

func TestGenerateURL(t *testing.T) {
	t.Parallel()

	j, srv := mockServer(t)
	defer srv.Close()

	tests := []struct {
		input *FetchRequest
		want  []string
	}{
		{NewRawSearch().Build(),
			[]string{"/api/v2.0/indexers/all/results/torznab?extended=1&t=search"}},
		{NewRawSearch().
			WithQuery("qqq").
			WithTrackers("aaa", "bbb").
			WithCategories(1, 2).
			Build(),
			[]string{
				"/api/v2.0/indexers/aaa/results/torznab?cat=1%2C2&extended=1&q=qqq&t=search",
				"/api/v2.0/indexers/bbb/results/torznab?cat=1%2C2&extended=1&q=qqq&t=search"}},
	}
	for _, test := range tests {
		got, err := j.generateFetchURLs(test.input)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(test.want) {
			t.Fatalf("expected %d urls, got %d", len(test.want), len(got))
		}
		for i := range test.want {
			if !strings.HasSuffix(got[i].String(), test.want[i]) {
				t.Errorf("strings.HasSuffix(generateURL(%+v), %q), want %q", test.input, got[i].String(), test.want[i])
			}
		}
	}
}

func TestDefaultTrackers(t *testing.T) {
	t.Parallel()

	j, srv := mockServer(t)
	defer srv.Close()

	// No default trackers and none defined on query
	urls, err := j.generateFetchURLs(NewRawSearch().Build())
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 1 {
		t.Fatalf("expected 1 url got %d", len(urls))
	}
	if !strings.Contains(urls[0].Path, "/all/") {
		t.Errorf("expected url to contain all tracker: %s", urls[0].Path)
	}

	// Default trackers but none defined on query
	j.cfg.DefaultTrackers = []string{"foo", "bar"}
	urls, err = j.generateFetchURLs(NewRawSearch().Build())
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 2 {
		t.Fatalf("expected 2 urls got %d", len(urls))
	}
	if !strings.Contains(urls[0].Path, "/foo/") {
		t.Errorf("expected url to contain foo tracker: %s", urls[0].Path)
	}
	if !strings.Contains(urls[1].Path, "/bar/") {
		t.Errorf("expected url to contain bar tracker: %s", urls[1].Path)
	}

	// Default trackers but query overrides
	j.cfg.DefaultTrackers = []string{"foo", "bar"}
	urls, err = j.generateFetchURLs(NewRawSearch().
		WithTrackers("baz").
		Build())
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 1 {
		t.Fatalf("expected 1 url got %d", len(urls))
	}
	if !strings.Contains(urls[0].Path, "/baz/") {
		t.Errorf("expected url to contain baz tracker: %s", urls[0].Path)
	}
}

func TestFetch(t *testing.T) {
	t.Parallel()

	j, srv := mockServer(t)
	defer srv.Close()

	input := NewRawSearch().Build()
	got, err := j.Fetch(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Errorf("len(Fetch(%+v).Results) = %v, want 4", input, len(got))
	}
}

func TestFetchProgress(t *testing.T) {
	t.Parallel()

	j, srv := mockServer(t)
	defer srv.Close()

	var trackers []string
	for i := range 100 {
		trackers = append(trackers, fmt.Sprintf("tracker-%d", i))
	}

	var gotComplete, gotTotal atomic.Uint32
	input := NewRawSearch().WithTrackers(trackers...).Build()
	got, err := j.Fetch(t.Context(), input, WithProgressReportFunc(func(complete, total uint) {
		gotComplete.Store(uint32(complete))
		gotTotal.Store(uint32(total))
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 100 {
		t.Errorf("len(Fetch(%+v).Results) = %v, want 100", input, len(got))
	}
	if gotComplete.Load() != gotTotal.Load() {
		t.Errorf("expected complete and total counts to be equal: %d != %d", gotComplete.Load(), gotTotal.Load())
	}
}

func TestListIndexers(t *testing.T) {
	t.Parallel()

	j, srv := mockServer(t)
	defer srv.Close()

	got, err := j.ListIndexers(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d indexers, expected 1", len(got))
	}
	if got[0].ID != "my-indexer" {
		t.Errorf("expected indexer to have id %q; got %q", "my-indexer", got[0].ID)
	}
}

func mockServer(t *testing.T) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2.0/indexers/all/results/torznab":
			if r.URL.Query().Get("t") == "indexers" {
				w.Write([]byte(exampleIndexerResponse))
				return
			}
			w.Write([]byte(exampleResponse))
		default:
			if r.URL.Query().Get("t") == "caps" {
				w.Write([]byte(exampleCapsResponse))
				return
			}
			tracker := extractTracker(r.URL.Path)
			if tracker == "" {
				// Not a known request pattern
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if tracker == "all" {
				t.Fatal("tracker 'all' should have been caught by an earlier case")
			}
			responseXML := fmt.Sprintf(exampleResponseTemplate, tracker)
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(responseXML))
			return
		}
	}))

	j, err := New(Settings{
		ApiURL: srv.URL,
		ApiKey: testAPIKey,
		Client: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return j, srv
}

const exampleResponse = `
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom" xmlns:torznab="http://torznab.com/schemas/2015/feed">
  <channel>
    <atom:link href="http://localhost:9117/" rel="self" type="application/rss+xml" />
    <title>Generic-Tracker (API)</title>
    <description>Generic tracker description</description>
    <link>https://generic-tracker.example/</link>
    <language>en-US</language>
    <category>search</category>
    <item>
      <title>Media S15E12 720p DSNP WEB-DL DDP 5.1 H.264-FLUX</title>
      <guid>https://generic-tracker.example/torrents/media-s15e12-720p-dsnp-web-dl-ddp-51-h264-flux.444921</guid>
      <jackettindexer id="generic-tracker-api">Generic-Tracker (API)</jackettindexer>
      <type>private</type>
      <comments>https://generic-tracker.example/torrents/media-s15e12-720p-dsnp-web-dl-ddp-51-h264-flux.444921</comments>
      <pubDate>Fri, 30 May 2025 22:50:38 +0000</pubDate>
      <size>398928401</size>
      <grabs>22</grabs>
      <description />
      <link>http://localhost:9117/dl/generic-tracker-api/?jackett_apikey=PLACEHOLDER_API_KEY&amp;path=PLACEHOLDER_ENCODED_PATH&amp;file=Media+S15E12+720p+DSNP+WEB-DL+DDP+5.1+H.264-FLUX</link>
      <category>5000</category>
      <category>100002</category>
      <enclosure url="http://localhost:9117/dl/generic-tracker-api/?jackett_apikey=PLACEHOLDER_API_KEY&amp;path=PLACEHOLDER_ENCODED_PATH&amp;file=Media+S15E12+720p+DSNP+WEB-DL+DDP+5.1+H.264-FLUX" length="398928401" type="application/x-bittorrent" />
      <torznab:attr name="category" value="5000" />
      <torznab:attr name="category" value="100002" />
      <torznab:attr name="imdb" value="1561755" />
      <torznab:attr name="imdbid" value="tt1561755" />
      <torznab:attr name="tmdbid" value="32726" />
      <torznab:attr name="seeders" value="20" />
      <torznab:attr name="peers" value="21" />
      <torznab:attr name="infohash" value="PLACEHOLDER_INFOHASH_1" />
      <torznab:attr name="downloadvolumefactor" value="1" />
      <torznab:attr name="uploadvolumefactor" value="1" />
    </item>
    <item>
      <title>Media S15E12 1080p DSNP WEB-DL DDP 5.1 H.264-FLUX</title>
      <guid>https://generic-tracker.example/torrents/media-s15e12-1080p-dsnp-web-dl-ddp-51-h264-flux.444918</guid>
      <jackettindexer id="generic-tracker-api">Generic-Tracker (API)</jackettindexer>
      <type>private</type>
      <comments>https://generic-tracker.example/torrents/media-s15e12-1080p-dsnp-web-dl-ddp-51-h264-flux.444918</comments>
      <pubDate>Fri, 30 May 2025 22:50:24 +0000</pubDate>
      <size>620177942</size>
      <grabs>61</grabs>
      <description />
      <link>http://localhost:9117/dl/generic-tracker-api/?jackett_apikey=PLACEHOLDER_API_KEY&amp;path=PLACEHOLDER_ENCODED_PATH&amp;file=Media+S15E12+1080p+DSNP+WEB-DL+DDP+5.1+H.264-FLUX</link>
      <category>5000</category>
      <category>100002</category>
      <enclosure url="http://localhost:9117/dl/generic-tracker-api/?jackett_apikey=PLACEHOLDER_API_KEY&amp;path=PLACEHOLDER_ENCODED_PATH&amp;file=Media+S15E12+1080p+DSNP+WEB-DL+DDP+5.1+H.264-FLUX" length="620177942" type="application/x-bittorrent" />
      <torznab:attr name="category" value="5000" />
      <torznab:attr name="category" value="100002" />
      <torznab:attr name="imdb" value="1561755" />
      <torznab:attr name="imdbid" value="tt1561755" />
      <torznab:attr name="tmdbid" value="32726" />
      <torznab:attr name="seeders" value="59" />
      <torznab:attr name="peers" value="60" />
      <torznab:attr name="infohash" value="PLACEHOLDER_INFOHASH_2" />
      <torznab:attr name="downloadvolumefactor" value="1" />
      <torznab:attr name="uploadvolumefactor" value="1" />
    </item>
    <item>
      <title>Media S15E12 1080p DSNP WEB-DL DDP 5.1 H.264-NTb</title>
      <guid>https://generic-tracker.example/torrents/media-s15e12-1080p-dsnp-web-dl-ddp-51-h264-ntb.444841</guid>
      <jackettindexer id="generic-tracker-api">Generic-Tracker (API)</jackettindexer>
      <type>private</type>
      <comments>https://generic-tracker.example/torrents/media-s15e12-1080p-dsnp-web-dl-ddp-51-h264-ntb.444841</comments>
      <pubDate>Fri, 30 May 2025 14:25:03 +0000</pubDate>
      <size>558112511</size>
      <grabs>97</grabs>
      <description />
      <link>http://localhost:9117/dl/generic-tracker-api/?jackett_apikey=PLACEHOLDER_API_KEY&amp;path=PLACEHOLDER_ENCODED_PATH&amp;file=Media+S15E12+1080p+DSNP+WEB-DL+DDP+5.1+H.264-NTb</link>
      <category>5000</category>
      <category>100002</category>
      <enclosure url="http://localhost:9117/dl/generic-tracker-api/?jackett_apikey=PLACEHOLDER_API_KEY&amp;path=PLACEHOLDER_ENCODED_PATH&amp;file=Media+S15E12+1080p+DSNP+WEB-DL+DDP+5.1+H.264-NTb" length="558112511" type="application/x-bittorrent" />
      <torznab:attr name="category" value="5000" />
      <torznab:attr name="category" value="100002" />
      <torznab:attr name="imdb" value="1561755" />
      <torznab:attr name="imdbid" value="tt1561755" />
      <torznab:attr name="tmdbid" value="32726" />
      <torznab:attr name="seeders" value="129" />
      <torznab:attr name="peers" value="129" />
      <torznab:attr name="infohash" value="PLACEHOLDER_INFOHASH_3" />
      <torznab:attr name="downloadvolumefactor" value="1" />
      <torznab:attr name="uploadvolumefactor" value="1" />
    </item>
    <item>
      <title>Media S15E12 1080p WEB-DL DDP 5.1 H.264-ETHEL</title>
      <guid>https://generic-tracker.example/torrents/media-s15e12-1080p-web-dl-ddp-51-h264-ethel.444752</guid>
      <jackettindexer id="generic-tracker-api">Generic-Tracker (API)</jackettindexer>
      <type>private</type>
      <comments>https://generic-tracker.example/torrents/media-s15e12-1080p-web-dl-ddp-51-h264-ethel.444752</comments>
      <pubDate>Fri, 30 May 2025 07:03:54 +0000</pubDate>
      <size>541779433</size>
      <grabs>200</grabs>
      <description />
      <link>http://localhost:9117/dl/generic-tracker-api/?jackett_apikey=PLACEHOLDER_API_KEY&amp;path=PLACEHOLDER_ENCODED_PATH&amp;file=Media+S15E12+1080p+WEB-DL+DDP+5.1+H.264-ETHEL</link>
      <category>5000</category>
      <category>100002</category>
      <enclosure url="http://localhost:9117/dl/generic-tracker-api/?jackett_apikey=PLACEHOLDER_API_KEY&amp;path=PLACEHOLDER_ENCODED_PATH&amp;file=Media+S15E12+1080p+WEB-DL+DDP+5.1+H.264-ETHEL" length="541779433" type="application/x-bittorrent" />
      <torznab:attr name="category" value="5000" />
      <torznab:attr name="category" value="100002" />
      <torznab:attr name="imdb" value="1561755" />
      <torznab:attr name="imdbid" value="tt1561755" />
      <torznab:attr name="tmdbid" value="32726" />
      <torznab:attr name="seeders" value="207" />
      <torznab:attr name="peers" value="207" />
      <torznab:attr name="infohash" value="PLACEHOLDER_INFOHASH_4" />
      <torznab:attr name="downloadvolumefactor" value="1" />
      <torznab:attr name="uploadvolumefactor" value="1" />
    </item>
  </channel>
</rss>`

const exampleResponseTemplate = `
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom" xmlns:torznab="http://torznab.com/schemas/2015/feed">
  <channel>
    <atom:link href="http://localhost:9117/" rel="self" type="application/rss+xml" />
    <title>%[1]s</title>
    <language>en-US</language>
    <category>search</category>
    <item>
      <title>A result</title>
      <guid>some-guid-string</guid>
      <jackettindexer id="%[1]s">%[1]s</jackettindexer>
      <type>private</type>
      <pubDate>Fri, 30 May 2025 22:50:38 +0000</pubDate>
      <size>39891245</size>
      <grabs>22</grabs>
      <description />
      <link>http://localhost:9117/dl/%[1]s</link>
      <category>5000</category>
      <category>100002</category>
      <enclosure url="http://localhost:9117/dl/%[1]s" length="39891245" type="application/x-bittorrent" />
      <torznab:attr name="category" value="5000" />
      <torznab:attr name="category" value="100002" />
      <torznab:attr name="imdb" value="12345" />
      <torznab:attr name="imdbid" value="tt1234" />
      <torznab:attr name="tmdbid" value="12345" />
      <torznab:attr name="seeders" value="20" />
      <torznab:attr name="peers" value="21" />
      <torznab:attr name="infohash" value="4141640e8f08d6f04df8071aa3c585e26e6328de" />
      <torznab:attr name="downloadvolumefactor" value="1" />
      <torznab:attr name="uploadvolumefactor" value="1" />
    </item>
  </channel>
</rss>`

const exampleIndexerResponse = `
<?xml version="1.0" encoding="UTF-8"?>
<indexers>
  <indexer id="my-indexer" configured="true">
    <title>My Indexer</title>
    <description>This is a fake indexer</description>
    <link>https://example.com</link>
    <language>en-US</language>
    <type>private</type>
    <caps>
      <server title="Jackett" />
      <limits default="100" max="100" />
      <searching>
        <search available="yes" supportedParams="q" />
        <tv-search available="yes" supportedParams="q,season,ep,tmdbid" />
        <movie-search available="yes" supportedParams="q,imdbid,tmdbid" />
        <music-search available="yes" supportedParams="q" />
        <audio-search available="yes" supportedParams="q" />
        <book-search available="yes" supportedParams="q" />
      </searching>
      <categories>
        <category id="2000" name="Movies" />
        <category id="5000" name="TV" />
        <category id="100001" name="Movies" />
        <category id="100002" name="TV" />
      </categories>
    </caps>
  </indexer>
</indexers>`

const exampleCapsResponse = `
<?xml version="1.0" encoding="UTF-8"?>
<caps>
  <server title="Jackett" />
  <limits default="100" max="100" />
  <searching>
    <search available="yes" supportedParams="q" />
    <tv-search available="yes" supportedParams="q,season,ep,tmdbid" />
    <movie-search available="yes" supportedParams="q,imdbid,tmdbid" />
    <music-search available="yes" supportedParams="q" />
    <audio-search available="yes" supportedParams="q" />
    <book-search available="yes" supportedParams="q" />
  </searching>
  <categories>
    <category id="2000" name="Movies" />
    <category id="5000" name="TV" />
    <category id="100001" name="Movies" />
    <category id="100002" name="TV" />
  </categories>
</caps>`
