package jackett

import (
	"encoding/xml"
	"reflect"
	"testing"
	"time"
)

func TestSearchResp_Unmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       searchResp
		expectError bool
		expectCount int
	}{
		{
			name: "successful response",
			input: searchResp{
				ErrorCode: 0,
				ErrorDesc: "",
				Channel: struct {
					AtomLink struct {
						Href string `xml:"href,attr"`
						Rel  string `xml:"rel,attr"`
						Type string `xml:"type,attr"`
					} `xml:"http://www.w3.org/2005/Atom link"`
					Title       string       `xml:"title"`
					Description string       `xml:"description"`
					Link        string       `xml:"link"`
					Language    string       `xml:"language"`
					Category    string       `xml:"category"`
					Items       []searchItem `xml:"item"`
				}{
					Items: []searchItem{
						{
							Title: "Test Item 1",
							GUID:  "guid-1",
						},
						{
							Title: "Test Item 2",
							GUID:  "guid-2",
						},
					},
				},
			},
			expectError: false,
			expectCount: 2,
		},
		{
			name: "error response with code",
			input: searchResp{
				ErrorCode: 100,
				ErrorDesc: "API limit exceeded",
			},
			expectError: true,
			expectCount: 0,
		},
		{
			name: "error response with description only",
			input: searchResp{
				ErrorCode: 0,
				ErrorDesc: "Something went wrong",
			},
			expectError: true,
			expectCount: 0,
		},
		{
			name: "empty response",
			input: searchResp{
				ErrorCode: 0,
				ErrorDesc: "",
				Channel: struct {
					AtomLink struct {
						Href string `xml:"href,attr"`
						Rel  string `xml:"rel,attr"`
						Type string `xml:"type,attr"`
					} `xml:"http://www.w3.org/2005/Atom link"`
					Title       string       `xml:"title"`
					Description string       `xml:"description"`
					Link        string       `xml:"link"`
					Language    string       `xml:"language"`
					Category    string       `xml:"category"`
					Items       []searchItem `xml:"item"`
				}{
					Items: []searchItem{},
				},
			},
			expectError: false,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := tt.input.Unmarshal()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(results) != tt.expectCount {
				t.Errorf("Expected %d results, got %d", tt.expectCount, len(results))
			}
		})
	}
}

