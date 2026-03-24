package handlers

import (
	"encoding/json"
	"envdash/internal/structs"
	"errors"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// Handler holds shared dependencies
type FSClient struct {
	Client *firestore.Client
}




func registrationsHandler(router *http.ServeMux) {

}

func (client *FSClient) HandleMessage(responseWriter http.ResponseWriter, request *http.Request) {
	switch request.Method {
	// case http.MethodPost:
	// h.addDocument(responseWriter, request)
	case http.MethodGet:
		client.getDocument(responseWriter, request)
	// case http.MethodDelete:
	// h.deleteDocument(responseWriter, request)
	default:
		http.Error(responseWriter, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ----------- GET /messages or /messages/{id} -----------
func (client *FSClient) getDocument(responseWriter http.ResponseWriter, request *http.Request) {
	log.Printf("Received %s request", request.Method)

	id := request.PathValue("id")
	ctx := request.Context()

	responseWriter.Header().Set("Content-Type", "application/json")

	// ----------- GET ALL -----------
	if id == "" {
		iter := client.Client.Collection(structs.CollectionName).Documents(ctx)
		defer iter.Stop()

		var results []map[string]interface{}

		for {
			doc, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				log.Printf("Error iterating documents: %v", err)
				http.Error(responseWriter, "Error retrieving data", http.StatusInternalServerError)
				return
			}

			results = append(results, doc.Data())
		}

		_ = json.NewEncoder(responseWriter).Encode(results)
		return
	}

	// ----------- GET ONE -----------
	doc, err := client.Client.Collection(structs.CollectionName).Doc(id).Get(ctx)
	if err != nil {
		log.Printf("Error retrieving document %s: %v", id, err)
		http.Error(responseWriter, "Document not found", http.StatusNotFound)
		return
	}

	_ = json.NewEncoder(responseWriter).Encode(doc.Data())
}
