package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	"os"
)

func main() {
	var s3Enabled bool
	s := os.Getenv("S3_INTEGRATION_ENABLED")
	if s != "" {
		s3Enabled = true
	}

	objectUri := os.Getenv("OBJECT_URI")
	if s3Enabled && objectUri == "" {
		log.Fatal("OBJECT_URI must be set when S3_INTEGRATION_ENABLED is true")
	}

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	if s3Enabled {
		// Start listening for changes in S3 to OBJECT_URI

		// Setup listener for S3 so we are notified of changes

		// Fetch new module binary

		// Signal stop listening for new Kafka records

		// Load new module binary

		// Resume processing kafka records
	}

	// Routes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	if !s3Enabled {
		// attach endpoint to the router that manually refreshes the wasm module
		r.Post("/update", func(writer http.ResponseWriter, request *http.Request) {
			// update the loaded wasm module

			// Fetch new module binary

			// Signal stop listening for new Kafka records

			// Load new module binary

			// Resume processing kafka records
		})
	}

	// Start Server
	log.Println("Starting server on :8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal("Server error", err)
	}
}
