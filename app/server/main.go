package main

import (
	"log"
	"plandex-server/routes"
	"plandex-server/setup"

	"github.com/gorilla/mux"
)

func main() {
	// Configure the default logger to include milliseconds in timestamps
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	r := mux.NewRouter()
	routes.AddHealthRoutes(r)
	routes.AddApiRoutes(r)
	routes.AddProxyableApiRoutes(r)
	setup.MustLoadIp()
	setup.MustInitDb()
	setup.StartServer(r, nil)
}
