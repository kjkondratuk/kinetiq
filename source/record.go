package source

type Record struct {
	Headers []RecordHeader
	Key     []byte
	Value   []byte
}

type RecordHeader struct {
	Key   string
	Value []byte
}

type RecordProcessor func(record *Record) error
