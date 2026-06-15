//go:build integration

package integration

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Eddrick-23/Logarithm/internal/ingester"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
)

type MockProducer struct {
	Err error
}

func (m *MockProducer) PublishLogs(context.Context, string, []byte, map[string][]string) error {
	return m.Err
}

func (m *MockProducer) PublishLiveTail(subject string, data []byte) error {
	return nil // not used
}

var testJsonPayload = struct {
	Key   string
	Value string
}{
	Key:   "test",
	Value: "test",
}

func setupTestApp(producerErr error) http.Handler {
	mockProducer := &MockProducer{producerErr}
	mux := http.NewServeMux()
	ingester.AddRoutes(mux, slog.Default(), mockProducer, "logs.")
	return mux
}

func marshalAndZipPayload(t *testing.T, payload any) ([]byte, []byte, []byte) {
	t.Helper()
	jsonBytes, err := json.Marshal(payload)

	if err != nil {
		t.Fatalf("error marshalling json body: %v", err)
	}

	var bufGzip bytes.Buffer
	gz := gzip.NewWriter(&bufGzip)
	_, err = gz.Write(jsonBytes)

	if err != nil {
		t.Fatalf("error writing gzip data: %v", err)
	}

	if err = gz.Close(); err != nil {
		t.Fatalf("error closing gzip writer: %v", err)
	}

	var bufZstd bytes.Buffer
	enc, err := zstd.NewWriter(&bufZstd)

	if err != nil {
		t.Fatalf("error creating zstd writer: %v", err)
	}

	_, err = enc.Write(jsonBytes)

	if err != nil {
		t.Fatalf("error writing zstd data: %v", err)
	}

	if err = enc.Close(); err != nil {
		t.Fatalf("error closing zstd writer: %v", err)
	}

	return jsonBytes, bufGzip.Bytes(), bufZstd.Bytes()
}

func TestIngestEndpoint(t *testing.T) {
	validBodyBytes, validGzipped, validZstd := marshalAndZipPayload(t, testJsonPayload)

	tests := []struct {
		name           string
		contentType    string
		body           []byte
		encoding       string
		producerErr    error
		expectedStatus int
		expectedBody   string
	}{
		{"valid json", "application/json", validBodyBytes, "", nil, http.StatusAccepted, "Log ingested successfully"},
		{"valid json gzip", "application/json", validGzipped, "gzip", nil, http.StatusAccepted, "Log ingested successfully"},
		{"valid json zstd", "application/json", validZstd, "zstd", nil, http.StatusAccepted, "Log ingested successfully"},
		{"wrong content type", "text/plain", validBodyBytes, "", nil, http.StatusUnsupportedMediaType, "Content-Type must be application/json"},
		{"missing content type", "", validBodyBytes, "", nil, http.StatusBadRequest, "Malformed/Missing Content-Type"},
		{"invalid gzip body", "application/json", []byte(`notgzip`), "gzip", nil, http.StatusBadRequest, "Malformed compressed body"},
		{"invalid zstd body", "application/json", []byte(`notzstd`), "zstd", nil, http.StatusAccepted, "Log ingested successfully"}, // zstd decompression is lazy, so let worker propagate the error, this passes
		{"publish failed", "application/json", validBodyBytes, "", fmt.Errorf("publish to nats failed"), http.StatusServiceUnavailable, "Message broker unavailable"},
		{"gzip publish failed", "application/json", validGzipped, "gzip", fmt.Errorf("publish to nats failed"), http.StatusServiceUnavailable, "Message broker unavailable"},
		{"zstd publish failed", "application/json", validZstd, "zstd", fmt.Errorf("publish to nats failed"), http.StatusServiceUnavailable, "Message broker unavailable"},
		{"unknown encoding", "application/json", validBodyBytes, "br", nil, http.StatusUnsupportedMediaType, "Unsupported Content-Encoding format"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			app := setupTestApp(tc.producerErr)
			req := httptest.NewRequest("POST", "/v1/logs", bytes.NewReader(tc.body))
			req.Header.Set("Content-type", tc.contentType)
			req.Header.Set("Content-Encoding", tc.encoding)

			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tc.expectedBody)
		})
	}
}
