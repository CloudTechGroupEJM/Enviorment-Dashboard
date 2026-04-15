package handlers

import (
	"context"
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/services/notification"
	"envdash/internal/structs"
	"errors"
	"net/http"

	"cloud.google.com/go/firestore"
)

type NotificationHandler struct {
	service notificationServiceAPI
}

// notificationServiceAPI defines the methods that the NotificationHandler expects from its service layer.
// This allows for easier testing and separation of concerns, as the handler can work with any implementation of this interface.
type notificationServiceAPI interface {
	Create(ctx context.Context, registration *structs.NotificationRegistration) (string, error)
	GetByID(ctx context.Context, id string) (*structs.NotificationRegistration, error)
	List(ctx context.Context) ([]structs.NotificationRegistration, error)
	DeleteByID(ctx context.Context, id string) error
}

// notificationsHandler sets up the HTTP handlers for notification-related endpoints and returns the initialized NotificationService.
// It registers handlers for creating, retrieving, and deleting notification registrations at the specified paths.
//
// Parameters:
//   - router: *http.ServeMux - The HTTP request multiplexer to register handlers with
//   - client: *firestore.Client - The Firestore client used by the NotificationService for database operations
//
// Returns:
//   - *services.NotificationService: The initialized NotificationService instance used by the handlers
func notificationsHandler(router *http.ServeMux, client *firestore.Client) *notification.NotificationService {
	service := notification.NewNotificationService(client, nil)
	handler := &NotificationHandler{service: service}

	router.HandleFunc(config.NOTIFICATION_PAGE_PATH, handler.handleNotifications)
	router.HandleFunc(config.NOTIFICATION_PAGE_PATH+"{id}", handler.handleNotifications)

	return service
}

// handleNotifications is the main HTTP handler for notification-related requests. It routes incoming requests to the appropriate method based on the HTTP method (POST, GET, DELETE).
// It handles creating new notifications, retrieving existing notifications (either all or by ID), and deleting notifications by ID.
//
// Parameters:
//   - writer: http.ResponseWriter - The HTTP response writer used to send responses back to the client
//   - request: *http.Request - The incoming HTTP request containing details about the operation to perform
func (handler *NotificationHandler) handleNotifications(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set(config.HEADER_CONTENT_TYPE, config.APPLICATION_JSON)

	switch request.Method {
	case http.MethodPost:
		handler.createNotification(writer, request)
	case http.MethodGet:
		handler.getNotifications(writer, request)
	case http.MethodDelete:
		handler.deleteNotification(writer, request)
	default:
		http.Error(writer, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// createNotification handles the creation of a new notification registration. It expects a JSON payload in the request body containing the details of the notification to be created.
// It decodes the JSON payload into a NotificationRegistration struct, calls the service layer to create the notification, and returns the ID of the newly created notification in the response.
//
// Parameters:
//   - writer: http.ResponseWriter - The HTTP response writer used to send responses back to the client
//   - request: *http.Request - The incoming HTTP request containing the JSON payload for creating a new notification
func (handler *NotificationHandler) createNotification(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	var registration structs.NotificationRegistration
	if err := json.NewDecoder(request.Body).Decode(&registration); err != nil {
		http.Error(writer, `{"error":"invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	id, err := handler.service.Create(request.Context(), &registration)
	if err != nil {
		http.Error(writer, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	writer.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(writer).Encode(map[string]string{"id": id})
}

// getNotifications handles the retrieval of notification registrations.
// If an ID is provided in the request path, it retrieves the specific notification with that ID. If no ID is provided, it retrieves all notifications.
// It returns the retrieved notification(s) in JSON format in the response.
//
// Parameters:
//   - writer: http.ResponseWriter - The HTTP response writer used to send responses back to the client
//   - request: *http.Request - The incoming HTTP request which may contain an ID in the path for retrieving a specific notification
func (handler *NotificationHandler) getNotifications(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if id == "" {
		registrations, err := handler.service.List(request.Context())
		if err != nil {
			http.Error(writer, `{"error":"failed to list notifications"}`, http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(writer).Encode(registrations)
		return
	}

	registration, err := handler.service.GetByID(request.Context(), id)
	if err != nil {
		if isNotFound(err) {
			http.Error(writer, `{"error":"notification not found"}`, http.StatusNotFound)
			return
		}
		http.Error(writer, `{"error":"failed to fetch notification"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(writer).Encode(registration)
}

// deleteNotification handles the deletion of a notification registration by its ID. It expects the ID to be provided in the request path.
// If the notification with the specified ID is found, it deletes it and returns a 204 No Content status. If the notification is not found, it returns a 404 Not Found status.
//
// Parameters:
//   - writer: http.ResponseWriter - The HTTP response writer used to send responses back to the client
//   - request: *http.Request - The incoming HTTP request which should contain an ID in the path for deleting a specific notification
func (handler *NotificationHandler) deleteNotification(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if id == "" {
		http.Error(writer, `{"error":"notification id is required"}`, http.StatusBadRequest)
		return
	}

	err := handler.service.DeleteByID(request.Context(), id)
	if err != nil {
		if isNotFound(err) {
			http.Error(writer, `{"error":"notification not found"}`, http.StatusNotFound)
			return
		}
		http.Error(writer, `{"error":"failed to delete notification"}`, http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

// isNotFound checks if the provided error is a "not found" error specific to notifications.
// It uses errors.Is to compare the error against the predefined ErrNotificationNotFound from the services package.
//
// Parameters:
//   - err: error - The error to check for being a "not found" error
//
// Returns:
//   - bool: true if the error is a "not found" error, false otherwise
func isNotFound(err error) bool {
	return errors.Is(err, notification.ErrNotificationNotFound)
}
