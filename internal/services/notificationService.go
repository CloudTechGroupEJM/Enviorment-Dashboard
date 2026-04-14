package services

import (
	"bytes"
	"context"
	"encoding/json"
	"envdash/internal/store"
	"envdash/internal/structs"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/firestore/apiv1/firestorepb"
)

// NotificationService provides methods for managing notification registrations and dispatching events to registered webhooks.
type NotificationService struct {
	client     *firestore.Client
	httpClient *http.Client
}

// ErrNotificationNotFound is returned when a requested notification registration cannot be found in the database.
var ErrNotificationNotFound = errors.New("notification not found")

// globalNotificationService holds a reference to the NotificationService for package-level access
var globalNotificationService *NotificationService

// NewNotificationService creates a new instance of NotificationService with the provided Firestore client and an optional HTTP client.
// If the httpClient parameter is nil, a default HTTP client with a 5-second timeout will be used.
// It also sets the global NotificationService instance for package-level access.
//
// Parameters:
//   - client: *firestore.Client - The Firestore client used for database interactions. Must not be nil.
//   - httpClient: *http.Client (optional) - The HTTP client used for sending webhook requests. If nil, a default client will be created.
//
// Returns:
//   - *NotificationService: A new instance of NotificationService initialized with the provided Firestore client and HTTP client.
func NewNotificationService(client *firestore.Client, httpClient *http.Client) *NotificationService {
	if client == nil {
		panic("firestore client cannot be nil")
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}

	service := &NotificationService{client: client, httpClient: httpClient}
	globalNotificationService = service
	return service
}

// Create registers a new notification based on the provided registration details.
// It validates the input, normalizes the event and country fields, and stores the registration in Firestore.
// If the registration is successful, it returns the generated ID of the new notification.
//
// Parameters:
//   - ctx: context.Context - The context for managing request deadlines and cancellation.
//   - registration: *structs.NotificationRegistration - The details of the notification to be registered. Must not be nil.
//
// Returns:
//   - string: The ID of the newly created notification registration.
//   - error: An error object if the creation fails due to validation errors or database issues; otherwise, nil.
func (service *NotificationService) Create(ctx context.Context, registration *structs.NotificationRegistration) (string, error) {
	if registration == nil {
		return "", errors.New("request body is required")
	}

	if err := validateNotificationRegistration(registration); err != nil {
		return "", err
	}

	registration.Event = strings.ToUpper(strings.TrimSpace(registration.Event))
	registration.Country = strings.ToUpper(strings.TrimSpace(registration.Country))
	registration.CreatedAt = time.Now()

	doc := service.client.Collection(store.NOTIFICATION_COLLECTION).NewDoc()
	registration.ID = doc.ID

	if _, err := doc.Set(ctx, registration); err != nil {
		return "", err
	}

	return doc.ID, nil
}

// GetByID retrieves a notification registration by its ID from Firestore.
// It validates the input ID, fetches the document from Firestore, and decodes it into a NotificationRegistration struct.
// If the registration is found, it returns the registration details; if not found, it returns a not found error.
//
// Parameters:
//   - ctx: context.Context - The context for managing request deadlines and cancellation.
//   - id: string - The ID of the notification registration to retrieve. Must not be empty or whitespace.
//
// Returns:
//   - *structs.NotificationRegistration: A pointer to the retrieved notification registration if found; otherwise, nil.
//   - error: An error object if the retrieval fails due to validation errors, not found errors, or database issues; otherwise, nil.
func (service *NotificationService) GetByID(ctx context.Context, id string) (*structs.NotificationRegistration, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("id is required")
	}

	doc, err := service.client.Collection(store.NOTIFICATION_COLLECTION).Doc(id).Get(ctx)
	if err != nil {
		if isNotFoundError(err) {
			return nil, fmt.Errorf("%w: %v", ErrNotificationNotFound, err)
		}
		return nil, err
	}

	var registration structs.NotificationRegistration
	if err := doc.DataTo(&registration); err != nil {
		return nil, err
	}

	if registration.ID == "" {
		registration.ID = doc.Ref.ID
	}

	return &registration, nil
}

