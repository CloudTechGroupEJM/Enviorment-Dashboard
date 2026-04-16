package handlers

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/services/apiKey"
	"envdash/internal/structs"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
)

// authHandler sets up authentication routes
// POST Register a new client and receive an API key
// DELETE Revoke an API key
//
// Parameters:
//   - router: HTTP request multiplexer to register routes
//   - firestoreClient: Firestore database client
func InitAuthentication(router *http.ServeMux, firestoreClient *firestore.Client) {
	apiKeyServiceInstance := apiKey.NewAPIKeyService(firestoreClient)

	// POST (Register new client)
	router.HandleFunc(http.MethodPost+ " " + config.AUTH_PAGE_PATH, func(writer http.ResponseWriter, request *http.Request) {
		registerNewClientHandler(writer, request, apiKeyServiceInstance)
	})

	// DELETE (Revoke API key)
	router.HandleFunc(http.MethodDelete + " " + config.AUTH_PAGE_PATH +"{apiKeyValue}", func(writer http.ResponseWriter, request *http.Request) {
		revokeAPIKeyHandler(writer, request, apiKeyServiceInstance)
	})
}

// registerNewClientHandler handles POST requests
// Accepts JSON with name and email, returns API key
//
// Parameters:
//   - writer: HTTP response writer
//   - request: HTTP request object
//   - apiKeyServiceInstance: Service for creating API keys
func registerNewClientHandler(writer http.ResponseWriter, request *http.Request, apiKeyServiceInstance *apiKey.APIKeyService) {

	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var registrationRequestData structs.APIKeyRegistration
	decodingError := json.NewDecoder(request.Body).Decode(&registrationRequestData)
	if decodingError != nil {
		log.Printf("error: %s ", decodingError)
		http.Error(writer, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Register client and generate API key
	apiKeyResponseData, registrationError := apiKeyServiceInstance.RegisterNewClient(request.Context(), registrationRequestData)
	if registrationError != nil {
		writer.WriteHeader(http.StatusBadRequest)
		http.Error(writer, registrationError.Error(), http.StatusBadRequest)
		return
	}

	// Return 201 Created with API key
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	json.NewEncoder(writer).Encode(apiKeyResponseData)
}

// revokeAPIKeyHandler handles DELETE requests
// Deactivates an API key
//
// Parameters:
//   - writer: HTTP response writer
//   - request: HTTP request object
//   - apiKeyServiceInstance: Service for revoking API keys
func revokeAPIKeyHandler(writer http.ResponseWriter, request *http.Request, apiKeyServiceInstance *apiKey.APIKeyService) {
	// Only allow DELETE
	if request.Method != http.MethodDelete {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Extract API key from URL path parameter
	apiKeyValue := request.PathValue("apiKeyValue")

	// Revoke the API key
	revocationError := apiKeyServiceInstance.RevokeAPIKey(request.Context(), apiKeyValue)
	if revocationError != nil {
		log.Println(revocationError)
		http.Error(writer, 
								revocationError.Error(), 
								http.StatusNotFound)
		return
	}

	log.Println("Api Key: " + apiKeyValue + " has been revoked.")
	writer.WriteHeader(http.StatusNoContent)
}
