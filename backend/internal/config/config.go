package config

import (
	"context"
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	IngesterHost              string          `env:"INGESTER_HOST, default=localhost"`
	IngesterPort              string          `env:"INGESTER_PORT, default=8090"`
	IngesterReadHeaderTimeout time.Duration   `env:"INGESTER_READ_HEADER_TIMEOUT, default=2s"`
	IngesterReadTimeout       time.Duration   `env:"INGESTER_READ_TIMEOUT, default=5s"`
	IngesterWriteTimeout      time.Duration   `env:"INGESTER_WRITE_TIMEOUT, default=10s"`
	IngesterIdleTimeout       time.Duration   `env:"INGESTER_IDLE_TIMEOUT, default=60s"`
	AppHost                   string          `env:"APP_HOST, default=dashboard-api"`
	AppPort                   string          `env:"APP_PORT, default=8091"`
	LiveTailRefreshInterval   int             `env:"LIVE_TAIL_REFRESH_INTERVAL, default=500"`
	LiveTailMaxBatch          int             `env:"LIVE_TAIL_MAX_BATCH, default=100"`
	DBAddress                 string          `env:"DB_ADDRESS, default=localhost:9000"`
	DBUser                    string          `env:"DB_USER, required"`
	DBPassword                string          `env:"DB_PASSWORD, required"`
	DBName                    string          `env:"DB_NAME, default=logarithm"`
	DBTableName               string          `env:"DB_TABLE_NAME, default=logs"`
	NatsURL                   string          `env:"NATS_URL, default=nats://127.0.0.1:4222"`
	NatsSubject               string          `env:"NATS_SUBJECT, default=logs.>"`
	NatsPublishSubjectPrefix  string          `env:"NATS_PUBLISH_PREFIX, default=logs."`
	NatsStreamMaxAge          time.Duration   `env:"NATS_STREAM_MAX_AGE, default=12h"`
	NatsMaxDeliver            int             `env:"NATS_MAX_DELIVER, default=10"`
	NatsBackoff               []time.Duration `env:"NATS_BACKOFF, default=5s,30s,60s,300s,3600s"`
	WorkerLogLevel            string          `env:"WORKER_LOG_LEVEL, default=INFO"`
	WorkerMaxBatch            int             `env:"WORKER_MAX_BATCH, default=10"`
	WorkerMaxWait             time.Duration   `env:"WORKER_MAX_WAIT, default=2s"`
	WorkerBackoff             []time.Duration `env:"WORKER_BACKOFF, default=5s,30s,60s,300s,3600s"`
	EnablePprof               bool            `env:"ENABLE_PPROF, default=false"`
	PprofHost                 string          `env:"PPROF_HOST, default=0.0.0.0"`
}

func LoadConfig(ctx context.Context) (*Config, error) {
	if err := godotenv.Load(".env.local", ".env"); err != nil {
		fmt.Println("Note: No .env found. using system environment variables with default fallbacks if needed.")
	}
	var c Config
	if err := envconfig.Process(ctx, &c); err != nil {
		return nil, fmt.Errorf("failed to load config from environment variables: %w", err)
	}
	return &c, nil
}
