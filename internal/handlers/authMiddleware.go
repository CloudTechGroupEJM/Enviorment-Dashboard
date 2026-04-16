package handlers

import (
	"context"
	"envdash/internal/services/apiKey"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
)

// ProtectedRouteMiddleware wraps an HTTP handler with API key validation
// Routes that don't require authentication (like status endpoint) should NOT use this middleware
//
// Parameters:
//   - protectedHandlerFunction: The actual handler to wrap
//   - apiKeyServiceInstance: Service for validating API keys
//   - firestoreClientInstance: Firestore client for database operations
//
// Returns:
//   - http.HandlerFunc: Wrapped handler that validates API key before executing the original handler
func ProtectedRouteMiddleware(protectedHandlerFunction http.HandlerFunc, apiKeyServiceInstance *apiKey.APIKeyService, firestoreClientInstance *firestore.Client) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		// Extract API key from X-API-Key header
		apiKeyFromHeader := request.Header.Get("X-API-Key")

		// Check if API key header is present
		if apiKeyFromHeader == "" {
			log.Println("missing X-API-Key in header")
			http.Error(writer, 
								"not authorized you need an api key and define it in header", 
								http.StatusUnauthorized )
			return
		}

		ctx := request.Context()

		// Validate the API key
		retrievedAPIKeyModel, isKeyActive, validationError := apiKeyServiceInstance.ValidateAPIKey(ctx, apiKeyFromHeader)

		if validationError != nil {
			log.Println("database error during key validation")
			http.Error(writer, 
								"database error during key validation", 
								http.StatusInternalServerError)
			return
		}

		// Check if key exists and is active
		if !isKeyActive || retrievedAPIKeyModel == nil {
			log.Println("invalid or revoked API key")
			http.Error(writer, 
								"invalid or revoked API key", 
								http.StatusForbidden)
			return
		}

		// Update last used timestamp in background (don't block request on this)
		go apiKeyServiceInstance.UpdateLastUsedTimestamp(ctx, retrievedAPIKeyModel.ID)

		// Add API key info to request context for later use if needed
		contextWithKeyInfo := context.WithValue(ctx, "apiKeyDocumentID", retrievedAPIKeyModel.ID)
		contextWithKeyInfo = context.WithValue(contextWithKeyInfo, "apiKeyName", retrievedAPIKeyModel.Name)

		// Create new request with updated context
		requestWithContext := request.WithContext(contextWithKeyInfo)

		protectedHandlerFunction(writer, requestWithContext)
	}
}
