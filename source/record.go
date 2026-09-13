package source

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Record struct {
	Ctx     context.Context
	Headers []RecordHeader
	Key     []byte
	Value   []byte

	// Src is the originating Kafka record, retained so its offset can be
	// marked for commit once the result has been written.
	Src *kgo.Record
}

type RecordHeader struct {
	Key   string
	Value []byte
}

type RecordProcessor func(record *Record) error
