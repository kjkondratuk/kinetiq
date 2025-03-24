package detection

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/kjkondratuk/kinetiq/loader"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSqsWatcher_StartEvents(t *testing.T) {
	mockClient := &MockSqsClient{}
	mockWatcher := &MockWatcher[S3EventNotification]{}
	mockListener := &MockListener[S3EventNotification]{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	validMessageBody := `{"Records":[{"eventName":"value"}]}`
	mockClient.On("ReceiveMessage", mock.Anything, mock.Anything, mock.Anything).
		Return(&sqs.ReceiveMessageOutput{
			Messages: []types.Message{
				{
					Body:          &validMessageBody,
					MessageId:     aws.String("message-1"),
					ReceiptHandle: aws.String("receipt-1"),
				},
			},
		}, nil).
		Once()

	mockClient.On("DeleteMessage", mock.Anything, mock.Anything, mock.Anything).
		Return(&sqs.DeleteMessageOutput{}, nil).
		Once()

	eventChan := make(chan S3EventNotification, 1)
	//errChan := make(chan error, 1)

	mockWatcher.On("EventsChan").Return(eventChan)
	//mockWatcher.On("ErrorsChan").Return(errChan)

	watcher := &sqsWatcher{
		client:              mockClient,
		watcher:             mockWatcher,
		listener:            mockListener,
		url:                 "test-queue-url",
		pollIntervalSeconds: 1,
	}

	go watcher.StartEvents(ctx)

	select {
	case event := <-eventChan:
		assert.Equal(t, "value", event.Records[0].EventName)
	case <-time.After(2 * time.Second):
		t.Error("expected event to be received")
	}

	mockClient.AssertExpectations(t)
	mockWatcher.AssertExpectations(t)
	mockListener.AssertExpectations(t)
}

func TestSqsWatcher_Process_InvalidMessage(t *testing.T) {
	mockClient := &MockSqsClient{}
	mockWatcher := &MockWatcher[S3EventNotification]{}

	ctx := context.Background()
	invalidMessageBody := `{invalid-json}`

	mockWatcher.On("ErrorsChan").Return(make(chan error, 1)).Twice()

	watcher := &sqsWatcher{
		client:  mockClient,
		watcher: mockWatcher,
		url:     "test-queue-url",
	}

	message := types.Message{
		Body:          aws.String(invalidMessageBody),
		MessageId:     aws.String("message-id"),
		ReceiptHandle: aws.String("receipt-handle"),
	}

	watcher.process(ctx, message)

	select {
	case err := <-mockWatcher.ErrorsChan():
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshalling")
	default:
		t.Error("expected error message in ErrorsChan")
	}

	mockClient.AssertNotCalled(t, "DeleteMessage", mock.Anything, mock.Anything)
	mockWatcher.AssertExpectations(t)
}

func TestSqsWatcher_Process_ValidMessage_DeleteFails(t *testing.T) {
	mockClient := &MockSqsClient{}
	mockWatcher := &MockWatcher[S3EventNotification]{}

	ctx := context.Background()
	validMessageBody := `{"Records":[{"eventName":"value"}]}`
	mockClient.On("DeleteMessage", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("delete error")).
		Once()
	mockWatcher.On("EventsChan").Return(make(chan S3EventNotification, 1))
	mockWatcher.On("ErrorsChan").Return(make(chan error, 1))

	watcher := &sqsWatcher{
		client:  mockClient,
		watcher: mockWatcher,
		url:     "test-queue-url",
	}

	message := types.Message{
		Body:          aws.String(validMessageBody),
		MessageId:     aws.String("message-id"),
		ReceiptHandle: aws.String("receipt-handle"),
	}

	watcher.process(ctx, message)

	eventChan := mockWatcher.EventsChan()
	select {
	case event := <-eventChan:
		assert.Equal(t, "value", event.Records[0].EventName)
	case <-time.After(1 * time.Second):
		t.Error("expected event to be processed")
	}

	errorChan := mockWatcher.ErrorsChan()
	select {
	case err := <-errorChan:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deleting S3 notification")
	default:
		t.Error("expected error in ErrorsChan")
	}

	mockClient.AssertExpectations(t)
	mockWatcher.AssertExpectations(t)
}

func TestNewS3SqsWatcher_WithInterval(t *testing.T) {
	mockClient := &MockSqsClient{}
	interval := 20

	watcher := NewS3SqsWatcher(mockClient, "test-url", WithInterval(interval))

	typedWatcher, ok := watcher.(*sqsWatcher)
	assert.True(t, ok)
	assert.Equal(t, interval, typedWatcher.pollIntervalSeconds)
}

