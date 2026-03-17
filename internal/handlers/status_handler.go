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
func probeEndPoint(url string) (int, error) {
	resp, err := http.Head(url)
	if err != nil {
		return http.StatusServiceUnavailable, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func probeAllEndpoints() map[string]int {
	endpoints := map[string]string{
		"countries": config.REST_COUNTRIES_API_PROBE,
		"metro":     config.METRO_API,
		"openaq":    config.OPENAQ_API,
		"nominatim": config.NOMINATIM_API,
		"currency":  config.CURRENCIES_API,
	}

	healthStatuses := make(map[string]int)
	for name, url := range endpoints {
		statusCode, err := probeEndPoint(url)
		if err != nil {
			statusCode = http.StatusServiceUnavailable
		}
		healthStatuses[name] = statusCode
	}
	return healthStatuses
}
