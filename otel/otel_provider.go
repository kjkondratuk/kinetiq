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
)

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
	traceExporter, err := otlptracegrpc.New(ctx)
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
	metricExporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter)),
	)
	return meterProvider, nil
}

func (op *otelProvider) newDefaultLoggerProvider(ctx context.Context) (*log.LoggerProvider, error) {
	//logExporter, err := stdoutlog.New()
	logExporter, err := otlploggrpc.New(ctx)
	if err != nil {
		return nil, err
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
	)
	return loggerProvider, nil
}
