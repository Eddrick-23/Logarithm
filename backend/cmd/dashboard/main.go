package main

import (
	"context"
	"encoding/hex"
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

	"github.com/Eddrick-23/Logarithm/internal/config"
	"github.com/Eddrick-23/Logarithm/internal/dashboard"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
	collectorlogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func NewServer(logger *slog.Logger, config *config.Config, logStore *storage.ClickHouseStore, broker *transport.NatsBroker) http.Handler {
	mux := http.NewServeMux()
	dashboard.AddRoutes(mux, logger, config, logStore, broker)

	var handler http.Handler = mux
	// add middlewares if any

	return handler
}

func mustDecodeHex(s string) []byte {
	b, _ := hex.DecodeString(s)
	return b
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
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				req := collectorlogspb.ExportLogsServiceRequest{
					ResourceLogs: []*logspb.ResourceLogs{
						{
							Resource: &resourcepb.Resource{
								Attributes: []*commonpb.KeyValue{
									{Key: "service.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "auth-service"}}},
									{Key: "environment", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "production"}}},
									{Key: "host.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "auth-worker-01"}}},
								},
							},
							ScopeLogs: []*logspb.ScopeLogs{
								{
									LogRecords: []*logspb.LogRecord{
										{
											TimeUnixNano:   uint64(time.Now().UnixNano()),
											TraceId:        mustDecodeHex("5b8aa5a2d2c8646c14e138a83416a41f"),
											SpanId:         mustDecodeHex("f96ea2a71a065463"),
											SeverityText:   "INFO",
											SeverityNumber: logspb.SeverityNumber_SEVERITY_NUMBER_INFO,
											Body:           &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "User authenticated successfully."}},
											Attributes: []*commonpb.KeyValue{
												{Key: "user_id", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "usr_987654321"}}},
												{Key: "ip_address", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "192.168.1.104"}}},
											},
										},
										{
											TimeUnixNano:   uint64(time.Now().UnixNano()),
											TraceId:        mustDecodeHex("1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d"),
											SpanId:         mustDecodeHex("a1b2c3d4e5f6a7b8"),
											SeverityText:   "ERROR",
											SeverityNumber: logspb.SeverityNumber_SEVERITY_NUMBER_ERROR,
											Body:           &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "Failed to connect to database cache."}},
											Attributes: []*commonpb.KeyValue{
												{Key: "cache_host", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "redis-cluster.local"}}},
												{Key: "timeout_ms", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "5000"}}},
											},
										},
										{
											TimeUnixNano:   uint64(time.Now().UnixNano()),
											TraceId:        mustDecodeHex("aabbccddeeff00112233445566778899"),
											SpanId:         mustDecodeHex("b1c2d3e4f5a6b7c8"),
											SeverityText:   "DEBUG",
											SeverityNumber: logspb.SeverityNumber_SEVERITY_NUMBER_DEBUG,
											Body:           &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "Cache lookup attempted for session token."}},
											Attributes: []*commonpb.KeyValue{
												{Key: "session_id", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "sess_112233445"}}},
												{Key: "cache_key", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "session:sess_112233445"}}},
											},
										},
										{
											TimeUnixNano:   uint64(time.Now().UnixNano()),
											TraceId:        mustDecodeHex("deadbeefcafebabe1234567890abcdef"),
											SpanId:         mustDecodeHex("c3d4e5f6a7b8c9d0"),
											SeverityText:   "WARNING",
											SeverityNumber: logspb.SeverityNumber_SEVERITY_NUMBER_WARN,
											Body:           &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "Auth token expiring soon, refresh recommended."}},
											Attributes: []*commonpb.KeyValue{
												{Key: "user_id", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "usr_111222333"}}},
												{Key: "expires_in_seconds", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "120"}}},
											},
										},
									},
								},
							},
						},
					},
				}

				// Marshal the struct into a JSON byte slice
				payload, err := protojson.Marshal(&req)
				if err != nil {
					natsLogger.Error("failed to marshal log payload:", "err", err)
					continue
				}

				// Publish the marshaled JSON bytes
				if err := broker.PublishLogs(ctx, "logs.auth-service", payload); err != nil {
					natsLogger.Error("publish error:", "err", err)
				}

				req2 := collectorlogspb.ExportLogsServiceRequest{
					ResourceLogs: []*logspb.ResourceLogs{
						{
							Resource: &resourcepb.Resource{
								Attributes: []*commonpb.KeyValue{
									{Key: "service.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "logging-service"}}},
									{Key: "environment", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "production"}}},
									{Key: "host.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "logging-worker-01"}}},
								},
							},
							ScopeLogs: []*logspb.ScopeLogs{
								{
									LogRecords: []*logspb.LogRecord{
										{
											TimeUnixNano:   uint64(time.Now().UnixNano()),
											TraceId:        mustDecodeHex("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"),
											SpanId:         mustDecodeHex("d1e2f3a4b5c6d7e8"),
											SeverityText:   "INFO",
											SeverityNumber: logspb.SeverityNumber_SEVERITY_NUMBER_INFO,
											Body:           &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "Log pipeline started successfully."}},
											Attributes: []*commonpb.KeyValue{
												{Key: "pipeline_id", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "pipe_001"}}},
												{Key: "source", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "otel-collector"}}},
											},
										},
										{
											TimeUnixNano:   uint64(time.Now().UnixNano()),
											TraceId:        mustDecodeHex("f1e2d3c4b5a6978869504030201f0e0d"),
											SpanId:         mustDecodeHex("e2f3a4b5c6d7e8f9"),
											SeverityText:   "DEBUG",
											SeverityNumber: logspb.SeverityNumber_SEVERITY_NUMBER_DEBUG,
											Body:           &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "Flushing log buffer to storage backend."}},
											Attributes: []*commonpb.KeyValue{
												{Key: "buffer_size", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "512"}}},
												{Key: "backend", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "clickhouse"}}},
											},
										},
										{
											TimeUnixNano:   uint64(time.Now().UnixNano()),
											TraceId:        mustDecodeHex("0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d"),
											SpanId:         mustDecodeHex("f3a4b5c6d7e8f9a0"),
											SeverityText:   "WARNING",
											SeverityNumber: logspb.SeverityNumber_SEVERITY_NUMBER_WARN,
											Body:           &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "Log ingestion rate approaching limit."}},
											Attributes: []*commonpb.KeyValue{
												{Key: "current_rate", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "4800"}}},
												{Key: "limit", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "5000"}}},
											},
										},
										{
											TimeUnixNano:   uint64(time.Now().UnixNano()),
											TraceId:        mustDecodeHex("1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f"),
											SpanId:         mustDecodeHex("a4b5c6d7e8f9a0b1"),
											SeverityText:   "ERROR",
											SeverityNumber: logspb.SeverityNumber_SEVERITY_NUMBER_ERROR,
											Body:           &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "Failed to write log batch to storage."}},
											Attributes: []*commonpb.KeyValue{
												{Key: "batch_id", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "batch_20260608_001"}}},
												{Key: "error", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "connection timeout"}}},
											},
										},
									},
								},
							},
						},
					},
				}

				// Marshal the struct into a JSON byte slice
				payload, err = protojson.Marshal(&req2)
				if err != nil {
					natsLogger.Error("failed to marshal log payload:", "err", err)
					continue
				}

				// Publish the marshaled JSON bytes
				if err := broker.PublishLogs(ctx, "logs.logging-service", payload); err != nil {
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
