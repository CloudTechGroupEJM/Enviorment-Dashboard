package handlers

import (
	"net/http"

	"cloud.google.com/go/firestore"
)

// SetupAllHandlers registers all HTTP handlers to the provided router.
// This function initializes auth, registrations, notification, status and dashboards handlers.
//
// Parameters:
//   - router: *http.ServeMux - The HTTP request multiplexer to register handlers with
//   - client: *firestore.Client - The Firestore client used by handlers that require database access
func SetupAllHandlers(router *http.ServeMux, client *firestore.Client) {
	authHandler(router)
	dispatcher := notificationsHandler(router, client)
	InitRegistration(router, client, dispatcher)
	StatusRouter(router)
	DashboardRouter(router, client)
}
