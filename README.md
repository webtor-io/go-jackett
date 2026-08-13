# go-jackett
[![GoDoc](https://godoc.org/github.com/webtor-io/go-jackett?status.svg)](https://godoc.org/github.com/webtor-io/go-jackett)
[![Go Report Card](https://goreportcard.com/badge/github.com/webtor-io/go-jackett)](https://goreportcard.com/report/github.com/webtor-io/go-jackett)
[![MIT license](http://img.shields.io/badge/license-MIT-brightgreen.svg)](http://opensource.org/licenses/MIT)

It is non-official Golang SDK for [Jackett](https://github.com/Jackett/Jackett),
and — through `NewTorznab` — a client for any [Torznab](https://torznab.github.io/spec-1.3-draft/torznab/Specification-v1.3.html)
endpoint: Prowlarr, NZBHydra2 or a tracker's own feed.

Example usage:

```go
package main

import (
	"log"
	"context"
	"github.com/webtor-io/go-jackett"
)

func main() {
    ctx := context.Background()
    j, err := jackett.New(jackett.Settings{
        ApiURL: "YOUR_API_URL",
        ApiKey: "YOUR_API_KEY",
    })
    if err != nil {
        panic(err)
    }
    resp, err := j.Fetch(ctx, jackett.NewBookSearch().
        WithTitle("Crime and Punishment").
        WithCategories(jackett.CategoryBooks).
        Build()
    })
    if err != nil {
        panic(err)
    }
    for _, r := range resp.Results {
        log.Printf("%+v", r)
    }
}
```

## Arbitrary Torznab endpoints

`New` expects a Jackett API root and builds
`/api/v2.0/indexers/<id>/results/torznab` paths from it. `NewTorznab` instead
takes the feed URL itself, so it also speaks to implementations that have no
such layout:

```go
j, err := jackett.NewTorznab(jackett.Settings{
    // Whatever the indexer's UI handed you — Jackett's "Copy Torznab Feed",
    // a Prowlarr indexer URL, a tracker's own feed.
    ApiURL:    "https://prowlarr.example.com/api/v1/indexer/3/newznab",
    ApiKey:    "YOUR_API_KEY",
    UserAgent: "my-app/1.0",
})
if err != nil {
    panic(err)
}

// Capabilities of the endpoint itself; the id is ignored in this mode.
caps, err := j.Caps(ctx, "")

results, err := j.Fetch(ctx, jackett.NewTVSearch().
    WithIMDBID("1839578").
    WithSeason(5).
    WithEpisode(14).
    Build(),
    // Skip the pre-flight capability probe when you already keep a snapshot
    // of the endpoint's caps — it saves a request per search.
    jackett.WithoutCapsValidation())
```

Query parameters already present in the URL are kept as defaults, except the
search parameters a request owns: a stale `q=` copied out of a browser
address bar never leaks into a later query. `Trackers`, `DefaultTrackers` and
`ListIndexers` are Jackett-specific and do not apply — the endpoint may
itself be an aggregate over many trackers.

As `ApiURL` just use root URL of your Jackett instance. `ApiKey` could be found at the top of Jackett UI.

It is also possible to get Jackett credentials from environment variables `JACKETT_API_URL` and `JACKETT_API_KEY`.
In this case just provide empty settings like so:

```go
j, err := jackett.New(jackett.Settings{})
```

### Querying

`jackett.Fetch()` takes a `jackett.FetchRequest` argument which describes all the possible query types that Jackett supports.

While you are free to build out a `jackett.FetchRequest` any way you choose, the package provides builder methods to make it easy.

```go
req := jackett.NewRawSearch().
    WithQuery("something special").
    Build()

req := jackett.NewMovieSearch().
    WithCategories(jackett.CategoryMoviesHD, jackett.CategoryMoviesUHD).
    WithIMDBID("tt12345").
    Build()

req := jackett.NewTVSearch().
    WithCategories(jackett.CategoryTVAnime).
    WithTVMazeID(123).
    Build()

req := jackett.NewMusicSearch().
    WithArtist("example").
    Build()

req := jackett.NewBookSearch().
    WithAuthor("example").
    WithTitle("example").
    Build()
```

### Targeting specific trackers

You can list all configured trackers on your instance using `Client.ListIndexers()`:

```go
j, err := jackett.New(jackett.Settings{})
indexers, err := j.ListIndexers(ctx)
```

You can use the `ID` of the desired tracker(s) to refine your query:

```go
req := jackett.New{Raw,Movie,TV,Music,Book}Search().
    WithTrackers(indexers[0].ID).
    Build()
```

By default if no trackers are specified in the query, all will be included.

Alternatively, you can define default trackers to be specified on all queries.

```go
j, err := jackett.New(jackett.Settings{
    DefaultTrackers: []string{"foo", "bar", "baz"},
    ...
})
```

With this setting, all queries will target these trackers unless specifically overridden via `WithTrackers(...)`.

### Parallelism

By default, `Client.Fetch` will query all defined trackers in parallel. The max number of concurrent requests per query is controlled by `WithMaxConcurrency`. If not set, it will default to `runtime.NumCPU()`. You can set it to `1` to effectively disable parallelism. Note that if you are not defining trackers via `WithTrackers(...)` or `Settings.DefaultTrackers`, the meta "all" tracker is used which always executes serially.

```go
resp, err := j.Fetch(ctx, input, jackett.WithMaxConcurrency(1)) 
```

You can also register a progress reporting function that will receive a callback whenever an indexer has completed. Note that this function must be safe to be called concurrently.

```go
resp, err := j.Fetch(ctx, input, jackett.WithProgressReportFunc(func(complete, total uint) {
    fmt.Printf("progress: %d/%d\n", complete, total)
}))
```
