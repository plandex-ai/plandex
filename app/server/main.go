package main

import (
	"log"
	"plandex-server/setup"

	"github.com/gorilla/mux"
)

func main() {
	// Configure the default logger to include milliseconds in timestamps
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	r := mux.NewRouter()

	setup.MustLoadIp()
	setup.MustInitDb()
	setup.StartServer(r)
}
