package main

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"github.com/kjkondratuk/kinetiq/plugin/functions"
	"github.com/tetratelabs/wazero"
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

	pluginRef := os.Getenv("PLUGIN_REF")
	if pluginRef == "" {
		log.Fatal("PLUGIN_REF must be set")
	}

	ctx := context.Background()

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

	plugin, err := v1.NewModuleServicePlugin(ctx, v1.WazeroModuleConfig(
		wazero.NewModuleConfig().
			WithStdout(os.Stdout).
			WithStderr(os.Stderr),
	))
	if err != nil {
		log.Fatal("Failed to setup plugin environment", err)
	}

	load, err := plugin.Load(ctx, pluginRef, functions.PluginFunctions{Http: http.DefaultClient})
	if err != nil {
		log.Fatal("Failed to load plugin", err)
	}
	defer load.Close(ctx)

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
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal("Server error", err)
	}
}
