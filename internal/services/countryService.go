package services

import (
	"envdash/internal/client"
	"envdash/internal/structs"
)

type CountryInternal struct {
	client *client.CountryClient
}

// NewCountryService
// Creates and returns a new CountryInternal service
// Used to talk to other services
func NewCountryService() *CountryInternal {
	return &CountryInternal{
		client: client.NewCountryClient(),
	}
}

// GetCountry
// Retrieves country information for the given country code
// Returns the first country from the API response
func (ci *CountryInternal) GetCountry(countryCode string) (*structs.IncomingCountry, error) {
	countries, err := ci.processCountryData(countryCode)
	if err != nil {
		return nil, err
	}
	if len(countries) == 0 {
		return nil, nil
	}
	return &countries[0], nil
}

// processCountryData
// Receives country struct from external API
func (ci *CountryInternal) processCountryData(countryCode string) ([]structs.IncomingCountry, error) {
	return ci.client.FetchCountryData(countryCode)
}
