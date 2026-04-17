// Package handlers contains the HTTP handlers and routing logic for the envdash application.
package handlers

import (
	"encoding/json"
	"envdash/internal/cache"
	"envdash/internal/config"
	"envdash/internal/services/apiKey"
	"envdash/internal/services/dashboard"
	"envdash/internal/structs"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
)

// dashboardService holds a package-level instance of DashBoardInternal to be used across dashboard requests.
var dashboardService *dashboard.DashBoardInternal

// DashboardRouter initializes the dashboard service and registers the dashboard endpoints
// onto the provided HTTP ServeMux. It injects the shared firestore client into the service.
//
// Parameters:
//   - router: The HTTP router (ServeMux) where the dashboard paths will be registered.
//   - client: The firestore client used for database operations.
func DashboardRouter(router *http.ServeMux, client *firestore.Client, dispatcher webhookDispatcher, apiKeyServiceInstance *apiKey.APIKeyService,cacheServiceInstance *cache.CacheService) {
	dashboardService = dashboard.NewDashboardService(client, cacheServiceInstance)
	// Register the dashboard endpoint. {p1} represents the path parameter (ID).

	//protected
	router.HandleFunc(config.DASHBOARDS_PAGE_PATH+"{id}",
		ProtectedRouteMiddleware(dashboardHandler(dashboardService, dispatcher), apiKeyServiceInstance, client))
}

// dashboardHandler creates and returns an HTTP handler function for processing dashboard requests.
// It ensures that only GET requests are allowed and utilizes the provided service to fetch
// dashboard data based on the path parameter "p1".
//
// Parameters:
//   - service: The internal dashboard service used to fetch dashboard data.
//
// Returns:
//   - http.HandlerFunc: A function that writes the dashboard JSON to the HTTP response,
//     or returns appropriate HTTP error codes (405 Method Not Allowed, 400 Bad Request, 404 Not Found) if issues occur.
func dashboardHandler(service *dashboard.DashBoardInternal, dispatcher webhookDispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Restrict to HTTP GET method only
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Call the service layer to retrieve dashboard data using the "id" path variable
		dashboardReceived, err := service.GetDashboard(r.Context(), r.PathValue("id"), r.Header.Get("x-api-key"))
		if err != nil {
			http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
			return
		}

		// If no data was found, return a 404 Not Found error
		if dashboardReceived == nil {
			http.Error(w, "Country not found", http.StatusNotFound)
			return
		}

		// Set the response Content-Type header and encode the retrieved struct into JSON
		dispatchDashboardWebhooks(dashboardReceived, dispatcher)
		w.Header().Set(config.HEADER_CONTENT_TYPE, config.APPLICATION_JSON)
		json.NewEncoder(w).Encode(dashboardReceived)
	}
}

// dispatchDashboardWebhooks is a helper function that dispatches webhooks based on the received dashboard data.
// It checks for the presence of the dispatcher and the dashboard data, extracts the ISO code,
// and dispatches lifecycle and threshold webhooks accordingly.
//
// Parameters:
//   - dashboardReceived: The dashboard data received from the service layer.
//   - dispatcher: The webhook dispatcher used to send notifications.
func dispatchDashboardWebhooks(dashboardReceived *structs.DashboardResponse, dispatcher webhookDispatcher) {
	if dispatcher == nil || dashboardReceived == nil {
		return
	}

	isoCode := strings.ToUpper(strings.TrimSpace(dashboardReceived.IsoCode))
	if isoCode == "" {
		return
	}

	dispatcher.DispatchLifecycleAsync(isoCode, structs.NotificationEventInvoke)

	if dashboardReceived.Features == nil {
		return
	}

	if dashboardReceived.Features.AirQuality != nil {
		dispatcher.DispatchThresholdAsync(isoCode, "pm25", dashboardReceived.Features.AirQuality.PM25)
		dispatcher.DispatchThresholdAsync(isoCode, "pm10", dashboardReceived.Features.AirQuality.PM10)
	}
	if dashboardReceived.Features.Temperature != nil {
		dispatcher.DispatchThresholdAsync(isoCode, "temperature", *dashboardReceived.Features.Temperature)
	}
	if dashboardReceived.Features.Precipitation != nil {
		dispatcher.DispatchThresholdAsync(isoCode, "precipitation", *dashboardReceived.Features.Precipitation)
	}
}
