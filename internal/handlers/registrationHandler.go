package handlers

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/store"
	"envdash/internal/structs"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
	"context"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type FirestoreHandler struct {
	Client *firestore.Client
}

// atomic counter (safe for concurrent use)
var requestCounter atomic.Int64

func InitRegistration(router *http.ServeMux) {

	// initalizing firebase
	client, clientErrInit := store.GetFirebaseClient()

	if clientErrInit != nil {
		log.Println("Error occurred when initializing Firebase client.")
		return
	}

	
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
		
	// client.Close()
}

func (firestoreHandler *FirestoreHandler) handleRegistrations(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodPost:
		firestoreHandler.addCountry(writer, request)
	case http.MethodGet:
		firestoreHandler.getRegistrations(writer, request)
	case http.MethodDelete:
		log.Println("testing Delete")
	default:
		http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// functionalities
func (FShandler *FirestoreHandler) addCountry(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	log.Printf("%s request recived", request.Method)

	var country structs.RegisterCountry

	if decoderErr := json.NewDecoder(request.Body).Decode(&country); decoderErr != nil {
		http.Error(writer, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

  country.IsoCode = strings.ToUpper(strings.TrimSpace(country.IsoCode))

	valdiationResult := countryValidation(&country, writer)

	if valdiationResult == true {

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

func countryValidation(country *structs.RegisterCountry, writer http.ResponseWriter) bool {

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

//retrive a specific document by id or all documents if no id i specified
func (FShandler *FirestoreHandler) getRegistrations(writer http.ResponseWriter, request *http.Request) {
	log.Printf("%s request recived", request.Method)

	registrationID := request.PathValue("id")

	context := request.Context()
	
	writer.Header().Set("Content-Type", "application/json")

	if registrationID == "" {
		retriveAllRegistrations(FShandler, context, writer)
		return
	} 

	retriveSingleRegistration(FShandler, context, registrationID, writer)
}





func retriveAllRegistrations(FShandler *FirestoreHandler, context context.Context, writer http.ResponseWriter){
	docIterator := FShandler.Client.Collection(store.REGISTRATIONCOLLECTION).Documents(context)
		defer docIterator.Stop()

		var results []map[string]interface{}

		for {
			registration, registrationErr := docIterator.Next()
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




func retriveSingleRegistration(FShandler *FirestoreHandler,context context.Context, registrationID string ,writer http.ResponseWriter){
	registration, registrationErr := FShandler.Client.Collection(store.REGISTRATIONCOLLECTION).Doc(registrationID).Get(context)
	if registrationErr != nil {
		log.Printf("Error retrieving document %s: %v", registrationID, registrationErr)
		http.Error(writer, "Document not found", http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(writer).Encode(registration.Data())
}