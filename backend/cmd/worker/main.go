package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/Eddrick-23/Logarithm/internal/config"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
)

func startPprof(logger *slog.Logger, config *config.Config) {
	if !config.EnablePprof {
		return
	}
	runtime.SetMutexProfileFraction(100)
	runtime.SetBlockProfileRate(100000)
	addr := net.JoinHostPort(config.PprofHost, "6061")
	go func() {
		logger.Info("pprof listening on", "addr", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			logger.Error("pprof server stopped", "err", err)
		}
	}()
}

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
	pprofLogger := logger.With("component", "pprof")
	dblogger := logger.With("component", "db")

	startPprof(pprofLogger, config)

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	store, err := storage.NewClickHouseStore(ctx,
		dblogger,
		config.DBAddress,
		config.DBName,
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

	stream, err := natsBroker.EnsureLogStream(ctx, transport.LogStreamName, transport.LogStreamSubject, config.NatsStreamMaxAge, int64(config.NatsLogStreamMaxBytes))
	if err != nil {
		return fmt.Errorf("failed to ensure log stream: %w", err)
	}

	consumer, err := natsBroker.NewDurableConsumer(ctx, stream, workerName, config.NatsMaxDeliver, config.NatsBackoff, config.NatsConsumerMaxAckPending)
	if err != nil {
		return fmt.Errorf("failed to create durable consumer: %w", err)
	}

	_, err = natsBroker.EnsureDLQStream(ctx, transport.DLQStreamName, transport.DLQSubject, config.NatsDLQMaxAge, int64(config.NatsDLQMaxBytes)) // set longer max age for debugging
	if err != nil {
		return fmt.Errorf("failed to ensure dlq stream: %w", err)
	}

	_, err = natsBroker.EnsureLiveTailStream(ctx, transport.LiveTailStreamName, transport.LiveTailSubject, int64(config.NatsLiveTailMaxBytes))
	if err != nil {
		return fmt.Errorf("failed to ensure live tail stream: %w", err)
	}
	consumeCallback, err := ConsumeCallback(workerLogger, store, natsBroker)
	if err != nil {
		return err
	}
	return consumer.ConsumeLogs(ctx,
		consumeCallback,
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
