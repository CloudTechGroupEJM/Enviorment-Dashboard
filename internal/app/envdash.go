package app

import (
	"envdash/internal/handlers"
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



    }else{
        log.Println("Port Oocupied!!!!!!!!!!")
    }
}

