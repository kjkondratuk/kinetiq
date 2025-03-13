package kafka

import (
	"context"
	"fmt"
	"github.com/kjkondratuk/kinetiq/source"
	"github.com/twmb/franz-go/pkg/kgo"
	"sync/atomic"
	"time"
)

type kafkaReader struct {
	client  *kgo.Client
	enabled atomic.Bool
	results chan source.Record
}

func NewKafkaReader(client *kgo.Client) source.Source {
	reader := &kafkaReader{
		client:  client,
		results: make(chan source.Record),
	}
	// Default to enabled
	reader.enabled.Store(true)
	return reader

}

func (r *kafkaReader) Output() <-chan source.Record {
	return r.results
}

func (r *kafkaReader) Read(ctx context.Context) {
	for {
		// Exit loop if disabled
		if !r.enabled.Load() {
			time.Sleep(100 * time.Millisecond) // Prevent busy-waiting
			continue
		}

		fetches := r.client.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			// All errors are retried internally when fetching, but non-retriable errors are
			// returned from polls so that users can notice and take action.
			panic(fmt.Sprint(errs))
		}

		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()

			headers := make([]source.RecordHeader, len(record.Headers))
			for i, header := range record.Headers {
				headers[i] = source.RecordHeader{
					Key:   header.Key,
					Value: header.Value,
				}
			}

			standardRec := source.Record{
				Headers: headers,
				Key:     record.Key,
				Value:   record.Value,
			}
			r.results <- standardRec
		}
	}
}

func (r *kafkaReader) Close() {
	close(r.results)
}

// Enable turns on the KafkaReader
func (r *kafkaReader) Enable() {
	r.enabled.Store(true)
}

// Disable turns off the KafkaReader
func (r *kafkaReader) Disable() {
	r.enabled.Store(false)
}
