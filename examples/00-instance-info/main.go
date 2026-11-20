// Example 00: the simplest request, reading information about the instance (GET /info).
//
// It shows how to create the client, send one request, check the response and read the result.
// The client is built by examples/client (see elabclient.New), which needs a server URL and an API key.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/elabftw/elabapi-go/v6/elabapi"
	elabclient "github.com/elabftw/elabapi-go/v6/examples/client"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run holds the actual logic and returns an error instead of exiting, so main stays the only place that exits.
func run() error {
	client, err := elabclient.New()
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	// The request is cancelled if it takes more than 10 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// The empty params struct means "no optional query parameters": we want the default statistics.
	// If err is not nil, the request itself failed (network, timeout...): the server did not answer.
	resp, err := client.GetInfoWithResponse(ctx, &elabapi.GetInfoParams{})
	if err != nil {
		return fmt.Errorf("get info: %w", err)
	}

	// JSON200 is only filled on a 200 with a valid body: nil means an error status (e.g. 401) or bad JSON.
	if resp.JSON200 == nil {
		return fmt.Errorf("get info: unexpected status %s: %s", resp.Status(), resp.Body)
	}

	// The spec says this endpoint can answer with one of two shapes (statistics, or histograms when the
	// "hist" parameter is set), so the generator gives a union type that keeps the raw JSON.
	// We pick the shape we asked for: Body0 is the statistics one.
	// The call does not check that the JSON really has that shape: fields that do not match stay nil.
	info, err := resp.JSON200.AsGetInfo200JSONResponseBody0()
	if err != nil {
		return fmt.Errorf("decode info: %w", err)
	}

	fmt.Printf("[*] The instance has %s teams and %s users.\n", display(info.TeamsCount), display(info.AllUsersCount))
	fmt.Printf("[*] Total size of uploaded files: %s\n", display(info.UploadsFilesizeSumFormatted))

	return nil
}

// === Helpers ===

// display returns the value p points to as a string, or "unknown" if p is nil.
// The generated types use pointers for optional fields (see elabapi/client.gen.go).
func display[T any](p *T) string {
	if p == nil {
		return "unknown"
	}
	return fmt.Sprint(*p)
}
