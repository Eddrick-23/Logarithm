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
	"sync"
	"syscall"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/config"
	"github.com/Eddrick-23/Logarithm/internal/ingester"
	"github.com/Eddrick-23/Logarithm/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
)

func NewHTTPServer(logger *slog.Logger, producer transport.Producer) http.Handler {
	mux := http.NewServeMux()
	ingester.AddRoutes(mux, logger, producer, transport.LogStreamSubjectPrefix)

	var handler http.Handler = mux
	// add middlewares if any

	return handler
}

func NewGRPCServer(logger *slog.Logger, producer transport.Producer) *grpc.Server {
	proxyHandler := ingester.NewProxyHandler(logger, producer, transport.LogStreamSubject)

	grpcServer := grpc.NewServer(
		grpc.ForceServerCodecV2(encoding.GetCodecV2(ingester.CodecName)),
		grpc.UnknownServiceHandler(proxyHandler.StreamHandler),
	)

	return grpcServer
}

func startPprof(logger *slog.Logger, config *config.Config) {
	if !config.EnablePprof {
		return
	}
	runtime.SetMutexProfileFraction(100)
	runtime.SetBlockProfileRate(100000)
	addr := net.JoinHostPort(config.PprofHost, "6060")
	go func() {
		logger.Info("pprof listening on", "addr", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			logger.Error("pprof server stopped", "err", err)
		}
	}()
}

func run(ctx context.Context, w io.Writer, args []string) error {
	logger := slog.New(
		slog.NewTextHandler(w, nil),
	)
	natsLogger := logger.With("component", "nats")
	httpLogger := logger.With("component", "ingester_http")
	grpcLogger := logger.With("component", "ingester_grpc")
	pprofLogger := logger.With("component", "pprof")

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	config, err := config.LoadConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	startPprof(pprofLogger, config)

	natsBroker, err := transport.NewNatsBroker(ctx, natsLogger, config.NatsURL)

	if err != nil {
		return fmt.Errorf("failed to crate nats broker: %w", err)
	}

	if _, err = natsBroker.EnsureLogStream(ctx,
		transport.LogStreamName, transport.LogStreamSubject,
		config.NatsStreamMaxAge, int64(config.NatsLogStreamMaxBytes)); err != nil {
		return fmt.Errorf("failed to ensure stream: %w", err)
	}

	srv := NewHTTPServer(httpLogger, natsBroker)

	httpServer := &http.Server{
		Addr:              net.JoinHostPort(config.IngesterHost, config.IngesterPortHTTP),
		Handler:           srv,
		ReadHeaderTimeout: config.IngesterReadHeaderTimeout,
		ReadTimeout:       config.IngesterReadTimeout,
		WriteTimeout:      config.IngesterWriteTimeout,
		IdleTimeout:       config.IngesterIdleTimeout,
	}

	grpcServer := NewGRPCServer(grpcLogger, natsBroker)

	// start http and grpc servers
	go func() {
		httpLogger.Info("listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpLogger.Error("http server failed", "err", err)
		}
	}()

	go func() {
		grpcLogger.Info("listening", "addr", "tcp"+":"+config.IngesterPortGRPC)
		lis, err := net.Listen("tcp", ":"+config.IngesterPortGRPC)
		if err != nil {
			grpcLogger.Error("failed to listen", "err", err)
		}
		if err := grpcServer.Serve(lis); err != nil {
			grpcLogger.Error("grpc server failed", "err", err)
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

		httpLogger.Info("Shutting down http server")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			httpLogger.Error("http server shutting down failed", "err", err)
		}

		grpcLogger.Info("Shutting down grpc server")
		grpcServer.GracefulStop()
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
