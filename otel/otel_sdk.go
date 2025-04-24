package otel

import (
	"context"
	"errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
)

type otelSdk struct {
	op OtelProvider
}

type OtelSdk interface {
	Configure() (ShutdownHandler, error)
}

type ShutdownHandler func(context.Context) error

func NewOtelSdk(op OtelProvider) OtelSdk {
	return &otelSdk{
		op: op,
	}
}

func (s *otelSdk) Configure() (shutdown ShutdownHandler, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// Set up propagator.
	otel.SetTextMapPropagator(s.op.Propagator())

	// Set up trace provider.
	tracerProvider := s.op.Tracer()
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up meter provider.
	meterProvider := s.op.Meter()
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	// Set up logger provider.
	loggerProvider := s.op.Logger()
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)

	// TODO : add options here to customize otel behavior

	return
}
