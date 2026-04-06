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
	aqSer    *AQInternal
	nomSer   *NomInternal
}

// NewDashboardService returns a new DashBoardInternal instance with a configured HTTP client.
func NewDashboardService() *DashBoardInternal {
	return &DashBoardInternal{
		counSer:  country.NewCountryService(),
		curSer:   NewCurrencyService(),
		metroSer: NewMetroService(),
		aqSer:    NewAqService(),
		nomSer:   NewNomService(),
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

	nomData, err := dashI.nomSer.GetCapitalCords(countryInfo.Capital[0])
	if err != nil {
		return nil, err
	}

	//uses capital, change to  countryInfo.Coordinates[0], countryInfo.Coordinates[1] for centroid
	metroInfo, err := dashI.metroSer.GetMetro(nomData.Lat, nomData.Lon) //uses capital, change to
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

	aq, errAq := dashI.aqSer.GetAQ(nomData.Lat, nomData.Lon)
	if errAq != nil {
		log.Printf("Error with AQ: %s", errAq)
		return nil, errAq
	}

	//todo: add some logic to call on register
	//note: we use redunant structs to keep logic clean
	//ie Area: registration(id)
	return &structs.DashboardResponse{
		Country: countryInfo.Name.Common,
		IsoCode: countryInfo.IsoCode,
		Features: structs.Features{
			Temperature:   metroInfo.MeanTemperature,
			Precipitation: metroInfo.MeanPrecipitation,
			AirQuality:    *aq, //use the aqRespose struct directly
			Capital:       countryInfo.Capital[0],
			Coordinates: structs.CoordinateDetails{ //todo: decide on which to use nominatim (capital) or country (centroid)
				Latitude:  nomData.Lat,
				Longitude: nomData.Lon,
			},
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
