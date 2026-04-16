package handlers

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/services/status"
	"log"
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

		// go to service layer to get status
		statusEndpoints := service.GetStatus()

		// Encode to memory first
		encodedData, err := json.Marshal(statusEndpoints)
		if err != nil {
			log.Printf("JSON Marshal error: %v", err)
			//Return internal server error if marshalling fails
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Only set headers and write if encoding was successful
		w.Header().Set(config.HEADER_CONTENT_TYPE, config.APPLICATION_JSON)
		w.Write(encodedData)
	}
}
