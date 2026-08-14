package jackett

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Result represents a single torrent or release item returned by Jackett.
// Fields are based on the Torznab API response structure.
// Note that most attributes are optional so not all Trackers will provide values for all fields.
// Ref: https://torznab.github.io/spec-1.3-draft/torznab/Specification-v1.3.html
type Result struct {
	// Tracker

	ID                   string        `json:"id,omitempty"`
	InfoHash             string        `json:"info_hash,omitempty"`
	Tracker              string        `json:"tracker,omitempty"`
	TrackerID            string        `json:"tracker_id,omitempty"`
	TrackerType          string        `json:"tracker_type,omitempty"`
	Grabs                uint          `json:"grabs,omitempty"`
	Peers                uint          `json:"peers,omitempty"`
	Seeders              uint          `json:"seeders,omitempty"`
	DownloadVolumeFactor float32       `json:"download_volume_factor,omitempty"`
	UploadVolumeFactor   float32       `json:"upload_volume_factor,omitempty"`
	MinimumRatio         float32       `json:"minimum_ratio,omitempty"`
	MinimumSeedTime      time.Duration `json:"minimum_seed_time,omitempty"`
	Tags                 []string      `json:"tags,omitempty"`

	// Links and metadata

	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Categories  []uint    `json:"categories,omitempty"`
	CoverURL    string    `json:"cover_url,omitempty"`
	BackdropURL string    `json:"backdrop_url,omitempty"`
	Link        string    `json:"link,omitempty"`
	MagnetURI   string    `json:"magnet_uri,omitempty"`
	Size        uint64    `json:"size,omitempty"`
	Files       uint      `json:"files,omitempty"`
	PublishDate time.Time `json:"publish_date"`

	// TV / Mov

	IMDBID    string   `json:"imdb_id,omitempty"`
	TVDBID    string   `json:"tvdb_id,omitempty"`
	TMDBID    uint     `json:"tmdb_id,omitempty"`
	TraktID   uint     `json:"trakt_id,omitempty"`
	DoubanID  uint     `json:"douban_id,omitempty"`
	TVMazeID  uint     `json:"tv_maze_id,omitempty"`
	Season    uint     `json:"season,omitempty"`
	Episode   uint     `json:"episode,omitempty"`
	Languages []string `json:"languages,omitempty"`
	Subs      []string `json:"subs,omitempty"`
	Genres    []string `json:"genres,omitempty"`

	// Music

	Artist    string   `json:"artist,omitempty"`
	Album     string   `json:"album,omitempty"`
	Publisher string   `json:"publisher,omitempty"`
	Tracks    []string `json:"tracks,omitempty"`

	// Books

	BookTitle string `json:"book_title,omitempty"`
	Author    string `json:"author,omitempty"`
	Pages     uint   `json:"pages,omitempty"`
}

type searchResp struct {
	// XMLName pins the root element. Without it any XML document — an
	// HTML error page, a JSON-ish blob, someone else's API — unmarshals
	// into an empty feed, and the caller cannot tell "not a feed" from
	// "no results".
	XMLName   xml.Name `xml:"rss"`
	Version   string   `xml:"version,attr"`
	ErrorCode int      `xml:"code,attr"`
	ErrorDesc string   `xml:"description,attr"`
	Channel   struct {
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
	} `xml:"channel"`
}

func (sr *searchResp) Unmarshal() ([]Result, error) {
	if sr.ErrorCode > 0 || sr.ErrorDesc != "" {
		return nil, fmt.Errorf("response indicates an error occurred: %d: %s",
			sr.ErrorCode, sr.ErrorDesc)
	}
	var results []Result
	for i, item := range sr.Channel.Items {
		r, err := item.Unmarshal()
		if err != nil {
			return nil, fmt.Errorf("unmarshal response item: %d: %w", i, err)
		}
		results = append(results, r)
	}
	return results, nil
}

type searchItem struct {
	Title          string `xml:"title"`
	GUID           string `xml:"guid"`
	JackettIndexer struct {
		ID   string `xml:"id,attr"`
		Name string `xml:",chardata"`
	} `xml:"jackettindexer"`
	Tags        []string `xml:"tag"`
	Type        string   `xml:"type"`
	Comments    string   `xml:"comments"`
	PubDate     string   `xml:"pubDate"`
	Size        uint64   `xml:"size"`
	Grabs       uint     `xml:"grabs"`
	Files       uint     `xml:"files"`
	Description string   `xml:"description"`
	Link        string   `xml:"link"`
	Categories  []uint   `xml:"category"`
	Enclosure   struct {
		URL    string `xml:"url,attr"`
		Length int64  `xml:"length,attr"`
		Type   string `xml:"type,attr"`
	} `xml:"enclosure"`
	// Namespace is deliberately left off: the same feed is served as
	// <torznab:attr> by Jackett and Prowlarr and as <newznab:attr> by
	// indexers exposing the Newznab flavour, and binding to the Torznab
	// namespace alone drops every attribute — seeders and infohash
	// included — on the latter.
	TorznabAttrs []struct {
		Name  string `xml:"name,attr"`
		Value string `xml:"value,attr"`
	} `xml:"attr"`
}

