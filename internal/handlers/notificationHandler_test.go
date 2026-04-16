package handlers

import (
	"context"
	"encoding/json"
	"envdash/internal/services/notification"
	"envdash/internal/structs"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeNotificationService struct {
	createFn func(ctx context.Context, registration *structs.NotificationRegistration) (string, error)
	getFn    func(ctx context.Context, id string) (*structs.NotificationRegistration, error)
	listFn   func(ctx context.Context) ([]structs.NotificationRegistration, error)
	deleteFn func(ctx context.Context, id string) error
}

func (fake *fakeNotificationService) Create(ctx context.Context, registration *structs.NotificationRegistration) (string, error) {
	return fake.createFn(ctx, registration)
}

func (fake *fakeNotificationService) GetByID(ctx context.Context, id string) (*structs.NotificationRegistration, error) {
	return fake.getFn(ctx, id)
}

func (fake *fakeNotificationService) List(ctx context.Context) ([]structs.NotificationRegistration, error) {
	return fake.listFn(ctx)
}

func (fake *fakeNotificationService) DeleteByID(ctx context.Context, id string) error {
	return fake.deleteFn(ctx, id)
}

func TestNotificationHandler(t *testing.T) {
	t.Run("POST returns 201 and id", func(t *testing.T) {
		handler := &NotificationHandler{service: &fakeNotificationService{
			createFn: func(ctx context.Context, registration *structs.NotificationRegistration) (string, error) {
				if registration.Event != structs.NotificationEventInvoke {
					t.Fatalf("expected INVOKE event")
				}
				return "notification-1", nil
			},
			getFn: func(ctx context.Context, id string) (*structs.NotificationRegistration, error) {
				return nil, errors.New("unused")
			},
			listFn: func(ctx context.Context) ([]structs.NotificationRegistration, error) {
				return nil, errors.New("unused")
			},
			deleteFn: func(ctx context.Context, id string) error {
				return errors.New("unused")
			},
		}}

		body := `{"url":"https://webhook.site/test","event":"INVOKE"}`
		req := httptest.NewRequest(http.MethodPost, "/envdash/v1/notifications/", strings.NewReader(body))
		rec := httptest.NewRecorder()

		handler.handleNotifications(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rec.Code)
		}

		var payload map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}
		if payload["id"] != "notification-1" {
			t.Fatalf("expected id notification-1, got %q", payload["id"])
		}
	})

	t.Run("GET by id returns 404 when not found", func(t *testing.T) {
		handler := &NotificationHandler{service: &fakeNotificationService{
			createFn: func(ctx context.Context, registration *structs.NotificationRegistration) (string, error) {
				return "", errors.New("unused")
			},
			getFn: func(ctx context.Context, id string) (*structs.NotificationRegistration, error) {
				return nil, notification.ErrNotificationNotFound
			},
			listFn: func(ctx context.Context) ([]structs.NotificationRegistration, error) {
				return nil, errors.New("unused")
			},
			deleteFn: func(ctx context.Context, id string) error {
				return errors.New("unused")
			},
		}}

		mux := http.NewServeMux()
		mux.HandleFunc("/envdash/v1/notifications/{id}", handler.handleNotifications)
		req := httptest.NewRequest(http.MethodGet, "/envdash/v1/notifications/missing", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("DELETE by id returns 204", func(t *testing.T) {
		handler := &NotificationHandler{service: &fakeNotificationService{
			createFn: func(ctx context.Context, registration *structs.NotificationRegistration) (string, error) {
				return "", errors.New("unused")
			},
			getFn: func(ctx context.Context, id string) (*structs.NotificationRegistration, error) {
				return nil, errors.New("unused")
			},
			listFn: func(ctx context.Context) ([]structs.NotificationRegistration, error) {
				return nil, errors.New("unused")
			},
			deleteFn: func(ctx context.Context, id string) error {
				return nil
			},
		}}

		mux := http.NewServeMux()
		mux.HandleFunc("/envdash/v1/notifications/{id}", handler.handleNotifications)
		req := httptest.NewRequest(http.MethodDelete, "/envdash/v1/notifications/notification-1", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
	})
}
