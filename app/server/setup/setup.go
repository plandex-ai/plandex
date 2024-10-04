package setup

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"plandex-server/db"
	"plandex-server/host"
	"plandex-server/model/plan"
	"syscall"
	"time"

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

var shutdownHooks []func()

func RegisterShutdownHook(hook func()) {
	shutdownHooks = append(shutdownHooks, hook)
}

func StartServer(handler http.Handler) {
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

	// Apply the maxBytesMiddleware to limit request size to 100 MB
	handler = maxBytesMiddleware(handler, 100<<20) // 100 MB limit

	// Enable CORS based on environment
	if os.Getenv("GOENV") == "development" {
		handler = cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization"},
			AllowCredentials: true,
		}).Handler(handler)
	} else {
		handler = cors.New(cors.Options{
			AllowedOrigins:   []string{fmt.Sprintf("https://%s.plandex.ai", os.Getenv("APP_SUBDOMAIN"))},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization"},
			AllowCredentials: true,
		}).Handler(handler)
	}

	server := &http.Server{
		Addr:              ":" + externalPort,
		Handler:           handler,
		MaxHeaderBytes:    1 << 20, // 1 MB
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Println("Started Plandex server on port " + externalPort)

	// Capture SIGTERM and SIGINT signals
	sigTermChan := make(chan os.Signal, 1)
	signal.Notify(sigTermChan, syscall.SIGTERM, syscall.SIGINT)

	<-sigTermChan
	log.Println("Plandex server shutting down gracefully...")

	// Context with a 5-second timeout to allow ongoing requests to finish
	// wait for active plans to finish for up to 2 hours
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()

	// Wait for active plans to complete or timeout
	log.Println("Waiting for any active plans to complete...")
	select {
	case <-ctx.Done():
		log.Println("Timeout waiting for active plans. Forcing shutdown.")
	case <-waitForActivePlans():
		log.Println("All active plans finished.")
	}

	log.Println("Shutting down http server...")
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Http server forced to shutdown: %v", err)
	}

	// Execute shutdown hooks
	for _, hook := range shutdownHooks {
		hook()
	}

	log.Println("Shutdown complete")
	os.Exit(0)
}

func waitForActivePlans() chan struct{} {
	done := make(chan struct{})
	go func() {
		for {
			if plan.NumActivePlans() == 0 {
				close(done)
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()
	return done
}

func maxBytesMiddleware(next http.Handler, maxBytes int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next.ServeHTTP(w, r)
	})
}
