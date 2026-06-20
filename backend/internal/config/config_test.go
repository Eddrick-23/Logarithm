package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestByteSizeDecode(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expected      int64
		expectedError bool
	}{
		{"parse GB", "50GB", 50 * 1024 * 1024 * 1024, false},
		{"parse MB", "10MB", 10 * 1024 * 1024, false},
		{"parse KB", "10KB", 10 * 1024, false},
		{"empty string", "", 0, true},
		{"invalid string", "invalidMB", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var b ByteSize
			err := b.EnvDecode(tc.input)

			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, b, ByteSize(tc.expected))
			}
		})
	}

}

func TestValidate(t *testing.T) {
	validConfig := Config{
		IngesterHost:              "localhost",
		IngesterPort:              "8090",
		IngesterReadHeaderTimeout: 2 * time.Second,
		IngesterReadTimeout:       10 * time.Second,
		IngesterWriteTimeout:      10 * time.Second,
		IngesterIdleTimeout:       60 * time.Second,
		AppHost:                   "localhost",
		AppPort:                   "8091",
		LiveTailRefreshInterval:   500,
		LiveTailMaxBatch:          100,
		DBAddress:                 "localhost:9000",
		DBUser:                    "default",
		DBPassword:                "password",
		DBName:                    "logarithm",
		NatsURL:                   "nats://localhost:4222",
		NatsStreamMaxAge:          24 * time.Hour,
		NatsDLQMaxAge:             24 * time.Hour,
		NatsMaxDeliver:            10,
		NatsBackoff:               []time.Duration{1 * time.Second},
		NatsLogStreamMaxBytes:     100,
		NatsDLQMaxBytes:           100,
		NatsConsumerMaxAckPending: 1000,
		WorkerLogLevel:            "INFO",
		WorkerMaxBatch:            1000,
		WorkerMaxWait:             2 * time.Second,
		WorkerBackoff:             []time.Duration{1 * time.Second},
		WorkerLiveTailCount:       3,
		WorkerLiveTailQueueSize:   10000,
		SeedSystem:                false,
		EnablePprof:               false,
		PprofHost:                 "0.0.0.0",
	}

	invalidConfigMaxAckPending := validConfig
	invalidConfigMaxAckPending.NatsConsumerMaxAckPending = 500

	invalidConfigLiveTailWorkers := validConfig
	invalidConfigLiveTailWorkers.WorkerLiveTailCount = 0

	invalidConfigLiveTailQueueSize := validConfig
	invalidConfigLiveTailQueueSize.WorkerLiveTailQueueSize = 0
	tests := []struct {
		name          string
		input         Config
		expectedError bool
	}{
		{
			name:          "valid config",
			input:         validConfig,
			expectedError: false,
		},
		{
			name:          "invalid config max ack pending too low",
			input:         invalidConfigMaxAckPending,
			expectedError: true,
		},
		{
			name:          "invalid config live tail workers",
			input:         invalidConfigLiveTailWorkers,
			expectedError: true,
		},
		{
			name:          "invalid config live tail queue size",
			input:         invalidConfigLiveTailQueueSize,
			expectedError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.validate()
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
