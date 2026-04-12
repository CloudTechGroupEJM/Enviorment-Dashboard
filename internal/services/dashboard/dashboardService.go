package dashboard

import (
	"context"
	"envdash/internal/services/country"
	"envdash/internal/services/currency"
	"envdash/internal/services/metro"
	"envdash/internal/services/nominatim"
	"envdash/internal/services/openaq"
	"envdash/internal/services/registration"
	"envdash/internal/structs"
	"errors"
	"log"
	"time"

	"cloud.google.com/go/firestore"
)

// DashBoardInternal is the internal implementation of the dashboard service,
// wrapping all services used by the dashboard.
type DashBoardInternal struct {
	counSer  *country.CountryInternal
	curSer   *currency.CurrencyInternal
	metroSer *metro.MetroInternal
	aqSer    *openaq.AQInternal
	nomSer   *nominatim.NomInternal
	firebase *registration.RegistrationService
}

// NewDashboardService returns a new DashBoardInternal instance with a configured HTTP client.
func NewDashboardService(client *firestore.Client) *DashBoardInternal {
	return &DashBoardInternal{
		counSer:  country.NewCountryService(),
		curSer:   currency.NewCurrencyService(),
		metroSer: metro.NewMetroService(),
		aqSer:    openaq.NewAqService(),
		nomSer:   nominatim.NewNomService(),
		firebase: registration.NewRegistrationService(client),
	}
}

func (dashI *DashBoardInternal) GetDashboard(id string) (*structs.DashboardResponse, error) {
	// 1. Get user preferences
	reg, err := dashI.firebase.GetByID(id, context.Background())
	if err != nil {
		return nil, err
	}

	// 2. Country data
	countryData, err := dashI.fetchCountryData(reg.IsoCode, reg.Features)
	if err != nil {
		return nil, err
	}

	// 3. Nominatim data (needs capital from countryData)
	nomiData, err := dashI.fetchNominatim(countryData, reg.Features)
	if err != nil {
		return nil, err
	}

	// 4. Concurrently fetch external data requiring coordinates/currencies
	metroData := dashI.fetchMetroData(nomiData, reg.Features)
	aqData := dashI.fetchAirQuality(nomiData, reg.Features)
	currencies := dashI.fetchCurrencies(countryData, reg.Features)

	// 5. Construct Final Response cleanly
	return buildDashboard(reg, countryData, nomiData, metroData, aqData, currencies), nil
}

// fetchCountryData only calls the API if any country-derived field is requested,
// or if a downstream call (nominatim → metro/aq, or currency) needs it.
func (dashI *DashBoardInternal) fetchCountryData(isoCode string, requested structs.BoolFeature) (*structs.IncomingCountry, error) {
	needsData := requested.Capital || requested.Area || len(requested.TargetCurrencies) > 0 ||
		requested.Temperature ||
		requested.Precipitation || requested.AirQuality || requested.Coordinates //todo: uncomment requested.Population
	if !needsData {
		return nil, nil
	}

	info, err := dashI.counSer.GetCountry(isoCode)
	if err != nil || info == nil {
		log.Printf("Failed to get country %s: %v", isoCode, err)
		return nil, errors.New("country not found")
	}
	return info, nil
}

// fetchNominatim only calls the API if a coordinate-derived field is requested.
func (dashI *DashBoardInternal) fetchNominatim(countryData *structs.IncomingCountry, requested structs.BoolFeature) (*structs.NomResponse, error) {
	needsData := requested.Temperature || requested.Precipitation || requested.AirQuality || requested.Coordinates
	if !needsData {
		return nil, nil
	}
	if countryData == nil || len(countryData.Capital) == 0 {
		return nil, errors.New("cannot geocode: missing capital")
	}

	info, err := dashI.nomSer.GetCapitalCords(countryData.Capital[0])
	if err != nil || info == nil {
		log.Printf("Failed to geocode capital %s: %v", countryData.Capital[0], err)
		return nil, errors.New("geocoding failed")
	}
	return info, nil
}

// fetchMetroData only calls the API if temperature or precipitation is requested.
func (dashI *DashBoardInternal) fetchMetroData(nomiData *structs.NomResponse, requested structs.BoolFeature) *structs.MetroResponse {
	needsData := requested.Temperature || requested.Precipitation
	if !needsData || nomiData == nil {
		return nil
	}

	info, err := dashI.metroSer.GetMetro(nomiData.Lat, nomiData.Lon)
	if err != nil {
		log.Printf("Failed to get metro data: %v", err)
		return nil
	}
	return info
}

// fetchAirQuality only calls the API if air quality is requested.
func (dashI *DashBoardInternal) fetchAirQuality(nomiData *structs.NomResponse, requested structs.BoolFeature) *structs.AqResponse {
	if !requested.AirQuality || nomiData == nil {
		return nil
	}

	info, err := dashI.aqSer.GetAQ(nomiData.Lat, nomiData.Lon)
	if err != nil {
		log.Printf("Failed to get air quality: %v", err)
		return nil
	}
	return info
}

// fetchCurrencies only calls the API if target currencies are requested.
func (dashI *DashBoardInternal) fetchCurrencies(countryData *structs.IncomingCountry, requested structs.BoolFeature) *structs.CurrencyResponse {
	if len(requested.TargetCurrencies) <= 0 {
		return nil
	}

	code := firstCurrency(countryData.Currencies)
	info, err := dashI.curSer.GetCurrency(code, []string{"USD", "EUR"})
	if err != nil {
		log.Printf("Failed to get currencies: %v", err)
		return nil
	}
	return info
}

// buildDashboard assembles the final response, populating only the features
// that were requested and successfully fetched.
func buildDashboard(
	reg *structs.RegisterCountry,
	countryData *structs.IncomingCountry,
	nomiData *structs.NomResponse,
	metroData *structs.MetroResponse,
	aqData *structs.AqResponse,
	currencies *structs.CurrencyResponse,
) *structs.DashboardResponse {
	features := structs.Features{}

	if metroData != nil {
		features.Temperature = metroData.MeanTemperature     // only non-nil if requested
		features.Precipitation = metroData.MeanPrecipitation // but see caveat below
	}

	if aqData != nil {
		features.AirQuality = *aqData
	}

	if currencies != nil {
		features.TargetCurrencies = currencies.TargetCurrencies
	}

	if reg.Features.Coordinates && nomiData != nil {
		features.Coordinates = structs.CoordinateDetails{
			Latitude:  nomiData.Lat,
			Longitude: nomiData.Lon,
		}
	}

	if countryData != nil {
		// if reg.Features.Population {
		//     features.Population = countryData.Population
		// }
		if reg.Features.Area {
			features.Area = countryData.Area
		}
	}

	if len(reg.Features.TargetCurrencies) > 0 && currencies != nil {
		features.TargetCurrencies = currencies.TargetCurrencies
	}

	return &structs.DashboardResponse{
		Country:       reg.Name,
		IsoCode:       reg.IsoCode,
		Features:      features,
		LastRetrieval: time.Now().Format(time.DateTime),
	}
}

// firstCurrency
// gets the first currency in the currency map
func firstCurrency(currencyMap map[string]struct{}) string {
	for code := range currencyMap {
		return code
	}
	return "" // return empty string if map is empty
}
