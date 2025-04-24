package otel

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"time"
)

const (
	// Instrumentation name and version used for all telemetry
	instrumentationName    = "github.com/kjkondratuk/kinetiq"
	instrumentationVersion = "0.1.0"
)

// Instrumentation provides access to OpenTelemetry instrumentation
type Instrumentation struct {
	tracer trace.Tracer
	meter  metric.Meter
	logger log.Logger
	name   string
}

// NewInstrumentation creates a new Instrumentation instance
func NewInstrumentation(name string) *Instrumentation {
	return &Instrumentation{
		tracer: otel.GetTracerProvider().Tracer(
			instrumentationName,
			trace.WithInstrumentationVersion(instrumentationVersion),
		),
		meter: otel.GetMeterProvider().Meter(
			instrumentationName,
			metric.WithInstrumentationVersion(instrumentationVersion),
		),
		logger: global.GetLoggerProvider().Logger(
			instrumentationName,
			log.WithInstrumentationVersion(instrumentationVersion),
		),
		name: name,
	}
}

// StartSpan starts a new span with the given name and returns the span and a context containing the span
func (i *Instrumentation) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return i.tracer.Start(ctx, name, opts...)
}

// RecordError records an error in the current span
func (i *Instrumentation) RecordError(span trace.Span, err error, opts ...trace.EventOption) {
	span.RecordError(err, opts...)
	span.SetStatus(codes.Error, err.Error())
}

// CreateCounter creates a new counter metric
func (i *Instrumentation) CreateCounter(name, description string, opts ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return i.meter.Int64Counter(name, metric.WithDescription(description), metric.WithUnit("1"))
}

// CreateUpDownCounter creates a new up/down counter metric
func (i *Instrumentation) CreateUpDownCounter(name, description string, opts ...metric.Int64UpDownCounterOption) (metric.Int64UpDownCounter, error) {
	return i.meter.Int64UpDownCounter(name, metric.WithDescription(description), metric.WithUnit("1"))
}

// CreateHistogram creates a new histogram metric
func (i *Instrumentation) CreateHistogram(name, description string, opts ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	return i.meter.Float64Histogram(name, metric.WithDescription(description), metric.WithUnit("ms"))
}

// MeasureExecutionTime measures the execution time of a function and records it in a histogram
func (i *Instrumentation) MeasureExecutionTime(ctx context.Context, histogram metric.Float64Histogram, attrs ...attribute.KeyValue) func() {
	start := time.Now()
	return func() {
		elapsed := float64(time.Since(start).Milliseconds())
		histogram.Record(ctx, elapsed, metric.WithAttributes(attrs...))
	}
}
