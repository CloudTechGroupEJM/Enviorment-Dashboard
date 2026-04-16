package country

import (
	"context"
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

type CountryClient struct {
	httpClient *http.Client
}

// NewCountryClient
// Creates a country client to execute HTTP calls.
func NewCountryClient() *CountryClient {
	return &CountryClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// FetchByCode retrieves country information by ISO 3166-1 alpha-2.
//
// Parameters:
//   - ctx: request context for cancellation and timeouts
//   - code: ISO 3166-1 alpha-2  code (e.g. "NO", "US")
func (cc *CountryClient) FetchByCode(ctx context.Context, code string) ([]structs.IncomingCountry, error) {
	return cc.fetch(ctx, config.PATH_REST_ALPHA, code)
}

// FetchByName retrieves country information by name (full or partial match).
//
// Parameters:
//   - ctx: request context for cancellation and timeouts
//   - name: country name (e.g. "Norway")
func (cc *CountryClient) FetchByName(ctx context.Context, name string) ([]structs.IncomingCountry, error) {
	return cc.fetch(ctx, config.PATH_REST_NAME, name)
}

// fetch performs the GET + decode against the given path segment and value.
func (cc *CountryClient) fetch(ctx context.Context, pathSegment, value string) ([]structs.IncomingCountry, error) {
	cURL, err := countryUrl(pathSegment, value)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", cURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building country request: %w", err)
	}

	resp, err := cc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching country info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("country API returned %d: %s", resp.StatusCode, body)
	}

	var countryInfo []structs.IncomingCountry
	if err := json.NewDecoder(resp.Body).Decode(&countryInfo); err != nil {
		return nil, fmt.Errorf("decoding country response: %w", err)
	}
	return countryInfo, nil
}

// countryUrl builds a country API URL for the given path segment (/alpha, /name) and value.
func countryUrl(pathSegment, value string) (string, error) {
	u, err := url.ParseRequestURI(config.REST_COUNTRIES_API)
	if err != nil {
		return "", fmt.Errorf("invalid country base URL: %w", err)
	}
	u.Path = path.Join(u.Path, pathSegment, value)
	return u.String(), nil
}
