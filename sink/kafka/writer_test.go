package kafka

import (
	"context"
	"errors"
	"github.com/kjkondratuk/kinetiq/processor"
	"github.com/stretchr/testify/mock"
	"github.com/twmb/franz-go/pkg/kgo"
	"sync/atomic"
	"testing"
	"time"
)

func TestKafkaWriter_Write(t *testing.T) {
	type testCase struct {
		name         string
		setupMocks   func() *MockKafkaClient
		input        chan processor.Result
		expectOutput func(t *testing.T, client *MockKafkaClient)
	}

	enabled := atomic.Bool{}
	enabled.Store(true)

	cases := []testCase{
		{
			name: "single successful produce",
			setupMocks: func() *MockKafkaClient {
				client := &MockKafkaClient{}
				client.On("Produce", mock.Anything, mock.Anything, mock.Anything).Return().Run(func(args mock.Arguments) {
					cb := args.Get(2).(func(*kgo.Record, error))
					cb(&kgo.Record{}, nil)
				})
				client.On("Close").Return()
				return client
			},
			input: func() chan processor.Result {
				ch := make(chan processor.Result, 1)
				ctx := context.Background()
				ch <- processor.Result{
					Key:   []byte("key"),
					Value: []byte("value"),
					Headers: []processor.RecordHeader{
						{Key: "header-key", Value: []byte("header-value")},
					},
					Ctx: ctx,
				}
				return ch
			}(),
			expectOutput: func(t *testing.T, client *MockKafkaClient) {
				client.AssertCalled(t, "Produce", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name: "produce error",
			setupMocks: func() *MockKafkaClient {
				client := &MockKafkaClient{}
				client.On("Produce", mock.Anything, mock.Anything, mock.Anything).Return().Run(func(args mock.Arguments) {
					cb := args.Get(2).(func(*kgo.Record, error))
					cb(nil, errors.New("produce error"))
				})
				client.On("Close").Return()
				return client
			},
			input: func() chan processor.Result {
				ch := make(chan processor.Result, 1)
				ctx := context.Background()
				ch <- processor.Result{
					Key:   []byte("key"),
					Value: []byte("value"),
					Ctx: ctx,
				}
				return ch
			}(),
			expectOutput: func(t *testing.T, client *MockKafkaClient) {
				client.AssertCalled(t, "Produce", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name: "empty channel",
			setupMocks: func() *MockKafkaClient {
				client := &MockKafkaClient{}
				client.On("Produce", mock.Anything, mock.Anything, mock.Anything).Return()
				client.On("Close").Return()
				return client
			},
			input: func() chan processor.Result {
				ch := make(chan processor.Result)
				return ch
			}(),
			expectOutput: func(t *testing.T, client *MockKafkaClient) {
				client.AssertNotCalled(t, "Produce", mock.Anything, mock.Anything, mock.Anything)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := tc.setupMocks()

			writer, err := NewKafkaWriter(mockClient, tc.input)
			if err != nil {
				t.Fatalf("Failed to create Kafka writer: %v", err)
			}
			defer close(tc.input)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			go writer.Write(ctx)
			<-ctx.Done() // Allow Write to process inputs
			writer.Close()

			tc.expectOutput(t, mockClient)
		})
	}
}

func TestKafkaWriter_Close(t *testing.T) {
	mockClient := &MockKafkaClient{}
	mockClient.On("Close").Return()

	writer := &kafkaWriter{
		client: mockClient,
	}

	writer.Close()

	mockClient.AssertCalled(t, "Close")
}
