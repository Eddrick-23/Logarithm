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
	"github.com/Eddrick-23/Logarithm/internal/dashboard"
	"github.com/Eddrick-23/Logarithm/internal/storage"
)

func NewServer(logger *slog.Logger, config *config.Config) http.Handler {
	mux := http.NewServeMux()
	dashboard.AddRoutes(mux, logger)

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

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	config, err := config.LoadConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logStore, err := storage.NewClickHouseStore(ctx, config.DBAddress, config.DBName, config.DBTableName, config.DBUser, config.DBPassword)
	if err != nil {
		databaseLogger.Error("failed to connect to db", "err", err)
		panic(err)
	}

	// seeding database with dummy data
	if err := logStore.InitDB(ctx); err != nil {
		databaseLogger.Error("initdb failed", "err", err)
	}

	srv := NewServer(httpLogger, config)

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
