package services

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"envdash/internal/store"
	"envdash/internal/structs"
	"envdash/internal/utils"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
)

// APIKeyService handles all API key operations
// Fields:
//   - firestoreClient: Reference to Firestore database client
type APIKeyService struct {
    client *firestore.Client
}

// NewAPIKeyService creates and returns a new APIKeyService instance
// 
// Parameters:
//   - firestoreClient: Firestore client (cannot be nil)
//
// Returns:
//   - *APIKeyService: Initialized service instance
//   - Panics if firestoreClient is nil
func NewAPIKeyService(client *firestore.Client) *APIKeyService {
    if client == nil {
        panic("firestore client cannot be nil")
    }
    return &APIKeyService{
        client: client,
    }
}

// RegisterNewClient creates a new API key for a client application
//
// Parameters:
//   - ctx: Context for database operation
//   - registrationData: APIKeyRegistration with client name and email
//
// Returns:
//   - *structs.APIKeyResponse: Contains generated key and creation timestamp
//   - error: Validation error or Firestore error
func (apiKeyService *APIKeyService) RegisterNewClient(ctx context.Context, registrationData structs.APIKeyRegistration) (*structs.APIKeyResponse, error) {
    // Validate input fields
    if strings.TrimSpace(registrationData.Name) == "" {
        return nil, errors.New("missing required field: name")
    }
    if strings.TrimSpace(registrationData.Email) == "" {
        return nil, errors.New("missing required field: email")
    }

    // Validate email format
    if emailValidationError := utils.ValidateEmailFormat(registrationData.Email); emailValidationError != nil {
        return nil, emailValidationError
    }

    
    // Check if email already exists in database
    emailExistsInDatabase, emailCheckError := apiKeyService.emailExists(ctx, registrationData.Email)
    if emailCheckError != nil {
        return nil, emailCheckError
    }
    if emailExistsInDatabase {
        return nil, errors.New("email already registered and has an active API key")
    }

    generatedAPIKey := generateAPIKey()
    
    currentTimestamp := time.Now().Format("20060102 15:04")

    // Create doc
    newDocumentReference := apiKeyService.client.Collection(store.APIKEYCOLLECTION).NewDoc()
    apiKeyStoreData := structs.APIKeyStoreModel{
        ID:        newDocumentReference.ID,
        Key:       generatedAPIKey,
        Name:      registrationData.Name,
        Email:     registrationData.Email,
        CreatedAt: currentTimestamp,
        IsActive:  true,
        LastUsed:  currentTimestamp,
    }

    _, writeError := newDocumentReference.Set(ctx, apiKeyStoreData)
    if writeError != nil {
        return nil, writeError
    }

    apiKeyResponseData := &structs.APIKeyResponse{
        Key:       generatedAPIKey,
        CreatedAt: currentTimestamp,
    }

    return apiKeyResponseData, nil
}

// generateAPIKey creates a unique API key using HMAC-SHA256
// Format: sk-envdash-{64-char-hex-hash}
//
// Returns:
//   - string: Generated API key
func generateAPIKey() string {
    keyPrefix := "sk-envdash-"
    
    randomBytesLength := 16
    randomBytes := make([]byte, randomBytesLength)
    
    _, readError := rand.Read(randomBytes)
    if readError != nil {
        // Fallback to timestamp
        randomBytes = []byte(time.Now().String())
    }
    
    secretKeyForHmac := []byte(time.Now().String())
    hmacHasher := hmac.New(sha256.New, secretKeyForHmac)
    hmacHasher.Write(randomBytes)
    
    hashSum := hmacHasher.Sum(nil)
    
    hexEncodedHash := hex.EncodeToString(hashSum)
    
    return keyPrefix + hexEncodedHash[:32]
}

