package main

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/fsnotify/fsnotify"
	"github.com/go-chi/chi/v5"
	chi_middleware "github.com/go-chi/chi/v5/middleware"
	app_config "github.com/kjkondratuk/kinetiq/config"
	"github.com/kjkondratuk/kinetiq/detection"
	"github.com/kjkondratuk/kinetiq/loader"
	"github.com/kjkondratuk/kinetiq/middleware"
	"github.com/kjkondratuk/kinetiq/otel"
	"github.com/kjkondratuk/kinetiq/processor"
	sink_kafka "github.com/kjkondratuk/kinetiq/sink/kafka"
	source_kafka "github.com/kjkondratuk/kinetiq/source/kafka"
	"github.com/twmb/franz-go/pkg/kgo"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
)

func main() {

	// Handle SIGINT gracefully.
	baseCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	dp, err := otel.NewDefaultOtelProvider(baseCtx)
	if err != nil {
		slog.Error("Failed to create default otel provider: %s", err)
		os.Exit(1)
	}

	// Set up OpenTelemetry.
	otelShutdown, err := otel.NewOtelSdk(dp).Configure()
	if err != nil {
		return
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(baseCtx))
	}()

	appCfg, err := app_config.DefaultConfigurator.Configure(baseCtx)
	if err != nil {
		slog.Error("Failed to load configuration: %s", err)
		os.Exit(1)
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
	defer dl.Close(baseCtx)

	log.Print("Plugin environment loaded...\n")

	consumerOpts := app_config.DefaultConfigurator.ConsumerConfig(appCfg)

	log.Printf("Connecting to kafka source brokers: %s", appCfg.Kafka.SourceBrokers)
	readerClient, err := kgo.NewClient(consumerOpts...)
	if err != nil {
		slog.Error("Failed to create kafka reader client", err)
		os.Exit(1)
	}
	defer readerClient.Close()

	log.Print("Reader client configured...")

	// Create Kafka reader with instrumentation
	reader, err := source_kafka.NewKafkaReader(readerClient)
	if err != nil {
		slog.Error("Failed to create kafka reader", err)
		os.Exit(1)
	}
	defer reader.Close()

	log.Print("Reader configured...")

	// Create WASM processor with instrumentation
	proc, err := processor.NewWasmProcessor(dl, reader.Output())
	if err != nil {
		slog.Error("Failed to create wasm processor", err)
		os.Exit(1)
	}
	defer proc.Close()

	log.Print("Processor configured...")

	log.Printf("Connecting to kafka dest brokers: %s", appCfg.Kafka.DestBrokers)

	producerOpts := app_config.DefaultConfigurator.ProducerConfig(appCfg)

	writerClient, err := kgo.NewClient(
		producerOpts...,
	)
	if err != nil {
		slog.Error("Failed to create kafka writer client", err)
		os.Exit(1)
	}
	defer writerClient.Close()

	log.Print("Writer client configured...")

	// Create Kafka writer with instrumentation
	writer, err := sink_kafka.NewKafkaWriter(writerClient, proc.Output())
	if err != nil {
		slog.Error("Failed to create kafka writer", err)
		os.Exit(1)
	}
	defer writer.Close()

	writerCtx, wc := context.WithCancel(baseCtx)
	defer wc()

	log.Print("Writer configured...")

	go writer.Write(writerCtx)
	log.Print("Writer started...")

	procCtx, pc := context.WithCancel(baseCtx)
	defer pc()

	go proc.Start(procCtx)

	log.Print("Processor started...")

	readCtx, rc := context.WithCancel(baseCtx)
	defer rc()

	go reader.Read(readCtx)

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
		go watcher.StartEvents(baseCtx)

		// Listen to events and process them
		go watcher.Listen(baseCtx, detection.S3NotificationPluginReloadResponder(baseCtx, appCfg.PluginRef, appCfg.S3.Bucket, dl))
	} else {
		// listen for changes from local file listener
		w, err := fsnotify.NewWatcher()
		if err != nil {
			slog.Error("failed to create path watcher: %s", err)
			os.Exit(1)
		}
		defer w.Close()

		parts := strings.Split(appCfg.PluginRef, "/")
		parentDir := strings.Join(parts[0:len(parts)-1], "/")

		err = w.Add(parentDir)
		if err != nil {
			slog.Error("failed to add path to watcher: %s", err)
			os.Exit(1)
		}

		watcher := detection.NewListener[fsnotify.Event](detection.NewWatcher(w.Events, w.Errors))

		go watcher.Listen(baseCtx, detection.FilesystemNotificationPluginReloadResponder(baseCtx, dl))
	}

	httpCtx, hc := context.WithCancel(baseCtx)
	defer hc()

	r := chi.NewRouter()

	r.Use(chi_middleware.Logger)
	r.Use(chi_middleware.Recoverer)

	// Add OpenTelemetry middleware
	httpInstr := otel.NewInstrumentation("http_server")
	r.Use(middleware.OtelMiddleware(httpInstr))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_, healthSpan := httpInstr.StartSpan(httpCtx, "HttpServer.HealthCheck")
		defer healthSpan.End()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Addr:        ":8080",
		BaseContext: func(_ net.Listener) context.Context { return httpCtx },
		// TODO : make server timeouts configurable
		//ReadTimeout:  time.Second,
		//WriteTimeout: 10 * time.Second,
		Handler: r,
	}
	srvErr := make(chan error, 1)
	go func() {
		srvErr <- srv.ListenAndServe()
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		return
	case <-baseCtx.Done():
		stop()
	}

	err = srv.Shutdown(context.Background())
	return
}