func (i *searchItem) Unmarshal() (Result, error) {
	var r Result
	r.ID = i.GUID
	r.InfoHash = i.tAttr("infohash")
	r.Tracker = i.JackettIndexer.Name
	r.TrackerID = i.JackettIndexer.ID
	r.TrackerType = i.Type
	r.Grabs = i.Grabs
	r.Peers = try(parseInt[uint](i.tAttr("peers")))
	r.Seeders = try(parseInt[uint](i.tAttr("seeders")))
	r.DownloadVolumeFactor = try(parseFloat(i.tAttr("downloadvolumefactor")))
	r.UploadVolumeFactor = try(parseFloat(i.tAttr("uploadvolumefactor")))
	r.MinimumRatio = try(parseFloat(i.tAttr("minimumratio")))
	r.MinimumSeedTime = time.Second *
		time.Duration(try(parseInt[uint](i.tAttr("minimumseedtime"))))
	r.Tags = i.Tags
	r.Title = i.Title
	r.Description = i.Description
	r.Categories = i.Categories
	r.CoverURL = i.tAttr("coverurl")
	r.BackdropURL = i.tAttr("backdropurl")
	r.Link = i.Link
	r.MagnetURI = i.tAttr("magneturl")
	if r.MagnetURI == "" {
		// Public trackers routinely ship the magnet as the item link or
		// the enclosure url instead of as an attribute.
		r.MagnetURI = firstMagnet(i.Link, i.Enclosure.URL)
	}
	r.Size = i.Size
	if r.Size == 0 && i.Enclosure.Length > 0 {
		r.Size = uint64(i.Enclosure.Length)
	}
	if r.Link == "" {
		r.Link = i.Enclosure.URL
	}
	r.Files = i.Files
	r.IMDBID = i.tAttr("imdbid")
	r.TVDBID = i.tAttr("tvdbid")
	r.TMDBID = try(parseInt[uint](i.tAttr("tmdbid")))
	r.TraktID = try(parseInt[uint](i.tAttr("tracktid")))
	r.DoubanID = try(parseInt[uint](i.tAttr("doubanid")))
	r.TVMazeID = try(parseInt[uint](i.tAttr("tvmazeid")))
	r.Season = try(parseInt[uint](i.tAttr("season")))
	r.Episode = try(parseInt[uint](i.tAttr("episode")))
	r.Languages = strings.Split(i.tAttr("language"), ",")
	r.Subs = strings.Split(i.tAttr("subs"), ",")
	r.Genres = strings.Split(i.tAttr("genres"), ",")
	r.Artist = i.tAttr("artist")
	r.Album = i.tAttr("album")
	r.Publisher = i.tAttr("publisher")
	r.Tracks = strings.Split(i.tAttr("tracks"), "|")
	r.BookTitle = i.tAttr("booktitle")
	r.PublishDate = try(i.parsePubDate())
	r.Author = i.tAttr("author")
	r.Pages = try(parseInt[uint](i.tAttr("pages")))
	return r, nil
}

func (i *searchItem) tAttr(name string) string {
	for _, attr := range i.TorznabAttrs {
		if strings.EqualFold(attr.Name, name) {
			return attr.Value
		}
	}
	return ""
}

// pubDateLayouts covers the spellings seen in the wild. The spec says
// RFC-822, but feeds disagree on whether the zone is numeric or named and
// a few emit RFC-3339 outright.
var pubDateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
	time.RFC822,
	time.RFC3339,
}

func (i *searchItem) parsePubDate() (time.Time, error) {
	var err error
	for _, layout := range pubDateLayouts {
		var t time.Time
		if t, err = time.Parse(layout, i.PubDate); err == nil {
			return t, nil
		}
	}
	return time.Time{}, err
}

// firstMagnet returns the first candidate that is a magnet URI.
func firstMagnet(candidates ...string) string {
	for _, c := range candidates {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(c)), "magnet:") {
			return c
		}
	}
	return ""
}

type indexersResp struct {
	XMLName  xml.Name         `xml:"indexers"`
	Indexers []IndexerDetails `xml:"indexer"`
}

// IndexerDetails represents detailed information about a configured indexer.
type IndexerDetails struct {
	ID          string      `xml:"id,attr"`
	Configured  string      `xml:"configured,attr"`
	Title       string      `xml:"title"`
	Description string      `xml:"description"`
	Link        string      `xml:"link"`
	Language    string      `xml:"language"`
	Type        string      `xml:"type"`
	Caps        IndexerCaps `xml:"caps"`
}

