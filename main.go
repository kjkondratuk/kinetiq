package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {
	var s3Enabled bool
	s := os.Getenv("S3_INTEGRATION_ENABLED")
	if s != "" {
		s3Enabled = true
	}

	sb := os.Getenv("S3_INTEGRATION_BUCKET")
	if s3Enabled && sb == "" {
		log.Fatal("S3_INTEGRATION_BUCKET must be set when S3_INTEGRATION_ENABLED is set")
	}

	cq := os.Getenv("S3_INTEGRATION_CHANGE_QUEUE")
	if s3Enabled && cq == "" {
		log.Fatal("S3_INTEGRATION_CHANGE_QUEUE must be set when S3_INTEGRATION_ENABLED is set")
	}

	pollingInterval := 10
	pollingIntervalStr := os.Getenv("S3_INTEGRATION_POLL_INTERVAL")
	if s3Enabled && pollingIntervalStr != "" {
		pollingInterval, _ = strconv.Atoi(pollingIntervalStr)
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

	if s3Enabled {
		downloadPlugin(ctx, sb, pluginRef)
	}

	plugin, err := v1.NewModuleServicePlugin(ctx, v1.WazeroModuleConfig(
		wazero.NewModuleConfig().
			WithStartFunctions("_initialize", "_start"). // unclear why adding this made things work... It should be doing this anyway...
			WithStdout(os.Stdout).
			WithStderr(os.Stderr),
	))
	if err != nil {
		log.Fatal("Failed to setup plugin environment", err)
	}
	log.Printf("Plugin environment setup: %s\n", plugin)

	load, err := plugin.Load(ctx, pluginRef, functions.PluginFunctions{})
	if err != nil {
		log.Fatal("Failed to load plugin", err)
	}
	defer load.Close(ctx)

	log.Printf("Plugin environment setup: %s\n", plugin)

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

	if !s3Enabled {
		// refresh wasm module from local file listener
	}

	go writer.Write(ctx)
	log.Print("Writer started...")

	go proc.Start(ctx)

	log.Print("Processor started...")

	go reader.Read(ctx)

	log.Print("Reader started...")

	if s3Enabled {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			log.Fatalf("failed to load AWS config for s3 module hotswap listener: %e", err)
		}

		sqsClient := sqs.NewFromConfig(cfg)
		go s3_detector.NewS3SqsListener(
			sqsClient,
			pollingInterval,
			cq,
			pluginRef).
			Listen(func(notif s3_detector.S3EventNotification) {
				for _, record := range notif.Records {
					if record.S3.Object.Key == pluginRef &&
						record.S3.Bucket.Name == sb &&
						record.EventName == "ObjectCreated:Put" {
						// disable the reader to stop the inflow of data during deployment
						//reader.Disable()

						// install new processor
						log.Printf("Loading new module: %s", record.S3.Object.ETag)

						//plugin, err = v1.NewModuleServicePlugin(ctx, v1.WazeroModuleConfig(
						//	wazero.NewModuleConfig().
						//		WithStartFunctions("_initialize", "_start"). // unclear why adding this made things work... It should be doing this anyway...
						//		WithStdout(os.Stdout).
						//		WithStderr(os.Stderr),
						//))
						//if err != nil {
						//	log.Fatal("Failed to refresh plugin environment", err)
						//}
						//
						downloadPlugin(ctx, sb, pluginRef)
						load, err = plugin.Load(ctx, pluginRef, functions.PluginFunctions{})
						if err != nil {
							log.Fatal("Failed to reload plugin", err)
						}
						proc.Update(load)
						//proc.Close()
						//proc = processor.NewWasmProcessor(load, reader.Output())

						// enable reader again
						//reader.Enable()
					}
				}
			})
		// Start listening for changes in S3 to PLUGIN_REF

		// Setup listener for S3 so we are notified of changes

		// Fetch new module binary

		// Signal stop listening for new Kafka records

		// Load new module binary

		// Resume processing kafka records
	}

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Routes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start Server
	log.Println("Starting server on :8080")
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal("Server error", err)
	}
}

func downloadPlugin(ctx context.Context, bucket string, pluginRef string) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	// Define a file to download to
	outFile, err := os.Create(pluginRef)
	if err != nil {
		log.Fatalf("failed to create file for S3 download: %v", err)
	}
	defer outFile.Close()

	// Download the file
	getObjectOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &pluginRef,
	})
	if err != nil {
		log.Fatalf("failed to download file from S3: %v", err)
	}
	defer getObjectOutput.Body.Close()

	// Write the data to the locally created file
	_, err = io.Copy(outFile, getObjectOutput.Body)
	if err != nil {
		log.Fatalf("failed to write downloaded file to local disk: %v", err)
	}

	log.Printf("Successfully downloaded %s from bucket %s to %s", pluginRef, bucket, pluginRef)
}
