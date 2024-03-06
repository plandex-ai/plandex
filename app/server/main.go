package main

import (
	"plandex-server/setup"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	setup.MustLoadIp()
	setup.MustInitDb()
	setup.StartServer(r)
}
