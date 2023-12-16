package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"plandex-server/db"

	openai "github.com/sashabaranov/go-openai"
)

var client *openai.Client

func main() {
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}

	err := db.Connect()
	if err != nil {
		log.Fatal("Error initializing database: ", err)
	}

	err = db.MigrationsUp()
	if err != nil {
		log.Fatal("Error running migrations: ", err)
	}

	routes := InitRoutes()

	// Get port from the environment variable or default to 8088
	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	log.Printf("Plandex server is running on :%s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), routes))
}
