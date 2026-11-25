# AGENTS.md

Guide for AI agents writing Go programs that use this library to talk to an eLabFTW instance (scripts for researchers: bulk imports, exports, metadata fixes, reports...).

## What this library is

- A Go client for the eLabFTW REST API v2, generated with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) from the `openapi.yaml` attached to each eLabFTW release.
- Module: `github.com/elabftw/elabapi-go/v6`. Package: `elabapi`, imported as `github.com/elabftw/elabapi-go/v6/elabapi`.
- All code is in one generated file: `elabapi/client.gen.go`. There is no hand-written layer: the patterns below are how to use the generated code.
- API reference (endpoints, parameters, permissions): https://doc.elabftw.net/api/v2/

## Setup

```bash
go mod init myscript
go get github.com/elabftw/elabapi-go/v6@latest
```

- The library version matches the eLabFTW version it was generated from (`v6.0.9` = eLabFTW 6.0.9). Ask which eLabFTW version the instance runs if an endpoint is missing or behaves differently, and pin that version: `go get github.com/elabftw/elabapi-go/v6@v6.0.9`.
- The `go` directive in this module's `go.mod` sets the minimum Go version.

## Creating the client

Copy [examples/client/client.go](examples/client/client.go) into the script's module (it is meant to be copied, not imported). Its `New()` reads:

| Variable | Meaning |
|---|---|
| `API_KEY` | eLabFTW API key, sent as the raw `Authorization` header (no `Bearer` prefix) |
| `API_HOST` | API base URL, **including** `/api/v2`, e.g. `https://eln.example.org/api/v2` |
| `CA_PATH` | optional PEM file of an extra CA to trust |
| `INSECURE=1` | skip TLS verification: local development instances only |

Never hardcode an API key in a script, and never print it. Read it from the environment.

## Script structure

Follow the examples: `main` only calls `run() error` and exits on error; one `context.WithTimeout` for the run; each step checks its result before the next one uses it.

| Example | Shows |
|---|---|
| [00-instance-info](examples/00-instance-info/main.go) | simplest GET, union response |
| [01-experiment-crud](examples/01-experiment-crud/main.go) | create, read, update (PATCH), delete; id from `Location` |
| [02-experiment-file-handling](examples/02-experiment-file-handling/main.go) | multipart upload, list, binary download |

From a clone of this repository, run one with `API_HOST=... API_KEY=... go run ./examples/01-experiment-crud`.

## Finding the method for an endpoint

Names come from the `operationId` in the spec. For an operation `Xxx`:

- `client.XxxWithResponse(ctx, pathParams..., params, body)`: call this one; it reads the response for you.
- `XxxParams`: optional query parameters (pass `&elabapi.XxxParams{}` for none).
- `XxxJSONRequestBody`: the JSON body type. It is often an alias of a schema type (`PostExperimentJSONRequestBody = EntityEditable`): look up that type to see the fields.
- `XxxWithBodyWithResponse(ctx, ..., contentType, io.Reader)`: send a raw body (multipart, or the union bodies below).
- `XxxResponse`: the result, with `StatusCode()`, `Status()`, raw `Body []byte`, and typed fields per status (`JSON200`, `Headers201`, `JSON401`...).

Search instead of reading the file (it is more than 60,000 lines). From a module that depends on the library, locate it first:

```bash
GEN="$(go list -m -f '{{.Dir}}' github.com/elabftw/elabapi-go/v6)/elabapi/client.gen.go"
grep -n 'func (c \*ClientWithResponses) .*Experiment.*WithResponse(' "$GEN"
grep -n '^type PostExperimentJSONRequestBody' "$GEN"   # = EntityEditable
grep -n '^type EntityEditable struct' -A 40 "$GEN"
```

Entity endpoints take the entity type as a plain string path parameter: `"experiments"`, `"items"`, `"experiments_templates"`, `"items_types"`.

## Checking responses

`err != nil` only means the request failed (network, timeout, TLS). An answer from the server is never an `err`, even a 4xx or 5xx: always check the response.

| Success | Check |
|---|---|
| 200 with JSON | `resp.JSON200 != nil` (nil on any other status or bad JSON) |
| 201 created | `resp.Headers201 != nil && resp.Headers201.Location != nil`; the new id is the last path element of `Location` (see `getIdFromUrl` in the examples) |
| 204 no content (delete...) | `resp.StatusCode() == http.StatusNoContent` |
| 200 non-JSON (file download) | `resp.StatusCode() == http.StatusOK`, content in `resp.Body` |

On failure, include `resp.Status()` and `resp.Body` in the error: eLabFTW explains the problem there.

## Pitfalls

1. **Optional fields are pointers.** Take the address of a variable (`title := "x"; Title: &title`); a nil field is left out of the JSON. Check for nil before dereferencing fields of a response.
2. **Union request bodies are sent as `{}` if passed directly.** These request bodies are unions (`anyOf`/`oneOf` in the spec): `PatchContainer`, `PatchEvent`, `PatchExperiment`, `PatchExperimentTemplate`, `PatchItem`, `PatchItemsType`, `PatchTeamgroup`, `PatchTeam`, `PatchUser`, `PostCompounds`, `PostInstance`, `PostTag`. Their `XxxJSONRequestBody` type does not get the union's `MarshalJSON`, so the server receives an empty object and the call silently does nothing. Build the `XxxJSONBody` with one of its `From...` methods, `json.Marshal` it, and send it with `XxxWithBodyWithResponse(ctx, ..., "application/json", bytes.NewReader(payload))`. See step 3 of example 01.
3. **Union responses** (for example `GET /info`) hold raw JSON: pick the expected shape with an `As...` method. It does not validate the shape: fields that do not match stay nil.
4. **Uploads** are `multipart/form-data`: build the body with `mime/multipart`, close the writer, and pass `mw.FormDataContentType()` as the content type (example 02).
5. **Downloads**: set `Format` to `ReadUploadParamsFormatBinary`, otherwise the API returns the JSON description of the upload. `resp.Body` holds the whole file in memory; for large files call `client.ReadUpload` (same arguments, without `WithResponse`), which returns the `*http.Response`: stream its `Body` to disk and close it.
6. **Timeouts**: the context deadline covers every request that uses it. Size it for the whole run, or create one context per request in long loops.

## Working with researchers' data

eLabFTW holds lab records, often with legal or regulatory value. For scripts that write:

- Test against a development instance or a few entries before a bulk run.
- For bulk PATCH, DELETE or imports, add a dry-run mode that prints what would change, and ask the user to confirm before writing.
- Prefer locking or archiving to deleting, and tell the user when an action cannot be undone (deleting uploads, timestamped or signed entries).
- Log the id of every entry changed, so the run can be audited.

## Working on this repository

- Never edit `elabapi/client.gen.go`: it is regenerated at each release (see the Dev section of [README.md](README.md)). Fix generation issues through `oapi-codegen.yaml` or upstream in the eLabFTW spec.
- Examples are part of the module: `go build ./...` and `go vet ./...` must pass.
