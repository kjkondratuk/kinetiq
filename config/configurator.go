package config

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/twmb/franz-go/pkg/kgo"
	kaws "github.com/twmb/franz-go/pkg/sasl/aws"
	"github.com/twmb/franz-go/pkg/sasl/oauth"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
	"log"
	"log/slog"
	"os"
	"slices"
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
	Aws       *aws.Config
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
	Topics []string
	Group  string
	Offset string
}

type SharedKafkaConfig struct {
	SASLMechanism   string
	SASLUser        string
	SASLPassword    string
	SASLToken       string
	SASLTokenAuth   bool
	OAuthExtensions map[string]string
	SASLZid         string
	DialTimeoutMs   int
	TLSEnabled      bool
}

var basicMskMechanism = func(conf *aws.Config) func(ctx context.Context) (kaws.Auth, error) {
	return func(ctx context.Context) (kaws.Auth, error) {
		cred, err := conf.Credentials.Retrieve(ctx)
		if err != nil {
			return kaws.Auth{}, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
		}

		return kaws.Auth{
			AccessKey:    cred.AccessKeyID,
			SecretKey:    cred.SecretAccessKey,
			SessionToken: cred.SessionToken,
		}, nil
	}
}

func (c configurator) Configure(ctx context.Context) (Config, error) {
	s3Enabled := getEnvOrDefault("S3_INTEGRATION_ENABLED", false)

	sb, err := getOrErrorOnValueAndCondition("S3_INTEGRATION_BUCKET", "", s3Enabled,
		errors.New("S3_INTEGRATION_BUCKET must be set when S3_INTEGRATION_ENABLED is set"))
	if err != nil {
		return Config{}, err
	}

	cq, err := getOrErrorOnValueAndCondition("S3_INTEGRATION_CHANGE_QUEUE", "", s3Enabled,
		errors.New("S3_INTEGRATION_CHANGE_QUEUE must be set when S3_INTEGRATION_ENABLED is set"))
	if err != nil {
		return Config{}, err
	}

	pollingInterval := getEnvOrDefault("S3_INTEGRATION_POLL_INTERVAL", 10)
	pluginRef, err := require("PLUGIN_REF", errors.New("PLUGIN_REF must be set"))
	if err != nil {
		return Config{}, err
	}

	sourceBrokers := getEnvOrDefault("KAFKA_SOURCE_BROKERS", "localhost:49092")
	sourceTopic, err := require("KAFKA_SOURCE_TOPIC", errors.New("KAFKA_SOURCE_TOPIC must be set"))
	if err != nil {
		return Config{}, err
	}

	destBrokers := getEnvOrDefault("KAFKA_DEST_BROKERS", sourceBrokers)
	destTopic, err := getOrErrorOnValueAndCondition("KAFKA_DEST_TOPIC", sourceTopic, sourceBrokers == destBrokers,
		errors.New("KAFKA_DEST_TOPIC must be different from KAFKA_SOURCE_TOPIC and must be set when "+
			"KAFKA_SOURCE_BROKERS and KAFKA_DEST_BROKERS are the same"))
	if err != nil {
		return Config{}, err
	}

	// setup producer
	producerCompression := os.Getenv("KAFKA_PRODUCER_COMPRESSION")
	producerPartitioner := os.Getenv("KAFKA_PRODUCER_PARTITIONER")
	producerDialTimeoutMs := getEnvOrDefault("KAFKA_PRODUCER_DIAL_TIMEOUT_MS", 0)
	producerTlsEnabled := truthy("KAFKA_PRODUCER_TLS_ENABLED")
	producerBatchMaxBytes := getEnvOrDefault("KAFKA_PRODUCER_BATCH_MAX_BYTES", 0)
	producerMaxBufferedRecords := getEnvOrDefault("KAFKA_PRODUCER_MAX_BUFFERED_RECORDS", 0)
	producerMaxBufferedBytes := getEnvOrDefault("KAFKA_PRODUCER_MAX_BUFFERED_BYTES", 0)
	producerRecordRetries := getEnvOrDefault("KAFKA_PRODUCER_RECORD_RETRIES", 0)
	producerTimeoutMs := getEnvOrDefault("KAFKA_PRODUCER_TIMEOUT_MS", 0)
	producerLingerMs := getEnvOrDefault("KAFKA_PRODUCER_LINGER_MS", 0)
	producerRequredAcks := os.Getenv("KAFKA_PRODUCER_REQUIRED_ACKS")
	producerSaslMechanism := os.Getenv("KAFKA_PRODUCER_SASL_MECHANISM")
	producerOAuthExtensions := parseMap("KAFKA_PRODUCER_OAUTH_EXTENSIONS")

	// setup consumer
	consumerDialTimeoutMs := getEnvOrDefault("KAFKA_CONSUMER_DIAL_TIMEOUT_MS", 0)
	consumerTlsEnabled := truthy("KAFKA_CONSUMER_TLS_ENABLED")
	consumerTopics := getList("KAFKA_CONSUMER_TOPICS")
	consumerGroup := getEnvOrDefault("KAFKA_CONSUMER_GROUP", "")
	consumerOffset := getEnvOrDefault("KAFKA_CONSUMER_OFFSET", "")
	consumerSaslMechanism := os.Getenv("KAFKA_CONSUMER_SASL_MECHANISM")
	consumerOAuthExtensions := parseMap("KAFKA_CONSUMER_OAUTH_EXTENSIONS")

	cfg := Config{
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
				SharedKafkaConfig: SharedKafkaConfig{
					SASLMechanism:   producerSaslMechanism,
					SASLUser:        os.Getenv("KAFKA_PRODUCER_SASL_USER"),
					SASLPassword:    os.Getenv("KAFKA_PRODUCER_SASL_PASSWORD"),
					SASLToken:       os.Getenv("KAFKA_PRODUCER_SASL_TOKEN"),
					SASLZid:         os.Getenv("KAFKA_PRODUCER_SASL_ZID"),
					SASLTokenAuth:   truthy("KAFKA_PRODUCER_SASL_TOKEN_AUTH"),
					OAuthExtensions: producerOAuthExtensions,
					DialTimeoutMs:   producerDialTimeoutMs,
					TLSEnabled:      producerTlsEnabled,
				},
				Partitioner:        producerPartitioner,
				Compression:        producerCompression,
				BatchMaxBytes:      producerBatchMaxBytes,
				MaxBufferedRecords: producerMaxBufferedRecords,
				MaxBufferedBytes:   producerMaxBufferedBytes,
				RecordRetries:      producerRecordRetries,
				TimeoutMs:          producerTimeoutMs,
				LingerMs:           producerLingerMs,
				RequiredAcks:       producerRequredAcks,
			},
			Consumer: ConsumerConfig{
				SharedKafkaConfig: SharedKafkaConfig{
					SASLMechanism:   consumerSaslMechanism,
					SASLUser:        os.Getenv("KAFKA_CONSUMER_SASL_USER"),
					SASLPassword:    os.Getenv("KAFKA_CONSUMER_SASL_PASSWORD"),
					SASLToken:       os.Getenv("KAFKA_CONSUMER_SASL_TOKEN"),
					SASLZid:         os.Getenv("KAFKA_CONSUMER_SASL_ZID"),
					SASLTokenAuth:   truthy("KAFKA_CONSUMER_SASL_TOKEN_AUTH"),
					OAuthExtensions: consumerOAuthExtensions,
					DialTimeoutMs:   consumerDialTimeoutMs,
					TLSEnabled:      consumerTlsEnabled,
				},
				Topics: consumerTopics,
				Group:  consumerGroup,
				Offset: consumerOffset,
			},
		},
	}

	// configure default AWS configuration for anything that requires it
	if s3Enabled || producerSaslMechanism == "aws-iam" || consumerSaslMechanism == "aws-iam" {
		awsCfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			slog.Error("failed to load AWS config", slog.String("err", err.Error()))
			os.Exit(1)
		}
		cfg.Aws = &awsCfg
	}

	return cfg, nil
}

