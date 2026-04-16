package structs

import "time"

const (
	NotificationEventRegister  = "REGISTER"
	NotificationEventChange    = "CHANGE"
	NotificationEventDelete    = "DELETE"
	NotificationEventInvoke    = "INVOKE"
	NotificationEventThreshold = "THRESHOLD"
)

type ThresholdRule struct {
	Field    string  `firestore:"field" json:"field"`
	Operator string  `firestore:"operator" json:"operator"`
	Value    float64 `firestore:"value" json:"value"`
}

type NotificationRegistration struct {
	ID        string         `firestore:"id,omitempty" json:"id,omitempty"`
	URL       string         `firestore:"url" json:"url"`
	Country   string         `firestore:"country,omitempty" json:"country,omitempty"`
	Event     string         `firestore:"event" json:"event"`
	Threshold *ThresholdRule `firestore:"threshold,omitempty" json:"threshold,omitempty"`
	CreatedAt time.Time      `firestore:"createdAt" json:"createdAt,omitempty"`
}

type ThresholdDispatchDetails struct {
	Field         string  `json:"field"`
	Operator      string  `json:"operator"`
	Threshold     float64 `json:"threshold"`
	MeasuredValue float64 `json:"measuredValue"`
}

type WebhookInvocationPayload struct {
	ID      string                    `json:"id"`
	Country string                    `json:"country"`
	Event   string                    `json:"event"`
	Time    string                    `json:"time"`
	Details *ThresholdDispatchDetails `json:"details,omitempty"`
}
