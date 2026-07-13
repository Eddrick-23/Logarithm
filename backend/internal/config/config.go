package config

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/go-units"
	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type ByteSize int64

func (b *ByteSize) EnvDecode(val string) error {
	parsed, err := units.RAMInBytes(val)

	if err != nil {
		return err
	}

	*b = ByteSize(parsed)
	return nil
}

type Config struct {
	IngesterHost              string          `env:"INGESTER_HOST, default=localhost"`
	IngesterPortGRPC          string          `env:"INGESTER_PORT_GRPC, default=8089"`
	IngesterPortHTTP          string          `env:"INGESTER_PORT_HTTP, default=8090"`
	IngesterLogLevel          string          `env:"INGESTER_LOG_LEVEL, default=INFO"`
	IngesterReadHeaderTimeout time.Duration   `env:"INGESTER_READ_HEADER_TIMEOUT, default=2s"`
	IngesterReadTimeout       time.Duration   `env:"INGESTER_READ_TIMEOUT, default=5s"`
	IngesterWriteTimeout      time.Duration   `env:"INGESTER_WRITE_TIMEOUT, default=10s"`
	IngesterIdleTimeout       time.Duration   `env:"INGESTER_IDLE_TIMEOUT, default=60s"`
	AppHost                   string          `env:"APP_HOST, default=dashboard-api"`
	AppPort                   string          `env:"APP_PORT, default=8091"`
	DashboardLogLevel         string          `env:"DASHBOARD_LOG_LEVEL, default=INFO"`
	LiveTailRefreshInterval   int             `env:"LIVE_TAIL_REFRESH_INTERVAL, default=500"`
	LiveTailMaxBatch          int             `env:"LIVE_TAIL_MAX_BATCH, default=100"`
	DBAddress                 string          `env:"DB_ADDRESS, default=localhost:9000"`
	DBUser                    string          `env:"DB_USER, required"`
	DBPassword                string          `env:"DB_PASSWORD, required"`
	DBName                    string          `env:"DB_NAME, default=logarithm"`
	DBBatchPoolSize           int             `env:"DB_BATCH_POOL_SIZE, default=5"`
	DBBatchPoolMaxRows        int             `env:"DB_BATCH_POOL_MAX_ROWS, default=2000"`
	NatsURL                   string          `env:"NATS_URL, default=nats://127.0.0.1:4222"`
	NatsStreamMaxAge          time.Duration   `env:"NATS_STREAM_MAX_AGE, default=12h"`
	NatsDLQMaxAge             time.Duration   `env:"NATS_DLQ_MAX_AGE, default=24h"`
	NatsMaxDeliver            int             `env:"NATS_MAX_DELIVER, default=10"`
	NatsBackoff               []time.Duration `env:"NATS_BACKOFF, default=5s,30s,60s,300s,3600s"`
	NatsLogStreamMaxBytes     ByteSize        `env:"NATS_LOG_STREAM_MAX_BYTES, default=50GB"`
	NatsDLQMaxBytes           ByteSize        `env:"NATS_DLQ_MAX_BYTES, default=10GB"`
	NatsConsumerMaxAckPending int             `env:"NATS_CONSUMER_MAX_ACK_PENDING, default=1000"`
	WorkerLogLevel            string          `env:"WORKER_LOG_LEVEL, default=INFO"`
	WorkerMaxBatch            int             `env:"WORKER_MAX_BATCH, default=10"`
	WorkerMaxWait             time.Duration   `env:"WORKER_MAX_WAIT, default=2s"`
	WorkerBackoff             []time.Duration `env:"WORKER_BACKOFF, default=5s,30s,60s,300s,3600s"`
	WorkerCount               int             `env:"WORKER_COUNT, default=1"`
	WorkerLiveTailCount       int             `env:"WORKER_LIVE_TAIL_COUNT, default=3"`
	WorkerLiveTailQueueSize   int             `env:"WORKER_LIVE_TAIL_QUEUE_SIZE, default=10000"`
	WorkerRowsPerBatch        int             `env:"WORKER_ROWS_PER_BATCH, default=1000"`
	SeedSystem                bool            `env:"SEED_SYSTEM, default=false"`
	EnablePprof               bool            `env:"ENABLE_PPROF, default=false"`
	PprofHost                 string          `env:"PPROF_HOST, default=0.0.0.0"`
}

