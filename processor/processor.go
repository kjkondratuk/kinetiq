package processor

import "context"

type Result struct {
	Headers []RecordHeader
	Key     []byte
	Value   []byte
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
