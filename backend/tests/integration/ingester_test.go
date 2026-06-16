//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Eddrick-23/Logarithm/internal/ingester"
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

func setupTestApp(producerErr error) http.Handler {
	mockProducer := &MockProducer{producerErr}
	mux := http.NewServeMux()
	ingester.AddRoutes(mux, slog.Default(), mockProducer, "logs.")
	return mux
}

func TestIngestEndpoint(t *testing.T) {
	dummyBody := []byte("data")
	tests := []struct {
		name           string
		contentType    string
		encoding       string
		producerErr    error
		expectedStatus int
		expectedBody   string
	}{
		{"json no encoding", "application/json", "", nil, http.StatusAccepted, "Log ingested successfully"},
		{"json gzip encoding", "application/json", "gzip", nil, http.StatusAccepted, "Log ingested successfully"},
		{"json with zstd encoding", "application/json", "zstd", nil, http.StatusAccepted, "Log ingested successfully"},
		{"protobuf no encoding", "application/x-protobuf", "", nil, http.StatusAccepted, "Log ingested successfully"},
		{"protobuf gzip encoding", "application/x-protobuf", "gzip", nil, http.StatusAccepted, "Log ingested successfully"},
		{"protobuf with zstd encoding", "application/x-protobuf", "zstd", nil, http.StatusAccepted, "Log ingested successfully"},
		{"unsupported content type", "text/plain", "", nil, http.StatusUnsupportedMediaType, "Unsupported Content-Type: text/plain"},
		{"missing content type", "", "", nil, http.StatusBadRequest, "Malformed/Missing Content-Type"},
		{"json publish failed", "application/json", "", fmt.Errorf("publish to nats failed"), http.StatusServiceUnavailable, "Message broker unavailable"},
		{"json unknown encoding", "application/json", "br", nil, http.StatusUnsupportedMediaType, "Unsupported Content-Encoding: br"},
		{"protobuf publish failed", "application/x-protobuf", "", fmt.Errorf("publish to nats failed"), http.StatusServiceUnavailable, "Message broker unavailable"},
		{"protobuf unknown encoding", "application/x-protobuf", "br", nil, http.StatusUnsupportedMediaType, "Unsupported Content-Encoding: br"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			app := setupTestApp(tc.producerErr)
			req := httptest.NewRequest("POST", "/v1/logs", bytes.NewReader(dummyBody))
			req.Header.Set("Content-type", tc.contentType)
			req.Header.Set("Content-Encoding", tc.encoding)

			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tc.expectedBody)
		})
	}
}
