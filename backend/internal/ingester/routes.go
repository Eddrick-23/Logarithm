package ingester

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/transport"
)

func AddRoutes(
	mux *http.ServeMux,
	logger *slog.Logger,
	producer transport.Producer,
	natsSubjectPrefix string,
) {
	mux.HandleFunc("GET /", handleRoot(logger))
	mux.HandleFunc("GET /health", handleHealth(logger))
	mux.Handle("POST /ingest", contentTypeMiddleware(gzipMiddleware(handleIngest(logger, producer, natsSubjectPrefix))))
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "gzip" {
			next.ServeHTTP(w, r)
			return
		}
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "Invalid gzip body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		defer gz.Close()

		r.Body = io.NopCloser(gz)        // r.Body.Close() will be handled here only
		r.Header.Del("Content-Encoding") // prevent double decodes

		next.ServeHTTP(w, r)
	})
}

func contentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			http.Error(w, "Malformed/Missing Content-Type", http.StatusBadRequest)
			return
		}

		if mediaType != "application/json" {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleRoot(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintln(w, "Ingestor API root!")

		if err != nil {
			logger.Error("failed to write response", "err", err)
		}
	}
}

func handleHealth(logger *slog.Logger) http.HandlerFunc {
	type response struct {
		Health string `json:"health"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		bytes, err := json.Marshal(&response{Health: "ok"})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to marshall json: %v", err), http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(bytes)
		if err != nil {
			logger.Error("failed to write response", "err", err)
		}
	}
}

func handleIngest(logger *slog.Logger, producer transport.Producer, natsSubjectTemplate string) http.HandlerFunc {
	// TODO push extra data to time how long it took to transfer payload
	// needed for latency numbers on the dashboard
	type partialIngestBody struct {
		ServiceName string `json:"serviceName"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		bytesPayload, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("failed to parse request body", "err", err)
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		var ingestBody partialIngestBody
		if err = json.Unmarshal(bytesPayload, &ingestBody); err != nil {
			logger.Error("invalid JSON body", "err", err)
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		if ingestBody.ServiceName == "" {
			http.Error(w, "No serviceName in payload", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err = producer.PublishLogs(ctx, natsSubjectTemplate+ingestBody.ServiceName, bytesPayload); err != nil {
			logger.Error("Error publishing to nats jetstream", "err", err)
			http.Error(w, "Error transporting json", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		if _, err = w.Write([]byte("Log ingested successfully")); err != nil {
			logger.Error("failed to write response", "err", err)
		}
	}
}
