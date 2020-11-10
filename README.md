# go-jackett

It is non-official Golang SDK for [Jackett](https://github.com/Jackett/Jackett).

Example usage:

```
package main

import (
	"log"

	"github.com/webtor-io/go-jackett"
)

func main() {
    ctx := context.Background()
    j := jackett.NewJackett(&jackett.Settings{
        ApiURL: "YOUR_API_URL",
        ApiKey: "YOUR_API_KEY",
    })
    resp, err := j.Fetch(ctx, &jackett.FetchRequest{
        Categories: []uint{7000},
        Query:      "Crime and Punishment",
    })
    if err != nil {
        panic(err)
    }
    for _, r := range resp.Results {
        log.Printf("%+v", r)
    }
}
```

As ApiUrl just use root url of your Jackett instance. ApiKey could be found at the top of Jackett UI.

It is also possible to get Jackett credentials from environment variables `JACKETT_API_URL` and `JACKETT_API_KEY`.
In this case just provide empty settings like so:

```
j := jackett.NewJackett(&jackett.Settings{})
```