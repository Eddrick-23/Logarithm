package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/Eddrick-23/Logarithm/internal/config"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
)

func run(ctx context.Context, w io.Writer) error {
	logger := slog.New(
		slog.NewTextHandler(w, nil),
	)
	workerLogger := logger.With("component", "worker")
	natsLogger := logger.With("component", "nats")
	// TODO refactor db to support logger via dependency injection

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

	if _, err = natsBroker.EnsureStream(ctx, config.NatsSubject); err != nil {
		return fmt.Errorf("failed to ensure stream: %w", err)
	}
	defer func() {
		natsBroker.Close()

		if err := store.Close(); err != nil {
			workerLogger.Error("failed to close clickhouse connection", "err", err)
		}
	}()

	// fmt.Println(store, natsBroker)
	// consume logic here
	return nil
}

func main() {

	run(context.Background(), os.Stdout)

}
