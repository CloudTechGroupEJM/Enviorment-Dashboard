package services

import (
	"context"
	"envdash/internal/store"
	"envdash/internal/structs"
	"errors"
	"strings"
	"time"

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
	registrationDoc, _, creationError := service.client.Collection(store.REGISTRATIONCOLLECTION).Add(ctx, registration)

	if creationError != nil {
		return "", creationError
	}
  registration.ID = registrationDoc.ID
	return registrationDoc.ID, nil
}

// validation checks that required fields (Name and IsoCode) are present and valid.
func validation(registration *structs.RegisterCountry) error {
	if registration.Name == "" {
		return errors.New("missing required field: name")
	}

	if registration.IsoCode == "" {
		return errors.New("missing required field: isoCode")
	}

	registration.IsoCode = strings.ToUpper(strings.TrimSpace(registration.IsoCode))
	if len(registration.IsoCode) != 2 {
		return errors.New("isoCode must be two letters")
	}

	return nil
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

  registration := service.client.Collection(store.REGISTRATIONCOLLECTION).
                  Doc(registrationID)

  dataUpdate["lastChange"] = time.Now()

  _ , updateErr := registration.Update(ctx, toUpdateFields(dataUpdate))
  	if updateErr != nil {
		return updateErr
	}

	return nil
}

// toUpdateFields converts a map to Firestore Update objects for partial updates.
func toUpdateFields(dataUpdate map[string]interface{}) []firestore.Update {
	updates := make([]firestore.Update, 0)
	for key, value := range dataUpdate {
		updates = append(updates, firestore.Update{
			Path:  key,
			Value: value,
		})
	}
	return updates
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
