package country

import (
	"context"
	"envdash/internal/client/country"
	"envdash/internal/structs"
	"fmt"
	"strings"
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

// GetCountry retrieves country information by either an ISO 3166-1 alpha-2/
// code or a country name. The input form is detected automatically: 2-letter iso code, and everything else as a name.
// Names have a minimum length of 3.
//
// Parameters:
//   - ctx: request context for cancellation and timeouts
//   - query: country ISO code (e.g. "NO", "US") or name (e.g. "Norway")
//
// Returns the first matching country, or an error if no match is found.
func (ci *CountryInternal) GetCountry(ctx context.Context, query string) (*structs.IncomingCountry, error) {
	query = strings.TrimSpace(query) //to remove space, removed so len() works
	if query == "" {
		return nil, fmt.Errorf("country query is required")
	}

	var countries []structs.IncomingCountry
	var err error

	switch {
	case len(query) == 2:
		countries, err = ci.client.FetchByCode(ctx, strings.ToUpper(query))
	case len(query) > 2:
		countries, err = ci.client.FetchByName(ctx, query)
	default:
		return nil, fmt.Errorf("country query %q is too short: must be a 2-letter ISO code or a full name", query)
	}

	if err != nil {
		return nil, fmt.Errorf("getting country %q: %w", query, err)
	}
	if len(countries) == 0 {
		return nil, fmt.Errorf("no country data returned for %q", query)
	}
	return &countries[0], nil
}
