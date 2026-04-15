package country

import (
	"context"
	"envdash/internal/client/country"
	"envdash/internal/structs"
	"fmt"
)

type CountryInternal struct {
	client *country.CountryClient
}

// NewCountryService
// Creates and returns a new CountryInternal service
// Used to talk to other services
func NewCountryService() *CountryInternal {
	return &CountryInternal{
		client: country.NewCountryClient(),
	}
}

// GetCountry
// Retrieves country information for the given two-letter country code.
// Returns the first country from the API response.
func (ci *CountryInternal) GetCountry(ctx context.Context, countryCode string) (*structs.IncomingCountry, error) {
	if countryCode == "" {
		return nil, fmt.Errorf("country code is required")
	}

	countries, err := ci.client.FetchCountryData(ctx, countryCode)
	if err != nil {
		return nil, fmt.Errorf("getting country %s: %w", countryCode, err)
	}
	if len(countries) == 0 {
		return nil, fmt.Errorf("no country data returned for %s", countryCode)
	}
	return &countries[0], nil
}
