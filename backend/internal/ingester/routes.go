package ingester

import (
	"compress/gzip"

	"github.com/klauspost/compress/zstd"

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
	mux.Handle("POST /v1/logs", newContentTypeMiddleware("application/json")(
		decompressMiddleware(handleOTLPLogs(logger, producer, natsSubjectPrefix))))
}

func decompressMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoding := r.Header.Get("Content-Encoding")

		if encoding == "" || encoding == "identity" {
			next.ServeHTTP(w, r)
			return
		}

		var uncompressedBody io.ReadCloser
		var err error
		switch encoding {
		case "gzip":
			uncompressedBody, err = gzip.NewReader(r.Body)
		case "zstd":
			var decoder *zstd.Decoder
			decoder, err = zstd.NewReader(r.Body)
			if err == nil {
				uncompressedBody = decoder.IOReadCloser()
			}
		default:
			http.Error(w, "Unsupported Content-Encoding format", http.StatusUnsupportedMediaType)
			return
		}

		if err != nil {
			http.Error(w, "Malformed compressed body", http.StatusBadRequest)
			return
		}

		defer r.Body.Close()
		defer uncompressedBody.Close()

		r.Body = io.NopCloser(uncompressedBody)
		r.ContentLength = -1
		r.Header.Del("Content-Encoding")

		next.ServeHTTP(w, r)
	})
}

func newContentTypeMiddleware(targetMediaType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")
			mediaType, _, err := mime.ParseMediaType(contentType)
			if err != nil {
				http.Error(w, "Malformed/Missing Content-Type", http.StatusBadRequest)
				return
			}

			if mediaType != targetMediaType {
				http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
				return
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

		// TODO add routing based on content-type in the future
		// now we assume all is json payload

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := producer.PublishLogs(ctx, natsSubjectTemplate+"raw", bodyBytes); err != nil {
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