// IndexerCaps represents the capabilities of a configured indexer.
type IndexerCaps struct {
	Server struct {
		Title string `xml:"title,attr"`
	} `xml:"server"`
	Limits struct {
		Default string `xml:"default,attr"`
		Max     string `xml:"max,attr"`
	} `xml:"limits"`
	Searching struct {
		Search struct {
			Available       string `xml:"available,attr"`
			SupportedParams string `xml:"supportedParams,attr"`
			SearchEngine    string `xml:"searchEngine,attr,omitempty"`
		} `xml:"search"`
		TVSearch struct {
			Available       string `xml:"available,attr"`
			SupportedParams string `xml:"supportedParams,attr"`
			SearchEngine    string `xml:"searchEngine,attr,omitempty"`
		} `xml:"tv-search"`
		MovieSearch struct {
			Available       string `xml:"available,attr"`
			SupportedParams string `xml:"supportedParams,attr"`
			SearchEngine    string `xml:"searchEngine,attr,omitempty"`
		} `xml:"movie-search"`
		MusicSearch struct {
			Available       string `xml:"available,attr"`
			SupportedParams string `xml:"supportedParams,attr"`
			SearchEngine    string `xml:"searchEngine,attr,omitempty"`
		} `xml:"music-search"`
		AudioSearch struct {
			Available       string `xml:"available,attr"`
			SupportedParams string `xml:"supportedParams,attr"`
			SearchEngine    string `xml:"searchEngine,attr,omitempty"`
		} `xml:"audio-search"`
		BookSearch struct {
			Available       string `xml:"available,attr"`
			SupportedParams string `xml:"supportedParams,attr"`
			SearchEngine    string `xml:"searchEngine,attr,omitempty"`
		} `xml:"book-search"`
	} `xml:"searching"`
	Categories struct {
		Categories []struct {
			ID      string `xml:"id,attr"`
			Name    string `xml:"name,attr"`
			Subcats []struct {
				ID   string `xml:"id,attr"`
				Name string `xml:"name,attr"`
			} `xml:"subcat"`
		} `xml:"category"`
	} `xml:"categories"`
}

func (c IndexerCaps) Validate(q url.Values) error {
	var params []string
	var supported bool
	t := q.Get("t")
	switch t {
	case "search":
		supported = strings.EqualFold(c.Searching.Search.Available, "yes")
		params = strings.Split(c.Searching.Search.SupportedParams, ",")
	case "movie":
		supported = strings.EqualFold(c.Searching.MovieSearch.Available, "yes")
		params = strings.Split(c.Searching.MovieSearch.SupportedParams, ",")
	case "tvsearch":
		supported = strings.EqualFold(c.Searching.TVSearch.Available, "yes")
		params = strings.Split(c.Searching.TVSearch.SupportedParams, ",")
	case "music":
		supported = strings.EqualFold(c.Searching.MusicSearch.Available, "yes")
		params = strings.Split(c.Searching.MusicSearch.SupportedParams, ",")
	case "book":
		supported = strings.EqualFold(c.Searching.BookSearch.Available, "yes")
		params = strings.Split(c.Searching.BookSearch.SupportedParams, ",")
	}
	if !supported {
		return newUnsupportedError(c,
			"search type %q not supported", t)
	}

	supportedParams := map[string]struct{}{
		// Some params are assumed to be supported by all indexers
		"extended": {},
		"t":        {},
		"cat":      {},
	}
	for _, p := range params {
		supportedParams[p] = struct{}{}
	}

	for k := range q {
		if _, ok := supportedParams[k]; !ok {
			return newUnsupportedError(c,
				"parameter %q is not supported; allowed: %v", k, params)
		}
	}

	return nil
}

type UnsupportedError struct {
	Caps    IndexerCaps
	Message string
}

func newUnsupportedError(c IndexerCaps, msg string, args ...any) *UnsupportedError {
	c.Categories.Categories = nil // elided for brevity
	return &UnsupportedError{
		Caps:    c,
		Message: fmt.Sprintf(msg, args...),
	}
}

func (m *UnsupportedError) Is(err error) bool {
	_, ok := err.(*UnsupportedError)
	return ok
}

func (m *UnsupportedError) Error() string {
	b, _ := json.Marshal(m.Caps)
	return m.Message + ": " + string(b)
}

func try[T any](v T, err error) T {
	var zero T
	if err != nil {
		return zero
	}
	return v
}

func parseFloat(s string) (float32, error) {
	n, err := strconv.ParseFloat(s, 32)
	return float32(n), err
}

func parseInt[T int | int32 | int64 | uint | uint32 | uint64](s string) (T, error) {
	var (
		n        T
		err      error
		i64      int64
		sentinel any = n
	)
	switch sentinel.(type) {
	case int, int64, uint, uint64:
		i64, err = strconv.ParseInt(s, 0, 64)
	case int32, uint32:
		i64, err = strconv.ParseInt(s, 0, 32)
	}
	if err != nil {
		return n, fmt.Errorf("parse %q as %T: %w", s, n, err)
	}
	return T(i64), nil

}
