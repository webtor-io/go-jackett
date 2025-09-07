package jackett

import (
	"fmt"
	"net/url"

	"github.com/google/go-querystring/query"
)

// FetchRequest represents a fetch or search request.
// Only one field should be provided depending on your search needs.
// You may build this struct by hand or use the provided builder functions.
type FetchRequest struct {
	Raw   *RawSearch
	Movie *MovieSearch
	TV    *TVSearch
	Music *MusicSearch
	Book  *BookSearch
}

// Validate returns an error if the FetchRequest is not valid.
func (fr *FetchRequest) Validate() error {
	if fr == nil {
		return fmt.Errorf("empty request")
	}
	var n int
	if fr.Raw != nil {
		n++
	}
	if fr.Movie != nil {
		n++
	}
	if fr.TV != nil {
		n++
	}
	if fr.Music != nil {
		n++
	}
	if fr.Book != nil {
		n++
	}
	if n > 1 {
		return fmt.Errorf("only one search type may be used at a time, but found %d", n)
	}
	if n < 1 {
		return fmt.Errorf("at least one search type must be specified")
	}
	return nil
}

func (fr *FetchRequest) Values() (url.Values, error) {
	if err := fr.Validate(); err != nil {
		return nil, err
	}
	var search string
	var err error
	var v url.Values
	if fr.Raw != nil {
		search = "search"
		v, err = query.Values(fr.Raw)
	}
	if fr.Movie != nil {
		search = "movie"
		v, err = query.Values(fr.Movie)
	}
	if fr.TV != nil {
		search = "tvsearch"
		v, err = query.Values(fr.TV)
	}
	if fr.Music != nil {
		search = "music"
		v, err = query.Values(fr.Music)
	}
	if fr.Book != nil {
		search = "book"
		v, err = query.Values(fr.Book)
	}
	if err != nil {
		return nil, err
	}

	v.Set("t", search)
	v.Set("extended", "1")
	return v, nil
}

func (fr *FetchRequest) Trackers() []string {
	if fr == nil {
		return nil
	}
	if fr.Raw != nil {
		return fr.Raw.Trackers
	}
	if fr.Movie != nil {
		return fr.Movie.Trackers
	}
	if fr.TV != nil {
		return fr.TV.Trackers
	}
	if fr.Music != nil {
		return fr.Music.Trackers
	}
	if fr.Book != nil {
		return fr.Book.Trackers
	}
	return nil
}

type RawSearch struct {
	Query      string   `url:"q,omitempty"`
	Trackers   []string `url:"-"`
	Categories []uint   `url:"cat,comma,omitempty"`
}

func NewRawSearch() *RawSearch {
	return &RawSearch{}
}

func (rs *RawSearch) WithQuery(query string) *RawSearch {
	copy := *rs
	copy.Query = query
	return &copy
}

func (rs *RawSearch) WithTrackers(trackers ...string) *RawSearch {
	copy := *rs
	copy.Trackers = trackers
	return &copy
}

func (rs *RawSearch) WithCategories(categories ...uint) *RawSearch {
	copy := *rs
	copy.Categories = categories
	return &copy
}

func (rs *RawSearch) Build() *FetchRequest {
	return &FetchRequest{Raw: rs}
}

type MovieSearch struct {
	RawSearch
	Year     uint   `url:"year,omitempty"`
	Genre    string `url:"genre,omitempty"`
	IMDBID   string `url:"imdbid,omitempty"`
	TracktID uint   `url:"tracktid,omitempty"`
	DoubanID uint   `url:"doubanid,omitempty"`
}

func NewMovieSearch() *MovieSearch {
	return &MovieSearch{}
}

func (ms *MovieSearch) WithQuery(query string) *MovieSearch {
	copy := *ms
	copy.Query = query
	return &copy
}

func (ms *MovieSearch) WithTrackers(trackers ...string) *MovieSearch {
	copy := *ms
	copy.Trackers = trackers
	return &copy
}

func (ms *MovieSearch) WithCategories(categories ...uint) *MovieSearch {
	copy := *ms
	copy.Categories = categories
	return &copy
}

func (ms *MovieSearch) WithYear(year uint) *MovieSearch {
	copy := *ms
	copy.Year = year
	return &copy
}

func (ms *MovieSearch) WithGenre(genre string) *MovieSearch {
	copy := *ms
	copy.Genre = genre
	return &copy
}

func (ms *MovieSearch) WithIMDBID(imdbID string) *MovieSearch {
	copy := *ms
	copy.IMDBID = imdbID
	return &copy
}

func (ms *MovieSearch) WithTracktID(tracktID uint) *MovieSearch {
	copy := *ms
	copy.TracktID = tracktID
	return &copy
}

func (ms *MovieSearch) WithDoubanID(doubanID uint) *MovieSearch {
	copy := *ms
	copy.DoubanID = doubanID
	return &copy
}

func (ms *MovieSearch) Build() *FetchRequest {
	return &FetchRequest{Movie: ms}
}

type TVSearch struct {
	MovieSearch
	Season   uint `url:"season,omitempty"`
	Episode  uint `url:"ep,omitempty"`
	TVDBID   uint `url:"tvdbid,omitempty"`
	RageID   uint `url:"rid,omitempty"`
	TMDBID   uint `url:"tmdbid,omitempty"`
	TVMazeID uint `url:"tvmazeid,omitempty"`
}

func NewTVSearch() *TVSearch {
	return &TVSearch{MovieSearch: MovieSearch{RawSearch: RawSearch{}}}
}

func (ts *TVSearch) WithQuery(query string) *TVSearch {
	copy := *ts
	copy.Query = query
	return &copy
}

