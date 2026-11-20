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

func run() error {
	client, err := elabclient.New()
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.GetInfoWithResponse(ctx, &elabapi.GetInfoParams{})
	if err != nil {
		return fmt.Errorf("get info: %w", err)
	}

	// JSON200 is only filled on a 200 with a valid body: nil means an error status (e.g. 401) or bad JSON.
	if resp.JSON200 == nil {
		return fmt.Errorf("get info: unexpected status %s: %s", resp.Status(), resp.Body)
	}

	info, err := resp.JSON200.AsGetInfo200JSONResponseBody0()
	if err != nil {
		return fmt.Errorf("decode info: %w", err)
	}

	fmt.Printf("[*] The instance has %s teams and %s users.\n", display(info.TeamsCount), display(info.AllUsersCount))
	fmt.Printf("[*] Total size of uploaded files: %s\n", display(info.UploadsFilesizeSumFormatted))

	return nil
}

// display returns the value p points to as a string, or "unknown" if p is nil.
// The generated types use pointers for optional fields (see elabapi/client.gen.go).
func display[T any](p *T) string {
	if p == nil {
		return "unknown"
	}
	return fmt.Sprint(*p)
}
