package main

import (
	"fmt"
	"log"
	"os"
	"plandex-server/setup"

	"github.com/gorilla/mux"
)

func main() {
	// Configure the default logger to include milliseconds in timestamps
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	var domain string
	if os.Getenv("DOMAIN") != "" {
		domain = os.Getenv("DOMAIN")
	} else if os.Getenv("GOENV") == "development" {
		domain = "localhost"
	} else {
		panic(fmt.Errorf("DOMAIN environment variable is required unless GOENV is set to development"))
	}

	r := mux.NewRouter()

	var apiRouter *mux.Router
	if os.Getenv("GOENV") == "development" {
		apiRouter = r
	} else {
		apiRouter = r.Host("api." + domain).Subrouter()
	}

	setup.MustLoadIp()
	setup.MustInitDb()
	setup.StartServer(apiRouter)
}
