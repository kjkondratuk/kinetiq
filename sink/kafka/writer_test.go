package kafka

import (
	"context"
	"errors"
	"github.com/kjkondratuk/kinetiq/processor"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	"sync"
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

			writer, err := NewKafkaWriter(mockClient, tc.input, nil)
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

// fakeMarker records which source records were marked for commit. The method is
// named to match *kgo.Client.MarkCommitRecords so the real client satisfies the
// same interface with no adapter.
type fakeMarker struct {
	mu     sync.Mutex
	marked []*kgo.Record
}

func (f *fakeMarker) MarkCommitRecords(recs ...*kgo.Record) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.marked = append(f.marked, recs...)
}

func (f *fakeMarker) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.marked)
}

// fakeProducer drives the callback outcome synchronously.
type fakeProducer struct{ err error }

func (f *fakeProducer) Produce(ctx context.Context, r *kgo.Record, cb func(*kgo.Record, error)) {
	cb(r, f.err)
}
func (f *fakeProducer) Close() {}

func TestKafkaWriter_MarksOnSuccessOnly(t *testing.T) {
	t.Run("successful_write_marks_source_record", func(t *testing.T) {
		marker := &fakeMarker{}
		producer := &fakeProducer{}

		input := make(chan processor.Result, 1)
		input <- processor.Result{
			Key:   []byte("key"),
			Value: []byte("value"),
			Ctx:   context.Background(),
			Src:   &kgo.Record{Offset: 7},
		}

		writer, err := NewKafkaWriter(producer, input, marker)
		if err != nil {
			t.Fatalf("Failed to create Kafka writer: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		go writer.Write(ctx)
		defer writer.Close()
		defer close(input)

		require.Eventually(t, func() bool {
			return marker.count() == 1
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("failed_write_does_not_mark", func(t *testing.T) {
		marker := &fakeMarker{}
		producer := &fakeProducer{err: errors.New("produce failed")}

		input := make(chan processor.Result, 1)
		input <- processor.Result{
			Key:   []byte("key"),
			Value: []byte("value"),
			Ctx:   context.Background(),
			Src:   &kgo.Record{Offset: 7},
		}

		writer, err := NewKafkaWriter(producer, input, marker)
		if err != nil {
			t.Fatalf("Failed to create Kafka writer: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		go writer.Write(ctx)
		<-ctx.Done()
		writer.Close()
		close(input)

		require.Equal(t, 0, marker.count(), "an unwritten record must not have its offset committed")
	})
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
