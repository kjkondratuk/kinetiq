package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/fsnotify/fsnotify"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	app_config "github.com/kjkondratuk/kinetiq/config"
	"github.com/kjkondratuk/kinetiq/detection"
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
	"strings"
)

func main() {
	appCfg, err := app_config.DefaultConfigurator.Configure()
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}

	ctx := context.Background()

	// download initial copy of the module
	if appCfg.S3.Enabled {
		// TODO : eat up SQS queue so we don't load changes more than once on startup (if there's a change backlog)
		downloadPlugin(ctx, appCfg.S3.Bucket, appCfg.PluginRef)
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
	log.Printf("Plugin environment setup...\n")

	load, err := plugin.Load(ctx, appCfg.PluginRef, functions.NewDefaultPluginFunctions())
	if err != nil {
		log.Fatal("Failed to load plugin", err)
	}
	defer load.Close(ctx)

	log.Print("Plugin environment loaded...\n")

	log.Printf("Connecting to kafka source brokers: %s", appCfg.Kafka.SourceBrokers)
	readerClient, err := kgo.NewClient(
		kgo.ConsumeTopics(appCfg.Kafka.SourceTopic),
		kgo.SeedBrokers(appCfg.Kafka.SourceBrokers...),
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

	log.Printf("Connecting to kafka dest brokers: %s", appCfg.Kafka.DestBrokers)
	writerClient, err := kgo.NewClient(
		kgo.DefaultProduceTopic(appCfg.Kafka.DestTopic),
		kgo.SeedBrokers(appCfg.Kafka.DestBrokers...),
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
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			log.Fatalf("failed to load AWS config for s3 module hotswap listener: %e", err)
		}
		sqsClient := sqs.NewFromConfig(cfg)

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
		go watcher.Listen(func(notif *detection.S3EventNotification, err error) {
			if err != nil {
				log.Printf("Failed to handle s3 watcher changes: %s", err)
				return
			}
			for _, record := range notif.Records {
				if record.S3.Object.Key == appCfg.PluginRef &&
					record.S3.Bucket.Name == appCfg.S3.Bucket &&
					record.EventName == "ObjectCreated:Put" {

					// install new processor
					log.Printf("Loading new module: %s", record.S3.Object.ETag)
					downloadPlugin(ctx, appCfg.S3.Bucket, appCfg.PluginRef)
					load, err = plugin.Load(ctx, appCfg.PluginRef, functions.NewDefaultPluginFunctions())
					if err != nil {
						log.Fatal("Failed to reload plugin", err)
					}
					proc.Update(load)
				}
			}
		})
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

		go watcher.Listen(func(notification *fsnotify.Event, err error) {
			if err != nil {
				log.Printf("Failed to handle file watcher changes: %s", err)
				return
			}
			if notification.Op.Has(fsnotify.Write) || notification.Op.Has(fsnotify.Create) {
				log.Printf("Detected change in %s", notification.Name)
				load, err = plugin.Load(ctx, notification.Name, functions.NewDefaultPluginFunctions())
				if err != nil {
					log.Fatal("Failed to reload plugin", err)
				}
				proc.Update(load)
			}
		})
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
