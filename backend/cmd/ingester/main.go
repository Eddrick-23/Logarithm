package main

import (
	"context"
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
	"github.com/Eddrick-23/Logarithm/internal/ingester"
	"github.com/Eddrick-23/Logarithm/internal/transport"
)

func NewServer(logger *slog.Logger, config *config.Config, producer transport.Producer) http.Handler {
	mux := http.NewServeMux()
	ingester.AddRoutes(mux, logger, producer, config.NatsPublishSubjectPrefix)

	var handler http.Handler = mux
	// add middlewares if any

	return handler
}

func run(ctx context.Context, w io.Writer, args []string) error {
	logger := slog.New(
		slog.NewTextHandler(w, nil),
	)
	natsLogger := logger.With("component", "nats")
	httpLogger := logger.With("component", "ingester")

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	config := config.LoadConfig()
	natsBroker, err := transport.NewNatsBroker(ctx, natsLogger, config.NatsURL)

	if err != nil {
		return fmt.Errorf("Failed to create Nats Broker: %w", err)
	}

	if _, err = natsBroker.EnsureStream(ctx, config.NatsSubject); err != nil {
		return fmt.Errorf("failed to ensure stream: %w", err)
	}

	srv := NewServer(httpLogger, config, natsBroker)

	httpServer := &http.Server{
		Addr:    net.JoinHostPort(config.IngesterHost, config.IngesterPort),
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

	natsBroker.Close() // close nats connection

	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stdout, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
