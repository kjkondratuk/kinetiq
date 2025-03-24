package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/fsnotify/fsnotify"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	app_config "github.com/kjkondratuk/kinetiq/config"
	"github.com/kjkondratuk/kinetiq/detection"
	"github.com/kjkondratuk/kinetiq/loader"
	"github.com/kjkondratuk/kinetiq/processor"
	sink_kafka "github.com/kjkondratuk/kinetiq/sink/kafka"
	source_kafka "github.com/kjkondratuk/kinetiq/source/kafka"
	"github.com/twmb/franz-go/pkg/kgo"
	"log"
	"net/http"
	"strings"
)

func main() {
	ctx := context.Background()

	appCfg, err := app_config.DefaultConfigurator.Configure(ctx)
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}

	var dl loader.Loader
	// download initial copy of the module
	if appCfg.S3.Enabled {
		s3Client := s3.NewFromConfig(*appCfg.Aws) // this value will be set if the configuration requires AWS

		// TODO : eat up SQS queue so we don't load changes more than once on startup (if there's a change backlog)
		dl = loader.NewS3Loader(s3Client, appCfg.S3.Bucket, appCfg.PluginRef)
	} else {
		dl = loader.NewBasicReloader(appCfg.PluginRef)
	}
	defer dl.Close(ctx)

	log.Print("Plugin environment loaded...\n")

	consumerOpts := app_config.DefaultConfigurator.ConsumerConfig(appCfg)

	log.Printf("Connecting to kafka source brokers: %s", appCfg.Kafka.SourceBrokers)
	readerClient, err := kgo.NewClient(consumerOpts...)
	if err != nil {
		log.Fatal("Failed to create kafka reader client", err)
	}
	defer readerClient.Close()

	log.Print("Reader client configured...")

	reader := source_kafka.NewKafkaReader(readerClient)
	defer reader.Close()

	log.Print("Reader configured...")

	proc := processor.NewWasmProcessor(dl, reader.Output())
	defer proc.Close()

	log.Print("Processor configured...")

	log.Printf("Connecting to kafka dest brokers: %s", appCfg.Kafka.DestBrokers)

	producerOpts := app_config.DefaultConfigurator.ProducerConfig(appCfg)

	writerClient, err := kgo.NewClient(
		producerOpts...,
	)
	if err != nil {
		log.Fatal("Failed to create kafka writer client", err)
	}
	defer writerClient.Close()

	log.Print("Writer client configured...")

	writer := sink_kafka.NewKafkaWriter(writerClient, proc.Output())
	defer writer.Close()

	log.Print("Writer configured...")

	go writer.Write(ctx)
	log.Print("Writer started...")

	go proc.Start(ctx)

	log.Print("Processor started...")

	go reader.Read(ctx)

	log.Print("Reader started...")

	if appCfg.S3.Enabled {
		// listen for changes from S3
		sqsClient := sqs.NewFromConfig(*appCfg.Aws) // this value will be configured if AWS SDK is required

		opts := []detection.SqsWatcherOpt{}

		if appCfg.S3.PollInterval != 0 {
			opts = append(opts, detection.WithInterval(appCfg.S3.PollInterval))
		}

		watcher := detection.NewS3SqsWatcher(
			sqsClient,
			appCfg.S3.ChangeQueue, opts...)

		// start watching for S3 Notification Events
		go watcher.StartEvents(ctx)

		// Listen to events and process them
		go watcher.Listen(ctx, detection.S3NotificationPluginReloadResponder(ctx, appCfg.PluginRef, appCfg.S3.Bucket, dl))
	} else {
		// listen for changes from local file listener
		w, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatalf("failed to create path watcher: %s", err)
		}
		defer w.Close()

		parts := strings.Split(appCfg.PluginRef, "/")
		parentDir := strings.Join(parts[0:len(parts)-1], "/")

		err = w.Add(parentDir)
		if err != nil {
			log.Fatalf("failed to add path to watcher: %s", err)
		}

		watcher := detection.NewListener[fsnotify.Event](detection.NewWatcher(w.Events, w.Errors))

		go watcher.Listen(ctx, detection.FilesystemNotificationPluginReloadResponder(ctx, dl))
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
