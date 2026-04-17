package registration

import (
	"context"
	"envdash/internal/config"
	"envdash/internal/store"
	"envdash/internal/structs"
	"envdash/internal/utils"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RegistrationService struct {
	client *firestore.Client
}

const UN_AUTHORIZED = "unauthorized this registration belongs to a different API key"
const REG_NOT_FOUND = "Registration not found"

// NewRegistrationService creates a new RegistrationService instance.
//
// Parameters:
//   - client: Firestore client (cannot be nil, will panic if nil)
//
// Returns:
//   - *RegistrationService: pointer to initialized RegistrationService instance
func NewRegistrationService(client *firestore.Client) *RegistrationService {
	if client == nil {
		panic("firestore client cannot be nil")
	}
	return &RegistrationService{
		client: client,
	}
}

// Post validates and creates a new registration in Firestore.
//
// Parameters:
//   - ctx: context for the operation
//   - registration: RegisterCountry struct with data to validate and store
//
// Returns:
//   - string: generated Firestore document ID on success
//   - error: validation error, duplicate ISO code error, or Firestore error
func (service *RegistrationService) Post(ctx context.Context, registration structs.RegisterCountry, usedApiKey string) (string, string, error) {
	if validationErr := utils.Validation(&registration); validationErr != nil {
		return "", "", validationErr
	}

	registration.LastChange = time.Now().Format(config.DATE_FORMAT)

	// Check if ISO code and country name are both provided or both missing
	hasIsoCode := registration.IsoCode != ""
	hasCountryName := registration.CountryName != ""

	if !hasIsoCode && !hasCountryName {
		return "", "", errors.New("either country name or ISO code must be provided")
	}

	if hasCountryName {
		countryExists, countryErr := service.countryNameExistsForApiKey(ctx, registration.CountryName, usedApiKey)
		if countryErr != nil {
			return "", "", countryErr
		}

		if countryExists {
			return "", "", errors.New("this API key has already registered this country")
		}
	}

	if hasIsoCode {
		isoExistsForApiKey, isoErr := service.isoCodeExistsForApiKey(ctx, registration.IsoCode, usedApiKey)
		if isoErr != nil {
			return "", "", isoErr
		}

		if isoExistsForApiKey {
			return "", "", errors.New("this API key has already registered this ISO code")
		}
	}

	if len(registration.Features.TargetCurrencies) == 0 {
		return "", "", errors.New("TargetCurrencies cant be empty, each country has a currency")
	}

	registrationDoc := service.client.Collection(store.REGISTRATIONCOLLECTION).NewDoc()
	registration.ID = registrationDoc.ID
	registration.ApiKeyID = usedApiKey

	_, creationError := registrationDoc.Set(ctx, registration)
	if creationError != nil {
		return "", "", creationError
	}
	return registrationDoc.ID, registration.LastChange, nil
}

// GetAll retrieves all registrations from the Firestore collection.
//
// Parameters:
//   - ctx: context for the operation
//
// Returns:
//   - []map[string]interface{}: slice of registration document maps on success
//   - error: error if collection is empty or query fails
func (service *RegistrationService) GetAll(ctx context.Context, usedApiKey string) ([]structs.RegisterCountry, error) {
	var allRegistrations []structs.RegisterCountry

	registrations, registrationsErr := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Where("apiKeyID", "==", usedApiKey).
		Documents(ctx).GetAll()
	if registrationsErr != nil {
		return nil, registrationsErr
	}

	// Check if collection is empty
	if len(registrations) == 0 {
		return nil, errors.New("collection is empty")
	}

	for _, registration := range registrations {
		var regStruct structs.RegisterCountry
		if err := registration.DataTo(&regStruct); err != nil {
			return nil, err
		}
		regStruct.ApiKeyID = ""
		allRegistrations = append(allRegistrations, regStruct)
	}
	return allRegistrations, nil
}

// GetByID retrieves a specific registration by its document ID.
//
// Parameters:
//   - registrationID: document ID to retrieve
//   - ctx: context for the operation
//
// Returns:
//   - *structs.RegisterCountry: registration document data on success
//   - error: error if not found or query fails
func (service *RegistrationService) GetByID(ctx context.Context, registrationID string, usedApiKey string) (*structs.RegisterCountry, error) {
	if strings.TrimSpace(registrationID) == "" {
		return nil, fmt.Errorf("registration id is required")
	}

	registrationSnapshot, err := service.client.
		Collection(store.REGISTRATIONCOLLECTION).
		Doc(registrationID).
		Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("registration with id %q not found", registrationID)
		}
		if status.Code(err) == codes.InvalidArgument {
			return nil, fmt.Errorf("invalid registration id %q", registrationID)
		}
		return nil, fmt.Errorf("loading registration %q: %w", registrationID, err)
	}

	// Create an empty struct and use DataTo to populate it
	var registrationStruct structs.RegisterCountry
	if err := registrationSnapshot.DataTo(&registrationStruct); err != nil {
		return nil, fmt.Errorf("decoding registration %q: %w", registrationID, err)
	}

	// Verify the API key matches
	if registrationStruct.ApiKeyID != usedApiKey {
		return nil, errors.New(UN_AUTHORIZED)
	}

	registrationStruct.ApiKeyID = ""

	return &registrationStruct, nil
}

