package handlers

import (
	"context"
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/services/registration"
	"envdash/internal/structs"
	"log"
	"net/http"
	"strconv"

	"strings"

	"cloud.google.com/go/firestore"
)

const SUCCESSFUL_EXECUTION = "Request successfully executed"
const INVALID_JSON = "Invalid JSON payload"

type RegistrationHandler struct {
	service    *registration.RegistrationService
	dispatcher webhookDispatcher
}

// InitRegistration initializes the registration service, handler and endpoints.
// Parameters: router - HTTP router, client - Firestore client
func InitRegistration(router *http.ServeMux, client *firestore.Client, dispatcher webhookDispatcher) {
	service := registration.NewRegistrationService(client)
	handler := &RegistrationHandler{
		service:    service,
		dispatcher: dispatcher,
	}

	router.HandleFunc(config.REGISTRATIONS_PAGE_PATH, handler.handleRegistrations)
	router.HandleFunc(config.REGISTRATIONS_PAGE_PATH+"{id}", handler.handleRegistrations)
}

// handleRegistrations routes HTTP requests to appropriate handler methods based on request type.
// Parameters: writer - HTTP response writer, request - HTTP request
func (handler *RegistrationHandler) handleRegistrations(writer http.ResponseWriter, request *http.Request) {
	log.Printf("%s request recived", request.Method)
	switch request.Method {
	case http.MethodPost:
		handler.createRegistration(writer, request)
	case http.MethodGet:
		handler.getRegistrations(writer, request)
	case http.MethodDelete:
		handler.deleteRegistrations(writer, request)
	case http.MethodPut:
		handler.putRegistration(writer, request)
	case http.MethodPatch:
		handler.patchRegistration(writer, request)
	case http.MethodHead:
		handler.headRegistrations(writer, request)
	default:
		http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// createRegistration handles HTTP POST requests to create a new country registration.
// Parameters: writer - HTTP response writer, request - HTTP request with registration data
// Returns: none (writes HTTP response)
func (handler *RegistrationHandler) createRegistration(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	var registration structs.RegisterCountry

	if decoderErr := json.NewDecoder(request.Body).Decode(&registration); decoderErr != nil {
		http.Error(writer, INVALID_JSON, http.StatusBadRequest)
		log.Println("Error when decoding registration")
		return
	}

	registrationID, CreationTime, creationErr := handler.service.Post(request.Context(), registration)
	if creationErr != nil {
		http.Error(writer, creationErr.Error(), http.StatusBadRequest)
		log.Println("Error when creating registration: " + creationErr.Error())
		return
	}

	writer.Header().Set(config.HEADER_CONTENT_TYPE, config.APPLICATION_JSON)
	writer.WriteHeader(http.StatusCreated)
	handler.dispatchLifecycleByID(request.Context(), registrationID, structs.NotificationEventRegister)

	_ = json.NewEncoder(writer).Encode(map[string]string{
		"id":         registrationID,
		"lastChange": CreationTime,
	})
	log.Println(SUCCESSFUL_EXECUTION)
}

// getRegistrations handles HTTP GET requests to retrieve a single registration or all registrations.
// Parameters: writer - HTTP response writer, request - HTTP request with optional registration ID
// Returns: none (writes HTTP response with registration data)
func (handler *RegistrationHandler) getRegistrations(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	writer.Header().Set(config.HEADER_CONTENT_TYPE, config.APPLICATION_JSON)

	if request.PathValue("id") == "" {
		allRegistrations, retrivingErr := handler.service.GetAll(request.Context())

		if retrivingErr != nil {
			log.Printf("Error retrieving registrations: %v", retrivingErr)
			http.Error(writer, "Error retrieving data", http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(writer).Encode(allRegistrations)
		return
	}

	singleRegistration, registrationErr := handler.service.GetByID(request.PathValue("id"), request.Context())

	if registrationErr != nil {
		log.Printf("Error retrieving registration %s: %v", request.PathValue("id"), registrationErr)
		http.Error(writer, "registration not found", http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(writer).Encode(singleRegistration)

	log.Println(SUCCESSFUL_EXECUTION)
}

// deleteRegistrations handles HTTP DELETE requests to remove a specific registration or all registrations.
// Parameters: writer - HTTP response writer, request - HTTP request with optional registration ID
// Returns: none (writes HTTP response)
func (handler *RegistrationHandler) deleteRegistrations(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	if request.PathValue("id") == "" {
		handler.deleteAllRegistrations(writer, request)
	} else {
		handler.deleteRegistrationByID(writer, request)
	}
	log.Println(SUCCESSFUL_EXECUTION)
}

// deleteAllRegistrations deletes all registrations and dispatches delete lifecycle events for each.
//
// Parameters:
//   - writer - HTTP response writer, request - HTTP request
func (handler *RegistrationHandler) deleteAllRegistrations(writer http.ResponseWriter, request *http.Request) {
	snapshot, lookupErr := handler.service.GetAll(request.Context())
	deletionErr := handler.service.DeleteAll(request.Context())
	if deletionErr != nil {
		log.Printf("Error nothing to delete %s", deletionErr)
		http.Error(writer, "Nothing to delete.", http.StatusNotFound)
		return
	}
	if lookupErr == nil {
		for _, registration := range snapshot {
			handler.dispatchLifecycleFromRegistration(registration, structs.NotificationEventDelete)
		}
	}
	http.Error(writer, "All registration deleted", http.StatusOK)
}

// deleteRegistrationByID deletes a specific registration by ID and dispatches a delete lifecycle event.
//
// Parameters:
//   - writer - HTTP response writer, request - HTTP request with registration ID in the path
func (handler *RegistrationHandler) deleteRegistrationByID(writer http.ResponseWriter, request *http.Request) {
	registration, lookupErr := handler.service.GetByID(request.PathValue("id"), request.Context())
	if lookupErr != nil {
		http.Error(writer, "Error when trying to delete.", http.StatusOK)
		log.Println("Deletion lookup error when deleting by ID")
		return
	}

	deleted, deletionErr := handler.service.DeleteByID(request.PathValue("id"), request.Context())
	if deletionErr != nil {
		http.Error(writer, "Error when trying to delete.", http.StatusOK)
		log.Println("Deletion error when deleting by ID")
		return
	}
	if deleted {
		handler.dispatchLifecycleFromRegistration(registration, structs.NotificationEventDelete)
		http.Error(writer, "registration has been deleted", http.StatusOK)
		log.Println("registration with id " + request.PathValue("id") + " has been deleted")
		return
	}

	http.Error(writer, "Error registration doesn't exist", http.StatusNoContent)
	log.Println("Registration: " + request.PathValue("id") + " doesnt exist")
}

// putRegistration handles HTTP PUT requests to replace an entire registration.
// Parameters: writer - HTTP response writer, request - HTTP request with registration ID and new data
// Returns: none (writes HTTP response)
func (handler *RegistrationHandler) putRegistration(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	writer.Header().Set("Content-Type", "application/json")

	var newRegistration structs.RegisterCountry
	if request.PathValue("id") != "" {
		decoderError := json.NewDecoder(request.Body).Decode(&newRegistration)
		if decoderError != nil {
			http.Error(writer, INVALID_JSON, http.StatusBadRequest)
			return
		}

		registrationID, replaceRegistrationErr := handler.service.Put(&newRegistration, request.PathValue("id"), request.Context())

		if replaceRegistrationErr != nil {
			log.Printf("registration doesn't exist: %s ", replaceRegistrationErr)
			http.Error(writer, "Couldn't replace registration: "+replaceRegistrationErr.Error(), http.StatusBadRequest)
			return
		} else {
			log.Printf("registration %s fully replaced", replaceRegistrationErr)
			writer.WriteHeader(http.StatusOK)
			handler.dispatchLifecycleByID(request.Context(), registrationID, structs.NotificationEventChange)

			_ = json.NewEncoder(writer).Encode(map[string]string{
				"id":     registrationID,
				"status": "updated",
			})
			log.Println(SUCCESSFUL_EXECUTION)
			return
		}
	}
	http.Error(writer, "Specify registration ID", http.StatusBadRequest)
	log.Println("No registration ID provided.")
}

// patchRegistration handles HTTP PATCH requests to update specific fields in a registration.
// Parameters: writer - HTTP response writer, request - HTTP request with registration ID and partial data
// Returns: none (writes HTTP response)
func (handler *RegistrationHandler) patchRegistration(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	writer.Header().Set("Content-Type", "application/json")
	if request.PathValue("id") == "" {
		log.Println("Need to specify a registration ID")
		http.Error(writer, "Missing registration ID", http.StatusBadRequest)
		return
	}

	var dataUpdate map[string]interface{}
	if decoderErr := json.NewDecoder(request.Body).Decode(&dataUpdate); decoderErr != nil {
		http.Error(writer, INVALID_JSON, http.StatusBadRequest)
		return
	}

	patchErr := handler.service.Patch(request.PathValue("id"), request.Context(), dataUpdate)
	if patchErr != nil {
		log.Println(patchErr)
		http.Error(writer, patchErr.Error(), http.StatusBadRequest)
		return
	}
	handler.dispatchLifecycleByID(request.Context(), request.PathValue("id"), structs.NotificationEventChange)
	_ = json.NewEncoder(writer).Encode(map[string]string{
		"id":     request.PathValue("id"),
		"status": "patched",
	})

	log.Println(SUCCESSFUL_EXECUTION)
}

// headRegistrations handles HTTP HEAD requests to retrieve registration count or existence status.
// Parameters: writer - HTTP response writer, request - HTTP request with optional registration ID
// Returns: none (writes HTTP response headers)
func (handler *RegistrationHandler) headRegistrations(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	if request.PathValue("id") == "" {
		totalRegistrations, totalErr := handler.service.HeadAllRegistrations(request.Context())
		if totalErr != nil {
			log.Printf("Error counting registrations: %v ", totalErr)
			http.Error(writer, "Error retrieving data", http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Registration-Count", strconv.Itoa(totalRegistrations))
		http.Error(writer, "Retrieved total registration", http.StatusOK)
	} else {
		_, registrationErr := handler.service.HeadOneRegistration(request.PathValue("id"), request.Context())
		if registrationErr != nil {
			log.Printf("Error retrieving registration %s: %v", request.PathValue("id"), registrationErr)
			http.Error(writer, "Registration not found", http.StatusNotFound)
			return
		}

		writer.Header().Set("Registration-Exists", "true")
		writer.WriteHeader(http.StatusOK)
	}
	log.Println(SUCCESSFUL_EXECUTION)
}

// dispatchLifecycleByID retrieves a registration by ID and dispatches a lifecycle event based on the registration's isoCode.
//
// Parameters:
//   - ctx: context for the operation
//   - registrationID: ID of the registration to retrieve and dispatch event for
//   - event: lifecycle event type to dispatch (e.g., "register", "change", "delete")
func (handler *RegistrationHandler) dispatchLifecycleByID(ctx context.Context, registrationID string, event string) {
	if handler.dispatcher == nil || strings.TrimSpace(registrationID) == "" {
		return
	}

	registration, err := handler.service.GetByID(registrationID, ctx)
	if err != nil {
		log.Printf("could not resolve registration %s for webhook dispatch: %v", registrationID, err)
		return
	}
	handler.dispatchLifecycleFromRegistration(registration, event)
}

// dispatchLifecycleFromRegistration dispatches a lifecycle event based on the isoCode in the registration data.
//
// Parameters:
//   - registration: map containing registration data, expected to have an "isoCode" field
//   - event: lifecycle event type to dispatch (e.g., "register", "change", "delete")
func (handler *RegistrationHandler) dispatchLifecycleFromRegistration(registration any, event string) {
	if handler.dispatcher == nil || registration == nil {
		return
	}

	isoCode, ok := extractISOCode(registration)
	if !ok || strings.TrimSpace(isoCode) == "" {
		log.Printf("skipping webhook dispatch for event %s: missing isoCode", event)
		return
	}

	handler.dispatcher.DispatchLifecycleAsync(strings.ToUpper(strings.TrimSpace(isoCode)), event)
}

// extractISOCode supports the two registration payload shapes used in this handler:
// map data from GetAll and struct data from GetByID.
func extractISOCode(registration any) (string, bool) {
	switch val := registration.(type) {
	case map[string]interface{}:
		isoCode, ok := val["isoCode"].(string)
		return isoCode, ok
	case *structs.RegisterCountry:
		return val.IsoCode, true
	default:
		return "", false
	}
}
