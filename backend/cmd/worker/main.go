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
	const logLevel = slog.LevelInfo // TODO set debug level in config
	const workerName = "worker"
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

	config := config.LoadConfig()
	store, err := storage.NewClickHouseStore(ctx,
		config.DBAddress,
		config.DBName,
		config.DBTableName,
		config.DBUser,
		config.DBPassword)

	if err != nil {
		return fmt.Errorf("failed to connect to clickhouse: %w", err)
	}

	natsBroker, err := transport.NewNatsBroker(ctx, natsLogger, config.NatsURL)

	if err != nil {
		return fmt.Errorf("failed to create Nats Broker: %w", err)
	}

	defer func() {
		natsBroker.Close()

		if err := store.Close(); err != nil {
			workerLogger.Error("failed to close clickhouse connection", "err", err)
		}
		workerLogger.Info("clickhouse connection closed")
	}()

	stream, err := natsBroker.EnsureStream(ctx, config.NatsSubject)
	if err != nil {
		return fmt.Errorf("failed to ensure stream: %w", err)
	}

	consumer, err := natsBroker.NewDurableConsumer(ctx, stream, workerName)
	if err != nil {
		return fmt.Errorf("failed to create durable consumer: %w", err)
	}

	return consumer.ConsumeLogs(ctx, ConsumeCallback(store))
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
