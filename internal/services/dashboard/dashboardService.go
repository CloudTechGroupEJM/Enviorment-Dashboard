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

// DashBoardInternal aggregates multiple specific data domain services to build
// the overall custom dashboard. It handles calls to country, currency,
// meteorological, air quality, geocoding, and database management services.
type DashBoardInternal struct {
	counSer  *country.CountryInternal
	curSer   *currency.CurrencyInternal
	metroSer *metro.MetroInternal
	aqSer    *openaq.AQInternal
	nomSer   *nominatim.NomInternal
	firebase *registration.RegistrationService
}

// NewDashboardService initializes and returns a new DashBoardInternal instance.
// It sets up all the sub-services needed for dashboard generation, injecting
// the provided Firestore client into the registration (database) service.
//
// Parameters:
//   - client: A pointer to the configured Firestore client.
//
// Returns:
//   - A pointer to the initialized DashBoardInternal service.
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

// GetDashboard retrieves a user's registered preferences and lazily fetches
// only the required information to populate a cohesive dashboard response.
//
// Parameters:
//   - id: The document ID of the user's registered dashboard in Firestore.
//
// Returns:
//   - *structs.DashboardResponse: The aggregated dashboard data struct.
//   - error: Returns an error if the registration isn't found or critical initial data (like country info) is unavailable.
func (d *DashBoardInternal) GetDashboard(id string) (*structs.DashboardResponse, error) {
	// Fetch user specifications from the database
	reg, err := d.firebase.GetByID(id, context.Background())
	if err != nil {
		return nil, err
	}
	feat := reg.Features

	// Fetch base country data depending on requirements
	countryData, err := d.fetchCountry(reg.IsoCode, feat)
	if err != nil {
		return nil, err
	}

	// Geocode the country's capital into coordinates if downstream data requires it
	nomiData, err := d.fetchNominatim(countryData, feat)
	if err != nil {
		return nil, err
	}

	// Concurrently independent fetches using retrieved foundational data (gracefully allowing nil returns)
	metroData := d.fetchMetro(nomiData, feat)
	aqData := d.fetchAirQuality(nomiData, feat)
	currencies := d.fetchCurrencies(countryData, feat)

	// Build the final struct view based on the returned data
	return buildDashboard(reg, countryData, nomiData, metroData, aqData, currencies), nil
}

// --- feature gates ---

// needsCountry determines if the country API must be called based on the requested features
// and dependencies of downstream services.
func needsCountry(f structs.BoolFeature) bool {
	return f.Capital || f.Area || f.Population || f.Coordinates ||
		f.Temperature || f.Precipitation || f.AirQuality ||
		len(f.TargetCurrencies) > 0
}

// needsCoords determines if latitude and longitude data is needed via the Geocoding API,
// either for direct display or as parameters for downstream coordinate-dependent APIs.
func needsCoords(f structs.BoolFeature) bool {
	return f.Coordinates || f.Temperature || f.Precipitation || f.AirQuality
}

// --- fetchers ---

// fetchCountry makes an API call to the country service to get foundational country data
// if the user requested fields that depend on it. Returns nil without error if unneeded.
func (d *DashBoardInternal) fetchCountry(iso string, f structs.BoolFeature) (*structs.IncomingCountry, error) {
	if !needsCountry(f) {
		return nil, nil
	}
	info, err := d.counSer.GetCountry(iso)
	if err != nil || info == nil {
		log.Printf("country lookup failed for %s: %v", iso, err)
		return nil, errors.New("country not found")
	}
	return info, nil
}

// fetchNominatim geocodes the given country's capital city to absolute coordinates.
// It is bypassed if coordinates aren't needed. It returns an error if geocoding fails
// because sequential steps strictly depend on these parameters.
func (d *DashBoardInternal) fetchNominatim(c *structs.IncomingCountry, f structs.BoolFeature) (*structs.NomResponse, error) {
	if !needsCoords(f) {
		return nil, nil
	}
	if c == nil || len(c.Capital) == 0 {
		return nil, errors.New("cannot geocode: missing capital")
	}
	info, err := d.nomSer.GetCapitalCords(c.Capital[0])
	if err != nil || info == nil {
		log.Printf("geocoding failed for %s: %v", c.Capital[0], err)
		return nil, errors.New("geocoding failed")
	}
	return info, nil
}

