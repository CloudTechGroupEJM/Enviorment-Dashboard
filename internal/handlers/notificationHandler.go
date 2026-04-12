package handlers

import (
	"context"
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/services"
	"envdash/internal/structs"
	"errors"
	"net/http"

	"cloud.google.com/go/firestore"
)

type NotificationHandler struct {
	service notificationServiceAPI
}

type notificationServiceAPI interface {
	Create(ctx context.Context, registration *structs.NotificationRegistration) (string, error)
	GetByID(ctx context.Context, id string) (*structs.NotificationRegistration, error)
	List(ctx context.Context) ([]structs.NotificationRegistration, error)
	DeleteByID(ctx context.Context, id string) error
}

func notificationsHandler(router *http.ServeMux, client *firestore.Client) *services.NotificationService {
	service := services.NewNotificationService(client, nil)
	handler := &NotificationHandler{service: service}

	router.HandleFunc(config.NOTIFICATION_PAGE_PATH, handler.handleNotifications)
	router.HandleFunc(config.NOTIFICATION_PAGE_PATH+"{id}", handler.handleNotifications)

	return service
}

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

func isNotFound(err error) bool {
	return errors.Is(err, services.ErrNotificationNotFound)
}
