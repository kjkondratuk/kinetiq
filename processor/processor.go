package processor

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Result struct {
	Ctx     context.Context
	Headers []RecordHeader
	Key     []byte
	Value   []byte

	// Src is the originating Kafka record, carried through so the writer can
	// mark it after a successful produce.
	Src *kgo.Record
}

type RecordHeader struct {
	Key   string
	Value []byte
}

type Processor interface {
	Start(ctx context.Context)
	Output() <-chan Result
	Close()
}
