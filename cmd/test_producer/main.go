package main

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
	"log"
	"math/rand"
	"time"
)

type samplePayload struct {
	Key   string `json:"key"`
	Value int    `json:"value"`
}

func main() {
	writerClient, err := kgo.NewClient(
		kgo.DefaultProduceTopic("kinetiq-test-topic"),
		kgo.SeedBrokers("localhost:49092"),
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

		writerClient.Produce(context.Background(), &kgo.Record{
			Key:   []byte(key),
			Value: payloadBytes,
		}, nil)

		time.Sleep(time.Second * 5)
	}
}
