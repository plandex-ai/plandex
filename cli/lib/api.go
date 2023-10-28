package lib

import (
	"os"
	"plandex/types"
)

var apiHost string

// Check that API implements the types.APIHandler interface
var Api types.APIHandler = (*API)(nil)

type API struct{}

func init() {
	var port = os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}
	apiHost = "http://localhost:" + port
}
