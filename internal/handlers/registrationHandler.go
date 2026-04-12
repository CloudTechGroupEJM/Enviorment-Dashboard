package handlers

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/services/registration"
	"envdash/internal/structs"
	"log"
	"net/http"
	"strconv"

	"cloud.google.com/go/firestore"
)

const SUCCESSFUL_EXECUTION = "Request successfully executed"
const INVALID_JSON = "Invalid JSON payload"

type RegistrationHandler struct {
	service *registration.RegistrationService
}

// InitRegistration initializes the registration service, handler and endpoints.
// Parameters: router - HTTP router, client - Firestore client
func InitRegistration(router *http.ServeMux, client *firestore.Client) {
	service := registration.NewRegistrationService(client)
	handler := &RegistrationHandler{
		service: service,
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

	registrationID, creationErr := handler.service.Post(request.Context(), registration)
	if creationErr != nil {
		http.Error(writer, creationErr.Error(), http.StatusBadRequest)
		log.Println("Error when creating registration: " + creationErr.Error())
		return
	}

	writer.Header().Set(config.HEADER_CONTENT_TYPE, config.APPLICATION_JSON)
	writer.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(writer).Encode(map[string]string{
		"id":         registrationID,
		"lastChange": registration.LastChange.String(),
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
		deletionErr := handler.service.DeleteAll(request.Context())
		if deletionErr != nil {
			log.Printf("Error nothing to delete %s", deletionErr)
			http.Error(writer, "Nothing to delete.", http.StatusNotFound)
			return
		} else {
			http.Error(writer, "All registration deleted", http.StatusOK)
		}
	} else {
		deleted, deletionErr := handler.service.DeleteByID(request.PathValue("id"), request.Context())
		if deletionErr != nil {
			http.Error(writer, "Error when trying to delete.", http.StatusOK)
			log.Println("Deletion error when deleting by ID")
			return
		}
		if deleted {
			http.Error(writer, "registration has been deleted", http.StatusOK)
			log.Println("registration with id " + request.PathValue("id") + " has been deleted")
		} else {
			http.Error(writer, "Error registration doesn't exist", http.StatusNoContent)
			log.Println("Registration: " + request.PathValue("id") + " doesnt exist")
			return
		}
	}
	log.Println(SUCCESSFUL_EXECUTION)
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
