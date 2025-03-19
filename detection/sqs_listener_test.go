package detection

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
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
