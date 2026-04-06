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

// NewRegistrationService creates and returns a new RegistrationService instance with the provided Firestore client.
func NewRegistrationService(client *firestore.Client) *RegistrationService {
	if client == nil {
		panic("firestore client cannot be nil")
	}
	return &RegistrationService{
		client: client,
	}
}

// Post validates and creates a new registration in Firestore, returning the generated document ID.
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

// validation validates and normalizes a RegisterCountry payload (name, ISO code, and target currencies), returning an error on the first invalid field.
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

	return nil
}

// validateName trims and validates the name field, returning an error if it is empty or contains non-letter characters.
func validateName(name string) (string, error) {
	name, nameErr := validString(name, "name")
	if nameErr != nil {
		return "", nameErr
	}
	return name, nil
}

// validateIsoCode trims, validates alphabetic content and exact length, then returns the ISO code in uppercase.
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

// validString removes whitespace and validates that the field is non-empty and contains letters only.
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
func (service *RegistrationService) GetByID(registrationID string, ctx context.Context) (map[string]interface{}, error) {
	registration, registrationErr := service.client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID).Get(ctx)
	if registrationErr != nil {
		return nil, registrationErr
	}
	return registration.Data(), nil
}

// DeleteAll removes all documents from the registrations collection using BulkWriter.
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
func (service *RegistrationService) registrationExists(registrationID string, ctx context.Context) error {
	_, registrationErr := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Doc(registrationID).Get(ctx)
	if registrationErr != nil {
		return registrationErr
	}
	return nil
}

// Put replaces an entire registration document with new data if it exists.
func (service *RegistrationService) Put(newRegistration *structs.RegisterCountry, registrationID string, ctx context.Context) (string, error) {
	exists := service.registrationExists(registrationID, ctx)
	if exists != nil {
		return "", exists
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
func (service *RegistrationService) Patch(registrationID string, ctx context.Context,
  dataUpdate map[string]interface{}) error {
  exists := service.registrationExists(registrationID, ctx)
  if exists != nil {
      return exists
  }

  // Validate only fields that are provided in the patch payload.
  // if err := validatePatchData(dataUpdate); err != nil {
  //     return err
  // }

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
func toUpdateFields(dataUpdate map[string]interface{}) ([]firestore.Update, error) {
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
func (service *RegistrationService) HeadAllRegistrations(ctx context.Context) (int, error) {
	totalRegistrations, err := service.client.Collection(store.REGISTRATIONCOLLECTION).
		Documents(ctx).GetAll()
	if err != nil {
		return 0, err
	}
	return len(totalRegistrations), nil
}

// HeadOneRegistration retrieves a single registration without modifying it.
func (service *RegistrationService) HeadOneRegistration(registrationID string,
	ctx context.Context) (map[string]interface{}, error) {
	registration, registrationErr := service.GetByID(registrationID, ctx)
	if registrationErr != nil {
		return nil, registrationErr
	}
	return registration, nil
}

// Check if the isoCode is already in database.
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
