package dashboard

import (
	"context"
	"envdash/internal/config"
	"envdash/internal/services/country"
	"envdash/internal/services/currency"
	"envdash/internal/services/metro"
	"envdash/internal/services/nominatim"
	"envdash/internal/services/openaq"
	"envdash/internal/services/registration"
	"envdash/internal/structs"
	"fmt"
	"log"
	"sort"
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
// Failure policy:
//   - Foundational fetches (registration, country, geocoding) abort the request
//     because downstream data depends on them.
//   - Optional fetches (metro, air quality, currencies) degrade gracefully:
//     errors are logged and the affected fields are simply omitted.
//
// Parameters:
//   - ctx: request context for cancellation and timeouts
//   - id: the document ID of the user's registered dashboard in Firestore
//
// Returns:
//   - *structs.DashboardResponse: the aggregated dashboard data
//   - error: if registration is not found or critical foundational data is unavailable
func (d *DashBoardInternal) GetDashboard(ctx context.Context, id string) (*structs.DashboardResponse, error) {
	reg, err := d.firebase.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	feat := reg.Features

	countryData, err := d.fetchCountry(ctx, reg.IsoCode, feat)
	if err != nil {
		return nil, err
	}

	nomiData, err := d.fetchNominatim(ctx, countryData, feat)
	if err != nil {
		return nil, err
	}

	metroData := d.fetchMetro(ctx, nomiData, feat)
	aqData := d.fetchAirQuality(ctx, nomiData, feat)
	currencies := d.fetchCurrencies(ctx, countryData, feat)

	return buildDashboard(reg, countryData, nomiData, metroData, aqData, currencies), nil
}

// --- feature gates ---
// Used to separate if a given registration actually needs to request from the external APIs.
// If a feature is reliant on data from another api, both calls need to be made.

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

// fetchCountry makes an API call to the country service to get foundational country data
// if the user requested fields that depend on it. Returns nil without error if unneeded.
func (d *DashBoardInternal) fetchCountry(ctx context.Context, iso string, f structs.BoolFeature) (*structs.IncomingCountry, error) {
	if !needsCountry(f) {
		return nil, nil
	}
	info, err := d.counSer.GetCountry(ctx, iso)
	if err != nil {
		return nil, fmt.Errorf("country lookup for %s: %w", iso, err)
	}
	if info == nil {
		return nil, fmt.Errorf("country %s not found", iso)
	}
	return info, nil
}

// fetchNominatim geocodes the given country's capital city to absolute coordinates.
// It is bypassed if coordinates aren't needed. It returns an error if geocoding fails
// because sequential steps strictly depend on these parameters.
func (d *DashBoardInternal) fetchNominatim(ctx context.Context, c *structs.IncomingCountry, f structs.BoolFeature) (*structs.NomResponse, error) {
	if !needsCoords(f) {
		return nil, nil
	}
	if c == nil || len(c.Capital) == 0 {
		return nil, fmt.Errorf("cannot geocode: missing capital")
	}
	info, err := d.nomSer.GetCapitalCoords(ctx, c.Capital[0])
	if err != nil {
		return nil, fmt.Errorf("geocoding %s: %w", c.Capital[0], err)
	}
	if info == nil {
		return nil, fmt.Errorf("geocoding %s: no result", c.Capital[0])
	}
	return info, nil
}

// fetchMetro retrieves meteorological data (weather) using provided coordinates.
// Failures to fetch this data are swallowed and logged, allowing for graceful partial degradation.
func (d *DashBoardInternal) fetchMetro(ctx context.Context, n *structs.NomResponse, f structs.BoolFeature) *structs.MetroResponse {
	if n == nil || !(f.Temperature || f.Precipitation) {
		return nil
	}
	info, err := d.metroSer.GetMetro(ctx, n.Lat, n.Lon)
	if err != nil {
		log.Printf("metro fetch failed: %v", err)
		return nil
	}
	return info
}

// fetchAirQuality retrieves localized air quality conditions using provided coordinates.
// Failures to fetch this data are swallowed and logged, returning nil.
func (d *DashBoardInternal) fetchAirQuality(ctx context.Context, n *structs.NomResponse, f structs.BoolFeature) *structs.AqResponse {
	if n == nil || !f.AirQuality {
		return nil
	}
	info, err := d.aqSer.GetAQ(ctx, n.Lat, n.Lon)
	if err != nil {
		log.Printf("air quality fetch failed: %v", err)
		return nil
	}
	return info
}

// fetchCurrencies retrieves currency conversion info between a primary currency
// and a list of target currencies. Failures are handled gracefully by returning nil.
func (d *DashBoardInternal) fetchCurrencies(ctx context.Context, c *structs.IncomingCountry, f structs.BoolFeature) *structs.CurrencyResponse {
	if c == nil || len(f.TargetCurrencies) == 0 {
		return nil
	}
	base := firstCurrency(c.Currencies)
	if base == "" {
		log.Printf("currency fetch skipped: no base currency for country")
		return nil
	}
	info, err := d.curSer.GetCurrency(ctx, base, f.TargetCurrencies)
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
		LastRetrieval: time.Now().UTC().Format(config.DATE_FORMAT),
	}
}

// firstCurrency returns the alphabetically-first currency code from the map.
// Sorting ensures deterministic output across calls — Go map iteration order
// is randomized, so without this two requests for the same country could
// pick different base currencies.
func firstCurrency(m map[string]struct{}) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys[0]
}
