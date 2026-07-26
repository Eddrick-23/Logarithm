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
		IngesterHost:                  "localhost",
		IngesterPortGRPC:              "8089",
		IngesterPortHTTP:              "8090",
		IngesterReadHeaderTimeout:     2 * time.Second,
		IngesterReadTimeout:           10 * time.Second,
		IngesterWriteTimeout:          10 * time.Second,
		IngesterIdleTimeout:           60 * time.Second,
		IngesterPresizeBuffer:         4 * 1024,
		IngesterBufferLimit:           20 * 1024,
		AppHost:                       "localhost",
		AppPort:                       "8091",
		LiveTailPresenceInterval:      1 * time.Second,
		LiveTailRefreshInterval:       500,
		LiveTailMaxBatch:              100,
		DBAddress:                     "localhost:9000",
		DBUser:                        "default",
		DBPassword:                    "password",
		DBName:                        "logarithm",
		DBBatchPoolSize:               5,
		DBBatchPoolMaxRows:            2000,
		NatsURL:                       "nats://localhost:4222",
		NatsStreamMaxAge:              24 * time.Hour,
		NatsDLQMaxAge:                 24 * time.Hour,
		NatsMaxDeliver:                10,
		NatsBackoff:                   []time.Duration{1 * time.Second},
		NatsLogStreamMaxBytes:         100,
		NatsDLQMaxBytes:               100,
		NatsConsumerMaxAckPending:     1000,
		WorkerLogLevel:                "INFO",
		WorkerMaxBatch:                1000,
		WorkerMaxWait:                 2 * time.Second,
		WorkerBackoff:                 []time.Duration{1 * time.Second},
		WorkerCount:                   1,
		WorkerLiveTailCount:           3,
		WorkerLiveTailQueueSize:       10000,
		WorkerRowsPerBatch:            1000,
		WorkerLiveTailPresenceTimeout: 1 * time.Second,
		SeedSystem:                    false,
		EnablePprof:                   false,
		PprofHost:                     "0.0.0.0",
	}

	invalidConfigIngesterPresizeBuffer := validConfig
	invalidConfigIngesterPresizeBuffer.IngesterPresizeBuffer = -10

	invalidConfigIngesterBufferLimit := validConfig
	invalidConfigIngesterBufferLimit.IngesterBufferLimit = validConfig.IngesterPresizeBuffer - 1

	invalidConfigMaxAckPending := validConfig
	invalidConfigMaxAckPending.NatsConsumerMaxAckPending = 500

	invalidConfigWorkerCount := validConfig
	invalidConfigWorkerCount.WorkerCount = 0

	invalidConfigLiveTailWorkers := validConfig
	invalidConfigLiveTailWorkers.WorkerLiveTailCount = 0

	invalidConfigLiveTailQueueSize := validConfig
	invalidConfigLiveTailQueueSize.WorkerLiveTailQueueSize = 0

	invalidConfigWorkerRowsPerBatch := validConfig
	invalidConfigWorkerRowsPerBatch.WorkerRowsPerBatch = 0

	invalidConfigDBBatchPoolSize := validConfig
	invalidConfigDBBatchPoolSize.DBBatchPoolSize = 0

	invalidConfigDBBatchPoolRowsTooSmall := validConfig
	invalidConfigDBBatchPoolRowsTooSmall.DBBatchPoolMaxRows = 1

	invalidConfigLiveTailPresenceInterval := validConfig
	invalidConfigLiveTailPresenceInterval.LiveTailPresenceInterval = 0

	invalidConfigWorkerLiveTailPresenceTimeout := validConfig
	invalidConfigWorkerLiveTailPresenceTimeout.WorkerLiveTailPresenceTimeout = 0

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
			name:          "invalid ingester presize buffer",
			input:         invalidConfigIngesterPresizeBuffer,
			expectedError: true,
		},
		{
			name:          "invalid ingester buffer limit smaller than presize buffer",
			input:         invalidConfigIngesterBufferLimit,
			expectedError: true,
		},
		{
			name:          "invalid config max ack pending too low",
			input:         invalidConfigMaxAckPending,
			expectedError: true,
		},
		{
			name:          "invalid config worker pool count",
			input:         invalidConfigWorkerCount,
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
		{
			name:          "invalid config worker rows per batch",
			input:         invalidConfigWorkerRowsPerBatch,
			expectedError: true,
		},
		{
			name:          "invalid config DB Batch Pool size",
			input:         invalidConfigDBBatchPoolSize,
			expectedError: true,
		},
		{
			name:          "invalid config DB Batch Pool Max Rows is smaller than worker rows per batch",
			input:         invalidConfigDBBatchPoolRowsTooSmall,
			expectedError: true,
		},
		{
			name:          "invalid config live tail presence interval",
			input:         invalidConfigLiveTailPresenceInterval,
			expectedError: true,
		},
		{
			name:          "invalid config worker live tail presence timeout",
			input:         invalidConfigWorkerLiveTailPresenceTimeout,
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
