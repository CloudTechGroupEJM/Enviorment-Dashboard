package services

import (
	"envdash/internal/services/country"
	"envdash/internal/structs"
	"log"
)

// StatusInternal
// get the start time
// holds a statusClient to get the HTTP functinaltiy
type DashBoardInternal struct {
	counSer *country.CountryInternal
	curSer  *CurrencyInternal
	//metroSer
	//aqSer
}

// StatusService
// start the status service, creates a client and sets startTime
// used as a receiver to organize the status related methods
// Needed to access the start time
func NewDashboardService() *DashBoardInternal {
	return &DashBoardInternal{
		counSer: country.NewCountryService(),
		curSer:  NewCurrencyService(),
		//metroSer: NewMetroService,
		//aqSer:    NewAqService,
	}
}

// GetStatus
// Construct the response for status endpoint
// todo: error handling _ ignores errors

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
			Temperature:      0,
			Precipitation:    0,
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

func firstCurrency(currencyMap map[string]struct{}) string {
	for code := range currencyMap {
		return code
	}
	return "" // return empty string if map is empty
}
