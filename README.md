# elabapi-go

Go library for eLabFTW REST API.

# Description

This repository contains a Go library to interact with [eLabFTW](https://github.com/elabftw/elabftw) REST API v2. It uses [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) to generate it based on the OpenApi specification of [eLabFTW REST API v2](https://doc.elabftw.net/api/v2/).

# Installation

```bash
# from your Go module
go get github.com/elabftw/elabapi-go
```

The package name is `elabapi`:

```go
import "github.com/elabftw/elabapi-go"
```

# Dev

## Get the OpenAPI specification

Download the `openapi.yaml` file attached to an eLabFTW release (here 5.6.15):

```bash
curl -fsSL -o openapi.yaml https://github.com/elabftw/elabftw/releases/download/5.6.15/openapi.yaml
```

Note: the 5.6.15 specification has an invalid type that oapi-codegen rejects. After downloading, replace `type: int` with `type: integer` for the `source` property of the `idp` schema.

## Generate the library

The options for the generator are in `oapi-codegen.yaml`. It is run with `go run` so its dependencies are not added to `go.mod`.

```bash
go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.2 -config oapi-codegen.yaml openapi.yaml
go mod tidy
```
