package processor

import (
	"context"
	"fmt"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	"github.com/kjkondratuk/kinetiq/loader"
	"github.com/kjkondratuk/kinetiq/source"
	"log"
)

type wasmProcessor struct {
	ldr loader.Loader
	//lock   sync.Mutex
	input  <-chan source.Record
	output chan Result
}

func NewWasmProcessor(ldr loader.Loader, channel <-chan source.Record) Processor {
	p := &wasmProcessor{
		ldr:    ldr,
		input:  channel,
		output: make(chan Result),
	}

	return p
}

func (p *wasmProcessor) Output() <-chan Result {
	return p.output
}

func (p *wasmProcessor) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case input := <-p.input:
			// TODO : can probably improve performance by not locking on each message and instead signalling a stop of this loop and a restart
			//p.lock.Lock()
			process, err := p.process(ctx, input)
			//p.lock.Unlock()
			if err != nil {
				log.Printf("error processing record: %s\n", err)
				continue
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
	}, nil
}

func (p *wasmProcessor) Close() {
	close(p.output)
}