func (ts *TVSearch) WithTrackers(trackers ...string) *TVSearch {
	copy := *ts
	copy.Trackers = trackers
	return &copy
}

func (ts *TVSearch) WithCategories(categories ...uint) *TVSearch {
	copy := *ts
	copy.Categories = categories
	return &copy
}

func (ts *TVSearch) WithYear(year uint) *TVSearch {
	copy := *ts
	copy.Year = year
	return &copy
}

func (ts *TVSearch) WithGenre(genre string) *TVSearch {
	copy := *ts
	copy.Genre = genre
	return &copy
}

func (ts *TVSearch) WithIMDBID(imdbID string) *TVSearch {
	copy := *ts
	copy.IMDBID = imdbID
	return &copy
}

func (ts *TVSearch) WithTracktID(tracktID uint) *TVSearch {
	copy := *ts
	copy.TracktID = tracktID
	return &copy
}

func (ts *TVSearch) WithDoubanID(doubanID uint) *TVSearch {
	copy := *ts
	copy.DoubanID = doubanID
	return &copy
}

func (ts *TVSearch) WithSeason(season uint) *TVSearch {
	copy := *ts
	copy.Season = season
	return &copy
}

func (ts *TVSearch) WithEpisode(episode uint) *TVSearch {
	copy := *ts
	copy.Episode = episode
	return &copy
}

func (ts *TVSearch) WithTVDBID(tvdbID uint) *TVSearch {
	copy := *ts
	copy.TVDBID = tvdbID
	return &copy
}

func (ts *TVSearch) WithRageID(rageID uint) *TVSearch {
	copy := *ts
	copy.RageID = rageID
	return &copy
}

func (ts *TVSearch) WithTMDBID(tmdbID uint) *TVSearch {
	copy := *ts
	copy.TMDBID = tmdbID
	return &copy
}

func (ts *TVSearch) WithTVMazeID(tvMazeID uint) *TVSearch {
	copy := *ts
	copy.TVMazeID = tvMazeID
	return &copy
}

func (ts *TVSearch) Build() *FetchRequest {
	return &FetchRequest{TV: ts}
}

type MusicSearch struct {
	RawSearch
	Album  string `url:"album,omitempty"`
	Artist string `url:"artist,omitempty"`
	Label  string `url:"label,omitempty"`
	Track  string `url:"track,omitempty"`
	Year   uint   `url:"year,omitempty"`
	Genre  string `url:"genre,omitempty"`
}

func NewMusicSearch() *MusicSearch {
	return &MusicSearch{RawSearch: RawSearch{}}
}

func (ms *MusicSearch) WithQuery(query string) *MusicSearch {
	copy := *ms
	copy.Query = query
	return &copy
}

func (ms *MusicSearch) WithTrackers(trackers ...string) *MusicSearch {
	copy := *ms
	copy.Trackers = trackers
	return &copy
}

func (ms *MusicSearch) WithCategories(categories ...uint) *MusicSearch {
	copy := *ms
	copy.Categories = categories
	return &copy
}

func (ms *MusicSearch) WithAlbum(album string) *MusicSearch {
	copy := *ms
	copy.Album = album
	return &copy
}

func (ms *MusicSearch) WithArtist(artist string) *MusicSearch {
	copy := *ms
	copy.Artist = artist
	return &copy
}

func (ms *MusicSearch) WithLabel(label string) *MusicSearch {
	copy := *ms
	copy.Label = label
	return &copy
}

func (ms *MusicSearch) WithTrack(track string) *MusicSearch {
	copy := *ms
	copy.Track = track
	return &copy
}

func (ms *MusicSearch) WithYear(year uint) *MusicSearch {
	copy := *ms
	copy.Year = year
	return &copy
}

func (ms *MusicSearch) WithGenre(genre string) *MusicSearch {
	copy := *ms
	copy.Genre = genre
	return &copy
}

func (ms *MusicSearch) Build() *FetchRequest {
	return &FetchRequest{Music: ms}
}

type BookSearch struct {
	RawSearch
	Title     string `url:"title,omitempty"`
	Author    string `url:"author,omitempty"`
	Publisher string `url:"publisher,omitempty"`
	Year      uint   `url:"year,omitempty"`
	Genre     string `url:"genre,omitempty"`
}

func NewBookSearch() *BookSearch {
	return &BookSearch{RawSearch: RawSearch{}}
}

func (bs *BookSearch) WithQuery(query string) *BookSearch {
	copy := *bs
	copy.Query = query
	return &copy
}

func (bs *BookSearch) WithTrackers(trackers ...string) *BookSearch {
	copy := *bs
	copy.Trackers = trackers
	return &copy
}

func (bs *BookSearch) WithCategories(categories ...uint) *BookSearch {
	copy := *bs
	copy.Categories = categories
	return &copy
}

func (bs *BookSearch) WithTitle(title string) *BookSearch {
	copy := *bs
	copy.Title = title
	return &copy
}

func (bs *BookSearch) WithAuthor(author string) *BookSearch {
	copy := *bs
	copy.Author = author
	return &copy
}

func (bs *BookSearch) WithPublisher(publisher string) *BookSearch {
	copy := *bs
	copy.Publisher = publisher
	return &copy
}

func (bs *BookSearch) WithYear(year uint) *BookSearch {
	copy := *bs
	copy.Year = year
	return &copy
}

func (bs *BookSearch) WithGenre(genre string) *BookSearch {
	copy := *bs
	copy.Genre = genre
	return &copy
}

func (bs *BookSearch) Build() *FetchRequest {
	return &FetchRequest{Book: bs}
}
