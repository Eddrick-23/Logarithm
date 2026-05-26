package ingester

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/transport"
)

func AddRoutes(
	mux *http.ServeMux,
	logger *slog.Logger,
	producer transport.Producer,
	natsSubjectPrefix string,
) {
	mux.HandleFunc("GET /", handleRoot(logger))
	mux.HandleFunc("GET /health", handleHealth())
	mux.HandleFunc("POST /ingest", handleIngest(logger, producer, natsSubjectPrefix))
}

func handleRoot(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintln(w, "Ingestor API root!")

		if err != nil {
			logger.Error("failed to write response", "err", err)
		}
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

func handleIngest(logger *slog.Logger, producer transport.Producer, natsSubjectTemplate string) http.HandlerFunc {
	// TODO1 optimisations, partial json decode to extract service name only, then immediate push
	// TODO2 push extra data to time how long it took to transfer payload
	// needed for latency numbers on the dashboard
	return func(w http.ResponseWriter, r *http.Request) {
		var payload core.LogIngestRequest

		err := json.NewDecoder(r.Body).Decode(&payload)

		if err != nil {
			logger.Error("Json decode error", "err", err)
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		bytesPayload, err := json.Marshal(payload)
		if err != nil {
			logger.Error("Error marshalling json", "err", err)
			http.Error(w, "Error marshalling json", http.StatusInternalServerError)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		streamSubject := natsSubjectTemplate + payload.ServiceName
		err = producer.PublishLogs(ctx, streamSubject, bytesPayload)

		if err != nil {
			logger.Error("Error publishing to nats jetstream", "err", err)
			http.Error(w, "Error transporting json", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte("log ingested successfully"))

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
		}
	}
}
