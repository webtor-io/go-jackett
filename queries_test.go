package jackett

import (
	"fmt"
	"maps"
	"net/url"
	"reflect"
	"slices"
	"sort"
	"testing"
)

func TestQueryBuilders(t *testing.T) {
	j := newTestJackett(t)

	t.Run("RawSearch", func(t *testing.T) {
		tests := []struct {
			name    string
			builder *RawSearch
			expect  expectations
		}{
			{
				name:    "empty",
				builder: NewRawSearch(),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"t":        []string{"search"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with query",
				builder: NewRawSearch().
					WithQuery("test query"),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"q":        []string{"test query"},
						"t":        []string{"search"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with trackers",
				builder: NewRawSearch().
					WithTrackers("tracker1", "tracker2"),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"t":        []string{"search"},
					},
					Trackers: []string{"tracker1", "tracker2"},
				},
			},
			{
				name: "with categories",
				builder: NewRawSearch().
					WithCategories(1000, 2000),
				expect: expectations{
					Query: url.Values{
						"cat":      []string{"1000,2000"},
						"extended": []string{"1"},
						"t":        []string{"search"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with all params",
				builder: NewRawSearch().
					WithQuery("complex").
					WithTrackers("tr").
					WithCategories(3000),
				expect: expectations{
					Query: url.Values{
						"q":        []string{"complex"},
						"cat":      []string{"3000"},
						"extended": []string{"1"},
						"t":        []string{"search"},
					},
					Trackers: []string{"tr"},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fr := tt.builder.Build()
				urlStrs, err := j.generateFetchURLs(fr)
				if err != nil {
					t.Fatalf("generateFetchURL() returned error: %v", err)
				}
				assertURLs(t, tt.expect, urlStrs...)
			})
		}
	})

	t.Run("MovieSearch", func(t *testing.T) {
		tests := []struct {
			name    string
			builder *MovieSearch
			expect  expectations
		}{
			{
				name:    "empty",
				builder: NewMovieSearch(),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"t":        []string{"movie"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with query",
				builder: NewMovieSearch().
					WithQuery("test movie title"),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"q":        []string{"test movie title"},
						"t":        []string{"movie"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with year",
				builder: NewMovieSearch().
					WithYear(2023),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"t":        []string{"movie"},
						"year":     []string{fmt.Sprintf("%d", 2023)},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with genre",
				builder: NewMovieSearch().
					WithGenre("Test Genre"),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"genre":    []string{"Test Genre"},
						"t":        []string{"movie"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with IMDB ID",
				builder: NewMovieSearch().
					WithIMDBID("tt123456"),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"imdbid":   []string{"tt123456"},
						"t":        []string{"movie"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with all params",
				builder: NewMovieSearch().
					WithQuery("test action movie").
					WithYear(2020).
					WithGenre("Test Genre").
					WithIMDBID("tt98765").
					WithTrackers("movietracker").
					WithCategories(CategoryMoviesHD),
				expect: expectations{
					Query: url.Values{
						"cat":      []string{fmt.Sprintf("%d", CategoryMoviesHD)},
						"extended": []string{"1"},
						"genre":    []string{"Test Genre"},
						"imdbid":   []string{"tt98765"},
						"q":        []string{"test action movie"},
						"t":        []string{"movie"},
						"year":     []string{fmt.Sprintf("%d", 2020)},
					},
					Trackers: []string{"movietracker"},
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fr := tt.builder.Build()
				urlStrs, err := j.generateFetchURLs(fr)
				if err != nil {
					t.Fatalf("generateFetchURL() returned error: %v", err)
				}
				assertURLs(t, tt.expect, urlStrs...)
			})
		}
	})

	t.Run("TVSearch", func(t *testing.T) {
		tests := []struct {
			name    string
			builder *TVSearch
			expect  expectations
		}{
			{
				name:    "empty",
				builder: NewTVSearch(),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"t":        []string{"tvsearch"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with query",
				builder: NewTVSearch().
					WithQuery("test tv show name"),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"q":        []string{"test tv show name"},
						"t":        []string{"tvsearch"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with season and episode",
				builder: NewTVSearch().
					WithSeason(1).
					WithEpisode(5),
				expect: expectations{
					Query: url.Values{
						"ep":       []string{"5"},
						"extended": []string{"1"},
						"season":   []string{"1"},
						"t":        []string{"tvsearch"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with TVDBID",
				builder: NewTVSearch().
					WithTVDBID(12345),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"t":        []string{"tvsearch"},
						"tvdbid":   []string{fmt.Sprintf("%d", 12345)},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with all params",
				builder: NewTVSearch().
					WithQuery("test tv show").
					WithYear(1994).
					WithSeason(1).
					WithEpisode(1).
					WithTVDBID(79168).
					WithCategories(CategoryTVHD),
				expect: expectations{
					Query: url.Values{
						"cat":      []string{fmt.Sprintf("%d", CategoryTVHD)},
						"ep":       []string{"1"},
						"extended": []string{"1"},
						"q":        []string{"test tv show"},
						"season":   []string{"1"},
						"t":        []string{"tvsearch"},
						"tvdbid":   []string{fmt.Sprintf("%d", 79168)},
						"year":     []string{fmt.Sprintf("%d", 1994)},
					},
					Trackers: []string{"all"},
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fr := tt.builder.Build()
				urlStrs, err := j.generateFetchURLs(fr)
				if err != nil {
					t.Fatalf("generateFetchURL() returned error: %v", err)
				}
				assertURLs(t, tt.expect, urlStrs...)
			})
		}
	})

	t.Run("MusicSearch", func(t *testing.T) {
		tests := []struct {
			name    string
			builder *MusicSearch
			expect  expectations
		}{
			{
				name:    "empty",
				builder: NewMusicSearch(),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"t":        []string{"music"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with query",
				builder: NewMusicSearch().
					WithQuery("test song name"),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"q":        []string{"test song name"},
						"t":        []string{"music"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with album",
				builder: NewMusicSearch().
					WithAlbum("Test Album"),
				expect: expectations{
					Query: url.Values{
						"album":    []string{"Test Album"},
						"extended": []string{"1"},
						"t":        []string{"music"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with artist and year",
				builder: NewMusicSearch().
					WithArtist("Test Artist").
					WithYear(1973),
				expect: expectations{
					Query: url.Values{
						"artist":   []string{"Test Artist"},
						"extended": []string{"1"},
						"t":        []string{"music"},
						"year":     []string{fmt.Sprintf("%d", 1973)},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with all params",
				builder: NewMusicSearch().
					WithQuery("Test Track Name").
					WithAlbum("Test Album Name").
					WithArtist("Test Artist").
					WithLabel("Test Label").
					WithTrack("4").
					WithYear(1973).
					WithGenre("Test Music Genre").
					WithCategories(CategoryAudioLossless),
				expect: expectations{
					Query: url.Values{
						"album":    []string{"Test Album Name"},
						"artist":   []string{"Test Artist"},
						"cat":      []string{fmt.Sprintf("%d", CategoryAudioLossless)},
						"extended": []string{"1"},
						"genre":    []string{"Test Music Genre"},
						"label":    []string{"Test Label"},
						"q":        []string{"Test Track Name"},
						"t":        []string{"music"},
						"track":    []string{"4"},
						"year":     []string{fmt.Sprintf("%d", 1973)},
					},
					Trackers: []string{"all"},
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fr := tt.builder.Build()
				urlStrs, err := j.generateFetchURLs(fr)
				if err != nil {
					t.Fatalf("generateFetchURL() returned error: %v", err)
				}
				assertURLs(t, tt.expect, urlStrs...)
			})
		}
	})

	t.Run("BookSearch", func(t *testing.T) {
		tests := []struct {
			name    string
			builder *BookSearch
			expect  expectations
		}{
			{
				name:    "empty",
				builder: NewBookSearch(),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"t":        []string{"book"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with title",
				builder: NewBookSearch().
					WithTitle("Test Book Title"),
				expect: expectations{
					Query: url.Values{
						"extended": []string{"1"},
						"t":        []string{"book"},
						"title":    []string{"Test Book Title"},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with author and year",
				builder: NewBookSearch().
					WithAuthor("Test Author Name").
					WithYear(1937),
				expect: expectations{
					Query: url.Values{
						"author":   []string{"Test Author Name"},
						"extended": []string{"1"},
						"t":        []string{"book"},
						"year":     []string{fmt.Sprintf("%d", 1937)},
					},
					Trackers: []string{"all"},
				},
			},
			{
				name: "with all params",
				builder: NewBookSearch().
					WithQuery("test fantasy book").
					WithTitle("Another Test Book Title").
					WithAuthor("Another Test Author Name").
					WithPublisher("Test Publisher").
					WithYear(1965).
					WithGenre("Test Book Genre").
					WithCategories(CategoryBooksEBook),
				expect: expectations{
					Query: url.Values{
						"author":    []string{"Another Test Author Name"},
						"cat":       []string{fmt.Sprintf("%d", CategoryBooksEBook)},
						"extended":  []string{"1"},
						"genre":     []string{"Test Book Genre"},
						"publisher": []string{"Test Publisher"},
						"q":         []string{"test fantasy book"},
						"t":         []string{"book"},
						"title":     []string{"Another Test Book Title"},
						"year":      []string{fmt.Sprintf("%d", 1965)},
					},
					Trackers: []string{"all"},
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fr := tt.builder.Build()
				urlStrs, err := j.generateFetchURLs(fr)
				if err != nil {
					t.Fatalf("generateFetchURL() returned error: %v", err)
				}
				assertURLs(t, tt.expect, urlStrs...)
			})
		}
	})
}

type expectations struct {
	Query    url.Values
	Trackers []string
}

func assertURLs(t *testing.T, exp expectations, urls ...url.URL) {
	t.Helper()

	seenTrackers := make(map[string]struct{})

	for _, u := range urls {
		tracker := extractTracker(u.Path)
		if tracker == "" {
			t.Errorf("expected tracker to be in path: %s", u.Path)
			continue
		}
		seenTrackers[tracker] = struct{}{}

		actualQuery := u.Query()

		// Check if all expected query parameters are present and correct
		for key, expectedValues := range exp.Query {
			actualValues, ok := actualQuery[key]
			if !ok {
				t.Errorf("expected query parameter '%s' not found in URL '%s'. Expected values: %v, Actual Query: %v", key, u.String(), expectedValues, actualQuery)
				continue
			}
			// Sort slices for consistent comparison, as order doesn't matter for multi-value params
			sort.Strings(actualValues)
			sort.Strings(expectedValues)
			if !reflect.DeepEqual(actualValues, expectedValues) {
				t.Errorf("query parameter '%s' mismatch. Got %v, want %v in URL '%s'", key, actualValues, expectedValues, u.String())
			}
		}

		// Check if there are any unexpected query parameters
		for key := range actualQuery {
			if _, ok := exp.Query[key]; !ok {
				t.Errorf("unexpected query parameter '%s' found in URL '%s'. Actual Query: %v", key, u.String(), actualQuery)
			}
		}
	}

	slices.Sort(exp.Trackers)
	sawTrackers := slices.Sorted(maps.Keys(seenTrackers))
	if !slices.Equal(exp.Trackers, sawTrackers) {
		t.Errorf("expected to see trackers %v; saw %v", exp.Trackers, sawTrackers)
	}
}

func newTestJackett(t *testing.T) *Client {
	t.Helper()
	j, err := New(Settings{ApiURL: "http://localhost:9117"})
	if err != nil {
		t.Fatal(err)
	}
	return j
}
