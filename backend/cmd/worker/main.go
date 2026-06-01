package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Eddrick-23/Logarithm/internal/config"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
)

func run(ctx context.Context, w io.Writer) error {
	workerName := "worker"

	config, err := config.LoadConfig(ctx)
	if err != nil {
		return err
	}

	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(config.WorkerLogLevel)); err != nil {
		logLevel = slog.LevelInfo
	}
	opt := &slog.HandlerOptions{
		Level: logLevel,
	}
	logger := slog.New(
		slog.NewTextHandler(w, opt),
	)
	workerLogger := logger.With("component", "worker")
	natsLogger := logger.With("component", "nats")
	// TODO refactor db to support logger via dependency injection

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	store, err := storage.NewClickHouseStore(ctx,
		config.DBAddress,
		config.DBName,
		config.DBTableName,
		config.DBUser,
		config.DBPassword)

	if err != nil {
		return fmt.Errorf("failed to connect to db: %w", err)
	}

	natsBroker, err := transport.NewNatsBroker(ctx, natsLogger, config.NatsURL)

	if err != nil {
		return fmt.Errorf("failed to create nats broker: %w", err)
	}

	defer func() {
		natsBroker.Close()

		if err := store.Close(); err != nil {
			workerLogger.Error("failed to close clickhouse connection", "err", err)
		}
		workerLogger.Info("clickhouse connection closed")
	}()

	stream, err := natsBroker.EnsureLogStream(ctx, transport.LogStreamName, config.NatsSubject, config.NatsStreamMaxAge)
	if err != nil {
		return fmt.Errorf("failed to ensure log stream: %w", err)
	}

	consumer, err := natsBroker.NewDurableConsumer(ctx, stream, workerName, config.NatsMaxDeliver, config.NatsBackoff)
	if err != nil {
		return fmt.Errorf("failed to create durable consumer: %w", err)
	}

	_, err = natsBroker.EnsureDLQStream(ctx, transport.DLQStreamName, transport.DLQSubject, 5*config.NatsStreamMaxAge) // set longer max age for debugging
	if err != nil {
		return fmt.Errorf("failed to ensure dlq stream: %w", err)
	}

	return consumer.ConsumeLogs(ctx,
		ConsumeCallback(workerLogger, store),
		DLQCallback(natsBroker, transport.DLQSubject),
		DelayCalculator(config.WorkerBackoff),
		config.WorkerMaxBatch,
		config.WorkerMaxWait,
	)
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
