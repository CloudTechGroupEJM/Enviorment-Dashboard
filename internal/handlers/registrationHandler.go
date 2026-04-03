package handlers

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/services"
	"envdash/internal/structs"
	"log"
	"net/http"
	"strconv"

	"cloud.google.com/go/firestore"
)

const SUCCESSFUL_EXECUTION = "Request successfully executed"

type RegistrationHandler struct {
	service *services.RegistrationService
}

// initlizing service, handler and
func InitRegistration(router *http.ServeMux, client *firestore.Client) {
	service := services.NewRegistrationService(client)
	handler := &RegistrationHandler{
		service: service,
	}

	router.HandleFunc(config.REGISTRATIONS_PAGE_PATH, handler.handleRegistrations)
	router.HandleFunc(config.REGISTRATIONS_PAGE_PATH+"{id}", handler.handleRegistrations)
}

// request method to assignt diffrent purposes of requests.
func (handler *RegistrationHandler) handleRegistrations(writer http.ResponseWriter, request *http.Request) {
	log.Println("---------------------------------")
	log.Printf("%s request recived", request.Method)
	switch request.Method {
	case http.MethodPost:
		handler.createRegistration(writer, request)
	case http.MethodGet:
		handler.getRegistrations(writer, request)
	case http.MethodDelete:
		handler.deleteRegistrations(writer, request)
	case http.MethodPut:
		handler.putRegistration(writer, request)
	case http.MethodPatch:
		handler.patchRegistration(writer, request)
	case http.MethodHead:
		handler.headRegistrations(writer, request)
	default:
		http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// Handles HTTP POST requests to create a new country registration.
func (handler *RegistrationHandler) createRegistration(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	var registration structs.RegisterCountry

	if decoderErr := json.NewDecoder(request.Body).Decode(&registration); decoderErr != nil {
		http.Error(writer, "Invalid JSON payload", http.StatusBadRequest)
		log.Println("Error recived when creating registration")
		return
	}

	registrationID, creationErr := handler.service.Post(request.Context(), registration)
	if creationErr != nil {
		http.Error(writer, creationErr.Error(), http.StatusBadRequest)
		log.Println("Error recived when creating registration")
		return
	}

	writer.Header().Set(config.HEADER_CONTENT_TYPE, config.APPLICATION_JSON)
	writer.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(writer).Encode(map[string]string{
		"id": registrationID,
	})
	log.Println(SUCCESSFUL_EXECUTION)
}

// Handles HTTP Get requests to retrive single or all registration/s.
func (handler *RegistrationHandler) getRegistrations(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	writer.Header().Set(config.HEADER_CONTENT_TYPE, config.APPLICATION_JSON)

	if request.PathValue("id") == "" {
		allRegistrations, retrivingErr := handler.service.GetAll(request.Context())

		if retrivingErr != nil {
			log.Printf("Error retriving registrations: %v", retrivingErr)
			http.Error(writer, "Error retrieving data", http.StatusInternalServerError)
			return	
		}

		_ = json.NewEncoder(writer).Encode(allRegistrations)
		return
	} else {
		singleRegistration, registrationErr := handler.service.GetByID(request.PathValue("id"), request.Context())

		if registrationErr != nil{
			log.Printf("Error retrieving registration %s: %v", request.PathValue("id"), registrationErr)
			http.Error(writer, "registration not found", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(writer).Encode(singleRegistration)
		
	}
	log.Println(SUCCESSFUL_EXECUTION)
}





// Handles HTTP Delete requests to remove a specific regisration or all that are stores.
func (handler *RegistrationHandler) deleteRegistrations(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	if request.PathValue("id") == "" {
		deletionErr := handler.service.DeleteAll(request.Context())
		if deletionErr != nil{
			log.Printf("Error nothing to delete %s", deletionErr)
			http.Error(writer, "Nothing to delete.", http.StatusNotFound)
			return
		} else{
			http.Error(writer, "All registration deleted", http.StatusOK)
		}
	} else{
		if handler.service.DeleteByID(request.PathValue("id"), request.Context()) == true{
			http.Error(writer,"Regisration has been deleted" , http.StatusOK)
			log.Println("Regisration with id "+ request.PathValue("id") + " has been deleted")
		} else{
			http.Error(writer,"Regisration doesnt exist" , http.StatusOK)
		}
	}
	log.Println(SUCCESSFUL_EXECUTION)
}


// Handles HTTP Put requests to replace existen registration.
func (handler *RegistrationHandler) putRegistration(writer http.ResponseWriter, request *http.Request){
	defer request.Body.Close()
	writer.Header().Set("Content-Type", "application/json")

	var newRegistration structs.RegisterCountry
	if request.PathValue("id") != ""{
		decoderError := json.NewDecoder(request.Body).Decode(&newRegistration)
		if decoderError != nil {
			http.Error(writer, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		registrationID, replaceRegistrationErr := handler.service.Put(&newRegistration, request.PathValue("id"), request.Context())

		if replaceRegistrationErr != nil{
			log.Printf("registration doesnt exist: %s ", replaceRegistrationErr)
			http.Error(writer, "Couldnt replace registration.", http.StatusBadRequest)

		}else{
			log.Printf("registration %s fully replaced", replaceRegistrationErr)
			writer.WriteHeader(http.StatusOK)

			_ = json.NewEncoder(writer).Encode(map[string]string{
				"id":     registrationID,
				"status": "updated",
			})
		}
	}else{
		http.Error(writer, "Specify registration ID", http.StatusBadRequest)
		return
	}
	log.Println(SUCCESSFUL_EXECUTION)
}



// Handles HTTP Patch requests to update information in the registration.
func (handler *RegistrationHandler) patchRegistration(writer http.ResponseWriter, request *http.Request){
	defer request.Body.Close()
	writer.Header().Set("Content-Type", "application/json")
	if request.PathValue("id") == "" {
		log.Println("Need to specify a registration ID")
		http.Error(writer, "Missing registration ID", http.StatusBadRequest)
		return
	}

	var dataUpdate map[string]interface{}
	if decoderErr := json.NewDecoder(request.Body).Decode(&dataUpdate); decoderErr != nil{
		http.Error(writer, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	patchErr := handler.service.Patch(request.PathValue("id"), request.Context(), &dataUpdate)
	if patchErr != nil{
		log.Println(patchErr)
		http.Error(writer, "registration not found", http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(writer).Encode(map[string]string{
		"id":     request.PathValue("id"),
		"status": "patched",
	})

	log.Println(SUCCESSFUL_EXECUTION)
}


// Handles HTTP Head requests to retrive header information.
func (handler *RegistrationHandler) headRegistrations(writer http.ResponseWriter, request *http.Request){
	defer request.Body.Close()
	if request.PathValue("id") == ""{
		totalRegistations, totalErr := handler.service.HeadAllRegistrations(request.Context())
		if totalErr != nil{
				log.Printf("Error counting registrations: %v ", totalErr)
				http.Error(writer, "Error retriving data", http.StatusInternalServerError)
				return
		}
		http.Error(writer, "Retrived total registration", http.StatusOK)
		writer.Header().Set("Registration-Count", strconv.Itoa(totalRegistations))
	}else{
		_, registrationErr := handler.service.HeadOneRegistration(request.PathValue("id"),request.Context())
		if registrationErr != nil{
			log.Printf("Error retrieving registration %s: %v", request.PathValue("id"), registrationErr)
			http.Error(writer, "Registration not found", http.StatusNotFound)
			return
		}

		writer.Header().Set("Registration-Exists", "true")
		writer.WriteHeader(http.StatusOK)
	}
	log.Println(SUCCESSFUL_EXECUTION)
}
