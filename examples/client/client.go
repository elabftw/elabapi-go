// This an example to copy-paste do not import as a dependency

package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	// to import a custom dependency, use the `go` command line
	// here : go get github.com/elabftw/elabapi-go/v6@latest (in your go module directory)
	"github.com/elabftw/elabapi-go/v6/elabapi"
)

// New builds an eLabFTW API client, configured from environment variables:
//   - API_KEY: the API key sent in the Authorization header (the default is only a value for local testing)
//   - API_HOST: the base URL of the API, including /api/v2
//   - CA_PATH and INSECURE: TLS options, see tlsConfig
//
// It returns a ClientWithResponses: its ...WithResponse methods read the HTTP response for you
// and fill a typed struct (see the examples that use it).
func New() (*elabapi.ClientWithResponses, error) {
	apiKey := getEnvOrDefault("API_KEY", "apiKey4Test")
	host := getEnvOrDefault("API_HOST", "https://elab.localhost:3148/api/v2")

	tlsCfg, err := tlsConfig()
	if err != nil {
		return nil, err
	}
	// Clone() keeps the default behavior (proxy from environment, timeouts)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsCfg

	// A request editor is a function the client calls on every request just before sending it.
	// Here it adds the API key, so we do not have to set the header in each call.
	setAuth := func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", apiKey)
		return nil
	}

	// WithHTTPClient makes the client use our transport (TLS settings above),
	// WithRequestEditorFn registers setAuth.
	return elabapi.NewClientWithResponses(
		host,
		elabapi.WithHTTPClient(&http.Client{Transport: transport}),
		elabapi.WithRequestEditorFn(setAuth),
	)
}

// tlsConfig verifies server certificates by default.
// CA_PATH: PEM file of an extra CA to trust (added to the system ones).
// INSECURE=1: skip verification entirely (development only).
func tlsConfig() (*tls.Config, error) {
	cfg := &tls.Config{InsecureSkipVerify: os.Getenv("INSECURE") == "1"}

	caPath := os.Getenv("CA_PATH")
	if caPath == "" {
		return cfg, nil
	}
	pem, err := os.ReadFile(caPath)
	if err != nil {
		return nil, err
	}
	// Start from the system's trusted CAs and add ours, instead of replacing them.
	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("no valid certificate found in %s", caPath)
	}
	cfg.RootCAs = pool

	return cfg, nil
}

// getEnvOrDefault returns the value of the environment variable k, or def if it is unset or empty.
func getEnvOrDefault(k string, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
