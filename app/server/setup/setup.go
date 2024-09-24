package setup

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"plandex-server/db"
	"plandex-server/host"
	"plandex-server/model/plan"
	"plandex-server/routes"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func MustLoadIp() {
	err := host.LoadIp()
	if err != nil {
		log.Fatal("Error loading IP: ", err)
	}
}

func MustInitDb() {
	err := db.Connect()
	if err != nil {
		log.Fatal("Error initializing database: ", err)
	}

	err = db.MigrationsUp()
	if err != nil {
		log.Fatal("Error running migrations: ", err)
	}

	err = db.CacheOrgRoleIds()
	if err != nil {
		log.Fatal("Error caching org role ids: ", err)
	}
}

func StartServer(r *mux.Router) {
	if os.Getenv("GOENV") == "development" {
		log.Println("In development mode.")
	}

	// Ensure database connection is closed
	defer func() {
		log.Println("Closing database connection...")
		err := db.Conn.Close()
		if err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
		log.Println("Database connection closed")
	}()

	// Get externalPort from the environment variable or default to 8080
	externalPort := os.Getenv("PORT")
	if externalPort == "" {
		externalPort = "8080"
	}

	routes.AddApiRoutes(r)

	// Enable CORS based on environment
	var corsHandler http.Handler
	if os.Getenv("GOENV") == "development" {
		corsHandler = cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
			AllowedHeaders:   []string{"Content-Type", "Authorization"},
			AllowCredentials: true,
		}).Handler(r)
	} else {
		corsHandler = cors.New(cors.Options{
			AllowedOrigins:   []string{"http://app.plandex.ai", "http://localhost:55000"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
			AllowedHeaders:   []string{"Content-Type", "Authorization"},
			AllowCredentials: true,
		}).Handler(r)
	}

	server := &http.Server{
		Addr:    ":" + externalPort,
		Handler: corsHandler,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Println("Started server on port " + externalPort)

	// Capture SIGTERM and SIGINT signals
	sigTermChan := make(chan os.Signal, 1)
	signal.Notify(sigTermChan, syscall.SIGTERM, syscall.SIGINT)

	<-sigTermChan
	log.Println("Shutting down server gracefully...")

	// Context with a 5-second timeout to allow ongoing requests to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Wait for active plans to complete
	for {
		l := plan.NumActivePlans()
		if l == 0 {
			break
		}
		log.Printf("Waiting for %d active plans to finish...\n", l)
		time.Sleep(1 * time.Second)
	}

	log.Println("Shutdown complete")
}