// ValidateAPIKey checks if the provided key is valid and active
//
// Parameters:
//   - ctx: Context for database operation
//   - apiKeyToValidate: The API key to check
//
// Returns:
//   - *structs.APIKeyStoreModel: Key data if valid and active
//   - bool: true if key exists and is active
//   - error: Firestore query error
func (apiKeyService *APIKeyService) ValidateAPIKey(ctx context.Context, apiKeyToValidate string) (*structs.APIKeyStoreModel, bool, error) {
    querySnapshot := apiKeyService.client.Collection(store.APIKEYCOLLECTION).
        Where("key", "==", apiKeyToValidate).
        Documents(ctx)

    retrievedDocuments, queryError := querySnapshot.GetAll()
    if queryError != nil {
        return nil, false, queryError
    }

    // No document found
    if len(retrievedDocuments) == 0 {
        return nil, false, nil
    }

    // Extract first (and should be only) document
    keyDocumentData := retrievedDocuments[0].Data()
    isActiveStatus := keyDocumentData["isActive"].(bool)

    // Reconstruct APIKeyStoreModel from Firestore data
    retrievedAPIKeyModel := &structs.APIKeyStoreModel{
        ID:        retrievedDocuments[0].Ref.ID,
        Key:       apiKeyToValidate,
        Name:      keyDocumentData["name"].(string),
        Email:     keyDocumentData["email"].(string),
        CreatedAt: keyDocumentData["createdAt"].(string),
        IsActive:  isActiveStatus,
        LastUsed:  keyDocumentData["lastUsed"].(string),
    }

    return retrievedAPIKeyModel, isActiveStatus, nil
}

// RevokeAPIKey deactivates an API key
//
// Parameters:
//   - ctx: Context for database operation
//   - apiKeyToRevoke: The API key to revoke
//
// Returns:
//   - error: Firestore error or "key not found" error
func (apiKeyService *APIKeyService) RevokeAPIKey(ctx context.Context, apiKeyToRevoke string) error {
    querySnapshot := apiKeyService.client.Collection(store.APIKEYCOLLECTION).
        Where("key", "==", apiKeyToRevoke).
        Documents(ctx)

    retrievedDocuments, queryError := querySnapshot.GetAll()
    if queryError != nil {
        return queryError
    }

    if len(retrievedDocuments) == 0 {
        return errors.New("api key not found")
    }

    documentReference := retrievedDocuments[0].Ref
    _, updateError := documentReference.Update(ctx, []firestore.Update{
        {
            Path:  "isActive",
            Value: false,
        },
    })

    return updateError
}

// UpdateLastUsedTimestamp updates the lastUsed field for an API key
//
// Parameters:
//   - ctx: Context for database operation
//   - apiKeyDocumentID: The Firestore document ID
//
// Returns:
//   - error: Firestore update error
func (apiKeyService *APIKeyService) UpdateLastUsedTimestamp(ctx context.Context, apiKeyDocumentID string) error {
    currentTimestamp := time.Now().Format("20060102 15:04:05")
    
    documentReference := apiKeyService.client.Collection(store.APIKEYCOLLECTION).Doc(apiKeyDocumentID)
    _, updateError := documentReference.Update(ctx, []firestore.Update{
        {
            Path:  "lastUsed",
            Value: currentTimestamp,
        },
    })

    return updateError
}


// emailExists checks if an email already has an active API key registered
//
// Parameters:
//   - ctx: Context for database operation
//   - emailAddressToCheck: The email to check for existing registration
//
// Returns:
//   - bool: true if email exists and is active
//   - error: Firestore query error
func (apiKeyService *APIKeyService) emailExists(ctx context.Context, emailAddressToCheck string) (bool, error) {
    querySnapshot := apiKeyService.client.Collection(store.APIKEYCOLLECTION).
        Where("email", "==", emailAddressToCheck).
        Where("isActive", "==", true).
        Documents(ctx)

    retrievedDocuments, queryError := querySnapshot.GetAll()
    if queryError != nil {
        return false, queryError
    }

    // If any documents found, email already exists
    return len(retrievedDocuments) > 0, nil
}