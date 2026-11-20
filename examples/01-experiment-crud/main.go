// Example 01: the full life cycle (CRUD) of an experiment.
//
// It creates an experiment, reads it back, updates its title and body, then deletes it.
// Each step shows how to send a request with the generated client and how to check that it succeeded.
//
// The client is built by examples/client (see elabclient.New), which needs a server URL and an API key.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
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
//
// Every step below follows the same pattern:
//  1. call a ...WithResponse method of the client
//  2. if err != nil, the request itself failed (network, timeout...)
//  3. otherwise the server answered: check the response to see if it was a success
//
// The ...WithResponse methods read and close the HTTP response body for us and fill a typed struct:
// JSON200, JSON401, JSON404... are only non-nil when the server answered with that status and a JSON body.
func run() error {
	client, err := elabclient.New()
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	// All requests share this context: if the whole example takes more than 10 seconds, it is cancelled.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// id is declared outside the steps' blocks so that step 1 can set it and the next steps can use it.
	var id int

	// Each step is wrapped in its own { } block to keep its variables (resp, title...) local to the step.
	// 1. create an experiment (POST /experiments)
	{
		// Optional fields of the generated types are pointers (*string), so we need variables to take their address.
		// A nil pointer means "not set" and is left out of the JSON sent to the server.
		title := "Example experiment"
		body := "example description"
		resp, err := client.PostExperimentWithResponse(ctx, elabapi.PostExperimentJSONRequestBody{
			Title: &title,
			Body:  &body,
		})
		if err != nil {
			return fmt.Errorf("create experiment: %w", err)
		}

		// A successful create is a 201 with no body: the new entry's URL is in the Location header.
		if resp.Headers201 == nil || resp.Headers201.Location == nil {
			return fmt.Errorf("create experiment: unexpected status %s: %s", resp.Status(), resp.Body)
		}

		fmt.Printf("[*] Created experiment: %s\n", *resp.Headers201.Location)

		// The Location header looks like https://<server>/api/v2/experiments/42: the id is its last part.
		// We keep it for the next steps. "=" and not ":=", so that we assign the outer id instead of creating a new one.
		id, err = getIdFromUrl(*resp.Headers201.Location)
		if err != nil {
			return fmt.Errorf("get id: %w", err)
		}
	}

	// 2. read the experiment (GET /experiments/{id})
	{
		// The empty params struct means "no optional query parameters".
		resp, err := client.GetExperimentWithResponse(ctx, id, &elabapi.GetExperimentParams{})
		if err != nil {
			return fmt.Errorf("get experiment: %w", err)
		}

		// A successful read is a 200 with the experiment as JSON.
		// JSON200 is only filled in that case, so checking it covers both the status and the decoding.
		if resp.JSON200 == nil {
			return fmt.Errorf("get experiment: unexpected status %s: %s", resp.Status(), resp.Body)
		}

		fmt.Printf("[*] Read experiment %d, new title: %s, and body: %s\n", id, *resp.JSON200.Title, *resp.JSON200.Body)
	}

	// 3. update an experiment (PATCH /experiments/{id})
	// This step is more involved than the others, because of the shape of the request body.
	{
		title := "Updated Title"
		body := "another description"

		// The API accepts two kinds of PATCH bodies: an action (lock, pin, sign...) or a set of fields to edit.
		// The OpenAPI spec describes this with "anyOf", which the generator turns into a union type
		// (PatchExperimentJSONBody) that hides the raw JSON. We pick the "fields to edit" variant:
		// EntityEditable.
		var patch elabapi.PatchExperimentJSONBody
		if err := patch.FromEntityEditable(elabapi.EntityEditable{Title: &title, Body: &body}); err != nil {
			return fmt.Errorf("build patch body: %w", err)
		}

		// We turn the union into JSON bytes ourselves instead of passing it to PatchExperimentWithResponse.
		// The generated PatchExperimentJSONRequestBody type does not have the union's MarshalJSON method
		// (a type declared as "type A B" does not inherit B's methods), so it would be sent as "{}".
		payload, err := json.Marshal(patch)
		if err != nil {
			return fmt.Errorf("marshal patch body: %w", err)
		}

		// The ...WithBody variant takes a content type and an io.Reader: we send exactly the bytes built above.
		resp, err := client.PatchExperimentWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("update experiment: %w", err)
		}

		// A successful update is a 200 with the updated experiment as JSON.
		if resp.JSON200 == nil {
			return fmt.Errorf("update experiment: unexpected status %s: %s", resp.Status(), resp.Body)
		}

		fmt.Printf("[*] Updated experiment %d, new title: %s, and body: %s\n", id, *resp.JSON200.Title, *resp.JSON200.Body)
	}

	// 4. delete the experiment (DELETE /experiments/{id})
	{
		resp, err := client.DeleteExperimentWithResponse(ctx, id)
		if err != nil {
			return fmt.Errorf("delete experiment: %w", err)
		}

		// A successful delete is a 204 (No Content): there is no body and no header to look at,
		// so there is no JSON2xx field to check. We compare the status code instead.
		if resp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("delete experiment: unexpected status %s: %s", resp.Status(), resp.Body)
		}

		fmt.Printf("[*] Deleted experiment: %d", id)
	}

	return nil
}

// === Helpers ===

// getIdFromUrl returns the id found at the end of a Location header,
// for example 42 for "https://elab.local/api/v2/experiments/42".
// path.Base returns the last element of a "/"-separated path, strconv.Atoi converts it to an int.
func getIdFromUrl(location string) (int, error) {
	id, err := strconv.Atoi(path.Base(location))
	if err != nil {
		return 0, fmt.Errorf("parse experiment id from %q: %w", location, err)
	}

	return id, nil
}
