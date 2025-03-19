package detection

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"log"

	"time"
)

type S3EventNotification struct {
	Records []S3EventRecord `json:"Records"`
}

type S3EventRecord struct {
	EventVersion string    `json:"eventVersion"`
	EventSource  string    `json:"eventSource"`
	AWSRegion    string    `json:"awsRegion"`
	EventTime    time.Time `json:"eventTime"`
	EventName    string    `json:"eventName"`
	S3           S3Entity  `json:"s3"`
}

type S3Entity struct {
	Bucket S3BucketEntity `json:"bucket"`
	Object S3ObjectEntity `json:"object"`
}

type S3BucketEntity struct {
	Name string `json:"name"`
	Arn  string `json:"arn"`
}

type S3ObjectEntity struct {
	Key       string `json:"key"`
	Size      int64  `json:"size"`
	ETag      string `json:"eTag"`
	VersionId string `json:"versionId,omitempty"`
}

type SqsClient interface {
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

// TODO : Finish testing this
func NewS3SqsWatcher(client SqsClient, intervalSeconds int, url string) Watcher {
	errChan := make(chan error)
	eventChan := make(chan S3EventNotification)
	fmt.Println("S3 notification listener started...")

	go func() {
		for {
			msgResult, err := client.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(url),
				MaxNumberOfMessages: 10,
				WaitTimeSeconds:     int32(intervalSeconds),
			})
			if err != nil {
				log.Printf("Failed to receive message from queue: %s", err)
				time.Sleep(time.Duration(intervalSeconds) * time.Second)
				continue
			}

			if len(msgResult.Messages) == 0 {
				time.Sleep(time.Duration(intervalSeconds) * time.Second)
				continue
			}

			for _, message := range msgResult.Messages {
				if message.Body != nil {
					notification := S3EventNotification{}
					err := json.Unmarshal([]byte(*message.Body), &notification)
					if err != nil {
						log.Printf("Error unmarshalling S3 notification: %s\n", err)
					}

					eventChan <- notification

					_, err = client.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
						QueueUrl:      aws.String(url),
						ReceiptHandle: message.ReceiptHandle,
					})
					if err != nil {
						log.Printf("Error deleting S3 notification: %s\n", err)
					} else {
						fmt.Printf("S3 notification deleted: %s\n", *message.MessageId)
					}
				} else {
					fmt.Printf("Received empty S3 message: %s\n", *message.MessageId)
				}
			}

			time.Sleep(time.Duration(intervalSeconds) * time.Second)
		}
	}()

	return NewWatcher(eventChan, errChan)
}