// DeleteAll removes all documents from the registrations collection.
//
// Parameters:
//   - ctx: context for the operation
//
// Returns:
//   - error: error if collection is empty or deletion fails
func (service *RegistrationService) DeleteAll(ctx context.Context, usedApiKey string) error {
	registrations, registrationsErr := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Where("apiKeyID", "==", usedApiKey).
		Documents(ctx).GetAll()
	if registrationsErr != nil {
		return registrationsErr
	}

	// Check if collection is empty
	if len(registrations) == 0 {
		return errors.New("collection is empty.")
	}

	bulkWriter := service.client.BulkWriter(ctx)
	for _, registration := range registrations {
		_, deleteErr := bulkWriter.Delete(registration.Ref)
		if deleteErr != nil {
			return deleteErr
		}
	}
	bulkWriter.End()
	return nil
}

// DeleteByID deletes a specific registration by its document ID.
//
// Parameters:
//   - registrationID: document ID to delete
//   - ctx: context for the operation
//
// Returns:
//   - bool: true if deleted successfully, false otherwise
//   - error: error if deletion failed or registration not found
func (service *RegistrationService) DeleteByID(registrationID string, ctx context.Context) (bool, error) {
	existsErr := service.registrationExists(registrationID, ctx)
	if existsErr != nil {
		return false, existsErr
	}

	_, deletionErr := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Doc(registrationID).Delete(ctx)
	if deletionErr != nil {
		return false, deletionErr
	}
	return true, nil
}

// registrationExists checks if a registration document exists by its ID.
//
// Parameters:
//   - registrationID: document ID to check
//   - ctx: context for the operation
//
// Returns:
//   - error: nil if registration exists, "Registration not found" error if it doesn't
func (service *RegistrationService) registrationExists(registrationID string, ctx context.Context) error {
	_, registrationErr := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Doc(registrationID).Get(ctx)
	if registrationErr != nil {
		return errors.New(REG_NOT_FOUND)
	}
	return nil
}

// Put replaces an entire registration document with new data.
//
// Parameters:
//   - newRegistration: RegisterCountry pointer with replacement data
//   - registrationID: document ID to replace
//   - ctx: context for the operation
//
// Returns:
//   - string: registration ID on success
//   - error: validation error, not found error, or Firestore error
func (service *RegistrationService) Put(newRegistration *structs.RegisterCountry, registrationID string, ctx context.Context, usedApiKey string) (string, error) {
	// Get the document to verify it exists and API key matches
	snapshot, err := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Doc(registrationID).Get(ctx)
	if err != nil {
		return "", errors.New(REG_NOT_FOUND)
	}

	// Verify the API key matches
	var registration structs.RegisterCountry
	if err := snapshot.DataTo(&registration); err != nil {
		return "", err
	}
	if registration.ApiKeyID != usedApiKey {
		return "", errors.New(UN_AUTHORIZED)
	}

	if validationErr := utils.Validation(newRegistration); validationErr != nil {
		return "", validationErr
	}
	newRegistration.LastChange = time.Now().Format(config.DATE_FORMAT)
	newRegistration.ID = registrationID
	newRegistration.ApiKeyID = usedApiKey

	_, registrationErr := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Doc(registrationID).Set(ctx, newRegistration)
	if registrationErr != nil {
		return "", registrationErr
	}
	return registrationID, nil
}

