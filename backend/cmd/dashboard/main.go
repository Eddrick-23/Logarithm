package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Eddrick-23/Logarithm/internal/config"
	"github.com/Eddrick-23/Logarithm/internal/dashboard"
	"github.com/Eddrick-23/Logarithm/internal/storage"
)

func main() {
	ctx := context.Background()
	router := dashboard.NewRouter()
	config, err := config.LoadConfig(ctx)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		panic(err)
	}

	logStore, err := storage.NewClickHouseStore(ctx, config.DBAddress, config.DBName, config.DBTableName, config.DBUser, config.DBPassword)
	if err != nil {
		slog.Error("failed to connect to db", "err", err)
		panic(err)
	}

	// seeding database with dummy data
	if err := logStore.InitDB(ctx); err != nil {
		slog.Error("initdb failed", "err", err)
	}

	port, err := strconv.Atoi(config.AppPort)
	if err != nil {
		slog.Error("Invalid non integer port provided, falling back to port 8091")
		port = 8091
	}
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Server listening on http://localhost%s\n", addr)
	err = http.ListenAndServe(addr, router)
	if err != nil {
		panic(err)
	}
}
