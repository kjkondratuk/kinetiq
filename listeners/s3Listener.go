package listeners

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"log"

	"time"
)

const (
	queueURL = "https://sqs.us-east-1.amazonaws.com/916325820950/kinetiq-updates-sqs"
	region   = "us-east-1"
)

func processMessage(message *sqsTypes.Message) {
	fmt.Printf("Received message: %v \n", message.Body)
	time.Sleep(5 * time.Second)
	fmt.Println("Processing complete")
}

func ListenForUpdates() error {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	sqsClient := sqs.NewFromConfig(cfg)
	fmt.Println("Listening for updates from SQS...")

	for {
		msgResult, err := sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     20,
		})
		if err != nil {
			log.Printf("Failed to receive message from queue: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		if len(msgResult.Messages) == 0 {
			fmt.Println("No messages found. Sleeping for 20 seconds...")
			time.Sleep(20 * time.Second)
			continue
		}

		message := msgResult.Messages[0]
		processMessage(&message)

		_, err = sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(queueURL),
			ReceiptHandle: message.ReceiptHandle,
		})
		if err != nil {
			log.Printf("Error deleting message: %v\n", err)
		} else {
			fmt.Printf("Message deleted: %s\n", *message.MessageId)
		}

		fmt.Println("Waiting 20 seconds before checking for new messages...")
		time.Sleep(60 * time.Second)
	}

	return nil
}
