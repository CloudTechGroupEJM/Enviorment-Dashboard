package handlers

import (
	"context"
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/store"
	"envdash/internal/structs"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type FirestoreHandler struct {
	Client *firestore.Client
}

func InitRegistration(router *http.ServeMux, client *firestore.Client) {

	firestoreHandler := &FirestoreHandler{
		Client: client,
	}

	router.HandleFunc(config.REGISTRATIONS_PAGE_PATH,
		func(writer http.ResponseWriter, request *http.Request) {
			firestoreHandler.handleRegistrations(writer, request)
	})

	router.HandleFunc(config.REGISTRATIONS_PAGE_PATH+"{id}",
		func(writer http.ResponseWriter, request *http.Request) {
			firestoreHandler.handleRegistrations(writer, request)
	})
}

func (firestoreHandler *FirestoreHandler) handleRegistrations(writer http.ResponseWriter, request *http.Request) {
	log.Printf("%s request recived", request.Method)
	switch request.Method {
	case http.MethodPost:
		firestoreHandler.addRegistration(writer, request)
	case http.MethodGet:
		firestoreHandler.getRegistrations(writer, request)
	case http.MethodDelete:
		firestoreHandler.deleteRegistrations(writer, request)
	case http.MethodPut:
		firestoreHandler.putRegistration(writer, request)
	case http.MethodPatch:
		firestoreHandler.patchRegistration(writer, request)
	default:
		http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// validate if name and isocode can go through as required
func registrationValidation(country *structs.RegisterCountry, writer http.ResponseWriter) bool {
	if country.Name == "" {
		http.Error(writer, "Missing required field: name", http.StatusBadRequest)
		return false
	}

	if country.IsoCode == "" {
		http.Error(writer, "Missing required field: isoCode", http.StatusBadRequest)
		return false
	}

	if len(country.IsoCode) != 2 {
		http.Error(writer, "IsoCode must be two letter.", http.StatusBadRequest)
		return false
	}

	return true
}

// POST function
func (FShandler *FirestoreHandler) addRegistration(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	var country structs.RegisterCountry

	log.Println(country)

	if decoderErr := json.NewDecoder(request.Body).Decode(&country); decoderErr != nil {
		http.Error(writer, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	country.IsoCode = strings.ToUpper(strings.TrimSpace(country.IsoCode))

	if registrationValidation(&country, writer) == true {

		country.LastChange = time.Now()

		context := request.Context()

		generatedID, _, idError := FShandler.Client.Collection(store.REGISTRATIONCOLLECTION).Add(context, country)
		if idError != nil {
			log.Printf("Failed to add document: %v", idError)
			http.Error(writer, "Failed to store document", http.StatusInternalServerError)
			return
		}

		log.Println("Document Created carries ID: " + generatedID.ID)

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)

		_ = json.NewEncoder(writer).Encode(map[string]string{
			"id": generatedID.ID,
		})

	} else {
		log.Println("Some values doesnt match the matched struct.")
		return
	}
}

// GET function
// retrive a specific document by id or all documents if no id i specified
func (FShandler *FirestoreHandler) getRegistrations(writer http.ResponseWriter, request *http.Request) {

	registrationID := request.PathValue("id")

	context := request.Context()

	writer.Header().Set("Content-Type", "application/json")

	if registrationID == "" {
		retriveAllRegistrations(FShandler, context, writer)
		return
	}

	retriveSingleRegistration(FShandler, context, registrationID, writer)
}

func retriveAllRegistrations(FShandler *FirestoreHandler, context context.Context, writer http.ResponseWriter) {
	regstrationIterator := FShandler.Client.Collection(store.REGISTRATIONCOLLECTION).Documents(context)
	defer regstrationIterator.Stop()

	var results []map[string]interface{}

	for {
		registration, registrationErr := regstrationIterator.Next()
		log.Println(registration)
		if errors.Is(registrationErr, iterator.Done) {
			break
		}
		if registrationErr != nil {
			log.Printf("Error iterating documents: %v", registrationErr)
			http.Error(writer, "Error retrieving data", http.StatusInternalServerError)
			return
		}

		results = append(results, registration.Data())
	}

	_ = json.NewEncoder(writer).Encode(results)
}

func retriveSingleRegistration(FShandler *FirestoreHandler, context context.Context, registrationID string, writer http.ResponseWriter) {
	registration, registrationErr := FShandler.Client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID).Get(context)
	if registrationErr != nil {
		log.Printf("Error retrieving document %s: %v", registrationID, registrationErr)
		http.Error(writer, "Document not found", http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(writer).Encode(registration.Data())
}

// DELETE function
func (FShandler *FirestoreHandler) deleteRegistrations(writer http.ResponseWriter, request *http.Request) {
	registrationID := request.PathValue("id")
	context := request.Context()

	if registrationID == "" {
		registrationIterator := FShandler.Client.Collection(store.REGISTRATIONCOLLECTION).Documents(context)

		defer registrationIterator.Stop()

		for {
			registration, registrationErr := registrationIterator.Next()
			if !errors.Is(registrationErr, iterator.Done) {
				if registrationErr != nil {
					log.Printf("Error iterating documents: %v", registrationErr)
					http.Error(writer, "Error deleting document", http.StatusInternalServerError)
					return
				}

				if _, err := registration.Ref.Delete(context); err != nil {
					log.Printf("Error deleting document: %v", err)
					http.Error(writer, "Error deleting document", http.StatusInternalServerError)
					return
				}
			} else {
				writer.WriteHeader(http.StatusNoContent)
				return
			}
		}
	} else{
		log.Println("whats here")

		_, existantsError := FShandler.Client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID).Get(context)
		if existantsError != nil {
    log.Printf("Document not found: %v", existantsError)
    http.Error(writer, "Document not found", http.StatusNotFound)
    return
		}	
		
		_, registrationErr := FShandler.Client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID).Delete(context)
		if registrationErr != nil {
			log.Printf("Error deleting document %s: %v", registrationID, registrationErr)
			http.Error(writer, "Error deleting document", http.StatusInternalServerError)
			return
		}else{
			log.Println("Registration deleted")
			http.Error(writer, "Registration deleted", http.StatusOK)
			return
		}
	}
}

// PUT (Full replacment of the given id)
func (FShandler *FirestoreHandler) putRegistration(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	if request.PathValue("id") != "" {	registrationID := request.PathValue("id")
		context := request.Context()

		var newRegistration structs.RegisterCountry

		decoderError := json.NewDecoder(request.Body).Decode(&newRegistration)
		if decoderError != nil {
			http.Error(writer, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		if registrationValidation(&newRegistration, writer) == true {
			newRegistration.LastChange = time.Now()

			_, err := FShandler.Client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID).Set(context, newRegistration)
			if err != nil {
				log.Printf("Failed to update document %s: %v", registrationID, err)
				http.Error(writer, "Failed to update document", http.StatusInternalServerError)
				return
			}

			log.Printf("Document %s fully replaced", registrationID)
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusOK)

			_ = json.NewEncoder(writer).Encode(map[string]string{
				"id":     registrationID,
				"status": "updated",
			})

		}

	} else{
		http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}


}


func (FShandler *FirestoreHandler) patchRegistration(writer http.ResponseWriter, request *http.Request){
	defer request.Body.Close()

	log.Printf("Received %s request", request.Method)

	registrationID := request.PathValue("id")
	context := request.Context()

	if registrationID == "" {
		http.Error(writer, "Missing registration ID", http.StatusBadRequest)
	
	return
	}

	var dataUpdate map[string]interface{}
	if decoderErr := json.NewDecoder(request.Body).Decode(&dataUpdate); decoderErr != nil{
		http.Error(writer, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Update only the provided fields
	_, err := FShandler.Client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID).Update(context, toUpdateFields(dataUpdate))
	if err != nil {
		log.Printf("Failed to patch document %s: %v", registrationID, err)
		http.Error(writer, "Failed to update document", http.StatusInternalServerError)
		return
	}

	log.Printf("Document %s partially updated", registrationID)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(writer).Encode(map[string]string{
		"id":     registrationID,
		"status": "patched",
	})

}


// Helper function to convert map to firestore updates
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

