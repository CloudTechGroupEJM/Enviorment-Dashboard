package handlers

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/services"
	"envdash/internal/structs"
	"errors"
	"log"
	"net/http"
	"sync/atomic"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// Handler holds shared dependencies
type FirestoreHandler struct {
	Client *firestore.Client
}

// atomic counter (safe for concurrent use)
var requestCounter atomic.Int64



func InitRegistration(router *http.ServeMux) {

	// initalizing firebase
	client, clientErrInit := services.GetFirebaseClient()

	if clientErrInit != nil {
		log.Println("Error occurred when initializing Firebase client.")
		return
	}

	// Ensure the client is properly closed when the application shuts down.
	defer client.Close() 

	firestore := &FirestoreHandler{
		Client: client,
	}

	router.HandleFunc(config.REGISTRATIONS_PAGE_PATH, firestore.HandleRegistration)




}



func (FShandler *FirestoreHandler) addCountry(writer http.ResponseWriter, request *http.Request){
	defer request.Body.Close()

	log.Printf("Recieved %s request", request.Method)

	var country *structs.RegisterCountry

	if err := json.NewDecoder(request.Body).Decode(&country); err != nil {
		http.Error(writer, "Invalid JSON payload", http.StatusBadRequest)
	}
}


func (FShandler *FirestoreHandler) HandleRegistration(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodPost:
		FShandler.addCountry(writer, request)
	case http.MethodGet:
		// h.getDocument(w, r)
	case http.MethodDelete:
		// h.deleteDocument(w, r)
	default:
		http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
	}
}