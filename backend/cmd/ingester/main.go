package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Eddrick-23/OrbitalTest/internal/config"
	"github.com/Eddrick-23/OrbitalTest/internal/ingester"
)

func main() {
	router := ingester.NewRouter()
	config := config.LoadConfig()

	port, err := strconv.Atoi(config.IngesterPort)
	if err != nil {
		slog.Error("Invalid non integer port provided, falling back to port 8090")
		port = 8090
	}
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Server listening on http://localhost%s\n", addr)
	err = http.ListenAndServe(addr, router)
	if err != nil {
		panic(err)
	}
}
