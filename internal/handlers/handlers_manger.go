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

	//Flow
	// HANDLER -> parse request, GET -> SERVICE ->
	// probe -> process status -> construct response -> HANDLER -> send response
	StatusRouter(router)
	dashboardsHandler(router)
}
