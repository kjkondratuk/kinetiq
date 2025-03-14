package processor

import (
	"context"
	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
)

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
	Update(module v1.ModuleService)
	Output() <-chan Result
	Close()
}
