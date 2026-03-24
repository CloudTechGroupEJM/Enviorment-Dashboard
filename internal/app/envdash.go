package app

import (
	"envdash/internal/handlers"
	"envdash/internal/services"
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
        handlers.SetupAllHandlers(router)
        log.Println("Starting HTTP server on port " + port)
        log.Fatal(http.ListenAndServe(":"+port, router))


        //
        initFirebase()
    }else{
        log.Println("Port Oocupied!!!!!!!!!!")
    }
}

func initFirebase(){
    client, clientErrInit := services.GetFirebaseClient()

    if clientErrInit != nil {
        log.Println("Error occurred when initializing Firebase client.")
    }

    defer client.Close()

    

}