// Patch performs a partial update of a registration document.
//
// Parameters:
//   - registrationID: document ID to update
//   - ctx: context for the operation
//   - dataUpdate: map of field names to update values
//
// Returns:
//   - error: not found error, validation error, or Firestore error
func (service *RegistrationService) Patch(registrationID string, ctx context.Context,
	dataUpdate map[string]any, usedApiKey string) error {

	// Get the document to verify API key ownership
	snapshot, err := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Doc(registrationID).Get(ctx)
	if err != nil {
		return errors.New(REG_NOT_FOUND)
	}

	// Verify the API key matches
	var registration structs.RegisterCountry
	if err := snapshot.DataTo(&registration); err != nil {
		return err
	}
	if registration.ApiKeyID != usedApiKey {
		return errors.New(UN_AUTHORIZED)
	}

	// Now proceed with the update
	registrationRef := service.client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID)
	dataUpdate["lastChange"] = time.Now().Format(config.DATE_FORMAT)

	updates, fieldsErr := toUpdateFields(dataUpdate)
	if fieldsErr != nil {
		return fieldsErr
	}

	_, updateErr := registrationRef.Update(ctx, updates)
	return updateErr
}

// toUpdateFields converts a map to Firestore Update objects for partial updates.
//
// Parameters:
//   - dataUpdate: map of field names to update values
//
// Returns:
//   - []firestore.Update: slice of Firestore Update objects for partial updates
//   - error: error from field validation
func toUpdateFields(dataUpdate map[string]any) ([]firestore.Update, error) {
	updates := make([]firestore.Update, 0, len(dataUpdate))

	for key, value := range dataUpdate {
		validatedVal, err := utils.ValidateFieldValue(key, value)
		if err != nil {
			return nil, err
		}
		updates = append(updates, firestore.Update{
			Path:  key,
			Value: validatedVal,
		})
	}

	// Sort by Path field alphabetically
	sort.Slice(updates, func(current, next int) bool {
		return updates[current].Path < updates[next].Path
	})
	return updates, nil
}

// HeadAllRegistrations returns the total count of all registrations in the collection.
//
// Parameters:
//   - ctx: context for the operation
//
// Returns:
//   - int: total number of registrations
//   - error: error if query fails
func (service *RegistrationService) HeadAllRegistrations(ctx context.Context, usedApiKey string) (int, error) {
	registrations, err := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Where("apiKeyID", "==", usedApiKey).
		Documents(ctx).GetAll()
	if err != nil {
		return 0, err
	}
	return len(registrations), nil
}

// HeadOneRegistration retrieves a single registration without modifying it (used for HEAD requests).
//
// Parameters:
//   - registrationID: document ID to check
//   - ctx: context for the operation
//
// Returns:
//   - map[string]interface{}: registration document data on success
//   - error: error if not found or query fails
func (service *RegistrationService) HeadOneRegistration(registrationID string,
	ctx context.Context, usedApiKey string) (*structs.RegisterCountry, error) {
	registration, registrationErr := service.GetByID(ctx, registrationID, usedApiKey)
	if registrationErr != nil {
		return nil, registrationErr
	}
	return registration, nil
}

// isoCodeExistsForApiKey checks if a specific API key has already registered an ISO code
func (service *RegistrationService) isoCodeExistsForApiKey(ctx context.Context, isoCode string, apiKeyID string) (bool, error) {
	querySnapshot := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Where("isoCode", "==", isoCode).
		Where("apiKeyID", "==", apiKeyID).
		Documents(ctx)

	retrievedDocuments, queryError := querySnapshot.GetAll()
	if queryError != nil {
		return false, queryError
	}

	return len(retrievedDocuments) > 0, nil
}

// countryNameExistsForApiKey checks if a specific API key has already registered a country name
func (service *RegistrationService) countryNameExistsForApiKey(ctx context.Context, countryName string, apiKeyID string) (bool, error) {
	querySnapshot := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Where("country", "==", countryName).
		Where("apiKeyID", "==", apiKeyID).
		Documents(ctx)

	retrievedDocuments, queryError := querySnapshot.GetAll()
	if queryError != nil {
		return false, queryError
	}

	return len(retrievedDocuments) > 0, nil
}