func (c configurator) CreateKafkaClientOptions(conf SharedKafkaConfig, name string, awsConf *aws.Config) []kgo.Opt {
	opts := []kgo.Opt{}

	// validate SASL opts
	if conf.SASLMechanism != "" {
		saslTypes := []string{"plain", "sasl-scram-512", "sasl-scram-256"}
		if slices.Contains(saslTypes, conf.SASLMechanism) &&
			conf.SASLUser == "" {
			slog.Error(fmt.Sprintf("%s_SASL_MECHANISM %s requires %s_SASL_USER to be set", name, conf.SASLMechanism, name))
			os.Exit(1)
		}

		if slices.Contains(saslTypes, conf.SASLMechanism) && conf.SASLPassword == "" {
			slog.Error(fmt.Sprintf("%s_SASL_MECHANISM %s requires %s_SASL_PASSWORD to be set", name, conf.SASLMechanism, name))
			os.Exit(1)
		}

		if conf.SASLMechanism == "oauth" && conf.SASLToken == "" {
			slog.Error(fmt.Sprintf("%s_SASL_MECHANISM %s requires %s_SASL_TOKEN to be set", name, conf.SASLMechanism, name))
			os.Exit(1)
		}

		if conf.SASLMechanism == "aws-iam" && awsConf == nil {
			slog.Error(fmt.Sprintf("%s_SASL_MECHANISM %s requires a valid AWS config", name, conf.SASLMechanism))
			os.Exit(1)
		}
	}

	// select and configure authentication mechanism based on
	switch conf.SASLMechanism {
	case "plain":
		opts = append(opts, kgo.SASL(plain.Auth{
			Zid:  conf.SASLZid,
			User: conf.SASLUser,
			Pass: conf.SASLPassword,
		}.AsMechanism()))
	case "sasl-scram-512":
		opts = append(opts, kgo.SASL(scram.Auth{
			Zid:     conf.SASLZid,
			User:    conf.SASLUser,
			Pass:    conf.SASLPassword,
			IsToken: conf.SASLTokenAuth,
		}.AsSha512Mechanism()))
	case "sasl-scram-256":
		opts = append(opts, kgo.SASL(scram.Auth{
			Zid:     conf.SASLZid,
			User:    conf.SASLUser,
			Pass:    conf.SASLPassword,
			IsToken: conf.SASLTokenAuth,
		}.AsSha256Mechanism()))
	case "oauth":
		opts = append(opts, kgo.SASL(oauth.Auth{
			Zid:        conf.SASLZid,
			Token:      conf.SASLToken,
			Extensions: conf.OAuthExtensions,
		}.AsMechanism()))
	case "aws-iam":
		opts = append(opts, kgo.SASL(kaws.ManagedStreamingIAM(basicMskMechanism(awsConf))))
	}

	if conf.TLSEnabled {
		opts = append(opts, kgo.DialTLS())
	}

	if conf.DialTimeoutMs != 0 {
		opts = append(opts, kgo.DialTimeout(time.Duration(conf.DialTimeoutMs)*time.Millisecond))
	}

	// TODO : allow configuration of these values
	//kgo.BrokerMaxReadBytes(),
	//kgo.MetadataMaxAge(),
	//kgo.AllowAutoTopicCreation(),
	//kgo.ClientID(),
	//kgo.ConnIdleTimeout(),
	//kgo.DialTLSConfig(),
	//kgo.MetadataMinAge(),
	//kgo.RequestRetries(),
	//kgo.RequestTimeoutOverhead(),
	//kgo.RetryTimeout(),
	//kgo.SoftwareNameAndVersion(),
	//kgo.WithHooks(),

	return opts
}