// List retrieves all notification registrations from Firestore.
// It fetches all documents from the notification collection, decodes them into NotificationRegistration structs, and returns a slice of registrations.
//
// Parameters:
//   - ctx: context.Context - The context for managing request deadlines and cancellation.
//
// Returns:
//   - []structs.NotificationRegistration: A slice containing all notification registrations retrieved from Firestore.
//   - error: An error object if the retrieval fails due to database issues; otherwise, nil.
func (service *NotificationService) List(ctx context.Context) ([]structs.NotificationRegistration, error) {
	docs, err := service.client.Collection(store.NOTIFICATION_COLLECTION).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	registrations := make([]structs.NotificationRegistration, 0, len(docs))
	for _, doc := range docs {
		var registration structs.NotificationRegistration
		if decodeErr := doc.DataTo(&registration); decodeErr != nil {
			return nil, decodeErr
		}
		if registration.ID == "" {
			registration.ID = doc.Ref.ID
		}
		registrations = append(registrations, registration)
	}

	return registrations, nil
}

// countNotifications retrieves the total number of notification registrations
// currently stored in Firestore using an aggregation query.
//
// Returns:
//   - int: The total number of notification registrations if retrieval is successful; otherwise, -1 if an error occurs.
func (service *NotificationService) countNotifications() int {
	ctx := context.Background()

	aggregationQuery := service.client.
		Collection(store.NOTIFICATION_COLLECTION).
		NewAggregationQuery().
		WithCount("count")

	results, err := aggregationQuery.Get(ctx)
	if err != nil {
		return -1
	}

	count, ok := results["count"]
	if !ok {
		return -1
	}

	return int(count.(*firestorepb.Value).GetIntegerValue())
}

// DeleteByID removes a notification registration from Firestore based on its ID.
// It validates the input ID, checks if the document exists, and deletes it if found.
// If the registration is not found, it returns a not found error.
//
// Parameters:
//   - ctx: context.Context - The context for managing request deadlines and cancellation.
//   - id: string - The ID of the notification registration to delete. Must not be empty or whitespace.
//
// Returns:
//   - error: An error object if the deletion fails due to validation errors, not found errors, or database issues; otherwise, nil.
func (service *NotificationService) DeleteByID(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("id is required")
	}

	if _, err := service.client.Collection(store.NOTIFICATION_COLLECTION).Doc(id).Get(ctx); err != nil {
		if isNotFoundError(err) {
			return fmt.Errorf("%w: %v", ErrNotificationNotFound, err)
		}
		return err
	}

	_, err := service.client.Collection(store.NOTIFICATION_COLLECTION).Doc(id).Delete(ctx)
	return err
}

// DispatchLifecycleAsync triggers asynchronous dispatch of lifecycle events (REGISTER, CHANGE, DELETE, INVOKE) to registered webhooks.
// It spawns a new goroutine to handle the dispatching, allowing the caller to continue without waiting for the dispatch to complete.
//
// Parameters:
//   - country: string - The country associated with the lifecycle event, used for filtering webhook registrations. Can be empty or whitespace for no country filter.
//   - event: string - The lifecycle event type (e.g., REGISTER, CHANGE, DELETE, INVOKE) to dispatch. Must not be empty or whitespace.
func (service *NotificationService) DispatchLifecycleAsync(country string, event string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = service.DispatchLifecycle(ctx, country, event)
	}()
}

