package services

import (
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
	"net/http"
	"time"
)

// get the start time
type StatusInternal struct {
	startTime time.Time
}

// StatusService
// start the status service, and sets startTime
// used as a receiver to organize the status related methods
// Need to access the start time
func StatusService(startTime time.Time) *StatusInternal {
	return &StatusInternal{startTime: startTime}
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
		Db_noti:      0,
		Version:      config.APPLICATION_VERSION,
		Uptime:       fmt.Sprintf("%.f", time.Since(s.startTime).Seconds()),
	}
}

// probeHeadEndPoint
// sends a head request to the endpoint to get the status
// only useful for endpoints that support it
func (s *StatusInternal) probeHeadEndPoint(url string) (int, error) {
	resp, err := http.Head(url)
	if err != nil {
		return http.StatusServiceUnavailable, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

// getStatus
// gets the status code, with configurable header key and header value
// some endpoint need this to allow for usage
// todo: check if switch to HEAD method works
func (s *StatusInternal) getStatus(url string, headerKey string, headerValue string) int {
	client := http.Client{}

	//check bad url
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return http.StatusServiceUnavailable
	}

	// set api key header if provided
	if headerValue != "" {
		req.Header.Set("X-API-Key", headerValue)
	}

	// set custom header if provided
	if headerKey != "" {
		req.Header.Set(headerKey, headerValue)
	}

	//check no request sent
	res, errRequest := client.Do(req)
	if errRequest != nil {
		return http.StatusServiceUnavailable
	}
	defer res.Body.Close()

	return res.StatusCode
}

// probeAllEndpoints
// checks all the endpoints using get probeHeadEndPoint and getStatus
// returns a map of the status codes for all endpoints
func (s *StatusInternal) probeAllEndpoints() map[string]int {
	endpoints := map[string]string{
		//cannot have the other methods as they need custom headers or Head not implemented
		"countries": config.REST_COUNTRIES_API_PROBE,
		"metro":     config.METRO_API,
	}

	healthStatuses := make(map[string]int)
	for name, url := range endpoints {
		statusCode, err := s.probeHeadEndPoint(url)
		if err != nil {
			statusCode = http.StatusServiceUnavailable
		}
		healthStatuses[name] = statusCode
	}

	healthStatuses["openaq"] = s.getStatus(config.OPENAQ_PROBE, "", config.OPENAQ_KEY)
	healthStatuses["nominatim"] = s.getStatus(config.NOMINATIM_PROBE, "User-Agent", "evdash/")
	healthStatuses["currency"] = s.getStatus(config.CURRENCIES_API_PROBE, "", "")

	return healthStatuses
}
