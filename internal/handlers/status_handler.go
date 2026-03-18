package handlers

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/services"
	"net/http"
	"time"
)

/*
Routes to the handler for the status endpoint
*/
func StatusRouter(router *http.ServeMux) {
	service := services.StatusService(time.Now())
	router.HandleFunc(config.STATUS_PAGE_PATH, statusHandler(service))
}

/*
Handles GET request to the /countryinfo/v1/status endpoint
*/
func statusHandler(service *services.StatusInternal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		healthStatuses := probeAllEndpoints()

		status := service.GetStatus(healthStatuses)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}
}

// sends a head request to the endpoint to get the status
func probeHeadEndPoint(url string) (int, error) {
	resp, err := http.Head(url)
	if err != nil {
		return http.StatusServiceUnavailable, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

// maybe return error, dont think its needed as if error set statuscode 503
// todo: check if switch to HEAD method works
func getStatus(url string, headerKey string, headerValue string) int {
	client := http.Client{}

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

	// runs GET request on URL
	res, err := client.Do(req)
	if err != nil {
		return http.StatusServiceUnavailable
	}
	defer res.Body.Close()

	return res.StatusCode
}

// returns a map of the status codes for all endpoints
func probeAllEndpoints() map[string]int {
	endpoints := map[string]string{
		//cannot have the other methods as they need custom headers or Head not implemented
		"countries": config.REST_COUNTRIES_API_PROBE,
		"metro":     config.METRO_API,
	}

	healthStatuses := make(map[string]int)
	for name, url := range endpoints {
		statusCode, err := probeHeadEndPoint(url)
		if err != nil {
			statusCode = http.StatusServiceUnavailable
		}
		healthStatuses[name] = statusCode
	}

	healthStatuses["openaq"] = getStatus(config.OPENAQ_PROBE, "", config.OPENAQ_KEY)
	healthStatuses["nominatim"] = getStatus(config.NOMINATIM_PROBE, "User-Agent", "evdash/")
	healthStatuses["currency"] = getStatus(config.CURRENCIES_API_PROBE, "", "")

	return healthStatuses
}
