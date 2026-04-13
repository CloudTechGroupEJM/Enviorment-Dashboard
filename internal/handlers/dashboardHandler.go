package handlers

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/services/dashboard"
	"net/http"
)

type DashboardHandler struct {
	service    *dashboard.DashBoardInternal
	dispatcher webhookDispatcher
}

func DashboardRouter(router *http.ServeMux, dispatcher webhookDispatcher) {
	handler := &DashboardHandler{
		service:    dashboard.NewDashboardService(),
		dispatcher: dispatcher,
	}
	router.HandleFunc(config.DASHBOARDS_PAGE_PATH+"{p1}", handler.handleDashboard)
}

func (handler *DashboardHandler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	service := handler.service
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
