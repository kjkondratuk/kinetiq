package kafka

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kjkondratuk/kinetiq/otel"
	"github.com/kjkondratuk/kinetiq/source"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"log/slog"
	"sync/atomic"
	"time"
)

type kafkaReader struct {
	ctx     context.Context
	client  KafkaClient
	enabled atomic.Bool
	results chan source.Record

	// Instrumentation
	instr                   *otel.Instrumentation
	recordsReadCounter      metric.Int64Counter
	readErrorsCounter       metric.Int64Counter
	processingTimeHistogram metric.Float64Histogram
}

type KafkaClient interface {
	PollFetches(ctx context.Context) kgo.Fetches
}

func NewKafkaReader(client KafkaClient) (source.Source, error) {
	// Create instrumentation
	instr := otel.NewInstrumentation("kafka_reader")

	// Create metrics
	recordsReadCounter, err := instr.CreateCounter(
		"kinetiq.source.kafka.records_read",
		"Number of records read from Kafka",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create records read counter: %w", err)
	}

	readErrorsCounter, err := instr.CreateCounter(
		"kinetiq.source.kafka.read_errors",
		"Number of errors encountered while reading from Kafka",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create read errors counter: %w", err)
	}

	processingTimeHistogram, err := instr.CreateHistogram(
		"kinetiq.source.kafka.processing_time",
		"Time taken to process a record from Kafka in milliseconds",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create processing time histogram: %w", err)
	}

	reader := &kafkaReader{
		client:                  client,
		results:                 make(chan source.Record),
		instr:                   instr,
		recordsReadCounter:      recordsReadCounter,
		readErrorsCounter:       readErrorsCounter,
		processingTimeHistogram: processingTimeHistogram,
	}
	// Default to enabled
	reader.enabled.Store(true)
	return reader, nil
}

func (r *kafkaReader) Read(ctx context.Context) {
	slog.Info("Starting Kafka reader")

	// Create a new context with a span
	//ctx, span := r.instr.StartSpan(ctx, "KafkaReader.Read")
	//defer span.End()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Exit loop if disabled
			if !r.enabled.Load() {
				time.Sleep(100 * time.Millisecond) // Prevent busy-waiting
				continue
			}

			fetches := r.client.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				// All errors are retried internally when fetching, but non-retriable errors are
				// returned from polls so that users can notice and take action.
				slog.Error("Error polling fetches from Kafka", "error", fmt.Errorf("%v", errs))
				r.readErrorsCounter.Add(ctx, 1)
				panic(fmt.Sprint(errs))
			}

			iter := fetches.RecordIter()
			for !iter.Done() {
				record := iter.Next()

				// Create a new context with a span for this record, with a correlation ID
				correlationID := uuid.New().String()
				corrCtx := context.WithValue(ctx, "correlation_id", correlationID)
				recordCtx, recordSpan := r.instr.StartSpan(corrCtx, "KafkaReader.ProcessRecord")

				// Measure processing time
				stopMeasure := r.instr.MeasureExecutionTime(recordCtx, r.processingTimeHistogram)

				headers := make([]source.RecordHeader, len(record.Headers))
				for i, header := range record.Headers {
					headers[i] = source.RecordHeader{
						Key:   header.Key,
						Value: header.Value,
					}
				}

				standardRec := source.Record{
					Headers: headers,
					Key:     record.Key,
					Value:   record.Value,
					Ctx:     recordCtx,
				}

				// Add attributes to the span
				recordSpan.SetAttributes(
					attribute.String("record.key", string(standardRec.Key)),
					attribute.Int("record.value.size", len(standardRec.Value)),
					attribute.Int("record.headers.count", len(standardRec.Headers)),
				)

				// Record successful read
				r.recordsReadCounter.Add(recordCtx, 1)

				r.results <- standardRec

				// End the span and stop measuring
				recordSpan.End()
				stopMeasure()
			}
		}
	}
}

func (r *kafkaReader) Output() <-chan source.Record {
	return r.results
}

func (r *kafkaReader) Close() {
	slog.Info("Closing Kafka reader")
	close(r.results)
}

// Enable turns on the KafkaReader
func (r *kafkaReader) Enable() {
	slog.Info("Enabling Kafka reader")
	r.enabled.Store(true)
}

// Disable turns off the KafkaReader
func (r *kafkaReader) Disable() {
	slog.Info("Disabling Kafka reader")
	r.enabled.Store(false)
}
