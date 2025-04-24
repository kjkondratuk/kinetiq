package processor

import (
	"context"
	"fmt"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"github.com/kjkondratuk/kinetiq/loader"
	"github.com/kjkondratuk/kinetiq/otel"
	"github.com/kjkondratuk/kinetiq/source"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"log/slog"
)

type wasmProcessor struct {
	ldr    loader.Loader
	input  <-chan source.Record
	output chan Result

	// Instrumentation
	instr                   *otel.Instrumentation
	processingTimeHistogram metric.Float64Histogram
	recordsProcessedCounter metric.Int64Counter
	processingErrorsCounter metric.Int64Counter
}

// NewWasmProcessor creates a new WASM processor with OpenTelemetry instrumentation
func NewWasmProcessor(ldr loader.Loader, channel <-chan source.Record) (Processor, error) {
	// Create instrumentation
	instr := otel.NewInstrumentation("wasm_processor")

	// Create metrics
	processingTimeHistogram, err := instr.CreateHistogram(
		"kinetiq.processor.processing_time",
		"Time taken to process a record in milliseconds",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create processing time histogram: %w", err)
	}

	recordsProcessedCounter, err := instr.CreateCounter(
		"kinetiq.processor.records_processed",
		"Number of records processed",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create records processed counter: %w", err)
	}

	processingErrorsCounter, err := instr.CreateCounter(
		"kinetiq.processor.processing_errors",
		"Number of errors encountered during processing",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create processing errors counter: %w", err)
	}

	p := &wasmProcessor{
		ldr:                     ldr,
		input:                   channel,
		output:                  make(chan Result),
		instr:                   instr,
		processingTimeHistogram: processingTimeHistogram,
		recordsProcessedCounter: recordsProcessedCounter,
		processingErrorsCounter: processingErrorsCounter,
	}

	return p, nil
}

func (p *wasmProcessor) Output() <-chan Result {
	return p.output
}

func (p *wasmProcessor) Start(ctx context.Context) {
	slog.Info("Starting WASM processor")

	// Create a new context with a span
	//ctx, span := p.instr.StartSpan(ctx, "WasmProcessor.Start")
	//defer span.End()

	for {
		select {
		case <-ctx.Done():
			return
		case record, ok := <-p.input:
			if !ok {
				return
			}

			// Create a new context with a span for this record
			processCtx, processSpan := p.instr.StartSpan(record.Ctx, "WasmProcessor.process")

			// Add attributes to the span
			processSpan.SetAttributes(
				attribute.String("record.key", string(record.Key)),
				attribute.Int("record.value.size", len(record.Value)),
				attribute.Int("record.headers.count", len(record.Headers)),
			)

			// Measure processing time
			stopMeasure := p.instr.MeasureExecutionTime(processCtx, p.processingTimeHistogram)

			process, err := p.process(processCtx, record)
			if err != nil {
				// Record the error
				p.instr.RecordError(processSpan, err)
				p.processingErrorsCounter.Add(processCtx, 1)
				slog.Error("Error processing record", "error", err)
				processSpan.End()
				stopMeasure()
				continue
			}

			// Record successful processing
			p.recordsProcessedCounter.Add(processCtx, 1)

			// Add result attributes to the span
			processSpan.SetAttributes(
				attribute.String("result.key", string(process.Key)),
				attribute.Int("result.value.size", len(process.Value)),
				attribute.Int("result.headers.count", len(process.Headers)),
			)

			p.output <- process

			// End the span and stop measuring
			processSpan.End()
			stopMeasure()
		}
	}
}

func (p *wasmProcessor) process(ctx context.Context, record source.Record) (Result, error) {
	// TODO : should probably not have to convert headers like this all over the place
	headers := make([]*v1.Headers, len(record.Headers))
	for i, header := range record.Headers {
		headers[i] = &v1.Headers{
			Key:   header.Key,
			Value: header.Value,
		}
	}

	req := &v1.ProcessRequest{
		Key:     record.Key,
		Value:   record.Value,
		Headers: headers,
	}

	processor, err := p.ldr.Get(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("error getting processor: %w", err)
	}

	res, err := processor.Process(ctx, req)
	if err != nil {
		return Result{}, fmt.Errorf("error processing record: %w", err)
	}

	hdr := make([]RecordHeader, len(res.Headers))
	for i, header := range res.Headers {
		hdr[i] = RecordHeader{
			Key:   header.Key,
			Value: header.Value,
		}
	}

	return Result{
		Key:     res.Key,
		Value:   res.Value,
		Headers: hdr,
		Ctx:     ctx,
	}, nil
}

func (p *wasmProcessor) Close() {
	slog.Info("Closing WASM processor")
	close(p.output)
}
