package services

import (
	"context"
	"envdash/internal/store"
	"envdash/internal/structs"
	"envdash/internal/utils"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
)

type RegistrationService struct {
	client *firestore.Client
}

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
func (service *RegistrationService) Post(ctx context.Context, registration structs.RegisterCountry) (string, error) {
	if validationErr := utils.Validation(&registration); validationErr != nil {
		return "", validationErr
	}

	registration.LastChange = time.Now()

	isoExists, isoErr := service.isoCodeExists(ctx, registration.IsoCode)
	if isoErr != nil {
		return "", isoErr
	}
	if isoExists {
		return "", errors.New("isoCode already exists in registration collection")
	}

	registrationDoc := service.client.Collection(store.REGISTRATIONCOLLECTION).NewDoc()
	registration.ID = registrationDoc.ID

	_, creationError := registrationDoc.Set(ctx, registration)
	if creationError != nil {
		return "", creationError
	}
	return registrationDoc.ID, nil
}

// GetAll retrieves all registrations from the Firestore collection.
//
// Parameters:
//   - ctx: context for the operation
//
// Returns:
//   - []map[string]interface{}: slice of registration document maps on success
//   - error: error if collection is empty or query fails
func (service *RegistrationService) GetAll(ctx context.Context) ([]map[string]interface{}, error) {
	var formattedRegistrations []map[string]interface{}

	registrations, registrationsErr := service.client.Collection(store.REGISTRATIONCOLLECTION).Documents(ctx).GetAll()
	if registrationsErr != nil {
		return nil, registrationsErr
	}

	// Check if collection is empty
	if len(registrations) == 0 {
		return nil, errors.New("collection is empty")
	}

	for _, registration := range registrations {
		formattedRegistrations = append(formattedRegistrations, registration.Data())
	}
	return formattedRegistrations, nil
}

// GetByID retrieves a specific registration by its document ID.
//
// Parameters:
//   - registrationID: document ID to retrieve
//   - ctx: context for the operation
//
// Returns:
//   - map[string]interface{}: registration document data on success
//   - error: error if not found or query fails
func (service *RegistrationService) GetByID(registrationID string, ctx context.Context) (map[string]interface{}, error) {
	registration, registrationErr := service.client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID).Get(ctx)
	if registrationErr != nil {
		return nil, registrationErr
	}
	return registration.Data(), nil
}

// DeleteAll removes all documents from the registrations collection.
//
// Parameters:
//   - ctx: context for the operation
//
// Returns:
//   - error: error if collection is empty or deletion fails
func (service *RegistrationService) DeleteAll(ctx context.Context) error {
	registrations, registrationsErr := service.client.Collection(store.REGISTRATIONCOLLECTION).
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
		return errors.New("Registration not found")
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
func (service *RegistrationService) Put(newRegistration *structs.RegisterCountry, registrationID string, ctx context.Context) (string, error) {
	exists := service.registrationExists(registrationID, ctx)
	if exists != nil {
		return "", exists
	}
	if validationErr := utils.Validation(newRegistration); validationErr != nil {
		return "", validationErr
	}
	newRegistration.LastChange = time.Now()
	_, registrationErr := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Doc(registrationID).Set(ctx, newRegistration)
	if registrationErr != nil {
		return "", registrationErr
	}
	return registrationID, registrationErr
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
	dataUpdate map[string]any) error {
	exists := service.registrationExists(registrationID, ctx)
	if exists != nil {
		return exists
	}

	registration := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Doc(registrationID)

	dataUpdate["lastChange"] = time.Now()

	updates, fieldsErr := toUpdateFields(dataUpdate)
	if fieldsErr != nil {
		return fieldsErr
	}

	_, updateErr := registration.Update(ctx, updates)
	if updateErr != nil {
		return updateErr
	}

	return nil
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
func (service *RegistrationService) HeadAllRegistrations(ctx context.Context) (int, error) {
	totalRegistrations, err := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Documents(ctx).GetAll()
	if err != nil {
		return 0, err
	}
	return len(totalRegistrations), nil
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
	ctx context.Context) (map[string]interface{}, error) {
	registration, registrationErr := service.GetByID(registrationID, ctx)
	if registrationErr != nil {
		return nil, registrationErr
	}
	return registration, nil
}

// isoCodeExists checks if an ISO code already exists in the registration collection.
//
// Parameters:
//   - ctx: context for the operation
//   - isoCode: ISO code to check for duplicates
//
// Returns:
//   - bool: true if ISO code already exists, false otherwise
//   - error: error if query fails
func (service *RegistrationService) isoCodeExists(ctx context.Context, isoCode string) (bool, error) {
	normalizedIsoCode := strings.ToUpper(strings.TrimSpace(isoCode))
	registration, registrationErr := service.client.
		Collection(store.REGISTRATIONCOLLECTION).
		Where("isoCode", "==", normalizedIsoCode).
		Limit(1).
		Documents(ctx).
		GetAll()
	if registrationErr != nil {
		return false, registrationErr
	}
	return len(registration) > 0, nil
}
