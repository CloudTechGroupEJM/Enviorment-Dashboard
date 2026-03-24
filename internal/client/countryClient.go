package client

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
	"net/http"
	"time"
)

type CountryClient struct {
	httpClient *http.Client
}

// NewCountryClient
// Creates a status client to execute HTTP calls.
func NewCountryClient() *CountryClient {
	return &CountryClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// FetchCountryData retrieves country information from the external country API
// for the given two-letter country code.
//
// Parameters:
//   - countryCode: a two-letter ISO 3166-1 alpha-2 country code (e.g. "NO", "US").
//
// Returns:
//   - []structs.IncomingCountry: a slice of country info structs populated with
//     the API response data.
//   - error: an error if the HTTP request fails or if the response body cannot be decoded.
func (cc *CountryClient) FetchCountryData(countryCode string) ([]structs.IncomingCountry, error) {

	resp, err := http.Get(config.REST_COUNTRIES_API + config.PATH_REST_ALPHA + countryCode)
	if err != nil {
		return nil, fmt.Errorf("error fetching country info: %w", err)
	}
	defer resp.Body.Close()

	var countryInfo []structs.IncomingCountry
	if err := json.NewDecoder(resp.Body).Decode(&countryInfo); err != nil {
		return nil, fmt.Errorf("error parsing country info: %w", err)
	}
	return countryInfo, nil
}