func TestSearchItem_Unmarshal(t *testing.T) {
	t.Parallel()

	item := searchItem{
		Title: "Test Movie 2023 1080p BluRay x264",
		GUID:  "test-guid-123",
		JackettIndexer: struct {
			ID   string `xml:"id,attr"`
			Name string `xml:",chardata"`
		}{
			ID:   "test-tracker-id",
			Name: "Test Tracker",
		},
		Tags:        []string{"freeleech", "internal"},
		Type:        "private",
		Comments:    "https://example.com/comments",
		PubDate:     "Mon, 02 Jan 2006 15:04:05 MST",
		Size:        1024000000,
		Grabs:       42,
		Files:       1,
		Description: "Test description",
		Link:        "https://example.com/download",
		Categories:  []uint{2000, 2010},
		Enclosure: struct {
			URL    string `xml:"url,attr"`
			Length int64  `xml:"length,attr"`
			Type   string `xml:"type,attr"`
		}{
			URL:    "https://example.com/torrent",
			Length: 1024000000,
			Type:   "application/x-bittorrent",
		},
		TorznabAttrs: []struct {
			Name  string `xml:"name,attr"`
			Value string `xml:"value,attr"`
		}{
			{Name: "infohash", Value: "abcd1234efgh5678"},
			{Name: "seeders", Value: "10"},
			{Name: "peers", Value: "5"},
			{Name: "downloadvolumefactor", Value: "0.5"},
			{Name: "uploadvolumefactor", Value: "2.0"},
			{Name: "minimumratio", Value: "1.0"},
			{Name: "minimumseedtime", Value: "86400"},
			{Name: "coverurl", Value: "https://example.com/cover.jpg"},
			{Name: "backdropurl", Value: "https://example.com/backdrop.jpg"},
			{Name: "magneturl", Value: "magnet:?xt=urn:btih:abcd1234"},
			{Name: "imdbid", Value: "tt1234567"},
			{Name: "tvdbid", Value: "12345"},
			{Name: "tmdbid", Value: "67890"},
			{Name: "tracktid", Value: "11111"},
			{Name: "doubanid", Value: "22222"},
			{Name: "tvmazeid", Value: "33333"},
			{Name: "season", Value: "1"},
			{Name: "episode", Value: "5"},
			{Name: "language", Value: "en,es"},
			{Name: "subs", Value: "en,fr"},
			{Name: "genres", Value: "Action,Drama"},
			{Name: "artist", Value: "Test Artist"},
			{Name: "album", Value: "Test Album"},
			{Name: "publisher", Value: "Test Publisher"},
			{Name: "tracks", Value: "Track 1|Track 2"},
			{Name: "booktitle", Value: "Test Book"},
			{Name: "author", Value: "Test Author"},
			{Name: "pages", Value: "300"},
		},
	}

	result, err := item.Unmarshal()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Test basic fields
	if result.ID != "test-guid-123" {
		t.Errorf("Expected ID %q, got %q", "test-guid-123", result.ID)
	}
	if result.InfoHash != "abcd1234efgh5678" {
		t.Errorf("Expected InfoHash %q, got %q", "abcd1234efgh5678", result.InfoHash)
	}
	if result.Tracker != "Test Tracker" {
		t.Errorf("Expected Tracker %q, got %q", "Test Tracker", result.Tracker)
	}
	if result.TrackerID != "test-tracker-id" {
		t.Errorf("Expected TrackerID %q, got %q", "test-tracker-id", result.TrackerID)
	}
	if result.TrackerType != "private" {
		t.Errorf("Expected TrackerType %q, got %q", "private", result.TrackerType)
	}

	// Test numeric fields
	if result.Grabs != 42 {
		t.Errorf("Expected Grabs %d, got %d", 42, result.Grabs)
	}
	if result.Peers != 5 {
		t.Errorf("Expected Peers %d, got %d", 5, result.Peers)
	}
	if result.Seeders != 10 {
		t.Errorf("Expected Seeders %d, got %d", 10, result.Seeders)
	}
	if result.DownloadVolumeFactor != 0.5 {
		t.Errorf("Expected DownloadVolumeFactor %f, got %f", 0.5, result.DownloadVolumeFactor)
	}
	if result.UploadVolumeFactor != 2.0 {
		t.Errorf("Expected UploadVolumeFactor %f, got %f", 2.0, result.UploadVolumeFactor)
	}
	if result.MinimumRatio != 1.0 {
		t.Errorf("Expected MinimumRatio %f, got %f", 1.0, result.MinimumRatio)
	}
	if result.MinimumSeedTime != time.Hour*24 {
		t.Errorf("Expected MinimumSeedTime %v, got %v", time.Hour*24, result.MinimumSeedTime)
	}

	// Test slice fields
	expectedTags := []string{"freeleech", "internal"}
	if !reflect.DeepEqual(result.Tags, expectedTags) {
		t.Errorf("Expected Tags %v, got %v", expectedTags, result.Tags)
	}

	expectedLanguages := []string{"en", "es"}
	if !reflect.DeepEqual(result.Languages, expectedLanguages) {
		t.Errorf("Expected Languages %v, got %v", expectedLanguages, result.Languages)
	}

	expectedTracks := []string{"Track 1", "Track 2"}
	if !reflect.DeepEqual(result.Tracks, expectedTracks) {
		t.Errorf("Expected Tracks %v, got %v", expectedTracks, result.Tracks)
	}

	// Test ID fields
	if result.IMDBID != "tt1234567" {
		t.Errorf("Expected IMDBID %q, got %q", "tt1234567", result.IMDBID)
	}
	if result.TMDBID != 67890 {
		t.Errorf("Expected TMDBID %d, got %d", 67890, result.TMDBID)
	}
	if result.Season != 1 {
		t.Errorf("Expected Season %d, got %d", 1, result.Season)
	}
	if result.Episode != 5 {
		t.Errorf("Expected Episode %d, got %d", 5, result.Episode)
	}
}

