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
	"github.com/stretchr/testify/assert"
)

type MockProducer struct {
	Err error
}

func (m *MockProducer) PublishLogs(context.Context, string, []byte) error {
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

func marshalAndZipPayload(t *testing.T, payload any) ([]byte, []byte) {
	t.Helper()
	jsonBytes, err := json.Marshal(payload)

	if err != nil {
		t.Fatalf("error marshalling json body: %v", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err = gz.Write(jsonBytes)

	if err != nil {
		t.Fatalf("error writing gzip data: %v", err)
	}

	if err = gz.Close(); err != nil {
		t.Fatalf("error closing gzip writer: %v", err)
	}

	return jsonBytes, buf.Bytes()
}

func TestIngestEndpoint(t *testing.T) {
	validBodyBytes, validGzipped := marshalAndZipPayload(t, testJsonPayload)

	tests := []struct {
		name           string
		contentType    string
		body           []byte
		gzipped        bool
		producerErr    error
		expectedStatus int
		expectedBody   string
	}{
		{"valid json", "application/json", validBodyBytes, false, nil, http.StatusAccepted, "Log ingested successfully"},
		{"valid json gzip", "application/json", validGzipped, true, nil, http.StatusAccepted, "Log ingested successfully"},
		{"wrong content type", "text/plain", validBodyBytes, false, nil, http.StatusUnsupportedMediaType, "Content-Type must be application/json"},
		{"missing content type", "", validBodyBytes, false, nil, http.StatusBadRequest, "Malformed/Missing Content-Type"},
		{"invalid gzip body", "application/json", []byte(`notgzip`), true, nil, http.StatusBadRequest, "Invalid gzip body"},
		{"publish failed", "application/json", validBodyBytes, false, fmt.Errorf("publish to nats failed"), http.StatusServiceUnavailable, "Message broker unavailable"},
		{"gzip publish failed", "application/json", validGzipped, true, fmt.Errorf("publish to nats failed"), http.StatusServiceUnavailable, "Message broker unavailable"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			app := setupTestApp(tc.producerErr)
			req := httptest.NewRequest("POST", "/v1/logs", bytes.NewReader(tc.body))
			req.Header.Set("Content-type", tc.contentType)
			if tc.gzipped {
				req.Header.Set("Content-Encoding", "gzip")
			}

			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tc.expectedBody)
		})
	}
}
