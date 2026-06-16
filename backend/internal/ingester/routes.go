package ingester

import (
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
	mux.Handle("POST /v1/logs",
		newContentTypeMiddleware("application/json", "application/x-protobuf")(
			newContentEncodingMiddleware("gzip", "zstd")(
				handleOTLPLogs(logger, producer, natsSubjectPrefix),
			),
		),
	)
}

// TODO allow protobuf bytes in the future, now assume all json bytes only
func newContentTypeMiddleware(supportedTypes ...string) func(http.Handler) http.Handler {
	supported := make(map[string]struct{}, len(supportedTypes))
	for _, s := range supportedTypes {
		supported[s] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")
			mediaType, _, err := mime.ParseMediaType(contentType)
			if err != nil {
				http.Error(w, "Malformed/Missing Content-Type", http.StatusBadRequest)
				return
			}

			if _, ok := supported[mediaType]; !ok {
				http.Error(w, fmt.Sprintf("Unsupported Content-Type: %s", mediaType), http.StatusUnsupportedMediaType)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func newContentEncodingMiddleware(supportedEncodings ...string) func(http.Handler) http.Handler {
	supported := make(map[string]struct{}, len(supportedEncodings))

	for _, s := range supportedEncodings {
		supported[s] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentEncoding := r.Header.Get("Content-Encoding")
			if contentEncoding == "" {
				next.ServeHTTP(w, r)
				return
			}

			if _, ok := supported[contentEncoding]; !ok {
				http.Error(w, fmt.Sprintf("Unsupported Content-Encoding: %s", contentEncoding), http.StatusUnsupportedMediaType)
			}
			next.ServeHTTP(w, r)
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
