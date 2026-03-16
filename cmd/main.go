package main

import (
	"envdash/internal/config"
	"envdash/internal/app"

)

/*
Starting Point
*/
func main() {
	app.StartServer(config.PORT)
}