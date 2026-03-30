package services

import (
	"envdash/internal/services/country"
	"envdash/internal/structs"
	"log"
)

// DashBoardInternal is the internal implementation of the dashbaord service,
// wrapping all services used by the dashboard.
type DashBoardInternal struct {
	counSer  *country.CountryInternal
	curSer   *CurrencyInternal
	metroSer *MetroInternal
	//aqSer
}

// NewDashboardService returns a new DashBoardInternal instance with a configured HTTP client.
func NewDashboardService() *DashBoardInternal {
	return &DashBoardInternal{
		counSer:  country.NewCountryService(),
		curSer:   NewCurrencyService(),
		metroSer: NewMetroService(),
		//aqSer:    NewAqService,
	}
}

// GetDashboard
// Used the other services to populate the dashboard fields, and returns the filled struct
func (dashI *DashBoardInternal) GetDashboard(country string) (*structs.DashboardResponse, error) {
	// Get country info and handle errors
	countryInfo, err := dashI.counSer.GetCountry(country)
	if err != nil {
		log.Printf("Error getting country info for %s: %v", country, err)
		return nil, err
	}
	if countryInfo == nil {
		log.Printf("Country not found: %s", country)
		return nil, err
	}

	metroInfo, err := dashI.metroSer.GetMetro(countryInfo.Coordinates[0], countryInfo.Coordinates[1])
	if err != nil {
		log.Printf("Error getting metro info: %v", err)
		return nil, err
	}

	// Get currency code
	currencyCode := firstCurrency(countryInfo.Currencies)
	if currencyCode == "" {
		log.Printf("No currency found for country: %s", country)
		return nil, nil
	}

	// Get currency info and handle errors
	curInfo, err := dashI.curSer.GetCurrency(currencyCode, []string{"USD", "EUR"})
	if err != nil {
		log.Printf("Error getting currency info: %v", err)
		return nil, err
	}
	if curInfo == nil {
		log.Printf("No currency data returned for: %s", currencyCode)
		return nil, nil
	}

	//todo: add some logic to call on register
	//ie Area: registration(id)
	return &structs.DashboardResponse{
		Country: countryInfo.Name.Common,
		IsoCode: countryInfo.IsoCode,
		Features: structs.Features{
			Temperature:      metroInfo.MeanTemperature,
			Precipitation:    metroInfo.MeanPrecipitation,
			AirQuality:       nil,
			Capital:          countryInfo.Capital[0],
			Coordinates:      nil,
			Population:       countryInfo.Population,
			Area:             countryInfo.Area,
			TargetCurrencies: curInfo.TargetCurrencies,
		},
		LastRetrieval: "",
	}, nil
}

// firstCurrency
// gets the first currency in the currency map
func firstCurrency(currencyMap map[string]struct{}) string {
	for code := range currencyMap {
		return code
	}
	return "" // return empty string if map is empty
}