// DispatchLifecycle triggers the dispatch of lifecycle events (REGISTER, CHANGE, DELETE, INVOKE) to registered webhooks.
// It retrieves all registrations for the specified event, filters them based on the country, and sends webhook payloads to the registered URLs.
//
// Parameters:
//   - ctx: context.Context - The context for managing request deadlines and cancellation.
//   - country: string - The country associated with the lifecycle event, used for filtering webhook registrations. Can be empty or whitespace for no country filter.
//   - event: string - The lifecycle event type (e.g., REGISTER, CHANGE, DELETE, INVOKE) to dispatch. Must not be empty or whitespace.
//
// Returns:
//   - error: An error object if the dispatching process encounters issues such as database retrieval errors; otherwise, nil. Individual webhook sending errors are logged but do not cause the entire dispatch to fail.
func (service *NotificationService) DispatchLifecycle(ctx context.Context, country string, event string) error {
	normalizedEvent := strings.ToUpper(strings.TrimSpace(event))
	normalizedCountry := strings.ToUpper(strings.TrimSpace(country))

	registrations, err := service.getRegistrationsForEvent(ctx, normalizedEvent)
	if err != nil {
		return err
	}

	for _, registration := range registrations {
		if !matchesCountryFilter(registration.Country, normalizedCountry) {
			continue
		}

		payload := structs.WebhookInvocationPayload{
			ID:      registration.ID,
			Country: normalizedCountry,
			Event:   normalizedEvent,
			Time:    time.Now().Format("20060102 15:04"),
		}

		if sendErr := service.sendWebhookPayload(ctx, registration.URL, payload); sendErr != nil {
			continue
		}
	}

	return nil
}

// DispatchThresholdAsync triggers asynchronous dispatch of threshold events to registered webhooks.
// It spawns a new goroutine to handle the dispatching, allowing the caller to continue without waiting for the dispatch to complete.
//
// Parameters:
//   - country: string - The country associated with the threshold event, used for filtering webhook registrations. Can be empty or whitespace for no country filter.
//   - field: string - The field (e.g., pm25, pm10, temperature, precipitation) associated with the threshold event. Must not be empty or whitespace.
//   - measuredValue: float64 - The measured value that triggered the threshold event, used for evaluating against registered threshold rules.
func (service *NotificationService) DispatchThresholdAsync(country string, field string, measuredValue float64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = service.DispatchThreshold(ctx, country, field, measuredValue)
	}()
}

// DispatchThreshold triggers the dispatch of threshold events to registered webhooks.
// It retrieves all registrations for the THRESHOLD event, filters them based on the country and threshold rules, and sends webhook payloads to the registered URLs.
//
// Parameters:
//   - ctx: context.Context - The context for managing request deadlines and cancellation.
//   - country: string - The country associated with the threshold event, used for filtering webhook registrations. Can be empty or whitespace for no country filter.
//   - field: string - The field (e.g., pm25, pm10, temperature, precipitation) associated with the threshold event. Must not be empty or whitespace.
//   - measuredValue: float64 - The measured value that triggered the threshold event, used for evaluating against registered threshold rules.
//
// Returns:
//   - error: An error object if the dispatching process encounters issues such as database retrieval errors; otherwise, nil. Individual webhook sending errors are logged but do not cause the entire dispatch to fail.
func (service *NotificationService) DispatchThreshold(ctx context.Context, country string, field string, measuredValue float64) error {
	registrations, err := service.getRegistrationsForEvent(ctx, structs.NotificationEventThreshold)
	if err != nil {
		return err
	}

	normalizedCountry := strings.ToUpper(strings.TrimSpace(country))
	normalizedField := strings.ToLower(strings.TrimSpace(field))

	for _, registration := range registrations {
		if !matchesCountryFilter(registration.Country, normalizedCountry) {
			continue
		}
		if registration.Threshold == nil {
			continue
		}
		if strings.ToLower(strings.TrimSpace(registration.Threshold.Field)) != normalizedField {
			continue
		}
		if !crossedThreshold(registration.Threshold.Operator, measuredValue, registration.Threshold.Value) {
			continue
		}

		payload := structs.WebhookInvocationPayload{
			ID:      registration.ID,
			Country: normalizedCountry,
			Event:   structs.NotificationEventThreshold,
			Time:    time.Now().Format("20060102 15:04"),
			Details: &structs.ThresholdDispatchDetails{
				Field:         normalizedField,
				Operator:      registration.Threshold.Operator,
				Threshold:     registration.Threshold.Value,
				MeasuredValue: measuredValue,
			},
		}

		if sendErr := service.sendWebhookPayload(ctx, registration.URL, payload); sendErr != nil {
			continue
		}
	}

	return nil
}