func (c configurator) ProducerConfig(conf Config) []kgo.Opt {
	producerOpts := []kgo.Opt{
		kgo.DefaultProduceTopic(conf.Kafka.DestTopic),
		kgo.SeedBrokers(conf.Kafka.DestBrokers...),
	}

	shared := c.CreateKafkaClientOptions(conf.Kafka.Producer.SharedKafkaConfig, "PRODUCER", conf.Aws)
	producerOpts = append(producerOpts, shared...)

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
		case "sticky-key":
			part = kgo.StickyKeyPartitioner(nil)
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
		kgo.SeedBrokers(conf.Kafka.SourceBrokers...),
	}

	shared := c.CreateKafkaClientOptions(conf.Kafka.Consumer.SharedKafkaConfig, "CONSUMER", conf.Aws)
	consumerOpts = append(consumerOpts, shared...)

	if len(conf.Kafka.Consumer.Topics) > 0 {
		topics := append(conf.Kafka.Consumer.Topics, conf.Kafka.SourceTopic)
		consumerOpts = append(consumerOpts, kgo.ConsumeTopics(topics...))
	} else {
		consumerOpts = append(consumerOpts, kgo.ConsumeTopics(conf.Kafka.SourceTopic))
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

func getEnvOrDefault[T any](key string, defaultValue T) T {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}

	var zero T
	switch any(zero).(type) {
	case int:
		if v, err := strconv.Atoi(valStr); err == nil {
			return any(v).(T)
		}
	case int64:
		if v, err := strconv.ParseInt(valStr, 10, 64); err == nil {
			return any(v).(T)
		}
	case float64:
		if v, err := strconv.ParseFloat(valStr, 64); err == nil {
			return any(v).(T)
		}
	case bool:
		if v, err := strconv.ParseBool(valStr); err == nil {
			return any(v).(T)
		}
	case string:
		return any(valStr).(T)
	}

	// fallback: return default if parsing fails or type is unsupported
	return defaultValue
}

func getOrErrorOnValueAndCondition(key string, value string, condition bool, err error) (string, error) {
	v := os.Getenv(key)
	if condition && v == value {
		return "", err
	}
	return v, nil
}

func require(key string, err error) (string, error) {
	k := os.Getenv(key)
	if k == "" {
		return "", err
	}
	return k, nil
}

func truthy(key string) bool {
	var r bool
	v := os.Getenv(key)
	if v != "" {
		r = true
	}
	return r
}

func getList(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return []string{}
	}
	return strings.Split(v, ",")
}

func parseMap(key string) map[string]string {
	result := make(map[string]string)
	if envValue := os.Getenv(key); envValue != "" {
		pairs := strings.Split(envValue, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				k := strings.TrimSpace(kv[0])
				v := strings.TrimSpace(kv[1])
				if k != "" {
					result[k] = v
				}
			} else {
				slog.Error("Invalid map value specified for map parameter", slog.String("key", key), slog.String("value", fmt.Sprintf("%+v", pair)))
			}
		}
	}
	return result
}
