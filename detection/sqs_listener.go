package detection

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/kjkondratuk/kinetiq/loader"
	"log"
	"log/slog"
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

var _ Listener[S3EventNotification] = &sqsWatcher{} // ensure sqsWatcher is an S3EventNotification listener
var _ Watcher[S3EventNotification] = &sqsWatcher{}  // ensure sqsWatcher is an S3EventNotification watcher

func S3NotificationPluginReloadResponder(ctx context.Context, pluginRef string, bucket string,
	dl loader.Loader) Responder[S3EventNotification] {

	return func(event *S3EventNotification, err error) {
		if err != nil {
			log.Printf("Failed to handle s3 watcher changes: %s", err)
			return
		}
		for _, record := range event.Records {
			if record.S3.Object.Key == pluginRef &&
				record.S3.Bucket.Name == bucket &&
				record.EventName == "ObjectCreated:Put" {

				// install new processor
				log.Printf("Loading new module: %s", record.S3.Object.ETag)
				err = dl.Reload(ctx)
				if err != nil {
					log.Printf("Failed to reload module: %s", err)
					return
				}
			}
		}
	}
}

type MessageProcessor func(ctx context.Context, message types.Message)

type SqsWatcherOpt func(w *sqsWatcher)

func WithInterval(intervalSeconds int) SqsWatcherOpt {
	return func(w *sqsWatcher) {
		w.pollIntervalSeconds = intervalSeconds
	}
}

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
	slog.Info("S3 notification listener started...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
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

		slog.Info("S3 notification deleted", slog.String("messageId", *message.MessageId))
		return
	}

	slog.Info("Received empty S3 message", slog.String("messageId", *message.MessageId))
}

// Listen : delegates to the underlying listener implementation
func (s *sqsWatcher) Listen(ctx context.Context, responder Responder[S3EventNotification]) {
	s.listener.Listen(ctx, responder)
}

// EventsChan : delegates to the underlying event channel implementation
func (s *sqsWatcher) EventsChan() chan S3EventNotification {
	return s.watcher.EventsChan()
}

// ErrorsChan : delegates to the underlying error channel implementation
func (s *sqsWatcher) ErrorsChan() chan error {
	return s.watcher.ErrorsChan()
}