// getRegistrationsForEvent retrieves all notification registrations from Firestore that match the specified event type.
// It queries the Firestore collection for documents where the "event" field matches the provided event, decodes the results into NotificationRegistration structs, and returns a slice of matching registrations.
//
// Parameters:
//   - ctx: context.Context - The context for managing request deadlines and cancellation.
//   - event: string - The event type (e.g., REGISTER, CHANGE, DELETE, INVOKE, THRESHOLD) to filter registrations by. Must not be empty or whitespace.
//
// Returns:
//   - []structs.NotificationRegistration: A slice containing all notification registrations that match the specified event type.
//   - error: An error object if the retrieval fails due to database issues; otherwise, nil.
func (service *NotificationService) getRegistrationsForEvent(ctx context.Context, event string) ([]structs.NotificationRegistration, error) {
	docs, err := service.client.Collection(store.NOTIFICATION_COLLECTION).
		Where("event", "==", strings.ToUpper(strings.TrimSpace(event))).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, err
	}

	registrations := make([]structs.NotificationRegistration, 0, len(docs))
	for _, doc := range docs {
		var registration structs.NotificationRegistration
		if decodeErr := doc.DataTo(&registration); decodeErr != nil {
			return nil, decodeErr
		}
		if registration.ID == "" {
			registration.ID = doc.Ref.ID
		}
		registrations = append(registrations, registration)
	}
	return registrations, nil
}

// sendWebhookPayload sends a JSON-encoded payload to the specified webhook URL using an HTTP POST request.
// It constructs the HTTP request with the appropriate headers, sends it using the configured HTTP client, and checks the response status code to ensure successful delivery.
//
// Parameters:
//   - ctx: context.Context - The context for managing request deadlines and cancellation.
//   - webhookURL: string - The URL of the webhook to which the payload should be sent. Must be a valid absolute URL.
//   - payload: structs.WebhookInvocationPayload - The payload data to be sent in the webhook request.
//
// Returns:
//   - error: An error object if the payload fails to send due to issues such as JSON encoding errors, HTTP request errors, or non-successful response status codes; otherwise, nil.
func (service *NotificationService) sendWebhookPayload(ctx context.Context, webhookURL string, payload structs.WebhookInvocationPayload) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := service.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// validateNotificationRegistration checks the validity of a NotificationRegistration struct.
// It ensures that required fields are present and correctly formatted, and that any threshold details are valid if the event type is THRESHOLD.
//
// Parameters:
//   - registration: *structs.NotificationRegistration - The notification registration to validate. Must not be nil.
//
// Returns:
//   - error: An error object if the validation fails due to missing or invalid fields; otherwise, nil.
func validateNotificationRegistration(registration *structs.NotificationRegistration) error {
	if strings.TrimSpace(registration.URL) == "" {
		return errors.New("url is required")
	}
	if _, err := url.ParseRequestURI(registration.URL); err != nil {
		return errors.New("url must be a valid absolute URL")
	}

	registration.Event = strings.ToUpper(strings.TrimSpace(registration.Event))
	if !isValidEvent(registration.Event) {
		return errors.New("event must be one of REGISTER, CHANGE, DELETE, INVOKE, THRESHOLD")
	}

	if registration.Event == structs.NotificationEventThreshold {
		if registration.Threshold == nil {
			return errors.New("threshold details are required for THRESHOLD event")
		}
		if err := validateThreshold(registration.Threshold); err != nil {
			return err
		}
	}

	return nil
}

