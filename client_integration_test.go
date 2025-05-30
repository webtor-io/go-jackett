package jackett

import (
	"errors"
	"math/rand"
	"os"
	"testing"
)

func TestIntegrationFetch(t *testing.T) {
	if os.Getenv(envAPIKey) == "" || os.Getenv(envAPIURL) == "" {
		t.Skipf("Test disabled unless %s and %s are set to point to a real Jackett server", envAPIKey, envAPIURL)
	}

	j, err := New(Settings{})
	if err != nil {
		t.Fatal(err)
	}

	idxs, err := j.ListIndexers(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var trackers []string
	for _, idx := range idxs {
		t.Logf("discovered indexer: %s", idx.ID)
		trackers = append(trackers, idx.ID)
	}

	if len(trackers) == 0 {
		t.Skip("No indexers available for testing")
	}

	randomTracker := func() string {
		return trackers[rand.Intn(len(trackers))]
	}

	t.Run("RawSearch", func(t *testing.T) {
		q := NewRawSearch().
			WithQuery("ubuntu").
			WithTrackers(randomTracker()).
			Build()

		resp, err := j.Fetch(t.Context(), q)
		if err != nil {
			if errors.Is(err, &UnsupportedError{}) {
				t.Skipf("query isn't supported by the test tracker: %s", err.Error())
			}
			t.Fatal(err)
		}
		validateResp(t, resp)
	})

	t.Run("MovieSearch", func(t *testing.T) {
		q := NewMovieSearch().
			WithQuery("the matrix").
			WithYear(1999).
			WithGenre("action").
			WithTrackers(randomTracker()).
			Build()

		resp, err := j.Fetch(t.Context(), q)
		if err != nil {
			if errors.Is(err, &UnsupportedError{}) {
				t.Skipf("query isn't supported by the test tracker: %s", err.Error())
			}
			t.Fatal(err)
		}
		validateResp(t, resp)
	})

	t.Run("MovieSearchWithIMDB", func(t *testing.T) {
		q := NewMovieSearch().
			WithIMDBID("tt0133093").
			WithTrackers(randomTracker()).
			Build()

		resp, err := j.Fetch(t.Context(), q)
		if err != nil {
			if errors.Is(err, &UnsupportedError{}) {
				t.Skipf("query isn't supported by the test tracker: %s", err.Error())
			}
			t.Fatal(err)
		}
		validateResp(t, resp)
	})

	t.Run("TVSearch", func(t *testing.T) {
		q := NewTVSearch().
			WithSeason(15).
			WithEpisode(12).
			WithTrackers(randomTracker()).
			Build()

		resp, err := j.Fetch(t.Context(), q)
		if err != nil {
			if errors.Is(err, &UnsupportedError{}) {
				t.Skipf("query isn't supported by the test tracker: %s", err.Error())
			}
			t.Fatal(err)
		}
		validateResp(t, resp)
	})

	t.Run("TVSearchWithTMDBID", func(t *testing.T) {
		q := NewTVSearch().
			WithTMDBID(32726).
			WithTrackers(trackers[0]).
			Build()

		resp, err := j.Fetch(t.Context(), q)
		if err != nil {
			if errors.Is(err, &UnsupportedError{}) {
				t.Skipf("query isn't supported by the test tracker: %s", err.Error())
			}
			t.Fatal(err)
		}
		validateResp(t, resp)
	})

	t.Run("MusicSearch", func(t *testing.T) {
		q := NewMusicSearch().
			WithQuery("pink floyd").
			WithAlbum("dark side of the moon").
			WithArtist("pink floyd").
			WithYear(1973).
			WithTrackers(randomTracker()).
			Build()

		resp, err := j.Fetch(t.Context(), q)
		if err != nil {
			if errors.Is(err, &UnsupportedError{}) {
				t.Skipf("query isn't supported by the test tracker: %s", err.Error())
			}
			t.Fatal(err)
		}
		validateResp(t, resp)
	})

	t.Run("MusicSearchByTrack", func(t *testing.T) {
		q := NewMusicSearch().
			WithTrack("money").
			WithArtist("pink floyd").
			WithTrackers(randomTracker()).
			Build()

		resp, err := j.Fetch(t.Context(), q)
		if err != nil {
			if errors.Is(err, &UnsupportedError{}) {
				t.Skipf("query isn't supported by the test tracker: %s", err.Error())
			}
			t.Fatal(err)
		}
		validateResp(t, resp)
	})

	t.Run("BookSearch", func(t *testing.T) {
		q := NewBookSearch().
			WithTitle("dune").
			WithAuthor("frank herbert").
			WithYear(1965).
			WithTrackers(randomTracker()).
			Build()

		resp, err := j.Fetch(t.Context(), q)
		if err != nil {
			if errors.Is(err, &UnsupportedError{}) {
				t.Skipf("query isn't supported by the test tracker: %s", err.Error())
			}
			t.Fatal(err)
		}
		validateResp(t, resp)
	})

	t.Run("BookSearchByAuthor", func(t *testing.T) {
		q := NewBookSearch().
			WithAuthor("stephen king").
			WithGenre("horror").
			WithTrackers(randomTracker()).
			Build()

		resp, err := j.Fetch(t.Context(), q)
		if err != nil {
			if errors.Is(err, &UnsupportedError{}) {
				t.Skipf("query isn't supported by the test tracker: %s", err.Error())
			}
			t.Fatal(err)
		}
		validateResp(t, resp)
	})

	t.Run("SearchWithCategories", func(t *testing.T) {
		q := NewRawSearch().
			WithQuery("bugs").
			WithCategories(CategoryMovies, CategoryMoviesForeign).
			WithTrackers(randomTracker()).
			Build()

		resp, err := j.Fetch(t.Context(), q)
		if err != nil {
			if errors.Is(err, &UnsupportedError{}) {
				t.Skipf("query isn't supported by the test tracker: %s", err.Error())
			}
			t.Fatal(err)
		}
		validateResp(t, resp)
	})
}

func validateResp(t *testing.T, results []Result) {
	t.Helper()
	t.Logf("[%s] got %d results", t.Name(), len(results))

	for i, result := range results {
		if result.Title == "" {
			t.Errorf("result[%s][%d]: Title is empty", t.Name(), i)
		}
		if result.Size == 0 {
			t.Errorf("result[%s][%d]: Size is 0", t.Name(), i)
		}
		if result.PublishDate.IsZero() {
			t.Errorf("result[%s][%d]: PublishDate is zero/unparsed", t.Name(), i)
		}
	}
}
