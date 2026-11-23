// Example 02: attach a file to an experiment, list the attachments, download one, then clean up.
//
// Go version of the Python example 03-experiment-file-handling.
// To stay self-contained, it creates a temporary experiment (see example 01) and uploads
// an in-memory file, so it does not depend on an existing experiment id or a file on disk.
//
// Doc: https://doc.elabftw.net/api/v2/#/Uploads
package main

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/elabftw/elabapi-go/v6/elabapi"
	elabclient "github.com/elabftw/elabapi-go/v6/examples/client"
)

// The entity type is part of the URL of every upload endpoint: /experiments/{id}/uploads
const entityType = "experiments"

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

	// 0. create a temporary experiment that will receive the file (POST /experiments)
	var expID int
	{
		title := "Example experiment for file handling"
		resp, err := client.PostExperimentWithResponse(ctx, elabapi.PostExperimentJSONRequestBody{Title: &title})
		if err != nil {
			return fmt.Errorf("create experiment: %w", err)
		}
		if resp.Headers201 == nil || resp.Headers201.Location == nil {
			return fmt.Errorf("create experiment: unexpected status %s: %s", resp.Status(), resp.Body)
		}

		expID, err = getIdFromUrl(*resp.Headers201.Location)
		if err != nil {
			return fmt.Errorf("get experiment id: %w", err)
		}
		fmt.Printf("[*] Created experiment: %d\n", expID)
	}

	// The file we will upload. It lives in memory: no file is needed on disk.
	filename := "hello.txt"
	content := []byte("Hello from the eLabFTW Go example!\n")

	// 1. upload the file to the experiment (POST /experiments/{id}/uploads)
	var uploadID int
	{
		// The upload endpoint expects a multipart/form-data body: a list of "parts", one per form field.
		// The generated client has no typed helper for it, so we build the body ourselves with
		// mime/multipart and send it with the ...WithBody variant, which takes a content type and an io.Reader.
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)

		// Part "file": the file itself. The name given here is what eLabFTW stores as real_name.
		fw, err := mw.CreateFormFile("file", filename)
		if err != nil {
			return fmt.Errorf("create file part: %w", err)
		}
		if _, err := fw.Write(content); err != nil {
			return fmt.Errorf("write file part: %w", err)
		}
		// Part "comment": an optional text field.
		if err := mw.WriteField("comment", "Uploaded with the Go example"); err != nil {
			return fmt.Errorf("write comment part: %w", err)
		}
		// Close writes the closing boundary of the body. Without it, the server receives an invalid body.
		if err := mw.Close(); err != nil {
			return fmt.Errorf("close multipart body: %w", err)
		}

		// FormDataContentType returns "multipart/form-data; boundary=...": the boundary tells the
		// server where each part starts and ends, so it must be the same one used to write the body.
		resp, err := client.PostUploadWithBodyWithResponse(ctx, entityType, expID, &elabapi.PostUploadParams{}, mw.FormDataContentType(), &buf)
		if err != nil {
			return fmt.Errorf("upload file: %w", err)
		}

		// A successful upload is a 201 with no body: the URL of the new upload is in the Location header.
		if resp.Headers201 == nil || resp.Headers201.Location == nil {
			return fmt.Errorf("upload file: unexpected status %s: %s", resp.Status(), resp.Body)
		}

		uploadID, err = getIdFromUrl(*resp.Headers201.Location)
		if err != nil {
			return fmt.Errorf("get upload id: %w", err)
		}
		fmt.Printf("[*] Uploaded %s: upload id %d\n", filename, uploadID)
	}

	// 2. list the files attached to the experiment (GET /experiments/{id}/uploads)
	{
		resp, err := client.ReadUploadsWithResponse(ctx, entityType, expID, &elabapi.ReadUploadsParams{})
		if err != nil {
			return fmt.Errorf("list uploads: %w", err)
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("list uploads: unexpected status %s: %s", resp.Status(), resp.Body)
		}

		// JSON200 points to a slice of Upload. The fields are pointers, so we dereference them
		// (fine here, the server always sends them for an existing upload).
		for _, u := range *resp.JSON200 {
			fmt.Printf("[*] Upload %d: name=%s comment=%s state=%d\n", *u.Id, *u.RealName, *u.Comment, *u.State)
		}
	}

	// 3. download the file (GET /experiments/{id}/uploads/{subid}?format=binary)
	{
		// Without format=binary, the server answers with the JSON description of the upload (as in step 2).
		// With it, the body is the content of the file.
		format := elabapi.ReadUploadParamsFormatBinary
		resp, err := client.ReadUploadWithResponse(ctx, entityType, expID, uploadID, &elabapi.ReadUploadParams{Format: &format})
		if err != nil {
			return fmt.Errorf("download upload: %w", err)
		}

		// The answer is not JSON, so JSON200 stays nil: we check the status code instead.
		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("download upload: unexpected status %s: %s", resp.Status(), resp.Body)
		}

		// resp.Body holds the whole file in memory (fine for small files; for big ones, use the plain
		// client.ReadUpload method, which gives an http.Response whose Body can be streamed to a file).
		// To keep the file, you would write it with os.WriteFile("name", resp.Body, 0o644).
		if !bytes.Equal(resp.Body, content) {
			return fmt.Errorf("download upload: content differs from what was uploaded")
		}
		fmt.Printf("[*] Downloaded %d bytes, identical to the uploaded file\n", len(resp.Body))
	}

	// 4. clean up: delete the upload, then the experiment
	{
		// DELETE /experiments/{id}/uploads/{subid}
		resp, err := client.DeleteUploadWithResponse(ctx, entityType, expID, uploadID)
		if err != nil {
			return fmt.Errorf("delete upload: %w", err)
		}
		if resp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("delete upload: unexpected status %s: %s", resp.Status(), resp.Body)
		}
		fmt.Printf("[*] Deleted upload: %d\n", uploadID)

		// DELETE /experiments/{id}
		delResp, err := client.DeleteExperimentWithResponse(ctx, expID)
		if err != nil {
			return fmt.Errorf("delete experiment: %w", err)
		}
		if delResp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("delete experiment: unexpected status %s: %s", delResp.Status(), delResp.Body)
		}
		fmt.Printf("[*] Deleted experiment: %d\n", expID)
	}

	return nil
}

// === Helpers ===

// getIdFromUrl returns the id found at the end of a Location header,
// for example 42 for "https://elab.local/api/v2/experiments/42".
func getIdFromUrl(location string) (int, error) {
	id, err := strconv.Atoi(path.Base(location))
	if err != nil {
		return 0, fmt.Errorf("parse id from %q: %w", location, err)
	}

	return id, nil
}
