package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/Eddrick-23/OrbitalTest/internal/config"
	"github.com/Eddrick-23/OrbitalTest/internal/ingester"
	"github.com/Eddrick-23/OrbitalTest/internal/transport"
)

func NewServer(config *config.Config, producer transport.Producer) http.Handler {
	mux := http.NewServeMux()
	ingester.AddRoutes(mux, producer)
	
	var handler http.Handler = mux
	// add middlewares if any

	return handler
}

func run(ctx context.Context, w io.Writer, args []string) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	config := config.LoadConfig()
	slog.Info(config.NatsURL)
	nb, err := transport.NewNatsBroker(ctx, config.NatsURL)

	if err != nil {
		return fmt.Errorf("Failed to create Nats Broker: %v", err)
	}

	srv := NewServer(config, nb)

	httpServer := &http.Server{
		Addr: fmt.Sprintf("localhost:%v", config.IngesterPort),
		Handler: srv,
	}

	go func() { // start server in a go routine
		slog.Info(fmt.Sprintf("listening on %s\n", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	// TODO check what is correct way to implement shutdown
	go func() { // shutdown job
		defer wg.Done()
		defer nb.Close() // close nats connection
		<- ctx.Done()
		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10 * time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
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
