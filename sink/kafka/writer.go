package kafka

import (
	"context"
	"github.com/kjkondratuk/kinetiq/processor"
	"github.com/kjkondratuk/kinetiq/sink"
	"github.com/twmb/franz-go/pkg/kgo"
	"log"
	"sync/atomic"
)

type kafkaWriter struct {
	client  *kgo.Client
	enabled atomic.Bool
	input   <-chan processor.Result
}

func NewKafkaWriter(client *kgo.Client, input <-chan processor.Result) sink.Sink {
	w := &kafkaWriter{
		client: client,
		input:  input,
	}
	w.enabled.Store(true)

	return w
}

func (w *kafkaWriter) Write(ctx context.Context) {
	for {
		select {
		case input := <-w.input:
			headers := make([]kgo.RecordHeader, len(input.Headers))
			for i, h := range input.Headers {
				headers[i] = kgo.RecordHeader{
					Key:   h.Key,
					Value: h.Value,
				}
			}

			w.client.Produce(ctx, &kgo.Record{
				Key:     input.Key,
				Value:   input.Value,
				Headers: headers,
			}, func(record *kgo.Record, err error) {
				if err != nil {
					log.Printf("Error producing to kafka: %s - %e", input.Key, err)
				}
			})
		}
	}
}

func (w *kafkaWriter) Close() {
	w.client.Close()
}
