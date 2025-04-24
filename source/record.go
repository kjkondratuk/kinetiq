package source

import "context"

type Record struct {
	Ctx     context.Context
	Headers []RecordHeader
	Key     []byte
	Value   []byte
}

type RecordHeader struct {
	Key   string
	Value []byte
}

type RecordProcessor func(record *Record) error
