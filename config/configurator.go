package config

import (
	"errors"
	"github.com/twmb/franz-go/pkg/kgo"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
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
	Producer      ProducerConfig
	Consumer      ConsumerConfig
}

type ProducerConfig struct {
	SharedKafkaConfig
	Partitioner        string
	Compression        string
	BatchMaxBytes      int
	MaxBufferedRecords int
	MaxBufferedBytes   int
	RecordRetries      int
	TimeoutMs          int
	LingerMs           int
	RequiredAcks       string
}

type ConsumerConfig struct {
	SharedKafkaConfig
	Topics string
	Group  string
	Offset string
}

type SharedKafkaConfig struct {
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

	compression := os.Getenv("KAFKA_PRODUCER_COMPRESSION")

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
			Producer: ProducerConfig{
				Compression: compression,
			},
			Consumer: ConsumerConfig{},
		},
	}, nil
}

func (c configurator) SharedKafkaConfig(conf Config) []kgo.Opt {
	opts := []kgo.Opt{}

	// TODO : allow configuration of these values
	//kgo.BrokerMaxReadBytes(),
	//kgo.MetadataMaxAge(),
	//kgo.AllowAutoTopicCreation(),
	//kgo.ClientID(),
	//kgo.DialTimeout(),
	//kgo.ConnIdleTimeout(),
	//kgo.DialTLS(),
	//kgo.DialTLSConfig(),
	//kgo.MetadataMinAge(),
	//kgo.RequestRetries(),
	//kgo.RequestTimeoutOverhead(),
	//kgo.RetryTimeout(),
	//kgo.SASL(),
	//kgo.SoftwareNameAndVersion(),
	//kgo.WithHooks(),

	return opts
}

func (c configurator) ProducerConfig(conf Config) []kgo.Opt {
	producerOpts := []kgo.Opt{
		kgo.DefaultProduceTopic(conf.Kafka.DestTopic),
		kgo.SeedBrokers(conf.Kafka.DestBrokers...),
	}

	// TODO : figure out how to test this or refactor since the configuration isn't exposed for validation
	// Setup compression, if configured
	if conf.Kafka.Producer.Compression != "" {
		var comp kgo.CompressionCodec
		switch conf.Kafka.Producer.Compression {
		case "gzip":
			comp = kgo.GzipCompression()
		case "lz4":
			comp = kgo.Lz4Compression()
		case "zstd":
			comp = kgo.ZstdCompression()
		case "none":
			comp = kgo.NoCompression()
		case "snappy":
			comp = kgo.SnappyCompression()
		default:
			log.Printf("Unsupported compression: %s - defaulting to snappy\n", conf.Kafka.Producer.Compression)
			comp = kgo.SnappyCompression()
		}
		producerOpts = append(producerOpts, kgo.ProducerBatchCompression(comp))
	}

	// Setup partitioner, if configured
	if conf.Kafka.Producer.Partitioner != "" {
		var part kgo.Partitioner
		switch conf.Kafka.Producer.Partitioner {
		case "round-robin":
			part = kgo.RoundRobinPartitioner()
		case "manual":
			part = kgo.ManualPartitioner()
		case "sticky":
			part = kgo.StickyPartitioner()
		case "least-backup":
			part = kgo.LeastBackupPartitioner()
		default:
			log.Printf("Unsupported partitioner: %s - defaulting to Kafka native key partitioning", conf.Kafka.Producer.Partitioner)
			part = kgo.StickyKeyPartitioner(nil)
		}
		producerOpts = append(producerOpts, kgo.RecordPartitioner(part))
	}

	// setup acks, if configured
	if conf.Kafka.Producer.RequiredAcks != "" {
		var acks kgo.Acks
		switch conf.Kafka.Producer.RequiredAcks {
		case "none":
			acks = kgo.NoAck()
		case "leader":
			acks = kgo.LeaderAck()
		case "all":
			acks = kgo.AllISRAcks()
		default:
			log.Printf("Unsupported RequiredAcks: %s - defaulting to all", conf.Kafka.Producer.RequiredAcks)
			acks = kgo.AllISRAcks()

		}
		producerOpts = append(producerOpts, kgo.RequiredAcks(acks))
	}

	if conf.Kafka.Producer.BatchMaxBytes != 0 {
		producerOpts = append(producerOpts, kgo.ProducerBatchMaxBytes(int32(conf.Kafka.Producer.BatchMaxBytes)))
	}

	if conf.Kafka.Producer.MaxBufferedRecords != 0 {
		producerOpts = append(producerOpts, kgo.MaxBufferedRecords(int(conf.Kafka.Producer.MaxBufferedRecords)))
	}

	if conf.Kafka.Producer.MaxBufferedBytes != 0 {
		producerOpts = append(producerOpts, kgo.MaxBufferedBytes(int(conf.Kafka.Producer.MaxBufferedBytes)))
	}

	if conf.Kafka.Producer.TimeoutMs != 0 {
		producerOpts = append(producerOpts, kgo.ProduceRequestTimeout(time.Duration(conf.Kafka.Producer.TimeoutMs)*time.Millisecond))
	}

	if conf.Kafka.Producer.RecordRetries != 0 {
		producerOpts = append(producerOpts, kgo.RecordRetries(conf.Kafka.Producer.RecordRetries))
	}

	if conf.Kafka.Producer.LingerMs != 0 {
		producerOpts = append(producerOpts, kgo.ProducerLinger(time.Duration(conf.Kafka.Producer.LingerMs)*time.Millisecond))
	}

	return producerOpts
}

func (c configurator) ConsumerConfig(conf Config) []kgo.Opt {
	consumerOpts := []kgo.Opt{
		kgo.ConsumeTopics(conf.Kafka.SourceTopic),
		kgo.SeedBrokers(conf.Kafka.SourceBrokers...),
	}

	if conf.Kafka.Consumer.Topics != "" {
		topics := strings.Split(conf.Kafka.Consumer.Topics, ",")
		consumerOpts = append(consumerOpts, kgo.ConsumeTopics(topics...))
	}

	if conf.Kafka.Consumer.Group != "" {
		consumerOpts = append(consumerOpts, kgo.ConsumerGroup(conf.Kafka.Consumer.Group))
	}

	if conf.Kafka.Consumer.Offset != "" {
		var offset kgo.Offset

		switch conf.Kafka.Consumer.Offset {
		case "start":
			offset = kgo.NewOffset().AtStart()
		case "end":
			offset = kgo.NewOffset().AtEnd()
		default:
			// if the value is numeric, treat it as an integer offset
			if v, err := strconv.Atoi(conf.Kafka.Consumer.Offset); err == nil {
				offset = kgo.NewOffset().At(int64(v))
			} else {
				offset = kgo.NewOffset().AtEnd()
			}
		}
		consumerOpts = append(consumerOpts, kgo.ConsumeResetOffset(offset))
	}

	return consumerOpts
}
