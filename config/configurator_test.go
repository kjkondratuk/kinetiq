package config

import (
	"os"
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
			errPrefix: "KAFKA_DEST_TOPIC must be different",
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
				os.Setenv(k, v)
			}
			// Clear environment variables after the test
			defer func() {
				for k := range tt.env {
					os.Unsetenv(k)
				}
			}()

			c := configurator{}
			_, err := c.Configure()
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