// fetchMetro retrieves meteorological data (weather) using provided coordinates.
// Failures to fetch this data are swallowed and logged, allowing for graceful partial degradation.
func (d *DashBoardInternal) fetchMetro(n *structs.NomResponse, f structs.BoolFeature) *structs.MetroResponse {
	if n == nil || !(f.Temperature || f.Precipitation) {
		return nil
	}
	info, err := d.metroSer.GetMetro(n.Lat, n.Lon)
	if err != nil {
		log.Printf("metro fetch failed: %v", err)
		return nil
	}
	return info
}

// fetchAirQuality retrieves localized air quality conditions using providing coordinates.
// Failures to fetch this data are swallowed and logged, returning nil.
func (d *DashBoardInternal) fetchAirQuality(n *structs.NomResponse, f structs.BoolFeature) *structs.AqResponse {
	if n == nil || !f.AirQuality {
		return nil
	}
	info, err := d.aqSer.GetAQ(n.Lat, n.Lon)
	if err != nil {
		log.Printf("air quality fetch failed: %v", err)
		return nil
	}
	return info
}

// fetchCurrencies retrieves currency conversion info between a primary currency
// and a list of target currencies. Failures are handled gracefully by returning nil.
func (d *DashBoardInternal) fetchCurrencies(c *structs.IncomingCountry, f structs.BoolFeature) *structs.CurrencyResponse {
	if c == nil || len(f.TargetCurrencies) == 0 {
		return nil
	}
	info, err := d.curSer.GetCurrency(firstCurrency(c.Currencies), f.TargetCurrencies)
	if err != nil {
		log.Printf("currency fetch failed: %v", err)
		return nil
	}
	return info
}

// --- response assembly ---

// buildDashboard maps the individually fetched responses into a single combined
// DashboardResponse payload. It strictly validates whether a piece of data
// was historically requested by the user, and if so cleanly assigns it.
//
// Unrequested features, or features for which failures gracefully bypassed assignment,
// will remain unpopulated/omitted in the JSON output depending on their struct tags.
func buildDashboard(
	reg *structs.RegisterCountry,
	country *structs.IncomingCountry,
	nom *structs.NomResponse,
	metro *structs.MetroResponse,
	aq *structs.AqResponse,
	cur *structs.CurrencyResponse) *structs.DashboardResponse {
	feats := &structs.Features{}
	req := reg.Features

	if metro != nil {
		temp := metro.MeanTemperature
		precip := metro.MeanPrecipitation
		feats.Temperature = &temp
		feats.Precipitation = &precip
	}
	if aq != nil {
		feats.AirQuality = aq
	}
	if cur != nil {
		feats.TargetCurrencies = cur.TargetCurrencies
	}
	if req.Coordinates && nom != nil {
		feats.Coordinates = &structs.CoordinateDetails{Latitude: nom.Lat, Longitude: nom.Lon}
	}
	if country != nil {
		if req.Population {
			feats.Population = country.Population
		}
		if req.Area {
			feats.Area = country.Area
		}
		if req.Capital && len(country.Capital) > 0 {
			feats.Capital = country.Capital[0]
		}
	}

	return &structs.DashboardResponse{
		Country:       reg.Name,
		IsoCode:       reg.IsoCode,
		Features:      feats,
		LastRetrieval: time.Now().Format(time.DateTime),
	}
}

// firstCurrency extracts the first currency code found in a currency mapping.
// Returns an empty string if the map is empty.
func firstCurrency(m map[string]struct{}) string {
	for code := range m {
		return code
	}
	return ""
}
