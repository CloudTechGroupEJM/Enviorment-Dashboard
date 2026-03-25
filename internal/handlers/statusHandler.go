package handlers

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/services/status"
	"net/http"
	"time"
)

// Hold the status service to be used by the handler
var statusService *status.StatusInternal

// executes once, on runtime
func init() {
	statusService = status.StatusService(time.Now())
}

// StatusRouter
// Routes to the statusHandler with the application-wide status service
func StatusRouter(router *http.ServeMux) {
	router.HandleFunc(config.STATUS_PAGE_PATH, statusHandler(statusService))
}

// statusHandler
// handles GET requests for the status endpoint
// gives error on all other methods
func statusHandler(service *status.StatusInternal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		//go to service layer to get status
		status := service.GetStatus()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}
}
