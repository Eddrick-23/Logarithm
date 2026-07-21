package ingester

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/transport"
)

// NewHTTPServer constructs the ingester's HTTP handler, wiring the OTLP
// log-ingestion routes defined in AddRoutes.
func NewHTTPServer(logger *slog.Logger, producer transport.Producer, presizeBuffer int64, bufferLimit int64) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, logger, producer, transport.LogStreamSubjectPrefix, presizeBuffer, bufferLimit)

	var handler http.Handler = mux
	// add middlewares if any

	return handler
}

func addRoutes(
	mux *http.ServeMux,
	logger *slog.Logger,
	producer transport.Producer,
	natsSubjectPrefix string,
	presizeBuffer int64,
	bufferLimit int64,
) {
	mux.HandleFunc("GET /", handleRoot(logger))
	mux.HandleFunc("GET /health", handleHealth(logger))
	mux.Handle("POST /v1/logs",
		newContentTypeMiddleware()(
			newContentEncodingMiddleware()(
				handleOTLPLogs(logger, producer, natsSubjectPrefix, presizeBuffer, bufferLimit),
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

func handleOTLPLogs(logger *slog.Logger, producer transport.Producer, natsSubjectTemplate string, presize int64, limit int64) http.HandlerFunc {
	bufPool := sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, 0, presize))
		},
	}

	natsSubject := natsSubjectTemplate + "raw"
	return func(w http.ResponseWriter, r *http.Request) {
		buf := bufPool.Get().(*bytes.Buffer)
		buf.Reset()
		defer func() {
			if int64(buf.Cap()) <= limit {
				bufPool.Put(buf)
			}
		}()
		defer r.Body.Close()

		if _, err := buf.ReadFrom(r.Body); err != nil {
			logger.Error("failed to read request body", "err", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		headers := map[string][]string{}
		if ct := r.Header.Get("Content-Type"); ct != "" {
			headers["Content-Type"] = []string{ct}
		}

		if ce := r.Header.Get("Content-Encoding"); ce != "" {
			headers["Content-Encoding"] = []string{ce}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := producer.PublishLogs(ctx, natsSubject, buf.Bytes(), headers); err != nil {
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
