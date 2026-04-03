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
func NewRegistrationService(client *firestore.Client) *RegistrationService{
	if client == nil{
		panic("firestone client cannot be nil")
	}
	return &RegistrationService{
		client: client,
	}
}


// Post validates and creates a new registration in Firestore, returning the generated document ID.
func (service *RegistrationService) Post(context context.Context, registration structs.RegisterCountry) (string, error){
	if validationErr := validation(&registration); validationErr != nil{
		return "", validationErr
	}
	
	registration.LastChange = time.Now()

	registrationDoc, _ , creationError := service.client.Collection(store.REGISTRATIONCOLLECTION).Add(context, registration)

	if creationError != nil{
		return "",creationError
	}

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
func (service *RegistrationService) GetAll(context context.Context) ([]map[string]interface{}, error){	
	var formatedRegistrations []map[string]interface{}

	registrations, registrationsErr := service.client.Collection(store.REGISTRATIONCOLLECTION).Documents(context).GetAll()
	if registrationsErr != nil{
		return nil, registrationsErr
	}

	// Check if collection is empty
	if len(registrations) == 0 {
			return nil, errors.New("collection is empty")
	}

	for _, registration := range registrations{
		formatedRegistrations = append(formatedRegistrations, registration.Data())
	}
	return formatedRegistrations, nil
}

// GetByID retrieves a specific registration by its document ID.
func (service *RegistrationService) GetByID(registrationID string, context context.Context) (map[string]interface{}, error){
	registration, registrationErr := service.client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID).Get(context)
	if registrationErr != nil {
		return nil, registrationErr
	}
	return registration.Data(), nil
}

// DeleteAll removes all documents from the registrations collection using BulkWriter.
func (service *RegistrationService) DeleteAll(context context.Context) error{
	registrations, registrationsErr := service.client.Collection(store.REGISTRATIONCOLLECTION).Documents(context).GetAll()
	if registrationsErr != nil {
			return registrationsErr
	}

	// Check if collection is empty
	if len(registrations) == 0 {
			return errors.New("collection is empty")
	}

  bulkWriter := service.client.BulkWriter(context)
	for _, registration := range registrations {
		_, deleteErr := bulkWriter.Delete(registration.Ref)
		if deleteErr != nil {
				return deleteErr
		}
	}
	bulkWriter.End()
	return nil
}

// DeleteByID removes a specific registration by its document ID after verifying it exists.
func (service *RegistrationService) DeleteByID(registrationID string, context context.Context) bool {
	if service.registrationExists(registrationID, context) == nil{
		service.client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID).Delete(context)
    return true
	}
	return false
}


// registrationExists checks if a registration document exists by its ID.
func (service *RegistrationService) registrationExists(registrationID string, context context.Context) error{
	  _, regisrationErr := service.client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID).Get(context)
    if regisrationErr != nil {
        return regisrationErr
    }
		return nil
}


// Put replaces an entire registration document with new data if it exists.
func (service *RegistrationService) Put(newRegistration *structs.RegisterCountry, registrationID string, context context.Context ) (string,error){
	exists := service.registrationExists(registrationID,context)
	if exists != nil{
		return "", exists
	}else{
		newRegistration.LastChange = time.Now()
		_, registrationErr := service.client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID).Set(context, newRegistration)
		if registrationErr != nil {
			return "", registrationErr
		}
		return registrationID, registrationErr
	}
}

// Patch updates only specified fields of a registration document.
func (service *RegistrationService) Patch(registrationID string, context context.Context, dataUpdate *map[string]interface{})error{
	exists := service.registrationExists(registrationID,context)
	if exists != nil{
		return exists
	}
	// Update only the provided fields
	_, updateErr :=	service.client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID).Update(context, toUpdateFields(dataUpdate))
	if updateErr != nil{
		return updateErr
	}
	return nil
}

// toUpdateFields converts a map to Firestore Update objects for partial updates.
func toUpdateFields(dataUpdate *map[string]interface{}) []firestore.Update {
	updates := make([]firestore.Update, 0)
	for key, value := range *dataUpdate {
		updates = append(updates, firestore.Update{
			Path:  key,
			Value: value,
		})
	}
	return updates
}

// HeadAllRegistrations returns the total count of all registrations in the collection.
func (service *RegistrationService) HeadAllRegistrations(context context.Context) (int, error) {
    totalRegistrations, err := service.client.Collection(store.REGISTRATIONCOLLECTION).Documents(context).GetAll()
    if err != nil {
        return 0, err
    }
    return len(totalRegistrations), nil
}

// HeadOneRegistration retrieves a single registration without modifying it.
func (service *RegistrationService) HeadOneRegistration(registrationID string, context context.Context)(map[string]interface{}, error) {
    registration, registrationErr := service.GetByID(registrationID, context)
    if registrationErr != nil {
			return nil, registrationErr
    }
		return registration, nil
}