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
	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/dashboard"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
	"github.com/Eddrick-23/Logarithm/internal/worker"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

func NewServer(logger *slog.Logger, config *config.Config, logStore *storage.ClickHouseStore, broker *transport.NatsBroker, appCtx context.Context) http.Handler {
	mux := http.NewServeMux()
	dashboard.AddRoutes(mux, logger, config, logStore, broker, appCtx)

	var handler http.Handler = mux
	// add middlewares if any

	return handler
}

func mustTraceID(hexStr string) pcommon.TraceID {
	var id [16]byte
	hex.Decode(id[:], []byte(hexStr))
	return pcommon.TraceID(id)
}

func mustSpanID(hexStr string) pcommon.SpanID {
	var id [8]byte
	hex.Decode(id[:], []byte(hexStr))
	return pcommon.SpanID(id)
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
		if !config.SeedSystem {
			return
		}

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		headers := map[string][]string{
			"Content-Type": {"application/json"},
		}

		for {
			select {
			case <-ticker.C:
				now := pcommon.Timestamp(time.Now().UnixNano())

				authLogs := plog.NewLogs()
				authRL := authLogs.ResourceLogs().AppendEmpty()
				authRL.Resource().Attributes().PutStr("service.name", "auth-service")
				authRL.Resource().Attributes().PutStr("environment", "production")
				authRL.Resource().Attributes().PutStr("host.name", "auth-worker-01")
				authSL := authRL.ScopeLogs().AppendEmpty()

				// Auth Log 1
				lr1 := authSL.LogRecords().AppendEmpty()
				lr1.SetTimestamp(now)
				lr1.SetTraceID(mustTraceID("5b8aa5a2d2c8646c14e138a83416a41f"))
				lr1.SetSpanID(mustSpanID("f96ea2a71a065463"))
				lr1.SetSeverityText("INFO")
				lr1.SetSeverityNumber(plog.SeverityNumberInfo)
				lr1.Body().SetStr("User authenticated successfully.")
				lr1.Attributes().PutStr("user_id", "usr_987654321")
				lr1.Attributes().PutStr("ip_address", "192.168.1.104")

				// Auth Log 2
				lr2 := authSL.LogRecords().AppendEmpty()
				lr2.SetTimestamp(now)
				lr2.SetTraceID(mustTraceID("1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d"))
				lr2.SetSpanID(mustSpanID("a1b2c3d4e5f6a7b8"))
				lr2.SetSeverityText("FATAL")
				lr2.SetSeverityNumber(plog.SeverityNumberError)
				lr2.Body().SetStr("Failed to connect to database cache.")
				lr2.Attributes().PutStr("cache_host", "redis-cluster.local")
				lr2.Attributes().PutStr("timeout_ms", "5000")

				// Auth Log 3
				lr3 := authSL.LogRecords().AppendEmpty()
				lr3.SetTimestamp(now)
				lr3.SetTraceID(mustTraceID("aabbccddeeff00112233445566778899"))
				lr3.SetSpanID(mustSpanID("b1c2d3e4f5a6b7c8"))
				lr3.SetSeverityText("DEBUG")
				lr3.SetSeverityNumber(plog.SeverityNumberDebug)
				lr3.Body().SetStr("Cache lookup attempted for session token.")
				lr3.Attributes().PutStr("session_id", "sess_112233445")
				lr3.Attributes().PutStr("cache_key", "sess_112233445")

				// Auth Log 4
				lr4 := authSL.LogRecords().AppendEmpty()
				lr4.SetTimestamp(now)
				lr4.SetTraceID(mustTraceID("deadbeefcafebabe1234567890abcdef"))
				lr4.SetSpanID(mustSpanID("c3d4e5f6a7b8c9d0"))
				lr4.SetSeverityText("WARNING")
				lr4.SetSeverityNumber(plog.SeverityNumberWarn)
				lr4.Body().SetStr("Auth token expiring soon, refresh recommmended")
				lr4.Attributes().PutStr("user_id", "usr_112233445")
				lr4.Attributes().PutStr("expires_in_seconds", "120")

				authReq := plogotlp.NewExportRequestFromLogs(authLogs)
				authPayload, err := authReq.MarshalJSON()
				if err != nil {
					natsLogger.Error("failed to marshal auth log payload:", "err", err)
					continue
				}

				if err := broker.PublishLogs(ctx, "logs.auth-service", authPayload, headers); err != nil {
					natsLogger.Error("publish error:", "err", err)
				}

				// --- 2. BUILD LOGGING SERVICE LOGS ---
				logLogs := plog.NewLogs()
				logRL := logLogs.ResourceLogs().AppendEmpty()
				logRL.Resource().Attributes().PutStr("service.name", "logging-service")
				logRL.Resource().Attributes().PutStr("environment", "production")
				logRL.Resource().Attributes().PutStr("host.name", "logging-worker-01")
				logSL := logRL.ScopeLogs().AppendEmpty()

				// Logging Log 1
				lr5 := logSL.LogRecords().AppendEmpty()
				lr5.SetTimestamp(now)
				lr5.SetTraceID(mustTraceID("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"))
				lr5.SetSpanID(mustSpanID("d1e2f3a4b5c6d7e8"))
				lr5.SetSeverityText("INFO")
				lr5.SetSeverityNumber(plog.SeverityNumberInfo)
				lr5.Body().SetStr("Log pipeline started successfully.")
				lr5.Attributes().PutStr("pipeline_id", "pipe_001")
				lr5.Attributes().PutStr("source", "otel-collector")

				// Logging Log 2
				lr6 := logSL.LogRecords().AppendEmpty()
				lr6.SetTimestamp(now)
				lr6.SetTraceID(mustTraceID("f1e2d3c4b5a6978869504030201f0e0d"))
				lr6.SetSpanID(mustSpanID("e2f3a4b5c6d7e8f9"))
				lr6.SetSeverityText("TRACE")
				lr6.SetSeverityNumber(plog.SeverityNumberDebug)
				lr6.Body().SetStr("Flushing log buffer to storage backend.")
				lr6.Attributes().PutStr("buffer_size", "512")
				lr6.Attributes().PutStr("backend", "clickhouse")

				// Logging Log 3
				lr7 := logSL.LogRecords().AppendEmpty()
				lr7.SetTimestamp(now)
				lr7.SetTraceID(mustTraceID("0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d"))
				lr7.SetSpanID(mustSpanID("f3a4b5c6d7e8f9a0"))
				lr7.SetSeverityText("WARNING")
				lr7.SetSeverityNumber(plog.SeverityNumberWarn)
				lr7.Body().SetStr("Log ingestion rate approaching limit")
				lr7.Attributes().PutStr("current_rate", "4800")
				lr7.Attributes().PutStr("limit", "5000")

				// Logging Log 4
				lr8 := logSL.LogRecords().AppendEmpty()
				lr8.SetTimestamp(now)
				lr8.SetTraceID(mustTraceID("1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f"))
				lr8.SetSpanID(mustSpanID("a4b5c6d7e8f9a0b1"))
				lr8.SetSeverityText("ERROR")
				lr8.SetSeverityNumber(plog.SeverityNumberError)
				lr8.Body().SetStr("Failed to write log batch to storage.")
				lr8.Attributes().PutStr("batch_id", "batch_20260608_001")
				lr8.Attributes().PutStr("error", "connection timeout")

				logReq := plogotlp.NewExportRequestFromLogs(logLogs)
				logPayload, err := logReq.MarshalJSON()
				if err != nil {
					natsLogger.Error("failed to marshal logging log payload:", "err", err)
					continue
				}

				if err := broker.PublishLogs(ctx, "logs.logging-service", logPayload, headers); err != nil {
					natsLogger.Error("publish error:", "err", err)
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	chConfig := storage.Config{
		Address:  config.DBAddress,
		Database: config.DBName,
		Username: config.DBUser,
		Password: config.DBPassword,
	}

	logStore, err := storage.NewClickHouseStore(ctx, chConfig, storage.WithLogger(databaseLogger))
	if err != nil {
		databaseLogger.Error("failed to connect to db", "err", err)
		panic(err)
	}

	// seeding database with dummy data
	if config.SeedSystem {
		if err := logStore.InitDB(ctx); err != nil {
			databaseLogger.Error("initdb failed", "err", err)
		}
	}

	srv := NewServer(httpLogger, config, logStore, broker, ctx)

	httpServer := &http.Server{
		Addr:    net.JoinHostPort(config.AppHost, config.AppPort),
		Handler: srv,
	}

	// Bind to the existing jetstream consumer and start consuming messages, mainly used to
	// extract out the consumer info to be saved into db to be displayed on the frontend
	jetStreamConsumer, err := broker.GetJetstreamConsumer(ctx, transport.LogStreamName, worker.WorkerName)
	if err != nil {
		natsLogger.Error("failed to load jetstream consumer", "err", err)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		ticker := time.NewTicker(15 * time.Second) // currently, it refetches info from consumer every 15 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// attempt to extract jetstream consumer info
				info, err := jetStreamConsumer.Info(ctx)
				if err != nil {
					natsLogger.Error("failed to get consumer info", "err", err)
					continue
				}

				natsQueueDepthConsumerInfo := core.NatsQueueDepthConsumerInfo{
					Name:           info.Name,
					Stream:         info.Stream,
					NumPending:     info.NumPending,
					NumAckPending:  uint64(info.NumAckPending),
					NumRedelivered: uint64(info.NumRedelivered),
				}

				// attempt to save the info into the database
				if err := logStore.SaveConsumerInfo(ctx, natsQueueDepthConsumerInfo); err != nil {
					databaseLogger.Error("failed to write consumer info to database", "err", err)
				}

			case <-ctx.Done():
				natsLogger.Info("stopping consumer info polling")
				return
			}
		}
	})

	go func() {
		httpLogger.Info("listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpLogger.Error("http server failed", "err", err)
		}
	}()

	wg.Go(func() {
		// shutdown job
		<-ctx.Done()

		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer cancel()

		httpLogger.Info("Shutting down server")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			httpLogger.Error("http server shutting down failed", "err", err)
		}
	})

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
