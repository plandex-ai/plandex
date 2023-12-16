package api

import (
	"os"
)

var apiHost string

var Client ApiClient = (*Api)(nil)

func init() {
	var port = os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}
	apiHost = "http://localhost:" + port
}
