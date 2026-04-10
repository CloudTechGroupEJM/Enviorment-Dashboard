package handlers

import (
	"net/http"
)

/*
SetupAllHandlers registers all HTTP handlers to the provided router.
This function initializes auth, registrations, notification, status and dashboards handlers.

Parameters:
  - router: *http.ServeMux - The HTTP request multiplexer to register handlers with
*/
func SetupAllHandlers(router *http.ServeMux) {
	authHandler(router)
	registrationsHandler(router)
	notificationsHandler(router)
	StatusRouter(router)
	DashboardRouter(router)
}
