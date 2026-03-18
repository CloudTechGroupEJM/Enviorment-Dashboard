package handlers

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
	"net/http"
)

// GetCountryInfoStruct retrieves country information for the given two-letter country code and returns it as an OutgoingCountry struct.
//
// Parameters:
//   - countryCode: a two-letter ISO 3166-1 alpha-2 country code (e.g. "NO", "US").
//
// Returns:
//   - structs.OutgoingCountry: a struct containing the country's name, ISO code,
//     capital, coordinates, population, area, and base currency.
//   - error: an error if the country information cannot be retrieved or processed.
func GetCountryInfoStruct(countryCode string) (structs.OutgoingCountry, error) {
	countryData, err := fetchCountryData(countryCode)
	if err != nil {
		return structs.OutgoingCountry{}, err
	}
	var outgoingCountry structs.OutgoingCountry = populateOutgoingCountry(countryData[0])
	return outgoingCountry, nil
}

// fetchCountryData retrieves country information from the external country API
// for the given two-letter country code.
//
// Parameters:
//   - countryCode: a two-letter ISO 3166-1 alpha-2 country code (e.g. "NO", "US").
//
// Returns:
//   - []structs.IncomingCountry: a slice of country info structs populated with
//     the API response data.
//   - error: an error if the HTTP request fails or if the response body cannot be decoded.
func fetchCountryData(countryCode string) ([]structs.IncomingCountry, error) {
	resp, err := http.Get(config.REST_COUNTRIES_API + "alpha/" + countryCode)
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

// populateOutgoingCountry converts an IncomingCountry struct to an OutgoingCountry struct.
//
// Parameters:
//   - incoming: a struct containing the raw country information as received from the API.
//
// Returns:
//   - structs.OutgoingCountry: a struct containing the processed country information,
//     including name, ISO code, capital, coordinates, population, area, and base currency.
func populateOutgoingCountry(incoming structs.IncomingCountry) structs.OutgoingCountry {
	outgoing := structs.OutgoingCountry{
		Name:    incoming.Name.Common,
		IsoCode: incoming.IsoCode,
		Capital: incoming.Capital[0],
		Coordinates: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{
			Latitude:  incoming.Coordinates[0],
			Longitude: incoming.Coordinates[1],
		},
		Population: incoming.Population,
		Area:       incoming.Area,
		BaseCurrency: func() string {
			for currency := range incoming.Currencies {
				return currency
			}
			return ""
		}(),
	}
	return outgoing
}
