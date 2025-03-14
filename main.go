package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	s3_detector "github.com/kjkondratuk/kinetiq/detection/s3"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"github.com/kjkondratuk/kinetiq/plugin/functions"
	"github.com/kjkondratuk/kinetiq/processor"
	sink_kafka "github.com/kjkondratuk/kinetiq/sink/kafka"
	source_kafka "github.com/kjkondratuk/kinetiq/source/kafka"
	"github.com/tetratelabs/wazero"
	"github.com/twmb/franz-go/pkg/kgo"
	"log"
	"net/http"
	"os"
	"strings"
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

	sourceBrokers := os.Getenv("KAFKA_SOURCE_BROKERS")
	if sourceBrokers == "" {
		sourceBrokers = "localhost:49092"
	}

	sourceTopic := os.Getenv("KAFKA_SOURCE_TOPIC")
	if sourceTopic == "" {
		log.Fatal("KAFKA_SOURCE_TOPIC must be set")
	}

	destBrokers := os.Getenv("KAFKA_DEST_BROKERS")
	if destBrokers == "" {
		destBrokers = sourceBrokers
	}

	destTopic := os.Getenv("KAFKA_DEST_TOPIC")
	if sourceBrokers == destBrokers && destTopic == sourceTopic {
		log.Fatal("KAFKA_DEST_TOPIC must be different from KAFKA_SOURCE_TOPIC when source and destination kafka sBrokers are the same")
	}

	ctx := context.Background()

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	plugin, err := v1.NewModuleServicePlugin(ctx, v1.WazeroModuleConfig(
		wazero.NewModuleConfig().
			WithStartFunctions("_initialize", "_start"). // unclear why adding this made things work... It should be doing this anyway...
			WithStdout(os.Stdout).
			WithStderr(os.Stderr),
	))
	if err != nil {
		log.Fatal("Failed to setup plugin environment", err)
	}

	load, err := plugin.Load(ctx, pluginRef, functions.PluginFunctions{})
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

	sBrokers := strings.Split(sourceBrokers, ",")
	log.Printf("Connecting to kafka source brokers: %s", sBrokers)
	readerClient, err := kgo.NewClient(
		kgo.ConsumeTopics(sourceTopic),
		kgo.SeedBrokers(sBrokers...),
	)
	if err != nil {
		log.Fatal("Failed to create kafka reader client", err)
	}
	defer readerClient.Close()

	log.Print("Reader client configured...")

	reader := source_kafka.NewKafkaReader(readerClient)
	defer reader.Close()

	log.Print("Reader configured...")

	proc := processor.NewWasmProcessor(load, reader.Output())
	defer proc.Close()

	log.Print("Processor configured...")

	dBrokers := strings.Split(destBrokers, ",")
	log.Printf("Connecting to kafka dest brokers: %s", dBrokers)
	writerClient, err := kgo.NewClient(
		kgo.DefaultProduceTopic(destTopic),
		kgo.SeedBrokers(dBrokers...),
	)
	if err != nil {
		log.Fatal("Failed to create kafka writer client", err)
	}
	defer writerClient.Close()

	log.Print("Writer client configured...")

	writer := sink_kafka.NewKafkaWriter(writerClient, proc.Output())
	defer writer.Close()

	log.Print("Writer configured...")

	if s3Enabled {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			log.Fatalf("failed to load AWS config for s3 module hotswap listener: %e", err)
		}

		sqsClient := sqs.NewFromConfig(cfg)
		go s3_detector.NewS3SqsListener(
			sqsClient,
			10,
			"https://sqs.us-east-1.amazonaws.com/916325820950/kinetiq-updates-sqs",
			"test_module.wasm").
			Listen(func(notif s3_detector.S3EventNotification) error {

				return nil
			})
		// Start listening for changes in S3 to OBJECT_URI

		// Setup listener for S3 so we are notified of changes

		// Fetch new module binary

		// Signal stop listening for new Kafka records

		// Load new module binary

		// Resume processing kafka records
	}

	log.Printf("Response: %s - %d : %s - %s", "code", process.ResponseCode, "message", process.Message)
	go writer.Write(ctx)
	log.Print("Writer started...")

	go proc.Start(ctx)

	log.Print("Processor started...")

	go reader.Read(ctx)

	log.Print("Reader started...")

	// Start Server
	log.Println("Starting server on :8080")
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal("Server error", err)
	}
}
