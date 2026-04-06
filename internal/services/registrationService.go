package services

import (
	"context"
	"envdash/internal/store"
	"envdash/internal/structs"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"cloud.google.com/go/firestore"
)

type RegistrationService struct {
	client *firestore.Client
}

// NewRegistrationService creates a new RegistrationService instance.
// Parameters:
//   - client: A Firestore client (cannot be nil, will panic if nil)
// Returns: A pointer to a new RegistrationService instance
func NewRegistrationService(client *firestore.Client) *RegistrationService {
	if client == nil {
		panic("firestore client cannot be nil")
	}
	return &RegistrationService{
		client: client,
	}
}

// Post validates and creates a new registration in Firestore.
// Parameters:
//   - ctx: Context for request cancellation and timeouts
//   - registration: RegisterCountry struct containing registration data to validate and store
// Returns:
//   - string: The generated Firestore document ID on success
//   - error: Validation error, ISO code duplicate error, or Firestore error
func (service *RegistrationService) Post(ctx context.Context, registration structs.RegisterCountry) (string, error) {
	if validationErr := validation(&registration); validationErr != nil {
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

// validation validates and normalizes a complete RegisterCountry payload.
// Parameters:
//   - registration: Pointer to RegisterCountry struct to validate (fields will be modified in place)
// Returns:
//   - error: Validation error if any field is invalid, nil if all fields pass validation
func validation(registration *structs.RegisterCountry) error {
	var err error

	if registration.Name, err = validateName(registration.Name); err != nil {
		return err
	}
	if registration.IsoCode, err = validateIsoCode(registration.IsoCode, "isoCode", 2); err != nil {
		return err
	}

	currencies := make([]string, 0, len(registration.Features.TargetCurrencies))
	for _, currency := range registration.Features.TargetCurrencies {
		currency, err = validateIsoCode(currency, "currency", 3)
		if err != nil {
			return err
		}
		currencies = append(currencies, currency)
	}
	registration.Features.TargetCurrencies = currencies

	if registration.Features.TargetCurrencies, err = validateTargetCurrencies(registration.Features.TargetCurrencies); err != nil {
		return err
	}
	return nil
}


// validateTargetCurrencies validates and normalizes a list of currency ISO codes.
// Parameters:
//   - oldCurrencies: Slice of currency codes to validate
// Returns:
//   - []string: Normalized currency codes in uppercase on success
//   - error: Validation error if any currency code is invalid
func validateTargetCurrencies(oldCurrencies []string) ([]string, error) {
	currencies := make([]string, 0, len(oldCurrencies))
	for _, currency := range oldCurrencies {
		var curErr error
		currency, curErr = validateIsoCode(currency, "currency", 3)
		if curErr != nil {
			return nil, curErr
		}
		currencies = append(currencies, currency)
	}
	return currencies, nil
}

// validateName validates and normalizes the name field.
// Parameters:
//   - name: The name string to validate
// Returns:
//   - string: Trimmed and validated name on success
//   - error: Error if name is empty or contains non-letter characters
func validateName(name string) (string, error) {
	name, nameErr := validString(name, "name")
	if nameErr != nil {
		return "", nameErr
	}
	return name, nil
}

// validateIsoCode validates and normalizes an ISO code field.
// Parameters:
//   - value: The ISO code string to validate
//   - field: Field name for error messages
//   - length: Expected length of the ISO code (e.g., 2 for country codes, 3 for currency codes)
// Returns:
//   - string: Normalized ISO code in uppercase on success
//   - error: Error if code is invalid, wrong length, or contains non-letter characters
func validateIsoCode(value, field string, length int) (string, error) {
	isoCode, isoCodeErr := validString(value, field)
	if isoCodeErr != nil {
		return "", isoCodeErr
	}
	if len(isoCode) != length {
		return "", fmt.Errorf("%s must be %d letters", field, length)
	}
	return strings.ToUpper(isoCode), nil
}

// validString removes whitespace and validates a string field.
// Parameters:
//   - input: The string to validate
//   - field: Field name for error messages
// Returns:
//   - string: Trimmed and validated string on success
//   - error: Error if string is empty after trimming or contains non-letter characters
func validString(input, field string) (string, error) {
	input = strings.Join(strings.Fields(input), "")
	if input == "" {
		return "", errors.New("missing required field: " + field)
	}
	for _, letter := range input {
		if !unicode.IsLetter(letter) {
			return "", errors.New(field + " must contain letters only")
		}
	}
	return input, nil
}


// GetAll retrieves all registrations from the Firestore collection.
// Parameters:
//   - ctx: Context for request cancellation and timeouts
// Returns:
//   - []map[string]interface{}: Slice of registration documents on success
//   - error: Error if collection is empty or Firestore query fails
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
// Parameters:
//   - registrationID: The Firestore document ID of the registration to retrieve
//   - ctx: Context for request cancellation and timeouts
// Returns:
//   - map[string]interface{}: The registration document data on success
//   - error: Error if document not found or Firestore query fails
func (service *RegistrationService) GetByID(registrationID string, ctx context.Context) (map[string]interface{}, error) {
	registration, registrationErr := service.client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID).Get(ctx)
	if registrationErr != nil {
		return nil, registrationErr
	}
	return registration.Data(), nil
}

// DeleteAll removes all documents from the registrations collection.
// Parameters:
//   - ctx: Context for request cancellation and timeouts
// Returns:
//   - error: Error if collection is empty or Firestore deletion fails
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
// Parameters:
//   - registrationID: The Firestore document ID of the registration to delete
//   - ctx: Context for request cancellation and timeouts
// Returns:
//   - bool: true if registration was deleted successfully, false otherwise
//   - error: Error indicating why deletion failed
func (service *RegistrationService) DeleteByID(registrationID string, ctx context.Context) (bool, error) {
	if service.registrationExists(registrationID, ctx) == nil {
		_, deletionErr := service.client.Collection(store.REGISTRATIONCOLLECTION).
			Doc(registrationID).Delete(ctx)
		if deletionErr != nil {
			return false, deletionErr
		}
		return true, nil
	}
	return false, service.registrationExists(registrationID, ctx)
}

// registrationExists checks if a registration document exists by its ID.
// Parameters:
//   - registrationID: The Firestore document ID to check
//   - ctx: Context for request cancellation and timeouts
// Returns:
//   - error: nil if registration exists, error message "Registration not found" if it doesn't
func (service *RegistrationService) registrationExists(registrationID string, ctx context.Context) error {
	_, registrationErr := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Doc(registrationID).Get(ctx)
	if registrationErr != nil {
		return errors.New("Registration not found")
	}
	return nil
}

// Put replaces an entire registration document with new data.
// Parameters:
//   - newRegistration: Pointer to RegisterCountry struct with replacement data (will be validated and modified)
//   - registrationID: The Firestore document ID of the registration to replace
//   - ctx: Context for request cancellation and timeouts
// Returns:
//   - string: The registration ID on success
//   - error: Validation error, registration not found error, or Firestore error
func (service *RegistrationService) Put(newRegistration *structs.RegisterCountry, registrationID string, ctx context.Context) (string, error) {
	exists := service.registrationExists(registrationID, ctx)
	if exists != nil {
		return "", exists
	}
	if validationErr := validation(newRegistration); validationErr != nil {
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


// Patch updates only specified fields of a registration document.
// Parameters:
//   - registrationID: The Firestore document ID of the registration to update
//   - ctx: Context for request cancellation and timeouts
//   - dataUpdate: Map of field names to new values to update (lastChange is automatically set)
// Returns:
//   - error: Validation error, registration not found error, or Firestore update error
func (service *RegistrationService) Patch(registrationID string, ctx context.Context,
	dataUpdate map[string]any) error {
	exists := service.registrationExists(registrationID, ctx)
	if exists != nil {
		return exists
	}

	if validationErr := validatePatchUpdate(dataUpdate); validationErr != nil {
		return validationErr
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

// validatePatchUpdate validates all fields in a PATCH update map.
// Parameters:
//   - dataUpdate: Map of field names to values being updated
// Returns:
//   - error: Validation error if any field is invalid, nil if all fields pass validation
func validatePatchUpdate(dataUpdate map[string]any) error {
	for key, value := range dataUpdate {
		if patchFailed := validatePatchField(key, value); patchFailed != nil {
			return patchFailed
		}
	}
	return nil
}

// validatePatchField validates a single field for PATCH updates.
// Parameters:
//   - key: The field name being updated (name, isoCode, features, etc.)
//   - value: The new value for the field
// Returns:
//   - error: Validation error if field is invalid, nil if field passes validation
func validatePatchField(key string, value any) error {
	switch key {
	case "name":
		return validatePatchName(value)
	case "isoCode":
		return validatePatchIsoCode(value)
	case "features":
		return validatePatchFeatures(value)
	}
	return nil
}

// validatePatchName validates the name field in a PATCH update.
func validatePatchName(value any) error {
	if name, ok := value.(string); ok {
		if _, nameErr := validateName(name); nameErr != nil {
			return nameErr
		}
	}
	return nil
}

// validatePatchIsoCode validates the isoCode field in a PATCH update.
func validatePatchIsoCode(value any) error {
	if isoCode, ok := value.(string); ok {
		if _, isoCodeErr := validateIsoCode(isoCode, "isoCode", 2); isoCodeErr != nil {
			return isoCodeErr
		}
	}
	return nil
}

// validatePatchFeatures validates the features field in a PATCH update.
func validatePatchFeatures(value any) error {
	if features, ok := value.(map[string]interface{}); ok {
		if targetCurrencies, exists := features["targetCurrencies"].([]interface{}); exists {
			currencies := make([]string, len(targetCurrencies))
			for key, currency := range targetCurrencies {
				if currencyStr, ok := currency.(string); ok {
					currencies[key] = currencyStr
				}
			}
			if _, err := validateTargetCurrencies(currencies); err != nil {
				return err
			}
		}
	}
	return nil
}

// toUpdateFields converts a map to Firestore Update objects for partial updates.
// Parameters:
//   - dataUpdate: Map of field names to update values
// Returns:
//   - []firestore.Update: Slice of Update objects ready for Firestore
//   - error: Always nil; included for consistency with other conversion functions
func toUpdateFields(dataUpdate map[string]any) ([]firestore.Update, error) {
	updates := make([]firestore.Update, 0, len(dataUpdate))

	for key, value := range dataUpdate {
		updates = append(updates, firestore.Update{
			Path:  key,
			Value: value,
		})
	}
	return updates, nil
}

// HeadAllRegistrations returns the total count of all registrations in the collection.
// Parameters:
//   - ctx: Context for request cancellation and timeouts
// Returns:
//   - int: The total number of registrations on success
//   - error: Error if Firestore query fails
func (service *RegistrationService) HeadAllRegistrations(ctx context.Context) (int, error) {
	totalRegistrations, err := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Documents(ctx).GetAll()
	if err != nil {
		return 0, err
	}
	return len(totalRegistrations), nil
}

// HeadOneRegistration retrieves a single registration without modifying it (used for HEAD requests).
// Parameters:
//   - registrationID: The Firestore document ID of the registration to check
//   - ctx: Context for request cancellation and timeouts
// Returns:
//   - map[string]interface{}: The registration document data on success
//   - error: Error if registration not found or Firestore query fails
func (service *RegistrationService) HeadOneRegistration(registrationID string,
	ctx context.Context) (map[string]interface{}, error) {
	registration, registrationErr := service.GetByID(registrationID, ctx)
	if registrationErr != nil {
		return nil, registrationErr
	}
	return registration, nil
}


// isoCodeExists checks if an ISO code already exists in the registration collection.
// Parameters:
//   - ctx: Context for request cancellation and timeouts
//   - isoCode: The ISO code to check for duplicates
// Returns:
//   - bool: true if ISO code already exists, false otherwise
//   - error: Error if Firestore query fails
func (service *RegistrationService) isoCodeExists(ctx context.Context, isoCode string) (bool, error) {
	normalizedIsoCode := strings.ToUpper(strings.TrimSpace(isoCode))
	registration, registrationErr := service.client.
		Collection(store.REGISTRATIONCOLLECTION).
		Where("IsoCode", "==", normalizedIsoCode).
		Limit(1).
		Documents(ctx).
		GetAll()
	if registrationErr != nil {
		return false, registrationErr
	}
	return len(registration) > 0, nil
}
