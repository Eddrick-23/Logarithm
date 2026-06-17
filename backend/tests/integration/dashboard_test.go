//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/config"
	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/dashboard"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func setupDashboardServer(t *testing.T, ctx context.Context) (*httptest.Server, *transport.NatsBroker) {
	t.Helper()

	logStore, err := storage.NewClickHouseStore(ctx, slog.Default(), dbAddr, dbname, user, password)
	if err != nil {
		t.Fatalf("failed to initialize ClickHouse store: %v", err)
	}

	setupTestDB(t, ctx, logStore)

	broker, err := transport.NewNatsBroker(ctx, slog.Default(), natsUrl)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}

	mux := http.NewServeMux()
	dashboard.AddRoutes(mux, slog.Default(), &config.Config{LiveTailRefreshInterval: 50}, logStore, broker, ctx)

	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.Close()
	})

	return server, broker
}

func TestDashboardHandleSearch(t *testing.T) {
	ctx := context.Background()
	server, _ := setupDashboardServer(t, ctx)

	searchTestCases := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedTotal  int    // Meta's totalRowCount
		expectedCount  int    // number of records in Data
		expectedFirst  string // TraceId of the 1st record in the array (to test sorting/matching)
	}{
		{
			name:           "Filter by Severity Text (WARNING)",
			queryParams:    "?severityText=WARNING&descending=false&limit=10&offset=0",
			expectedStatus: http.StatusOK,
			expectedTotal:  1,
			expectedCount:  1,
			expectedFirst:  testRecord2.TraceId,
		},
		{
			name:           "Filter by Body Text",
			queryParams:    "?body=timeout&descending=false&limit=10&offset=0",
			expectedStatus: http.StatusOK,
			expectedTotal:  1,
			expectedCount:  1,
			expectedFirst:  testRecord1.TraceId,
		},
		{
			name:           "Pagination - Limit 2",
			queryParams:    "?descending=false&limit=2&offset=0",
			expectedStatus: http.StatusOK,
			expectedTotal:  len(seedData),
			expectedCount:  2,
			expectedFirst:  testRecord1.TraceId,
		},
		{
			name:           "Pagination - Offset 2",
			queryParams:    "?descending=false&limit=2&offset=2",
			expectedStatus: http.StatusOK,
			expectedTotal:  len(seedData),
			expectedCount:  len(seedData) - 2,
			expectedFirst:  testRecord3.TraceId,
		},
		{
			name:           "Filter by Time Range",
			queryParams:    "?startTime=2024-05-19T00:00:00&endTime=2024-12-31T23:59:59&descending=false&limit=10&offset=0",
			expectedStatus: http.StatusOK,
			expectedTotal:  2,
			expectedCount:  2,
			expectedFirst:  testRecord1.TraceId,
		},
		{
			name:           "Sorting - Descending by Timestamp",
			queryParams:    "?orderBy=timestamp&descending=true&limit=10&offset=0",
			expectedStatus: http.StatusOK,
			expectedTotal:  len(seedData),
			expectedCount:  len(seedData),
			expectedFirst:  testRecord4.TraceId,
		},
		{
			name:           "Bad Request - Invalid Limit",
			queryParams:    "?limit=not-a-number&offset=0",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Bad Request - Invalid Offset",
			queryParams:    "?limit=2&offset=abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Bad Request - Invalid Start Time",
			queryParams:    "?startTime=bad-time-format&limit=10&offset=0",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Bad Request - Invalid End Time",
			queryParams:    "?endTime=very-bad-time-format&limit=10&offset=0",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range searchTestCases {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := server.URL + "/api/search" + tc.queryParams
			resp, err := http.Get(reqURL)
			if err != nil {
				t.Fatalf("failed to make GET request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("expected status %v, got %v", tc.expectedStatus, resp.StatusCode)
				return
			}

			if tc.expectedStatus != http.StatusOK {
				return
			}

			var result struct {
				Data []core.LogRecord `json:"data"`
				Meta struct {
					TotalRowCount int `json:"totalRowCount"`
				} `json:"meta"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("failed to decode JSON: %v", err)
			}

			if result.Meta.TotalRowCount != tc.expectedTotal {
				t.Errorf("expected total rows %v, got %v", tc.expectedTotal, result.Meta.TotalRowCount)
			}

			if len(result.Data) != tc.expectedCount {
				t.Fatalf("expected %v logs in data array, got %v", tc.expectedCount, len(result.Data))
			}

			if tc.expectedCount > 0 && result.Data[0].TraceId != tc.expectedFirst {
				t.Errorf("expected first trace ID %s, got %s", tc.expectedFirst, result.Data[0].TraceId)
			}
		})
	}
}

func TestDashboardHandleDistinctServices(t *testing.T) {
	ctx := context.Background()
	server, _ := setupDashboardServer(t, ctx)

	reqURL := server.URL + "/api/services"
	resp, err := http.Get(reqURL)
	if err != nil {
		t.Fatalf("failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %v", resp.StatusCode)
	}

	var result map[string][]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	services := result["services"]
	if len(services) != 2 {
		t.Fatalf("expected 2 distinct services, got %v", len(services))
	}

	// order of services returned may vary, hence need to check both values
	if services[0] != "payment-service" && services[1] != "payment-service" {
		t.Errorf("expected 'payment-service' but it was not found")
	}

	if services[0] != "test-service" && services[1] != "test-service" {
		t.Errorf("expected 'test-service' but it was not found")
	}
}

func TestDashboardHandleMetrics(t *testing.T) {
	ctx := context.Background()
	server, _ := setupDashboardServer(t, ctx)

	reqURL := server.URL + "/api/ingestion-metrics"
	resp, err := http.Get(reqURL)
	if err != nil {
		t.Fatalf("failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %v", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)

	var result core.IngestionMetricsEvent
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	metrics := result.Graph.Metrics["payment-service"]
	// 60 records due to the backfill of null timings
	if len(metrics) != 60 {
		t.Fatalf("expected 60 log metrics, got %v", len(metrics))
	}
}

func TestDashboardHandleLiveTail(t *testing.T) {
	type LogMessage struct {
		Message string `json:"message"`
	}

	ctx := context.Background()
	server, broker := setupDashboardServer(t, ctx)

	// convert httptestURL (http://) to websocket URL (ws://)
	wsURL := "ws" + server.URL[4:] + "/ws/logs/tail"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to websocket: %v", err)
	}
	defer ws.Close()
	time.Sleep(100 * time.Millisecond) // wait for server to establish Jetstream consumer

	testPayload := LogMessage{
		Message: "hello from testcaes!",
	}

	testData, err := json.Marshal(testPayload)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	err = broker.PublishLiveTail(transport.LiveTailSubjectPrefix+"test.app", testData)
	if err != nil {
		t.Fatalf("Failed to publish to NATS: %v", err)
	}

	// message should come within 2 seconds, otherwise websocket is down
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))

	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read from websocket (timed out?): %v", err)
	}

	var batch []json.RawMessage
	if err := json.Unmarshal(message, &batch); err != nil {
		t.Fatalf("Failed to unmarshal websocket payload: %v", err)
	}

	if len(batch) != 1 {
		t.Fatalf("Expected 1 log in batch, got %d", len(batch))
	}

	// check that the payload we sent matches the one we received
	var receivedPayload LogMessage
	if err := json.Unmarshal(batch[0], &receivedPayload); err != nil {
		t.Fatalf("Failed to unmarshal received payload: %v", err)
	}
	assert.Equal(t, testPayload, receivedPayload)
}
