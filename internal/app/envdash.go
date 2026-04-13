package app

import (
	"envdash/internal/handlers"
	"envdash/internal/store"
	"envdash/internal/utils"
	"log"
	"net/http"
	"os"
)

// StartServer
// Initializing the server
// Loads the .env file, checks if the port is available, initializes the firebase client and sets up the handlers
func StartServer(port string) {
	// Load the .env file
	if _, err := os.Stat(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")); os.IsNotExist(err) {
		log.Fatal("Environment variables: GOOGLE_APPLICATION_CREDENTIALS doesn't exist!")
	}
	if os.Getenv("OPEN_AQ_API_KEY") == "" {
		log.Fatal("Environment variables: OPEN_AQ_API_KEY doesn't exist or is empty!")
	}
	if utils.IsPortAvailable(port) == true {
		router := http.NewServeMux()

		// initializing firebase
		client, clientErrInit := store.GetFirebaseClient()
		if clientErrInit != nil {
			log.Println("Error occurred when initializing Firebase client.")
			return
		}
		defer client.Close()

		handlers.SetupAllHandlers(router, client)

		log.Println("Starting HTTP server on port " + port)
		log.Fatal(http.ListenAndServe(":"+port, router))
	} else {
		log.Println("Port Oocupied!!!!!!!!!!")
	}
}
