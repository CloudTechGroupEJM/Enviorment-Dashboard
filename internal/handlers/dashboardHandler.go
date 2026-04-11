package handlers

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/services/dashboard"
	"net/http"
)

// todo: this is just a test
// todo: fix to work with the configuration of dashboard
var dashboardService *dashboard.DashBoardInternal

func init() {
	dashboardService = dashboard.NewDashboardService()
}

func DashboardRouter(router *http.ServeMux) {
	router.HandleFunc(config.DASHBOARDS_PAGE_PATH+"{p1}", dashboardHandler(dashboardService))
}

func dashboardHandler(service *dashboard.DashBoardInternal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		//go to service layer to get status
		dashboardRecived, err := service.GetDashboard(r.PathValue("p1"))
		if err != nil {
			http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if dashboardRecived == nil {
			http.Error(w, "Country not found", http.StatusNotFound)
			return
		}

		w.Header().Set(config.HEADER_CONTENT_TYPE, config.APPLICATION_JSON)
		json.NewEncoder(w).Encode(dashboardRecived)
	}
}
