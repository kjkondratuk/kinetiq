package otel

import (
	"context"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	"os"
	"strconv"
	"time"
)

// getOtlpEndpoint returns the OTLP endpoint from environment variables
// It checks for signal-specific endpoint first, then falls back to the general endpoint
// If no environment variable is set, it returns the default endpoint
func getOtlpEndpoint(signal string) string {
	// Check for signal-specific endpoint
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_" + signal + "_ENDPOINT"); endpoint != "" {
		return endpoint
	}

	// Check for general endpoint
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		return endpoint
	}

	// Default endpoint
	return "localhost:4317"
}

// isOtlpInsecure returns whether to use an insecure connection from environment variables
// It checks for signal-specific setting first, then falls back to the general setting
// If no environment variable is set, it returns the default (true)
func isOtlpInsecure(signal string) bool {
	// Check for signal-specific setting
	if insecureStr := os.Getenv("OTEL_EXPORTER_OTLP_" + signal + "_INSECURE"); insecureStr != "" {
		insecure, err := strconv.ParseBool(insecureStr)
		if err == nil {
			return insecure
		}
	}

	// Check for general setting
	if insecureStr := os.Getenv("OTEL_EXPORTER_OTLP_INSECURE"); insecureStr != "" {
		insecure, err := strconv.ParseBool(insecureStr)
		if err == nil {
			return insecure
		}
	}

	// Default to insecure
	return true
}

type otelProvider struct {
	tracerProvider *trace.TracerProvider
	meterProvider  *metric.MeterProvider
	loggerProvider *log.LoggerProvider
	prop           propagation.TextMapPropagator
}

type OtelProvider interface {
	Tracer() *trace.TracerProvider
	Meter() *metric.MeterProvider
	Logger() *log.LoggerProvider
	Propagator() propagation.TextMapPropagator
}

func NewDefaultOtelProvider(ctx context.Context) (OtelProvider, error) {
	op := otelProvider{}
	tp, err := op.newDefaultTracerProvider(ctx)
	if err != nil {
		return nil, err
	}

	mp, err := op.newDefaultMeterProvider(ctx)
	if err != nil {
		return nil, err
	}

	lp, err := op.newDefaultLoggerProvider(ctx)
	if err != nil {
		return nil, err
	}

	op.meterProvider = mp
	op.loggerProvider = lp
	op.tracerProvider = tp
	op.prop = op.newDefaultPropagator()

	return &op, nil
}

func (op *otelProvider) Tracer() *trace.TracerProvider {
	return op.tracerProvider
}

func (op *otelProvider) Meter() *metric.MeterProvider {
	return op.meterProvider
}

func (op *otelProvider) Logger() *log.LoggerProvider {
	return op.loggerProvider
}

func (op *otelProvider) Propagator() propagation.TextMapPropagator {
	return op.prop
}

func (op *otelProvider) newDefaultPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func (op *otelProvider) newDefaultTracerProvider(ctx context.Context) (*trace.TracerProvider, error) {
	endpoint := getOtlpEndpoint("TRACES")
	insecure := isOtlpInsecure("TRACES")

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
	}

	if insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	traceExporter, err := otlptracegrpc.New(ctx, opts...)
	//stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
	)
	return tracerProvider, nil
}

func (op *otelProvider) newDefaultMeterProvider(ctx context.Context) (*metric.MeterProvider, error) {
	//metricExporter, err := stdoutmetric.New()
	endpoint := getOtlpEndpoint("METRICS")
	insecure := isOtlpInsecure("METRICS")

	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(endpoint),
	}

	if insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	metricExporter, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(
			metricExporter,
			// Set a shorter interval for more frequent metric exports
			metric.WithInterval(15*time.Second),
		)),
	)
	return meterProvider, nil
}

func (op *otelProvider) newDefaultLoggerProvider(ctx context.Context) (*log.LoggerProvider, error) {
	//logExporter, err := stdoutlog.New()
	endpoint := getOtlpEndpoint("LOGS")
	insecure := isOtlpInsecure("LOGS")

	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(endpoint),
	}

	if insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}

	logExporter, err := otlploggrpc.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
	)
	return loggerProvider, nil
}
