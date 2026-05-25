package ingester

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Eddrick-23/Logarithm/internal/transport"
)

func AddRoutes(
	mux *http.ServeMux,
	logger *slog.Logger,
	producer transport.Producer,
) {
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/health", handleHealth())
	mux.HandleFunc("/ingest", handleIngest(logger, producer)) // has dependency: producer
	mux.HandleFunc("/api/data", apiDataHandler)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintln(w, "Ingestor API root!")

	if err != nil {
		slog.Error("failed to write response", "err", err)
	}
}

func handleHealth() http.HandlerFunc {
	type response struct {
		Health string `json:"health"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		bytes, err := json.Marshal(&response{Health: "ok"})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to marshall json: %v", err), http.StatusInternalServerError)
		}
		fmt.Println()
		_, err = w.Write(bytes)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
		}
	}
}

func handleIngest(logger *slog.Logger, producer transport.Producer) http.HandlerFunc {
	// should this be POST/PUT?
	// receive payload, unmarshall to core.LogIngestRequest
	// publish payload into nats jetstream
	// return immediately

	// TODO push extra data to time how long it took to transfer payload
	// needed for latency numbers on the dashboard
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func apiDataHandler(w http.ResponseWriter, r *http.Request) {
	data := "Some data from the API"
	_, err := fmt.Fprintln(w, data)

	if err != nil {
		slog.Error("failed to write response", "err", err)
	}
}
