# elabapi-go

Go library for eLabFTW REST API.

# Description

This repository contains a Go library to interact with [eLabFTW](https://github.com/elabftw/elabftw) REST API v2. It uses [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) to generate it based on the OpenApi specification of [eLabFTW REST API v2](https://doc.elabftw.net/api/v2/).

# Installation

Library versions follow eLabFTW releases: `v6.0.9` is generated from the OpenAPI specification attached to the eLabFTW 6.0.9 release. The major version is part of the module path (`/v6`), as Go requires for v2 and above.

```bash
# from your Go module
go get github.com/elabftw/elabapi-go/v6@latest
```

`@latest` is the highest stable release. Pre-releases (such as `6.1.0-beta`) are only used if requested explicitly, for example `@v6.1.0-beta`.

The package name is `elabapi`:

```go
import "github.com/elabftw/elabapi-go/v6/elabapi"
```

The module is `github.com/elabftw/elabapi-go/v6` and the package lives in its `elabapi/` directory.

# Using an AI assistant

If you write your scripts with an AI assistant, ask it to read [AGENTS.md](https://github.com/elabftw/elabapi-go/blob/master/AGENTS.md) first: it explains how to use this library and its pitfalls.

# Dev

## Layout

Everything is on `master`, including the generated `elabapi/client.gen.go`: never edit it by hand, it is overwritten at each release. Go downloads module source from git tags, so the generated file has to be committed. The generator options are in `oapi-codegen.yaml`.

## Release

A release is made by the [Release workflow](.github/workflows/release.yml). Give it the eLabFTW version to generate from:

```bash
gh workflow run release.yml -f version=6.0.9
```

It generates the client from the `openapi.yaml` attached to that eLabFTW release (the specification is read from its URL, never stored here), commits the result on `master`, pushes the tag `v6.0.9` and creates the GitHub release. A version with a suffix (`-beta`, `-alpha`) is marked as a pre-release. A tag must never be deleted and recreated: the Go module proxy caches it, so ship a new patch version instead.

## Generate the library locally

The workflow does this; run it by hand to try a change to `oapi-codegen.yaml`. The generator is run with `go run` so its dependencies are not added to `go.mod`. The `/v6` of the module path must match the major version of the eLabFTW release.

```bash
VERSION=6.0.9
go mod edit -module github.com/elabftw/elabapi-go/v${VERSION%%.*}
go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.2 -config oapi-codegen.yaml https://github.com/elabftw/elabftw/releases/download/$VERSION/openapi.yaml
go mod tidy
```

The oapi-codegen version (`v2.8.2`) is set here and in the workflow (`OAPI_CODEGEN_VERSION`): change both together.
