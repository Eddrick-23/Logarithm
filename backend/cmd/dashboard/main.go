package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Eddrick-23/Logarithm/internal/config"
	"github.com/Eddrick-23/Logarithm/internal/dashboard"
)

func main() {
	router := dashboard.NewRouter()
	config := config.LoadConfig()

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
