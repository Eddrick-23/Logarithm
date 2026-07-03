package ingester

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/transport"
)

func NewHTTPServer(logger *slog.Logger, producer transport.Producer) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, logger, producer, transport.LogStreamSubjectPrefix)

	var handler http.Handler = mux
	// add middlewares if any

	return handler
}

func addRoutes(
	mux *http.ServeMux,
	logger *slog.Logger,
	producer transport.Producer,
	natsSubjectPrefix string,
) {
	mux.HandleFunc("GET /", handleRoot(logger))
	mux.HandleFunc("GET /health", handleHealth(logger))
	mux.Handle("POST /v1/logs",
		newContentTypeMiddleware()(
			newContentEncodingMiddleware()(
				handleOTLPLogs(logger, producer, natsSubjectPrefix),
			),
		),
	)
}

func newContentTypeMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")

			if strings.HasPrefix(contentType, "application/x-protobuf") || strings.HasPrefix(contentType, "application/json") {
				next.ServeHTTP(w, r)
				return
			}

			http.Error(w, fmt.Sprintf("Unsupported or Missing Content-Type: %s", contentType), http.StatusUnsupportedMediaType)
		})
	}
}

func newContentEncodingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentEncoding := r.Header.Get("Content-Encoding")
			if contentEncoding == "" || contentEncoding == "identity" {
				next.ServeHTTP(w, r)
				return
			}

			if contentEncoding == "zstd" || contentEncoding == "gzip" {
				next.ServeHTTP(w, r)
				return
			}

			http.Error(w, fmt.Sprintf("Unsupported Content-Encoding: %s", contentEncoding), http.StatusUnsupportedMediaType)
		})
	}
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

func handleOTLPLogs(logger *slog.Logger, producer transport.Producer, natsSubjectTemplate string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		headers := map[string][]string{}
		if ct := r.Header.Get("Content-Type"); ct != "" {
			headers["Content-Type"] = []string{ct}
		}

		if ce := r.Header.Get("Content-Encoding"); ce != "" {
			headers["Content-Encoding"] = []string{ce}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := producer.PublishLogs(ctx, natsSubjectTemplate+"raw", bodyBytes, headers); err != nil {
			logger.Error("failed to publish to nats", "err", err)
			http.Error(w, "Message broker unavailable", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		if _, err := w.Write([]byte("Log ingested successfully")); err != nil {
			logger.Error("failed to write response", "err", err)
		}
	}
}
