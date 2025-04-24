package kafka

import (
	"context"
	"errors"
	"github.com/kjkondratuk/kinetiq/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/twmb/franz-go/pkg/kgo"
	"testing"
	"time"
)

func TestKafkaReader_Read(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		setupMocks    func() *MockKafkaClient
		expectedPanic bool
		expected      []source.Record
	}{
		{
			name:    "disabled should sleep and skip fetch",
			enabled: false,
			setupMocks: func() *MockKafkaClient {
				mockClient := new(MockKafkaClient)
				mockClient.On("PollFetches", mock.Anything).Times(0)
				return mockClient
			},
			expectedPanic: false,
			expected:      nil,
		},
		{
			name:    "fetch error should panic",
			enabled: true,
			setupMocks: func() *MockKafkaClient {
				mockClient := new(MockKafkaClient)
				mockFetches := kgo.Fetches{
					{
						Topics: []kgo.FetchTopic{
							{
								Topic: "some-topic",
								Partitions: []kgo.FetchPartition{
									{
										Partition: 0,
										Err:       errors.New("fetch error"),
									},
								},
							},
						},
					},
				}
				mockClient.On("PollFetches", mock.Anything).Return(mockFetches).Once()
				return mockClient
			},
			expectedPanic: true,
			expected:      nil,
		},
		{
			name:    "process fetched records",
			enabled: true,
			setupMocks: func() *MockKafkaClient {
				mockClient := new(MockKafkaClient)
				mockRecord := &kgo.Record{
					Headers: []kgo.RecordHeader{
						{Key: "header1", Value: []byte("value1")},
					},
					Key:   []byte("key1"),
					Value: []byte("value1"),
				}

				mockFetches := kgo.Fetches{
					{
						Topics: []kgo.FetchTopic{
							{
								Topic: "some-topic",
								Partitions: []kgo.FetchPartition{
									{
										Partition: 0,
										Records:   []*kgo.Record{mockRecord},
									},
								},
							},
						},
					},
				}
				mockClient.On("PollFetches", mock.Anything).Return(mockFetches)
				return mockClient
			},
			expectedPanic: false,
			expected: []source.Record{
				{
					Headers: []source.RecordHeader{
						{
							Key:   "header1",
							Value: []byte("value1"),
						},
					},
					Key:   []byte("key1"),
					Value: []byte("value1"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := NewKafkaReader(tt.setupMocks())
			assert.NoError(t, err)

			ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
			defer cancel()

			go func(ctx context.Context) {
				select {
				case <-ctx.Done():
					return
				default:
					if tt.expectedPanic {
						assert.Panics(t, func() {
							reader.Read(ctx)
						})
					} else {
						reader.Read(ctx)
					}
				}
			}(ctx)

			// Assert results
			if !tt.expectedPanic && tt.enabled {
				for _, expected := range tt.expected {
					select {
					case result := <-reader.Output():
						// Compare only the fields we care about, ignoring the context
						assert.Equal(t, expected.Headers, result.Headers)
						assert.Equal(t, expected.Key, result.Key)
						assert.Equal(t, expected.Value, result.Value)
					case <-time.After(1 * time.Second):
						t.Fatal("timed out waiting for results")
					}
				}
			}
		})
	}
}

func TestKafkaReader_Output(t *testing.T) {
	results := make(chan source.Record, 5)
	reader := &kafkaReader{
		results: results,
	}
	assert.EqualValues(t, results, reader.Output())
}

func TestKafkaReader_EnableDisable(t *testing.T) {
	reader := &kafkaReader{}
	assert.False(t, reader.enabled.Load())

	reader.Enable()
	assert.True(t, reader.enabled.Load())

	reader.Disable()
	assert.False(t, reader.enabled.Load())
}

func TestKafkaReader_Close(t *testing.T) {
	results := make(chan source.Record, 5)
	reader := &kafkaReader{
		results: results,
	}

	reader.Close()

	select {
	case _, ok := <-reader.results:
		assert.False(t, ok)
	default:
		t.Fatal("expected channel to close but it did not")
	}
}

func TestNewKafkaReader(t *testing.T) {
	mockClient := new(MockKafkaClient)
	reader, err := NewKafkaReader(mockClient)

	assert.NoError(t, err)
	assert.NotNil(t, reader)
	assert.IsType(t, &kafkaReader{}, reader)
	assert.True(t, reader.(*kafkaReader).enabled.Load())
	assert.NotNil(t, reader.Output())
}