// validateThreshold checks the validity of a ThresholdRule struct.
// It ensures that the field is one of the allowed types (pm25, pm10, temperature, precipitation) and that the operator is either ">" or "<".
//
// Parameters:
//   - rule: *structs.ThresholdRule - The threshold rule to validate. Must not be nil.
//
// Returns:
//   - error: An error object if the validation fails due to invalid field or operator; otherwise, nil.
func validateThreshold(rule *structs.ThresholdRule) error {
	field := strings.ToLower(strings.TrimSpace(rule.Field))
	switch field {
	case "pm25", "pm10", "temperature", "precipitation":
	default:
		return errors.New("threshold.field must be one of pm25, pm10, temperature, precipitation")
	}

	rule.Operator = strings.TrimSpace(rule.Operator)
	if rule.Operator != ">" && rule.Operator != "<" {
		return errors.New("threshold.operator must be '>' or '<'")
	}

	return nil
}

// matchesCountryFilter checks if a given country matches the filter specified in a notification registration.
// It normalizes both the filter and the country by trimming whitespace and converting to uppercase before comparison.
// If the filter is empty, it returns true, indicating that the registration should be considered a match for any country.
//
// Parameters:
//   - filter: string - The country filter specified in the notification registration. Can be empty or whitespace for no filter.
//   - country: string - The country associated with the event being evaluated against the registration.
//
// Returns:
//   - bool: True if the country matches the filter or if the filter is empty; otherwise, false.
func matchesCountryFilter(filter string, country string) bool {
	normalizedFilter := strings.ToUpper(strings.TrimSpace(filter))
	if normalizedFilter == "" {
		return true
	}
	return normalizedFilter == strings.ToUpper(strings.TrimSpace(country))
}

// crossedThreshold evaluates whether a measured value crosses a specified threshold based on the provided operator.
// It supports ">" and "<" operators to determine if the measured value exceeds or falls below the threshold, respectively.
//
// Parameters:
//   - operator: string - The operator indicating the type of threshold comparison (e.g., ">" or "<"). Must not be empty or whitespace.
//   - measuredValue: float64 - The value that was measured and is being evaluated against the threshold.
//   - threshold: float64 - The threshold value that the measured value is being compared to.
//
// Returns:
//   - bool: True if the measured value crosses the threshold based on the operator; otherwise, false.
func crossedThreshold(operator string, measuredValue float64, threshold float64) bool {
	switch strings.TrimSpace(operator) {
	case ">":
		return measuredValue > threshold
	case "<":
		return measuredValue < threshold
	default:
		return false
	}
}

// isValidEvent checks if the provided event string is one of the allowed notification event types (REGISTER, CHANGE, DELETE, INVOKE, THRESHOLD).
// It normalizes the input by trimming whitespace and converting to uppercase before comparison.
//
// Parameters:
//   - event: string - The event type to validate. Must not be empty or whitespace.
//
// Returns:
//   - bool: True if the event is valid; otherwise, false.
func isValidEvent(event string) bool {
	switch event {
	case structs.NotificationEventRegister,
		structs.NotificationEventChange,
		structs.NotificationEventDelete,
		structs.NotificationEventInvoke,
		structs.NotificationEventThreshold:
		return true
	default:
		return false
	}
}

// isNotFoundError checks if the provided error indicates a "not found" condition.
// It does this by examining the error message for common "not found" indicators, such as "notfound" or "not found", in a case-insensitive manner.
//
// Parameters:
//   - err: error - The error to check for a "not found" condition. Can be nil.
//
// Returns:
//   - bool: True if the error message contains indicators of a "not found" condition; otherwise, false.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "notfound") || strings.Contains(errText, "not found")
}

// GetNotificationCount retrieves the total number of notification registrations from the global NotificationService instance.
// This is a package-level function that provides access to the notification count for endpoints that need it.
//
// Returns:
//   - int: The total number of notification registrations if retrieval is successful; otherwise, -1 if an error occurs or the service is not initialized.
func GetNotificationCount() int {
	if globalNotificationService == nil {
		return -1
	}
	return globalNotificationService.countNotifications()
}
