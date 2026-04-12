package services

import (
	"context"
	"encoding/json"
	"envdash/internal/structs"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateNotificationRegistration(t *testing.T) {
	tests := []struct {
		name      string
		input     structs.NotificationRegistration
		wantError bool
	}{
		{
			name: "valid invoke registration",
			input: structs.NotificationRegistration{
				URL:   "https://webhook.site/abc",
				Event: structs.NotificationEventInvoke,
			},
			wantError: false,
		},
		{
			name: "invalid URL",
			input: structs.NotificationRegistration{
				URL:   "not-a-url",
				Event: structs.NotificationEventInvoke,
			},
			wantError: true,
		},
		{
			name: "threshold missing details",
			input: structs.NotificationRegistration{
				URL:   "https://webhook.site/abc",
				Event: structs.NotificationEventThreshold,
			},
			wantError: true,
		},
		{
			name: "valid threshold",
			input: structs.NotificationRegistration{
				URL:   "https://webhook.site/abc",
				Event: structs.NotificationEventThreshold,
				Threshold: &structs.ThresholdRule{
					Field:    "pm25",
					Operator: ">",
					Value:    35,
				},
			},
			wantError: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			input := testCase.input
			err := validateNotificationRegistration(&input)
			if (err != nil) != testCase.wantError {
				t.Fatalf("validateNotificationRegistration() error = %v, wantError %v", err, testCase.wantError)
			}
		})
	}
}

func TestCrossedThreshold(t *testing.T) {
	tests := []struct {
		name     string
		operator string
		value    float64
		limit    float64
		want     bool
	}{
		{name: "greater than match", operator: ">", value: 10, limit: 5, want: true},
		{name: "greater than no match", operator: ">", value: 2, limit: 5, want: false},
		{name: "less than match", operator: "<", value: 2, limit: 5, want: true},
		{name: "less than no match", operator: "<", value: 8, limit: 5, want: false},
		{name: "unsupported operator", operator: "=", value: 8, limit: 5, want: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := crossedThreshold(testCase.operator, testCase.value, testCase.limit)
			if got != testCase.want {
				t.Fatalf("crossedThreshold(%q, %f, %f) = %v, want %v", testCase.operator, testCase.value, testCase.limit, got, testCase.want)
			}
		})
	}
}

func TestSendWebhookPayload(t *testing.T) {
	t.Run("success on 2xx response", func(t *testing.T) {
		captured := structs.WebhookInvocationPayload{}
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", request.Method)
			}
			if request.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("expected application/json content type")
			}
			if err := json.NewDecoder(request.Body).Decode(&captured); err != nil {
				t.Fatalf("failed to decode payload: %v", err)
			}
			writer.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		service := &NotificationService{httpClient: server.Client()}
		payload := structs.WebhookInvocationPayload{ID: "id-1", Country: "NO", Event: structs.NotificationEventInvoke, Time: "20260101 12:00"}

		err := service.sendWebhookPayload(context.Background(), server.URL, payload)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if captured.ID != payload.ID || captured.Event != payload.Event || captured.Country != payload.Country {
			t.Fatalf("captured payload mismatch: %+v", captured)
		}
	})

	t.Run("returns error on non-2xx response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusBadGateway)
		}))
		defer server.Close()

		service := &NotificationService{httpClient: server.Client()}
		payload := structs.WebhookInvocationPayload{ID: "id-1", Country: "NO", Event: structs.NotificationEventInvoke, Time: "20260101 12:00"}

		err := service.sendWebhookPayload(context.Background(), server.URL, payload)
		if err == nil {
			t.Fatalf("expected error for non-2xx webhook response")
		}
	})
}

