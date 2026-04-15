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

// FetchCountryData retrieves country information from the external country API
// for the given two-letter country code.
//
// Parameters:
//   - ctx: request context for cancellation and timeouts
//   - countryCode: a two-letter ISO 3166-1 alpha-2 country code (e.g. "NO", "US").
//
// Returns:
//   - []structs.IncomingCountry: a slice of country info structs populated with
//     the API response data.
//   - error: an error if the HTTP request fails or if the response body cannot be decoded.
func (cc *CountryClient) FetchCountryData(ctx context.Context, countryCode string) ([]structs.IncomingCountry, error) {
	cURL, err := countryUrl(countryCode)
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

	// check response code
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

// countryUrl builds the country lookup URL from base + alpha path + code
func countryUrl(countryCode string) (string, error) {
	u, err := url.ParseRequestURI(config.REST_COUNTRIES_API)
	if err != nil {
		return "", fmt.Errorf("invalid country base URL: %w", err)
	}
	u.Path = path.Join(u.Path, config.PATH_REST_ALPHA, countryCode)
	return u.String(), nil
}
