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
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/ingester"
	"github.com/stretchr/testify/assert"
)

type MockProducer struct {
	Err error
}

func (m *MockProducer) PublishLogs(context.Context, string, []byte) error {
	return m.Err
}

var testLogRecordDTO core.LogRecordDTO = core.LogRecordDTO{
	Timestamp:      time.Date(2024, 5, 20, 10, 0, 0, 0, time.UTC),
	TraceId:        "4bf92f3577b34da6a3ce929d0e0e4736",
	SpanId:         "00f067aa0ba902b7",
	SeverityText:   "ERROR",
	SeverityNumber: 17,
	Body:           "Failed to process transaction due to timeout",
	LogAttributes:  []core.KeyValue{{Key: "http.method", Value: "POST"}},
}

var testIngestRequest core.LogIngestRequest = core.LogIngestRequest{
	ServiceName:        "test-service",
	ResourceAttributes: []core.KeyValue{{Key: "host.name", Value: "prod-payment-02"}},
	Records:            []core.LogRecordDTO{testLogRecordDTO},
}
var testIngestRequestNoServiceName core.LogIngestRequest = core.LogIngestRequest{
	ServiceName:        "",
	ResourceAttributes: []core.KeyValue{{Key: "host.name", Value: "prod-payment-02"}},
	Records:            []core.LogRecordDTO{testLogRecordDTO},
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
	validBodyBytes, validGzipped := marshalAndZipPayload(t, testIngestRequest)
	bodyMissingServiceNameBytes, missingGzipped := marshalAndZipPayload(t, testIngestRequestNoServiceName)

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
		{"missing service name", "application/json", bodyMissingServiceNameBytes, false, nil, http.StatusBadRequest, "No serviceName in payload"},
		{"missing service name gzip", "application/json", missingGzipped, true, nil, http.StatusBadRequest, "No serviceName in payload"},
		{"wrong content type", "text/plain", validBodyBytes, false, nil, http.StatusUnsupportedMediaType, "Content-Type must be application/json"},
		{"missing content type", "", validBodyBytes, false, nil, http.StatusBadRequest, "Malformed/Missing Content-Type"},
		{"invalid json", "application/json", []byte(`{bad}`), false, nil, http.StatusBadRequest, "Invalid JSON body"},
		{"invalid gzip body", "application/json", []byte(`notgzip`), true, nil, http.StatusBadRequest, "Invalid gzip body"},
		{"valid json publish failed", "application/json", validBodyBytes, false, fmt.Errorf("publish to nats failed"), http.StatusInternalServerError, "Error transporting json"},
		{"valid json gzip publish failed", "application/json", validGzipped, true, fmt.Errorf("publish to nats failed"), http.StatusInternalServerError, "Error transporting json"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			app := setupTestApp(tc.producerErr)
			req := httptest.NewRequest("POST", "/ingest", bytes.NewReader(tc.body))
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
