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

	setAuth := func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", apiKey)
		return nil
	}

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

func getEnvOrDefault(k string, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