func TestSearchItem_tAttr(t *testing.T) {
	t.Parallel()

	item := searchItem{
		TorznabAttrs: []struct {
			Name  string `xml:"name,attr"`
			Value string `xml:"value,attr"`
		}{
			{Name: "seeders", Value: "10"},
			{Name: "PEERS", Value: "5"}, // Test case insensitive
			{Name: "downloadVolumeFactor", Value: "0.5"},
		},
	}

	tests := []struct {
		name     string
		attrName string
		expected string
	}{
		{
			name:     "exact match",
			attrName: "seeders",
			expected: "10",
		},
		{
			name:     "case insensitive match",
			attrName: "peers",
			expected: "5",
		},
		{
			name:     "camelCase match",
			attrName: "downloadvolumefactor",
			expected: "0.5",
		},
		{
			name:     "not found",
			attrName: "nonexistent",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := item.tAttr(tt.attrName)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSearchItem_parsePubDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		pubDate     string
		expectError bool
		expected    time.Time
	}{
		{
			name:        "valid RFC1123Z format",
			pubDate:     "Mon, 02 Jan 2006 15:04:05 -0700",
			expectError: false,
			expected:    time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("MST", -7*3600)),
		},
		{
			name:        "valid RFC1123Z with UTC",
			pubDate:     "Fri, 30 May 2025 22:50:38 +0000",
			expectError: false,
			expected:    time.Date(2025, 5, 30, 22, 50, 38, 0, time.UTC),
		},
		{
			name:        "invalid format",
			pubDate:     "2006-01-02 15:04:05",
			expectError: true,
			expected:    time.Time{},
		},
		{
			name:        "empty string",
			pubDate:     "",
			expectError: true,
			expected:    time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := searchItem{PubDate: tt.pubDate}
			result, err := item.parsePubDate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !result.Equal(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIndexersResp_XMLUnmarshaling(t *testing.T) {
	t.Parallel()

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<indexers>
  <indexer id="test-indexer" configured="true">
    <title>Test Indexer</title>
    <description>A test indexer</description>
    <link>https://example.com</link>
    <language>en-US</language>
    <type>private</type>
    <caps>
      <server title="Jackett" />
      <limits default="100" max="100" />
      <searching>
        <search available="yes" supportedParams="q" />
        <tv-search available="yes" supportedParams="q,season,ep" />
        <movie-search available="no" supportedParams="" />
      </searching>
      <categories>
        <category id="2000" name="Movies">
          <subcat id="2010" name="Movies/Foreign" />
        </category>
        <category id="5000" name="TV" />
      </categories>
    </caps>
  </indexer>
</indexers>`

	var resp indexersResp
	err := xml.Unmarshal([]byte(xmlData), &resp)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	if len(resp.Indexers) != 1 {
		t.Fatalf("Expected 1 indexer, got %d", len(resp.Indexers))
	}

	indexer := resp.Indexers[0]
	if indexer.ID != "test-indexer" {
		t.Errorf("Expected ID %q, got %q", "test-indexer", indexer.ID)
	}
	if indexer.Configured != "true" {
		t.Errorf("Expected Configured %q, got %q", "true", indexer.Configured)
	}
	if indexer.Title != "Test Indexer" {
		t.Errorf("Expected Title %q, got %q", "Test Indexer", indexer.Title)
	}
	if indexer.Type != "private" {
		t.Errorf("Expected Type %q, got %q", "private", indexer.Type)
	}

	// Test caps structure
	if indexer.Caps.Server.Title != "Jackett" {
		t.Errorf("Expected Server Title %q, got %q", "Jackett", indexer.Caps.Server.Title)
	}
	if indexer.Caps.Limits.Default != "100" {
		t.Errorf("Expected Limits Default %q, got %q", "100", indexer.Caps.Limits.Default)
	}
	if indexer.Caps.Searching.Search.Available != "yes" {
		t.Errorf("Expected Search Available %q, got %q", "yes", indexer.Caps.Searching.Search.Available)
	}
	if indexer.Caps.Searching.MovieSearch.Available != "no" {
		t.Errorf("Expected MovieSearch Available %q, got %q", "no", indexer.Caps.Searching.MovieSearch.Available)
	}

	// Test categories
	if len(indexer.Caps.Categories.Categories) != 2 {
		t.Fatalf("Expected 2 categories, got %d", len(indexer.Caps.Categories.Categories))
	}
	if indexer.Caps.Categories.Categories[0].ID != "2000" {
		t.Errorf("Expected Category ID %q, got %q", "2000", indexer.Caps.Categories.Categories[0].ID)
	}
	if len(indexer.Caps.Categories.Categories[0].Subcats) != 1 {
		t.Errorf("Expected 1 subcat, got %d", len(indexer.Caps.Categories.Categories[0].Subcats))
	}
}

func TestSearchRespXMLUnmarshaling(t *testing.T) {
	t.Parallel()

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom" xmlns:torznab="http://torznab.com/schemas/2015/feed">
  <channel>
    <atom:link href="http://localhost:9117/" rel="self" type="application/rss+xml" />
    <title>Test Tracker</title>
    <description>Test description</description>
    <link>https://example.com</link>
    <language>en-US</language>
    <category>search</category>
    <item>
      <title>Test Movie 2023</title>
      <guid>test-guid</guid>
      <jackettindexer id="test-tracker">Test Tracker</jackettindexer>
      <type>private</type>
      <pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
      <size>1024000000</size>
      <grabs>42</grabs>
      <files>1</files>
      <description>Test description</description>
      <link>https://example.com/download</link>
      <category>2000</category>
      <torznab:attr name="seeders" value="10" />
      <torznab:attr name="peers" value="5" />
    </item>
  </channel>
</rss>`

	var resp searchResp
	err := xml.Unmarshal([]byte(xmlData), &resp)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	if resp.Version != "2.0" {
		t.Errorf("Expected Version %q, got %q", "2.0", resp.Version)
	}
	if resp.Channel.Title != "Test Tracker" {
		t.Errorf("Expected Channel Title %q, got %q", "Test Tracker", resp.Channel.Title)
	}
	if len(resp.Channel.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(resp.Channel.Items))
	}

	item := resp.Channel.Items[0]
	if item.Title != "Test Movie 2023" {
		t.Errorf("Expected Item Title %q, got %q", "Test Movie 2023", item.Title)
	}
	if item.Size != 1024000000 {
		t.Errorf("Expected Item Size %d, got %d", 1024000000, item.Size)
	}
}
