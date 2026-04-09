package app

import (
	"envdash/internal/handlers"
	"envdash/internal/store"
	"envdash/internal/utils"
	"log"
	"net/http"
)

/*
Initializing the server
*/
func StartServer(port string) {
    if utils.IsPortAvailable(port) == true{
        router := http.NewServeMux()


        // initalizing firebase
        client, clientErrInit := store.GetFirebaseClient()
        if clientErrInit != nil {
            log.Println("Error occurred when initializing Firebase client.")
            return
        }
        defer client.Close()
        
        handlers.SetupAllHandlers(router, client)


        log.Println("Starting HTTP server on port " + port)
        log.Fatal(http.ListenAndServe(":"+port, router))
    }else{
        log.Println("Port Oocupied!!!!!!!!!!")
    }
}

