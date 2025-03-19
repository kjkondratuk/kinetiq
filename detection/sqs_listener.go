package detection

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"log"

	"time"
)

type SqsClient interface {
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

type sqsWatcher struct {
	watcher             Watcher[S3EventNotification]
	listener            Listener[S3EventNotification]
	client              SqsClient
	url                 string
	pollIntervalSeconds int
}

type SqsWatcher interface {
	Listener[S3EventNotification]
	Watcher[S3EventNotification]
	StartEvents(ctx context.Context)
}

var _ Listener[S3EventNotification] = &sqsWatcher{}

type MessageProcessor func(ctx context.Context, message types.Message)

type SqsWatcherOpt func(w *sqsWatcher)

func WithInterval(intervalSeconds int) SqsWatcherOpt {
	return func(w *sqsWatcher) {
		w.pollIntervalSeconds = intervalSeconds
	}
}

// TODO : Finish testing this
func NewS3SqsWatcher(client SqsClient, url string, opts ...SqsWatcherOpt) SqsWatcher {
	errChan := make(chan error)
	eventChan := make(chan S3EventNotification)
	w := NewWatcher[S3EventNotification](eventChan, errChan)
	watch := &sqsWatcher{
		client:              client,
		watcher:             w,
		listener:            NewListener[S3EventNotification](w),
		url:                 url,
		pollIntervalSeconds: 10,
	}

	for _, opt := range opts {
		opt(watch)
	}

	return watch
}

// StartEvents : starts listening for S3 change event notifications on the specified SQS topic, publishing
// events to the listener event channel when one is received
func (s *sqsWatcher) StartEvents(ctx context.Context) {
	fmt.Println("S3 notification listener started...")

	go func() {
		for {
			msgResult, err := s.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(s.url),
				MaxNumberOfMessages: 10,
				WaitTimeSeconds:     int32(s.pollIntervalSeconds),
			})
			if err != nil {
				log.Printf("Failed to receive message from queue: %s", err)
				time.Sleep(time.Duration(s.pollIntervalSeconds) * time.Second)
				continue
			}

			if len(msgResult.Messages) == 0 {
				time.Sleep(time.Duration(s.pollIntervalSeconds) * time.Second)
				continue
			}

			for _, message := range msgResult.Messages {
				s.process(ctx, message)
			}

			time.Sleep(time.Duration(s.pollIntervalSeconds) * time.Second)
		}
	}()
}

func (s *sqsWatcher) process(ctx context.Context, message types.Message) {
	if message.Body != nil {
		notification := S3EventNotification{}
		err := json.Unmarshal([]byte(*message.Body), &notification)
		if err != nil {
			s.watcher.ErrorsChan() <- fmt.Errorf("error unmarshalling S3 notification: %w", err)
			return
		}

		s.watcher.EventsChan() <- notification

		// not waiting for deletion on a success response here because if we can't process it, do we really
		// care to keep the message?
		_, err = s.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(s.url),
			ReceiptHandle: message.ReceiptHandle,
		})
		if err != nil {
			s.watcher.ErrorsChan() <- fmt.Errorf("error deleting S3 notification: %w", err)
			return
		}

		fmt.Printf("S3 notification deleted: %s\n", *message.MessageId)
		return
	}

	fmt.Printf("Received empty S3 message: %s\n", *message.MessageId)
}

// Listen : delegates to the underlying listener implementation
func (s *sqsWatcher) Listen(responder Responder[S3EventNotification]) {
	s.listener.Listen(responder)
}

// EventsChan : delegates to the underlying event channel implementation
func (s *sqsWatcher) EventsChan() chan S3EventNotification {
	return s.watcher.EventsChan()
}

// ErrorsChan : delegates to the underlying error channel implementation
func (s *sqsWatcher) ErrorsChan() chan error {
	return s.watcher.ErrorsChan()
}
