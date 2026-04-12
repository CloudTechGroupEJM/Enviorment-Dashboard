package status

import (
	"envdash/internal/client/status"
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
	"net/http"
	"os"
	"time"
)

// StatusInternal
// get the start time
// holds a statusClient to get the HTTP functinaltiy
type StatusInternal struct {
	startTime time.Time
	client    *status.StatusClient
}

// StatusService
// start the status service, creates a client and sets startTime
// used as a receiver to organize the status related methods
// Needed to access the start time
func StatusService(startTime time.Time) *StatusInternal {
	return &StatusInternal{
		startTime: startTime,
		client:    status.NewStatusClient(),
	}
}

// GetStatus
// Construct the response for status endpoint
func (s *StatusInternal) GetStatus() *structs.StatusResponse {
	healthStatus := s.probeAllEndpoints()
	return &structs.StatusResponse{
		CountriesApi: healthStatus["countries"],
		MetroAPI:     healthStatus["metro"],
		AqAPI:        healthStatus["openaq"],
		Nominatim:    healthStatus["nominatim"],
		CurrencyAPI:  healthStatus["currency"],
		Db_noti:      0, //todo implement status of the firestore instance
		Version:      config.APPLICATION_VERSION,
		Uptime:       fmt.Sprintf("%.f", time.Since(s.startTime).Seconds()),
	}
}

// probeAllEndpoints
// checks all the endpoints using get probeHeadEndPoint and getStatus
// returns a map of the status codes for all endpoints
func (s *StatusInternal) probeAllEndpoints() map[string]int {
	endpoints := map[string]string{
		"countries": config.REST_COUNTRIES_API_PROBE,
		"metro":     config.METRO_API,
		"nominatim": config.NOMINATIM_PROBE,
	}

	healthStatuses := make(map[string]int)
	for name, url := range endpoints {
		statusCode, err := s.client.ProbeHeadEndpoint(url)
		if err != nil {
			statusCode = http.StatusServiceUnavailable
		}
		healthStatuses[name] = statusCode
	}

	healthStatuses["openaq"] = s.client.ProbeGetEndpoint(config.OPENAQ_PROBE, "", os.Getenv("OPEN_AQ_API_KEY"))
	healthStatuses["currency"] = s.client.ProbeGetEndpoint(config.CURRENCIES_API_PROBE, "", "")

	return healthStatuses
}
