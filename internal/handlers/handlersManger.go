package handlers

import (
	"net/http"

	"cloud.google.com/go/firestore"
)

/*
SetupAllHandlers registers all HTTP handlers to the provided router.
This function initializes auth, registrations, notification, status and dashboards handlers.

Parameters:
  - router: *http.ServeMux - The HTTP request multiplexer to register handlers with
*/
func SetupAllHandlers(router *http.ServeMux, client *firestore.Client) {
	authHandler(router)
	InitRegistration(router, client)
	notificationsHandler(router)
	StatusRouter(router)
	DashboardRouter(router, client)
}
