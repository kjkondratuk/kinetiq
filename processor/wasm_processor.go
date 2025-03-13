package processor

import (
	"context"
	"fmt"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"github.com/kjkondratuk/kinetiq/source"
	"log"
	"sync/atomic"
)

type wasmProcessor struct {
	plugin  v1.ModuleService
	enabled atomic.Bool
	input   <-chan source.Record
	output  chan Result
}

func NewWasmProcessor(plugin v1.ModuleService, channel <-chan source.Record) Processor {
	p := &wasmProcessor{
		plugin: plugin,
		input:  channel,
		output: make(chan Result),
	}

	p.enabled.Store(true)

	return p
}

func (p *wasmProcessor) Output() <-chan Result {
	return p.output
}

func (p *wasmProcessor) Start(ctx context.Context) {
	for {
		select {
		case input := <-p.input:
			process, err := p.process(ctx, input)
			if err != nil {
				log.Println("error processing record: %w", err)
			}
			p.output <- process
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
	res, err := p.plugin.Process(ctx, req)
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
		Key:     record.Key,
		Value:   record.Value,
		Headers: hdr,
	}, nil
}

func (p *wasmProcessor) Close() {
	close(p.output)
}

// Enable turns on the processor
func (p *wasmProcessor) Enable() {
	p.enabled.Store(true)
}

// Disable turns off the processor
func (p *wasmProcessor) Disable() {
	p.enabled.Store(false)
}