type PublicConfig struct {
	LiveTailRefreshInterval   int    `json:"liveTailRefreshInterval"`
	LiveTailMaxBatch          int    `json:"liveTailMaxBatch"`
	NatsStreamMaxAge          string `json:"natsStreamMaxAge"`
	NatsDLQMaxAge             string `json:"natsDLQMaxAge"`
	NatsMaxDeliver            int    `json:"natsMaxDeliver"`
	NatsBackoff               string `json:"natsBackoff"`
	NatsLogStreamMaxBytes     string `json:"natsLogStreamMaxBytes"`
	NatsDLQMaxBytes           string `json:"natsDLQMaxBytes"`
	NatsConsumerMaxAckPending int    `json:"natsConsumerMaxAckPending"`
	WorkerLogLevel            string `json:"workerLogLevel"`
	WorkerMaxBatch            int    `json:"workerMaxBatch"`
	WorkerBackoff             string `json:"workerBackoff"`
	WorkerRowsPerBatch        int    `json:"workerRowsPerBatch"`
}

func LoadConfig(ctx context.Context) (*Config, error) {
	if err := godotenv.Load(".env.local", ".env"); err != nil {
		fmt.Println("Note: No .env found. using system environment variables with default fallbacks if needed.")
	}
	var c Config
	if err := envconfig.Process(ctx, &c); err != nil {
		return nil, fmt.Errorf("failed to load config from environment variables: %w", err)
	}

	if err := c.validate(); err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *Config) validate() error {
	if c.NatsConsumerMaxAckPending < c.WorkerMaxBatch {
		safeLimit := c.WorkerMaxBatch * 2
		return fmt.Errorf("NATS_CONSUMER_MAX_ACK_PENDING (%d) is dangerously low. To prevent deadlocks between nats and worker, set it to at least (%d)",
			c.NatsConsumerMaxAckPending,
			safeLimit,
		)
	}

	if c.WorkerCount <= 0 {
		return fmt.Errorf("WORKER_COUNT (%d) must be a positive number. Recommended (1)",
			c.WorkerLiveTailCount,
		)
	}

	if c.WorkerLiveTailCount <= 0 {
		return fmt.Errorf("WORKER_LIVE_TAIL_COUNT (%d) must be a positive number for the live tail feature to work. Recommended (3)",
			c.WorkerLiveTailCount,
		)
	}

	if c.WorkerLiveTailQueueSize <= 0 {
		return fmt.Errorf("WORKER_LIVE_TAIL_QUEUE_SIZE (%d) must be a positive number for the live tail feature to work. Recommended (10000)",
			c.WorkerLiveTailQueueSize,
		)
	}

	if c.WorkerRowsPerBatch <= 0 {
		return fmt.Errorf("WORKER_ROWS_PER_BATCH (%d) must be a positive number for effective buffer presizing. Recommended(2000)",
			c.WorkerRowsPerBatch,
		)
	}

	if c.DBBatchPoolSize <= 0 {
		return fmt.Errorf("DB_BATCH_POOL_SIZE (%d) must be a positive number",
			c.DBBatchPoolSize,
		)
	}

	if c.DBBatchPoolMaxRows < c.WorkerRowsPerBatch {
		return fmt.Errorf("DB_BATCH_POOL_MAX_ROWS (%d) must be larger than WORKER_ROWS_PER_BATCH (%d) for effective buffer reuse. Recommended (%d)",
			c.DBBatchPoolMaxRows, c.WorkerRowsPerBatch, 2*c.WorkerRowsPerBatch,
		)
	}
	return nil
}

func (c *Config) Public() PublicConfig {
	return PublicConfig{
		LiveTailRefreshInterval:   c.LiveTailRefreshInterval,
		LiveTailMaxBatch:          c.LiveTailMaxBatch,
		NatsStreamMaxAge:          formatDuration(c.NatsStreamMaxAge),
		NatsDLQMaxAge:             formatDuration(c.NatsDLQMaxAge),
		NatsMaxDeliver:            c.NatsMaxDeliver,
		NatsBackoff:               formatBackoff(c.NatsBackoff),
		NatsLogStreamMaxBytes:     units.BytesSize(float64(c.NatsLogStreamMaxBytes)),
		NatsDLQMaxBytes:           units.BytesSize(float64(c.NatsDLQMaxBytes)),
		NatsConsumerMaxAckPending: c.NatsConsumerMaxAckPending,
		WorkerLogLevel:            c.WorkerLogLevel,
		WorkerMaxBatch:            c.WorkerMaxBatch,
		WorkerBackoff:             formatBackoff(c.WorkerBackoff),
		WorkerRowsPerBatch:        c.WorkerRowsPerBatch,
	}
}
