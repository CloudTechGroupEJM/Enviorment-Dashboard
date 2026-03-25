package services

import (
	"envdash/internal/services/country"
	"envdash/internal/structs"
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
func (dashI *DashBoardInternal) GetDashboard(country string) *structs.DashboardResponse {
	countryInfo, _ := dashI.counSer.GetCountry(country)
	curInfo, _ := dashI.curSer.GetCurrency(firstCurrency(countryInfo.Currencies), []string{"USD", "EUR"})

	return &structs.DashboardResponse{
		Country: countryInfo.Name.Common,
		IsoCode: countryInfo.IsoCode,
		Features: structs.Features{
			Temperature:      0,
			Precipitation:    0,
			Capital:          countryInfo.Capital[0],
			Coordinates:      nil,
			Population:       countryInfo.Population,
			Area:             countryInfo.Area,
			TargetCurrencies: curInfo.TargetCurrencies,
		},
		LastRetrieval: "",
	}
}

func firstCurrency(currencyMap map[string]struct{}) string {
	for code := range currencyMap {
		return code
	}
	return "" // return empty string if map is empty
}
