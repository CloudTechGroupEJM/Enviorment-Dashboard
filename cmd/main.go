package main

import (
	"envdash/internal/app"
	"envdash/internal/config"
)

/*
Starting Point
*/
func main() {
	app.StartServer(config.PORT)
}
