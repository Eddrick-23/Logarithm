package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
)

var dbAddr string
var user string
var password string
var dbname string
var natsUrl string

var testRecordEveryField core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Date(2024, 5, 20, 10, 0, 0, 0, time.UTC),
	ObservedTimestamp: time.Date(2024, 5, 20, 10, 0, 0, 0, time.UTC),
	TraceId:           "4bf92f3577b34da6a3ce929d0e0e4736",
	SpanId:            "00f067aa0ba902b7",
	SeverityText:      "ERROR",
	SeverityNumber:    17,
	ServiceName:       "test-service",
	Body:              "Failed to process transaction due to timeout",
	BodyType:          "string",
	ScopeName:         "test",
	ScopeVersion:      "1.0.0",
	LogAttrKeys:       []string{"http.method", "http.status_code", "retry_count"},
	LogAttrValues:     []string{"POST", "504", "3"},
	ResAttrKeys:       []string{"service.name"},
	ResAttrValues:     []string{"test-service"},
}

var testRecord1 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Date(2024, 5, 20, 10, 0, 0, 0, time.UTC),
	ObservedTimestamp: time.Date(2024, 5, 20, 10, 0, 0, 0, time.UTC),
	TraceId:           "4bf92f3577b34da6a3ce929d0e0e4736",
	SpanId:            "00f067aa0ba902b7",
	SeverityText:      "ERROR",
	SeverityNumber:    17,
	ServiceName:       "test-service",
	Body:              "Failed to process transaction due to timeout",
	BodyType:          "string",
	ScopeName:         "test",
	ScopeVersion:      "1.0.0",
	LogAttrKeys:       []string{"http.method", "http.status_code", "retry_count"},
	LogAttrValues:     []string{"POST", "504", "3"},
	ResAttrKeys:       []string{},
	ResAttrValues:     []string{},
}

var testRecord2 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Date(2024, 5, 23, 10, 0, 0, 0, time.UTC),
	ObservedTimestamp: time.Date(2024, 5, 23, 10, 0, 0, 0, time.UTC),
	TraceId:           "4bf92f3577b37da6a3ce929d0f0e4736",
	SpanId:            "01f067ef0ba402b7",
	SeverityText:      "WARNING",
	SeverityNumber:    13,
	ServiceName:       "test-service",
	Body:              "extra information",
	BodyType:          "string",
	ScopeName:         "test",
	ScopeVersion:      "1.0.0",
	LogAttrKeys:       []string{"http.method", "http.status_code", "retry_count"},
	LogAttrValues:     []string{"POST", "504", "3"},
	ResAttrKeys:       []string{},
	ResAttrValues:     []string{},
}

var testRecord3 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Date(2025, 5, 20, 10, 0, 0, 0, time.UTC),
	ObservedTimestamp: time.Date(2025, 5, 20, 10, 0, 0, 0, time.UTC),
	TraceId:           "8bf92f3577b34da6d3ce921d0e0e4536",
	SpanId:            "02y067aa0ba902h3",
	SeverityText:      "INFO",
	SeverityNumber:    9,
	ServiceName:       "test-service",
	Body:              "Just some test body",
	BodyType:          "string",
	ScopeName:         "test",
	ScopeVersion:      "1.0.0",
	LogAttrKeys:       []string{"http.method", "http.status_code", "retry_count"},
	LogAttrValues:     []string{"GET", "500", "3"},
	ResAttrKeys:       []string{},
	ResAttrValues:     []string{},
}
var now = time.Now().UTC()
var testRecord4 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         now,
	ObservedTimestamp: now,
	TraceId:           "3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b",
	SpanId:            "a1b2c3d4e5f6a7b8",
	SeverityText:      "ERROR",
	SeverityNumber:    17,
	ServiceName:       "payment-service",
	Body:              "Payment processing failed due to server outage.",
	BodyType:          "string",
	ScopeName:         "test",
	ScopeVersion:      "1.0.0",
	LogAttrKeys:       []string{"payment_id", "error_code", "retry_attempt"},
	LogAttrValues:     []string{"pay_abc123", "TIMEOUT", "2"},
	ResAttrKeys:       []string{"host.name"},
	ResAttrValues:     []string{"payment-worker-01"},
}

var seedData = []core.FlatLogRecord{testRecord1, testRecord2, testRecord3, testRecord4}

func setupTestDB(t *testing.T, ctx context.Context, store storage.LogStore) {
	t.Helper()

	conn, err := getRawDBConn()
	if err != nil {
		t.Fatalf("failed to get raw db conn in setup: %v", err)
	}
	defer conn.Close()

	err = conn.Exec(ctx, "TRUNCATE TABLE logarithm.logs")
	if err != nil {
		t.Fatalf("failed to truncate table: %v", err)
	}

	err = store.BatchInsert(ctx, seedData)
	if err != nil {
		t.Fatalf("failed to seed test data: %v", err)
	}
}
func getRawDBConn() (driver.Conn, error) {
	ctx := context.Background()
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{dbAddr},
		Auth: clickhouse.Auth{
			Database: dbname,
			Username: user,
			Password: password,
		},
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to configure clickhouse: %v", err)
	}

	if err := conn.Ping(ctx); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return nil, err
	}

	return conn, nil
}
