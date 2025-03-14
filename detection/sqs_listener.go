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

type s3SqsListener struct {
	queueURL        string
	objectKey       string
	intervalSeconds time.Duration
	client          *sqs.Client
}

func NewS3SqsListener(client *sqs.Client, intervalSeconds int, url string, objectKey string) *s3SqsListener {
	return &s3SqsListener{
		client:          client,
		intervalSeconds: time.Duration(intervalSeconds),
		objectKey:       objectKey,
		queueURL:        url,
	}
}

func (l *s3SqsListener) Listen(responder Responder[S3EventNotification]) {

	fmt.Println("S3 notification listener started...")

	for {
		msgResult, err := l.client.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(l.queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     int32(l.intervalSeconds),
		})
		if err != nil {
			log.Printf("Failed to receive message from queue: %s", err)
			time.Sleep(l.intervalSeconds * time.Second)
			continue
		}

		if len(msgResult.Messages) == 0 {
			time.Sleep(l.intervalSeconds * time.Second)
			continue
		}

		for _, message := range msgResult.Messages {
			if message.Body != nil {
				notification := S3EventNotification{}
				err := json.Unmarshal([]byte(*message.Body), &notification)
				if err != nil {
					log.Printf("Error unmarshalling S3 notification: %s\n", err)
				}

				responder(notification)

				_, err = l.client.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(l.queueURL),
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

		time.Sleep(l.intervalSeconds * time.Second)
	}
}

func (l *s3SqsListener) Close() {
	fmt.Println("S3 notification listener stopped...")
}
