package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
	kaws "github.com/twmb/franz-go/pkg/sasl/aws"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

type samplePayload struct {
	Key   string `json:"key"`
	Value int    `json:"value"`
}

func main() {
	ctx := context.Background()
	brokerStr := os.Getenv("KAFKA_BROKERS")
	if brokerStr == "" {
		brokerStr = "localhost:49092"
	}
	brokers := strings.Split(brokerStr, ",")

	saslMechanism := os.Getenv("KAFKA_SASL_MECHANISM")

	opts := []kgo.Opt{
		kgo.DefaultProduceTopic("kinetiq-test-topic"),
		kgo.SeedBrokers(brokers...),
	}

	if saslMechanism != "" {
		switch saslMechanism {
		case "aws-iam":
			awsCfg, err := config.LoadDefaultConfig(ctx)
			if err != nil {
				log.Fatal("Failed to load AWS config", err)
			}
			opts = append(opts, kgo.DialTLS(), kgo.SASL(kaws.ManagedStreamingIAM(func(ctx context.Context) (kaws.Auth, error) {
				creds, err := awsCfg.Credentials.Retrieve(ctx)
				if err != nil {
					return kaws.Auth{}, err
				}
				return kaws.Auth{
					AccessKey:    creds.AccessKeyID,
					SecretKey:    creds.SecretAccessKey,
					SessionToken: creds.SessionToken,
				}, nil
			})))
		case "":
			break
		default:
			log.Fatalf("Unsupported SASL mechanism: %s", saslMechanism)
		}
	}

	writerClient, err := kgo.NewClient(
		opts...,
	)
	if err != nil {
		log.Fatal("Failed to create kafka writer client", err)
	}
	defer writerClient.Close()

	for {
		key := uuid.New().String()

		val := rand.Intn(100) // Generates a random number between 0 and 99
		payload := samplePayload{
			Key:   "abc123",
			Value: val,
		}

		payloadBytes, _ := json.Marshal(payload)

		writerClient.Produce(ctx, &kgo.Record{
			Key:   []byte(key),
			Value: payloadBytes,
		}, func(r *kgo.Record, err error) {
			if err != nil {
				log.Printf("Error producing to kafka: %s - %e", key, err)
			} else {
				log.Printf("Produced message: %s - %s", key, string(payloadBytes))
			}
		})

		time.Sleep(time.Millisecond * 300)
	}
}
