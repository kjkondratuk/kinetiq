package s3

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqs_types "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"log"

	"time"
)

type S3ChangeEvent struct {
	Records []S3ChangeEventRecord `json:"Records"`
}

type S3ChangeEventRecord struct {
	EventName string `json:"eventName"`
}

type s3SqsListener struct {
	queueURL        string
	objectUri       string
	intervalSeconds time.Duration
	client          *sqs.Client
}

type S3SqsListener interface {
	Listen(responder S3SqsListenerResponder)
}

type S3SqsListenerResponder func(message *sqs_types.Message) error

func NewS3SqsListener(client *sqs.Client, intervalSeconds int, url string, objectUri string) *s3SqsListener {
	return &s3SqsListener{
		client:          client,
		intervalSeconds: time.Duration(intervalSeconds),
		objectUri:       objectUri,
		queueURL:        url,
	}
}

func (l *s3SqsListener) Listen(responder S3SqsListenerResponder) {

	fmt.Println("Listening for updates from SQS...")

	for {
		msgResult, err := l.client.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(l.queueURL),
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     int32(l.intervalSeconds),
		})
		if err != nil {
			log.Printf("Failed to receive message from queue: %s", err)
			time.Sleep(l.intervalSeconds * time.Second)
			continue
		}

		if len(msgResult.Messages) == 0 {
			time.Sleep(10 * time.Second)
			continue
		}

		for _, message := range msgResult.Messages {
			err = responder(&message)
			if err != nil {
				log.Printf("Error responding to hotswap notification: %v\n", err)
			}

			_, err = l.client.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(l.queueURL),
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				log.Printf("Error deleting hotswap notification: %v\n", err)
			} else {
				fmt.Printf("Message deleted: %s\n", *message.MessageId)
			}
		}

		time.Sleep(10 * time.Second)
	}
}