func TestS3NotificationPluginReloadResponder(t *testing.T) {
	type constructArgs struct {
		ctx       context.Context
		pluginRef string
		bucket    string
		dl        func() *loader.MockLoader
	}
	type callArgs struct {
		event *S3EventNotification
		err   error
	}
	tests := []struct {
		name          string
		constructArgs constructArgs
		callArgs      callArgs
		validate      func(t *testing.T, mockLoader *loader.MockLoader)
	}{
		{
			"should handle errors from listener",
			constructArgs{
				ctx:       t.Context(),
				pluginRef: "test_plugin",
				bucket:    "some_bucket",
				dl: func() *loader.MockLoader {
					return &loader.MockLoader{}
				},
			},
			callArgs{
				event: nil,
				err:   errors.New("something went wrong"),
			},
			func(t *testing.T, mockLoader *loader.MockLoader) {
				mockLoader.AssertNotCalled(t, "Get", mock.Anything)
			},
		}, {
			"should handle no records",
			constructArgs{
				ctx:       t.Context(),
				pluginRef: "test_plugin",
				bucket:    "some_bucket",
				dl: func() *loader.MockLoader {
					ldr := &loader.MockLoader{}

					return ldr
				},
			},
			callArgs{
				event: &S3EventNotification{
					Records: make([]S3EventRecord, 0),
				},
				err: nil,
			},
			func(t *testing.T, mockLoader *loader.MockLoader) {
				mockLoader.AssertNotCalled(t, "Get", mock.Anything)
			},
		}, {
			"should not respond to events for other buckets",
			constructArgs{
				ctx:       t.Context(),
				pluginRef: "test_plugin",
				bucket:    "some_bucket",
				dl: func() *loader.MockLoader {
					ldr := &loader.MockLoader{}

					return ldr
				},
			},
			callArgs{
				event: &S3EventNotification{
					Records: []S3EventRecord{
						{
							S3: S3Entity{
								Bucket: S3BucketEntity{
									Name: "some_other_bucket",
								},
							},
						},
					},
				},
				err: nil,
			},
			func(t *testing.T, mockLoader *loader.MockLoader) {
				mockLoader.AssertNotCalled(t, "Get", mock.Anything)
			},
		}, {
			"should not respond to events for other objects",
			constructArgs{
				ctx:       t.Context(),
				pluginRef: "test_plugin",
				bucket:    "some_bucket",
				dl: func() *loader.MockLoader {
					ldr := &loader.MockLoader{}

					return ldr
				},
			},
			callArgs{
				event: &S3EventNotification{
					Records: []S3EventRecord{
						{
							S3: S3Entity{
								Bucket: S3BucketEntity{
									Name: "some_bucket",
								},
								Object: S3ObjectEntity{
									Key: "some_other_object",
								},
							},
						},
					},
				},
				err: nil,
			},
			func(t *testing.T, mockLoader *loader.MockLoader) {
				mockLoader.AssertNotCalled(t, "Get", mock.Anything)
			},
		}, {
			"should handle errors to reload the module",
			constructArgs{
				ctx:       t.Context(),
				pluginRef: "test_plugin",
				bucket:    "some_bucket",
				dl: func() *loader.MockLoader {
					ldr := &loader.MockLoader{}
					ldr.On("Reload", mock.Anything).Return(errors.New("something bad happened"))
					return ldr
				},
			},
			callArgs{
				event: &S3EventNotification{
					Records: []S3EventRecord{
						{
							EventName: "ObjectCreated:Put",
							S3: S3Entity{
								Bucket: S3BucketEntity{
									Name: "some_bucket",
								},
								Object: S3ObjectEntity{
									Key: "test_plugin",
								},
							},
						},
					},
				},
				err: nil,
			},
			func(t *testing.T, mockLoader *loader.MockLoader) {
				mockLoader.AssertCalled(t, "Reload", mock.Anything)
			},
		}, {
			"should handle successful calls to reload the module",
			constructArgs{
				ctx:       t.Context(),
				pluginRef: "test_plugin",
				bucket:    "some_bucket",
				dl: func() *loader.MockLoader {
					ldr := &loader.MockLoader{}
					ldr.On("Reload", mock.Anything).Return(nil)
					return ldr
				},
			},
			callArgs{
				event: &S3EventNotification{
					Records: []S3EventRecord{
						{
							EventName: "ObjectCreated:Put",
							S3: S3Entity{
								Bucket: S3BucketEntity{
									Name: "some_bucket",
								},
								Object: S3ObjectEntity{
									Key: "test_plugin",
								},
							},
						},
					},
				},
				err: nil,
			},
			func(t *testing.T, mockLoader *loader.MockLoader) {
				mockLoader.AssertCalled(t, "Reload", mock.Anything)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ldr := tt.constructArgs.dl()
			responder := S3NotificationPluginReloadResponder(tt.constructArgs.ctx, tt.constructArgs.pluginRef, tt.constructArgs.bucket, ldr)
			responder(tt.callArgs.event, tt.callArgs.err)
			tt.validate(t, ldr)
		})
	}
}
