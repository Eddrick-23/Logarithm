package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Eddrick-23/Logarithm/api/schemas"
	"github.com/Eddrick-23/Logarithm/internal/config"
	"github.com/Eddrick-23/Logarithm/internal/dashboard"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
)

func NewServer(logger *slog.Logger, config *config.Config, logStore *storage.ClickHouseStore, broker *transport.NatsBroker) http.Handler {
	mux := http.NewServeMux()
	dashboard.AddRoutes(mux, logger, config, logStore, broker)

	var handler http.Handler = mux
	// add middlewares if any

	return handler
}

func run(ctx context.Context, w io.Writer, args []string) error {
	logger := slog.New(
		slog.NewTextHandler(w, nil),
	)
	httpLogger := logger.With("component", "dashboard")
	databaseLogger := logger.With("component", "database")
	natsLogger := logger.With("component", "nats")

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	config, err := config.LoadConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	broker, err := transport.NewNatsBroker(ctx, natsLogger, config.NatsURL)
	if err != nil {
		natsLogger.Error("failed to initialise NATS broker", "err", err)
		os.Exit(1)
	}
	defer broker.Close()

	// Mocking live updates
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				req := schemas.LogIngestRequest{
					ServiceName: "auth-service",
					ResourceAttributes: []schemas.KeyValue{
						{Key: "environment", Value: "production"},
						{Key: "host.name", Value: "auth-worker-01"},
					},
					Records: []schemas.LogRecordDTO{
						{
							Timestamp:      time.Now(),
							TraceId:        "5b8aa5a2d2c8646c14e138a83416a41f",
							SpanId:         "f96ea2a71a065463",
							SeverityText:   "INFO",
							SeverityNumber: 9,
							Body:           "User authenticated successfully.",
							LogAttributes: []schemas.KeyValue{
								{Key: "user_id", Value: "usr_987654321"},
								{Key: "ip_address", Value: "192.168.1.104"},
							},
						},
						{
							Timestamp:      time.Now(),
							TraceId:        "1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d",
							SpanId:         "a1b2c3d4e5f6a7b8",
							SeverityText:   "ERROR",
							SeverityNumber: 17,
							Body:           "Failed to connect to database cache.",
							LogAttributes: []schemas.KeyValue{
								{Key: "cache_host", Value: "redis-cluster.local"},
								{Key: "timeout_ms", Value: "5000"},
							},
						},
					},
				}

				// Marshal the struct into a JSON byte slice
				payload, err := json.Marshal(req)
				if err != nil {
					natsLogger.Error("failed to marshal log payload:", "err", err)
					continue
				}

				// Publish the marshaled JSON bytes
				if err := broker.PublishLogs(ctx, "logs.service-a", payload); err != nil {
					natsLogger.Error("publish error:", "err", err)
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	logStore, err := storage.NewClickHouseStore(ctx, config.DBAddress, config.DBName, config.DBTableName, config.DBUser, config.DBPassword)
	if err != nil {
		databaseLogger.Error("failed to connect to db", "err", err)
		panic(err)
	}

	// seeding database with dummy data
	if err := logStore.InitDB(ctx); err != nil {
		databaseLogger.Error("initdb failed", "err", err)
	}

	srv := NewServer(httpLogger, config, logStore, broker)

	httpServer := &http.Server{
		Addr:    net.JoinHostPort(config.AppHost, config.AppPort),
		Handler: srv,
	}

	go func() {
		httpLogger.Info("listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpLogger.Error("http server failed", "err", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() { // shutdown job
		defer wg.Done()
		<-ctx.Done()

		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer cancel()

		httpLogger.Info("Shutting down server")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			httpLogger.Error("http server shutting down failed", "err", err)
		}
	}()

	wg.Wait()

	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stdout, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
