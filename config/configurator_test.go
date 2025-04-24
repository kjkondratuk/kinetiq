package config

import (
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/twmb/franz-go/pkg/kgo"
	"os"
	"reflect"
	"testing"
)

func TestConfigurator_Configure(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		wantErr   bool
		errPrefix string
	}{
		{
			name: "valid configuration with S3 integration enabled",
			env: map[string]string{
				"S3_INTEGRATION_ENABLED":       "true",
				"S3_INTEGRATION_BUCKET":        "bucket_name",
				"S3_INTEGRATION_CHANGE_QUEUE":  "change_queue",
				"S3_INTEGRATION_POLL_INTERVAL": "30",
				"PLUGIN_REF":                   "plugin",
				"KAFKA_SOURCE_BROKERS":         "source_broker",
				"KAFKA_SOURCE_TOPIC":           "source_topic",
				"KAFKA_DEST_TOPIC":             "dest_topic",
			},
			wantErr: false,
		},
		{
			name: "missing S3 bucket when S3 integration enabled",
			env: map[string]string{
				"S3_INTEGRATION_ENABLED":      "true",
				"S3_INTEGRATION_CHANGE_QUEUE": "change_queue",
				"PLUGIN_REF":                  "plugin",
				"KAFKA_SOURCE_BROKERS":        "source_broker",
				"KAFKA_SOURCE_TOPIC":          "source_topic",
				"KAFKA_DEST_TOPIC":            "dest_topic",
			},
			wantErr:   true,
			errPrefix: "S3_INTEGRATION_BUCKET must be set",
		},
		{
			name: "missing change queue when S3 integration enabled",
			env: map[string]string{
				"S3_INTEGRATION_ENABLED": "true",
				"S3_INTEGRATION_BUCKET":  "bucket_name",
				"PLUGIN_REF":             "plugin",
				"KAFKA_SOURCE_BROKERS":   "source_broker",
				"KAFKA_SOURCE_TOPIC":     "source_topic",
				"KAFKA_DEST_TOPIC":       "dest_topic",
			},
			wantErr:   true,
			errPrefix: "S3_INTEGRATION_CHANGE_QUEUE must be set",
		},
		{
			name: "missing plugin ref",
			env: map[string]string{
				"KAFKA_SOURCE_BROKERS": "source_broker",
				"KAFKA_SOURCE_TOPIC":   "source_topic",
				"KAFKA_DEST_TOPIC":     "dest_topic",
			},
			wantErr:   true,
			errPrefix: "PLUGIN_REF must be set",
		},
		{
			name: "missing kafka source topic",
			env: map[string]string{
				"PLUGIN_REF":       "plugin",
				"KAFKA_DEST_TOPIC": "dest_topic",
			},
			wantErr:   true,
			errPrefix: "KAFKA_SOURCE_TOPIC must be set",
		},
		{
			name: "source and destination topics identical",
			env: map[string]string{
				"PLUGIN_REF":           "plugin",
				"KAFKA_SOURCE_BROKERS": "broker",
				"KAFKA_SOURCE_TOPIC":   "topic",
				"KAFKA_DEST_BROKERS":   "broker",
				"KAFKA_DEST_TOPIC":     "topic",
			},
			wantErr:   true,
			errPrefix: "KAFKA_DEST_TOPIC must be different from KAFKA_SOURCE_TOPIC and must be set when KAFKA_SOURCE_BROKERS and KAFKA_DEST_BROKERS are the same",
		},
		{
			name: "default kafka source brokers",
			env: map[string]string{
				"PLUGIN_REF":         "plugin",
				"KAFKA_SOURCE_TOPIC": "source_topic",
				"KAFKA_DEST_TOPIC":   "dest_topic",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.env {
				_ = os.Setenv(k, v)
			}
			// Clear environment variables after the test
			defer func() {
				for k := range tt.env {
					_ = os.Unsetenv(k)
				}
			}()

			c := configurator{}
			_, err := c.Configure(t.Context())
			if (err != nil) != tt.wantErr {
				t.Errorf("Configure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errPrefix != "" && err.Error()[:len(tt.errPrefix)] != tt.errPrefix {
				t.Errorf("unexpected error = %v, wanted prefix %v", err.Error(), tt.errPrefix)
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue interface{}
		envValue     string
		want         interface{}
	}{
		{
			name:         "string value exists",
			key:          "TEST_STRING",
			defaultValue: "default",
			envValue:     "test",
			want:         "test",
		},
		{
			name:         "string value does not exist",
			key:          "TEST_STRING_EMPTY",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
		{
			name:         "int value exists",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "20",
			want:         20,
		},
		{
			name:         "int value does not exist",
			key:          "TEST_INT_EMPTY",
			defaultValue: 10,
			envValue:     "",
			want:         10,
		},
		{
			name:         "bool value exists",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "true",
			want:         true,
		},
		{
			name:         "bool value does not exist",
			key:          "TEST_BOOL_EMPTY",
			defaultValue: true,
			envValue:     "",
			want:         true,
		},
		{
			name:         "invalid int value",
			key:          "TEST_INVALID_INT",
			defaultValue: 10,
			envValue:     "not-an-int",
			want:         10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			switch tt.defaultValue.(type) {
			case string:
				got := getEnvOrDefault(tt.key, tt.defaultValue.(string))
				if got != tt.want.(string) {
					t.Errorf("getEnvOrDefault() = %v, want %v", got, tt.want)
				}
			case int:
				got := getEnvOrDefault(tt.key, tt.defaultValue.(int))
				if got != tt.want.(int) {
					t.Errorf("getEnvOrDefault() = %v, want %v", got, tt.want)
				}
			case bool:
				got := getEnvOrDefault(tt.key, tt.defaultValue.(bool))
				if got != tt.want.(bool) {
					t.Errorf("getEnvOrDefault() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestRequire(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		envValue string
		err     error
		want    string
		wantErr bool
	}{
		{
			name:     "value exists",
			key:      "TEST_REQUIRE",
			envValue: "test",
			err:      errors.New("TEST_REQUIRE must be set"),
			want:     "test",
			wantErr:  false,
		},
		{
			name:     "value does not exist",
			key:      "TEST_REQUIRE_EMPTY",
			envValue: "",
			err:      errors.New("TEST_REQUIRE_EMPTY must be set"),
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			got, err := require(tt.key, tt.err)
			if (err != nil) != tt.wantErr {
				t.Errorf("require() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("require() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOrErrorOnValueAndCondition(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		condition bool
		err       error
		envValue  string
		want      string
		wantErr   bool
	}{
		{
			name:      "condition true and value matches",
			key:       "TEST_CONDITION",
			value:     "",
			condition: true,
			err:       errors.New("error"),
			envValue:  "",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "condition true but value doesn't match",
			key:       "TEST_CONDITION",
			value:     "",
			condition: true,
			err:       errors.New("error"),
			envValue:  "test",
			want:      "test",
			wantErr:   false,
		},
		{
			name:      "condition false",
			key:       "TEST_CONDITION",
			value:     "",
			condition: false,
			err:       errors.New("error"),
			envValue:  "",
			want:      "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			got, err := getOrErrorOnValueAndCondition(tt.key, tt.value, tt.condition, tt.err)
			if (err != nil) != tt.wantErr {
				t.Errorf("getOrErrorOnValueAndCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getOrErrorOnValueAndCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTruthy(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envValue string
		want     bool
	}{
		{
			name:     "value exists",
			key:      "TEST_TRUTHY",
			envValue: "anything",
			want:     true,
		},
		{
			name:     "value does not exist",
			key:      "TEST_TRUTHY_EMPTY",
			envValue: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			if got := truthy(tt.key); got != tt.want {
				t.Errorf("truthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetList(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envValue string
		want     []string
	}{
		{
			name:     "value exists with multiple items",
			key:      "TEST_LIST",
			envValue: "item1,item2,item3",
			want:     []string{"item1", "item2", "item3"},
		},
		{
			name:     "value exists with single item",
			key:      "TEST_LIST_SINGLE",
			envValue: "item1",
			want:     []string{"item1"},
		},
		{
			name:     "value does not exist",
			key:      "TEST_LIST_EMPTY",
			envValue: "",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			got := getList(tt.key)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMap(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envValue string
		want     map[string]string
	}{
		{
			name:     "value exists with multiple items",
			key:      "TEST_MAP",
			envValue: "key1=value1,key2=value2,key3=value3",
			want:     map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"},
		},
		{
			name:     "value exists with single item",
			key:      "TEST_MAP_SINGLE",
			envValue: "key1=value1",
			want:     map[string]string{"key1": "value1"},
		},
		{
			name:     "value with spaces",
			key:      "TEST_MAP_SPACES",
			envValue: " key1 = value1 , key2 = value2 ",
			want:     map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name:     "value with invalid format",
			key:      "TEST_MAP_INVALID",
			envValue: "key1=value1,invalid,key2=value2",
			want:     map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name:     "value does not exist",
			key:      "TEST_MAP_EMPTY",
			envValue: "",
			want:     map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			got := parseMap(tt.key)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateKafkaClientOptions(t *testing.T) {
	c := configurator{}
	awsConfig := &aws.Config{}

	tests := []struct {
		name     string
		conf     SharedKafkaConfig
		clientName string
		awsConf  *aws.Config
		envSetup func()
		envCleanup func()
		checkOpts func(t *testing.T, opts []kgo.Opt)
	}{
		{
			name: "plain SASL mechanism",
			conf: SharedKafkaConfig{
				SASLMechanism: "plain",
				SASLUser:      "user",
				SASLPassword:  "password",
				SASLZid:       "zid",
				DialTimeoutMs: 1000,
				TLSEnabled:    true,
			},
			clientName: "TEST",
			awsConf:    nil,
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				// We can't directly check the options as they're not exposed,
				// but we can verify the length is as expected
				if len(opts) != 3 { // SASL + TLS + DialTimeout
					t.Errorf("Expected 3 options, got %d", len(opts))
				}
			},
		},
		{
			name: "sasl-scram-512 mechanism",
			conf: SharedKafkaConfig{
				SASLMechanism: "sasl-scram-512",
				SASLUser:      "user",
				SASLPassword:  "password",
				SASLTokenAuth: true,
				DialTimeoutMs: 0,
				TLSEnabled:    false,
			},
			clientName: "TEST",
			awsConf:    nil,
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				if len(opts) != 1 { // SASL only
					t.Errorf("Expected 1 option, got %d", len(opts))
				}
			},
		},
		{
			name: "sasl-scram-256 mechanism",
			conf: SharedKafkaConfig{
				SASLMechanism: "sasl-scram-256",
				SASLUser:      "user",
				SASLPassword:  "password",
				DialTimeoutMs: 0,
				TLSEnabled:    false,
			},
			clientName: "TEST",
			awsConf:    nil,
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				if len(opts) != 1 { // SASL only
					t.Errorf("Expected 1 option, got %d", len(opts))
				}
			},
		},
		{
			name: "oauth mechanism",
			conf: SharedKafkaConfig{
				SASLMechanism:   "oauth",
				SASLToken:       "token",
				OAuthExtensions: map[string]string{"ext1": "val1"},
				DialTimeoutMs:   0,
				TLSEnabled:      false,
			},
			clientName: "TEST",
			awsConf:    nil,
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				if len(opts) != 1 { // SASL only
					t.Errorf("Expected 1 option, got %d", len(opts))
				}
			},
		},
		{
			name: "aws-iam mechanism",
			conf: SharedKafkaConfig{
				SASLMechanism: "aws-iam",
				DialTimeoutMs: 0,
				TLSEnabled:    false,
			},
			clientName: "TEST",
			awsConf:    awsConfig,
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				if len(opts) != 1 { // SASL only
					t.Errorf("Expected 1 option, got %d", len(opts))
				}
			},
		},
		{
			name: "no SASL with TLS and dial timeout",
			conf: SharedKafkaConfig{
				SASLMechanism: "",
				DialTimeoutMs: 2000,
				TLSEnabled:    true,
			},
			clientName: "TEST",
			awsConf:    nil,
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				if len(opts) != 2 { // TLS + DialTimeout
					t.Errorf("Expected 2 options, got %d", len(opts))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.envSetup()
			defer tt.envCleanup()

			opts := c.CreateKafkaClientOptions(tt.conf, tt.clientName, tt.awsConf)
			tt.checkOpts(t, opts)
		})
	}
}

func TestProducerConfig(t *testing.T) {
	c := configurator{}

	tests := []struct {
		name     string
		conf     Config
		envSetup func()
		envCleanup func()
		checkOpts func(t *testing.T, opts []kgo.Opt)
	}{
		{
			name: "basic producer config",
			conf: Config{
				Kafka: KafkaConfig{
					DestBrokers: []string{"broker1:9092", "broker2:9092"},
					DestTopic:   "test-topic",
					Producer: ProducerConfig{
						SharedKafkaConfig: SharedKafkaConfig{
							SASLMechanism: "",
							TLSEnabled:    false,
						},
						Compression:        "",
						Partitioner:        "",
						BatchMaxBytes:      0,
						MaxBufferedRecords: 0,
						MaxBufferedBytes:   0,
						RecordRetries:      0,
						TimeoutMs:          0,
						LingerMs:           0,
						RequiredAcks:       "",
					},
				},
			},
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				// Basic producer config should have at least 2 options (seed brokers and default topic)
				if len(opts) < 2 {
					t.Errorf("Expected at least 2 options, got %d", len(opts))
				}
			},
		},
		{
			name: "producer with compression",
			conf: Config{
				Kafka: KafkaConfig{
					DestBrokers: []string{"broker1:9092"},
					DestTopic:   "test-topic",
					Producer: ProducerConfig{
						SharedKafkaConfig: SharedKafkaConfig{},
						Compression:        "gzip",
					},
				},
			},
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				// Should have at least 3 options (seed brokers, default topic, compression)
				if len(opts) < 3 {
					t.Errorf("Expected at least 3 options, got %d", len(opts))
				}
			},
		},
		{
			name: "producer with partitioner",
			conf: Config{
				Kafka: KafkaConfig{
					DestBrokers: []string{"broker1:9092"},
					DestTopic:   "test-topic",
					Producer: ProducerConfig{
						SharedKafkaConfig: SharedKafkaConfig{},
						Partitioner:        "round-robin",
					},
				},
			},
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				// Should have at least 3 options (seed brokers, default topic, partitioner)
				if len(opts) < 3 {
					t.Errorf("Expected at least 3 options, got %d", len(opts))
				}
			},
		},
		{
			name: "producer with required acks",
			conf: Config{
				Kafka: KafkaConfig{
					DestBrokers: []string{"broker1:9092"},
					DestTopic:   "test-topic",
					Producer: ProducerConfig{
						SharedKafkaConfig: SharedKafkaConfig{},
						RequiredAcks:       "all",
					},
				},
			},
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				// Should have at least 3 options (seed brokers, default topic, required acks)
				if len(opts) < 3 {
					t.Errorf("Expected at least 3 options, got %d", len(opts))
				}
			},
		},
		{
			name: "producer with batch settings",
			conf: Config{
				Kafka: KafkaConfig{
					DestBrokers: []string{"broker1:9092"},
					DestTopic:   "test-topic",
					Producer: ProducerConfig{
						SharedKafkaConfig:  SharedKafkaConfig{},
						BatchMaxBytes:      1024,
						MaxBufferedRecords: 100,
						MaxBufferedBytes:   10240,
						RecordRetries:      3,
						TimeoutMs:          5000,
						LingerMs:           100,
					},
				},
			},
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				// Should have at least 8 options (seed brokers, default topic, + 6 batch settings)
				if len(opts) < 8 {
					t.Errorf("Expected at least 8 options, got %d", len(opts))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.envSetup()
			defer tt.envCleanup()

			opts := c.ProducerConfig(tt.conf)
			tt.checkOpts(t, opts)
		})
	}
}

func TestConsumerConfig(t *testing.T) {
	c := configurator{}

	tests := []struct {
		name     string
		conf     Config
		envSetup func()
		envCleanup func()
		checkOpts func(t *testing.T, opts []kgo.Opt)
	}{
		{
			name: "basic consumer config",
			conf: Config{
				Kafka: KafkaConfig{
					SourceBrokers: []string{"broker1:9092", "broker2:9092"},
					SourceTopic:   "source-topic",
					Consumer: ConsumerConfig{
						SharedKafkaConfig: SharedKafkaConfig{
							SASLMechanism: "",
							TLSEnabled:    false,
						},
						Topics: []string{},
						Group:  "",
						Offset: "",
					},
				},
			},
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				// Basic consumer config should have at least 2 options (seed brokers and consume topics)
				if len(opts) < 2 {
					t.Errorf("Expected at least 2 options, got %d", len(opts))
				}
			},
		},
		{
			name: "consumer with additional topics",
			conf: Config{
				Kafka: KafkaConfig{
					SourceBrokers: []string{"broker1:9092"},
					SourceTopic:   "source-topic",
					Consumer: ConsumerConfig{
						SharedKafkaConfig: SharedKafkaConfig{},
						Topics: []string{"topic1", "topic2"},
					},
				},
			},
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				// Should have at least 2 options (seed brokers, consume topics)
				if len(opts) < 2 {
					t.Errorf("Expected at least 2 options, got %d", len(opts))
				}
			},
		},
		{
			name: "consumer with group",
			conf: Config{
				Kafka: KafkaConfig{
					SourceBrokers: []string{"broker1:9092"},
					SourceTopic:   "source-topic",
					Consumer: ConsumerConfig{
						SharedKafkaConfig: SharedKafkaConfig{},
						Group: "test-group",
					},
				},
			},
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				// Should have at least 3 options (seed brokers, consume topics, consumer group)
				if len(opts) < 3 {
					t.Errorf("Expected at least 3 options, got %d", len(opts))
				}
			},
		},
		{
			name: "consumer with offset start",
			conf: Config{
				Kafka: KafkaConfig{
					SourceBrokers: []string{"broker1:9092"},
					SourceTopic:   "source-topic",
					Consumer: ConsumerConfig{
						SharedKafkaConfig: SharedKafkaConfig{},
						Offset: "start",
					},
				},
			},
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				// Should have at least 3 options (seed brokers, consume topics, offset)
				if len(opts) < 3 {
					t.Errorf("Expected at least 3 options, got %d", len(opts))
				}
			},
		},
		{
			name: "consumer with offset end",
			conf: Config{
				Kafka: KafkaConfig{
					SourceBrokers: []string{"broker1:9092"},
					SourceTopic:   "source-topic",
					Consumer: ConsumerConfig{
						SharedKafkaConfig: SharedKafkaConfig{},
						Offset: "end",
					},
				},
			},
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				// Should have at least 3 options (seed brokers, consume topics, offset)
				if len(opts) < 3 {
					t.Errorf("Expected at least 3 options, got %d", len(opts))
				}
			},
		},
		{
			name: "consumer with numeric offset",
			conf: Config{
				Kafka: KafkaConfig{
					SourceBrokers: []string{"broker1:9092"},
					SourceTopic:   "source-topic",
					Consumer: ConsumerConfig{
						SharedKafkaConfig: SharedKafkaConfig{},
						Offset: "100",
					},
				},
			},
			envSetup:   func() {},
			envCleanup: func() {},
			checkOpts: func(t *testing.T, opts []kgo.Opt) {
				// Should have at least 3 options (seed brokers, consume topics, offset)
				if len(opts) < 3 {
					t.Errorf("Expected at least 3 options, got %d", len(opts))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.envSetup()
			defer tt.envCleanup()

			opts := c.ConsumerConfig(tt.conf)
			tt.checkOpts(t, opts)
		})
	}
}
