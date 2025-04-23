package kafka

import (
	"context"
	"fmt"
	"github.com/kjkondratuk/kinetiq/otel"
	"github.com/kjkondratuk/kinetiq/processor"
	"github.com/kjkondratuk/kinetiq/sink"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"sync/atomic"
)

type kafkaWriter struct {
	client  KafkaClient
	enabled atomic.Bool
	input   <-chan processor.Result

	// Instrumentation
	instr                    *otel.Instrumentation
	recordsWrittenCounter    metric.Int64Counter
	writeErrorsCounter       metric.Int64Counter
	processingTimeHistogram  metric.Float64Histogram
}

type KafkaClient interface {
	Produce(ctx context.Context, record *kgo.Record, cb func(record *kgo.Record, err error))
	Close()
}

func NewKafkaWriter(client KafkaClient, input <-chan processor.Result) (sink.Sink, error) {
	// Create instrumentation
	instr := otel.NewInstrumentation("kafka_writer")

	// Create metrics
	recordsWrittenCounter, err := instr.CreateCounter(
		"kinetiq.sink.kafka.records_written",
		"Number of records written to Kafka",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create records written counter: %w", err)
	}

	writeErrorsCounter, err := instr.CreateCounter(
		"kinetiq.sink.kafka.write_errors",
		"Number of errors encountered while writing to Kafka",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create write errors counter: %w", err)
	}

	processingTimeHistogram, err := instr.CreateHistogram(
		"kinetiq.sink.kafka.processing_time",
		"Time taken to process a record to Kafka in milliseconds",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create processing time histogram: %w", err)
	}

	w := &kafkaWriter{
		client:                  client,
		input:                   input,
		instr:                   instr,
		recordsWrittenCounter:   recordsWrittenCounter,
		writeErrorsCounter:      writeErrorsCounter,
		processingTimeHistogram: processingTimeHistogram,
	}
	w.enabled.Store(true)

	return w, nil
}

func (w *kafkaWriter) Write(ctx context.Context) {
	w.instr.LogInfo("Starting Kafka writer")

	// Create a new context with a span
	ctx, span := w.instr.StartSpan(ctx, "KafkaWriter.Write")
	defer span.End()

	for {
		select {
		case <-ctx.Done():
			return
		case input, ok := <-w.input:
			if !ok {
				w.instr.LogInfo("Kafka producer input channel closed")
				return
			}

			// Create a new context with a span for this record
			writeCtx, writeSpan := w.instr.StartSpan(ctx, "KafkaWriter.WriteRecord")

			// Add attributes to the span
			writeSpan.SetAttributes(
				attribute.String("result.key", string(input.Key)),
				attribute.Int("result.value.size", len(input.Value)),
				attribute.Int("result.headers.count", len(input.Headers)),
			)

			// Measure processing time
			stopMeasure := w.instr.MeasureExecutionTime(writeCtx, w.processingTimeHistogram)

			headers := make([]kgo.RecordHeader, len(input.Headers))
			for i, h := range input.Headers {
				headers[i] = kgo.RecordHeader{
					Key:   h.Key,
					Value: h.Value,
				}
			}

			w.client.Produce(writeCtx, &kgo.Record{
				Key:     input.Key,
				Value:   input.Value,
				Headers: headers,
			}, func(record *kgo.Record, err error) {
				if err != nil {
					// Record the error
					w.instr.RecordError(writeSpan, err)
					w.writeErrorsCounter.Add(writeCtx, 1)
					w.instr.LogError("Error producing to Kafka", err)
				} else {
					// Record successful write
					w.recordsWrittenCounter.Add(writeCtx, 1)
				}

				// End the span and stop measuring
				writeSpan.End()
				stopMeasure()
			})
		}
	}
}

func (w *kafkaWriter) Close() {
	w.instr.LogInfo("Closing Kafka writer")
	w.client.Close()
}
