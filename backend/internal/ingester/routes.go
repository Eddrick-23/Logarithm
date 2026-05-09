package ingester

import (
	"fmt"
	"log/slog"
	"net/http"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/api/data", apiDataHandler)

	return mux;
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintln(w, "Ingestor API root!")

	if err != nil {
		slog.Error("failed to write response", "err", err)
	}
}

func apiDataHandler(w http.ResponseWriter, r *http.Request) {
	data := "Some data from the API"
	_, err := fmt.Fprintln(w, data)
	
	if err != nil {
		slog.Error("failed to write response", "err", err)
	}
}
