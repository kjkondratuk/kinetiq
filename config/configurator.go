package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

var (
	DefaultConfigurator = configurator{}
)

type configurator struct{}

type Configurator interface {
	Configure() error
}

type Config struct {
	S3        S3Config
	PluginRef string
	Kafka     KafkaConfig
}

type S3Config struct {
	Bucket       string
	Enabled      bool
	ChangeQueue  string
	PollInterval int
}

type KafkaConfig struct {
	SourceBrokers []string
	SourceTopic   string
	DestBrokers   []string
	DestTopic     string
}

func (c configurator) Configure() (Config, error) {
	var s3Enabled bool
	s := os.Getenv("S3_INTEGRATION_ENABLED")
	if s != "" {
		s3Enabled = true
	}

	sb := os.Getenv("S3_INTEGRATION_BUCKET")
	if s3Enabled && sb == "" {
		return Config{}, errors.New("S3_INTEGRATION_BUCKET must be set when S3_INTEGRATION_ENABLED is set")
	}

	cq := os.Getenv("S3_INTEGRATION_CHANGE_QUEUE")
	if s3Enabled && cq == "" {
		return Config{}, errors.New("S3_INTEGRATION_CHANGE_QUEUE must be set when S3_INTEGRATION_ENABLED is set")
	}

	pollingInterval := 10
	pollingIntervalStr := os.Getenv("S3_INTEGRATION_POLL_INTERVAL")
	if s3Enabled && pollingIntervalStr != "" {
		pollingInterval, _ = strconv.Atoi(pollingIntervalStr)
	}

	pluginRef := os.Getenv("PLUGIN_REF")
	if pluginRef == "" {
		return Config{}, errors.New("PLUGIN_REF must be set")
	}

	sourceBrokers := os.Getenv("KAFKA_SOURCE_BROKERS")
	if sourceBrokers == "" {
		sourceBrokers = "localhost:49092"
	}

	sourceTopic := os.Getenv("KAFKA_SOURCE_TOPIC")
	if sourceTopic == "" {
		return Config{}, errors.New("KAFKA_SOURCE_TOPIC must be set")
	}

	destBrokers := os.Getenv("KAFKA_DEST_BROKERS")
	if destBrokers == "" {
		destBrokers = sourceBrokers
	}

	destTopic := os.Getenv("KAFKA_DEST_TOPIC")
	if sourceBrokers == destBrokers && destTopic == sourceTopic {
		return Config{}, errors.New("KAFKA_DEST_TOPIC must be different from KAFKA_SOURCE_TOPIC when source and destination kafka sBrokers are the same")
	}

	return Config{
		S3: S3Config{
			Bucket:       sb,
			Enabled:      s3Enabled,
			ChangeQueue:  cq,
			PollInterval: pollingInterval,
		},
		PluginRef: pluginRef,
		Kafka: KafkaConfig{
			SourceBrokers: strings.Split(sourceBrokers, ","),
			SourceTopic:   sourceTopic,
			DestBrokers:   strings.Split(destBrokers, ","),
			DestTopic:     destTopic,
		},
	}, nil
}
