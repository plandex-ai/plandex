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
	"plandex-server/notify"
	"plandex-server/shutdown"
	"runtime/debug"
	"syscall"
	"time"
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

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for monitoring endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/version" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		log.Printf("\n\nRequest: %s %s\n\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("\n\nCompleted: %s %s in %v\n\n", r.Method, r.URL.Path, time.Since(start))
	})
}

func StartServer(handler http.Handler, configureFn func(handler http.Handler) http.Handler, afterStart func()) {
	if os.Getenv("GOENV") == "development" {
		log.Println("In development mode.")
	}

	shutdown.ShutdownCtx, shutdown.ShutdownCancel = context.WithCancel(context.Background())
	defer shutdown.ShutdownCancel()

	// Ensure database connection is closed
	defer func() {
		log.Println("Closing database connection...")
		err := db.Conn.Close()
		if err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
		log.Println("Database connection closed")
	}()

	// Get externalPort from the environment variable or default to 8099
	externalPort := os.Getenv("PORT")
	if externalPort == "" {
		externalPort = "8099"
	}

	// Add logging middleware before the maxBytes middleware
	handler = loggingMiddleware(handler)

	// Apply the maxBytesMiddleware to limit request size to 1 GB
	handler = maxBytesMiddleware(handler, 1000<<20) // 1 GB limit

	if configureFn != nil {
		handler = configureFn(handler)
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

	if afterStart != nil {
		afterStart()
	}

	// Capture SIGTERM and SIGINT signals
	sigTermChan := make(chan os.Signal, 1)

	signal.Notify(sigTermChan, syscall.SIGTERM, syscall.SIGINT)

	sig := <-sigTermChan
	log.Printf("Received signal %v, shutting down gracefully...\n", sig)

	// Create a channel to track completion of active plans
	plansDone := make(chan struct{})

	// Start goroutine to monitor active plans
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in waitForActivePlans: %v\n%s", r, debug.Stack())
				go notify.NotifyErr(notify.SeverityError, fmt.Errorf("panic in waitForActivePlans: %v\n%s", r, debug.Stack()))
			}
			close(plansDone)
		}()

		// First wait for active plans to complete or timeout
		log.Println("Waiting for active plans to complete...")
		activePlansCtx, cancel := context.WithTimeout(shutdown.ShutdownCtx, 60*time.Second)
		defer cancel()

		select {
		case <-activePlansCtx.Done():
			if activePlansCtx.Err() == context.DeadlineExceeded {
				log.Println("Timeout waiting for active plans. Forcing shutdown.")
			}
		case <-waitForActivePlans():
			log.Println("All active plans finished.")
		}

		// Then clean up any remaining locks
		log.Println("Cleaning up any remaining locks...")
		if err := db.CleanupActiveLocks(shutdown.ShutdownCtx); err != nil {
			log.Printf("Error cleaning up locks: %v", err)
		}
	}()

	// Wait for plans to finish or timeout
	select {
	case <-shutdown.ShutdownCtx.Done():
		log.Println("Global shutdown timeout reached")
	case <-plansDone:
		log.Println("All cleanup tasks completed")
	}

	// Shutdown the HTTP server
	log.Println("Shutting down http server...")
	httpCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(httpCtx); err != nil {
		log.Printf("Http server forced to shutdown: %v", err)
	}

	// Execute shutdown hooks
	log.Println("Executing shutdown hooks...")
	for _, hook := range shutdownHooks {
		hook()
	}

	log.Println("Shutdown complete")
}

func waitForActivePlans() chan struct{} {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if plan.NumActivePlans() == 0 {
					close(done)
					return
				}
			}
		}
	}()
	return done
}

func maxBytesMiddleware(next http.Handler, maxBytes int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// log the size of the request body
		// log.Printf("Request body size: %d", r.ContentLength)

		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next.ServeHTTP(w, r)
	})
}
