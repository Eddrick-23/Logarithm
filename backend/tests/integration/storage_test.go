//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2" // alias to avoid naming conflict
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/testcontainers/testcontainers-go"
	chmodule "github.com/testcontainers/testcontainers-go/modules/clickhouse" // alias to avoid naming conflict
)

var dbAddr string
var user string
var password string
var dbname string
var dbtablename string

var _ storage.LogStore = (*storage.ClickHouseStore)(nil) // make sure satisfies interface methods before any tests

var testRecord1 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:      time.Date(2024, 5, 20, 10, 0, 0, 0, time.UTC),
	TraceId:        "4bf92f3577b34da6a3ce929d0e0e4736",
	SpanId:         "00f067aa0ba902b7",
	SeverityText:   "ERROR",
	SeverityNumber: 17,
	ServiceName:    "test-service",
	Body:           "Failed to process transaction due to timeout",
	LogAttrKeys:    []string{"http.method", "http.status_code", "retry_count"},
	LogAttrValues:  []string{"POST", "504", "3"},
	ResAttrKeys:    []string{},
	ResAttrValues:  []string{},
}

var testRecord2 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:      time.Date(2024, 5, 23, 10, 0, 0, 0, time.UTC),
	TraceId:        "4bf92f3577b37da6a3ce929d0f0e4736",
	SpanId:         "01f067ef0ba402b7",
	SeverityText:   "WARNING",
	SeverityNumber: 13,
	ServiceName:    "test-service",
	Body:           "extra information",
	LogAttrKeys:    []string{"http.method", "http.status_code", "retry_count"},
	LogAttrValues:  []string{"POST", "504", "3"},
	ResAttrKeys:    []string{},
	ResAttrValues:  []string{},
}

var testRecord3 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:      time.Date(2025, 5, 20, 10, 0, 0, 0, time.UTC),
	TraceId:        "8bf92f3577b34da6d3ce921d0e0e4536",
	SpanId:         "02y067aa0ba902h3",
	SeverityText:   "INFO",
	SeverityNumber: 9,
	ServiceName:    "test-service",
	Body:           "Just some test body",
	LogAttrKeys:    []string{"http.method", "http.status_code", "retry_count"},
	LogAttrValues:  []string{"GET", "500", "3"},
	ResAttrKeys:    []string{},
	ResAttrValues:  []string{},
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	user = "clickhouse"
	password = "password"
	dbname = "logarithm"
	dbtablename = "logs"

	clickHouseContainer, err := chmodule.Run(ctx,
		"clickhouse/clickhouse-server:26.3-alpine",
		chmodule.WithUsername(user),
		chmodule.WithPassword(password),
		chmodule.WithDatabase(dbname),
		chmodule.WithInitScripts(filepath.Join("testdata", "init-db.sql")),
	)
	defer func() {
		if clickHouseContainer != nil {
			if err := testcontainers.TerminateContainer(clickHouseContainer); err != nil {
				fmt.Printf("failed to terminate container: %s", err)
			}
		}
	}()
	if err != nil {

		fmt.Printf("failed to start container: %s", err)
		return
	}

	host, err := clickHouseContainer.Host(ctx)
	if err != nil {
		fmt.Printf("failed to get host: %v", err)
	}
	port, err := clickHouseContainer.MappedPort(ctx, "9000/tcp")
	if err != nil {
		fmt.Printf("failed to get port: %v", err)
	}

	dbAddr = fmt.Sprintf("%s:%s", host, port.Port())
	exitVal := m.Run()

	os.Exit(exitVal)
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

func setupTestDB(t *testing.T, ctx context.Context, store storage.LogStore) {
	t.Helper()

	conn, err := getRawDBConn()

	if err != nil {
		t.Fatalf("failed to get raw db conn in setup: %v", err)
	}

	err = conn.Exec(ctx, "TRUNCATE TABLE logarithm.logs")
	if err != nil {
		t.Fatalf("failed to truncate table: %v", err)
	}

	seedData := []core.FlatLogRecord{testRecord1, testRecord2, testRecord3}

	err = store.BatchInsert(ctx, seedData)
	if err != nil {
		t.Fatalf("failed to seed test data: %v", err)
	}
}

func TestNewClickHouseStore(t *testing.T) {
	_, err := storage.NewClickHouseStore(context.Background(), dbAddr, dbname, dbtablename, user, password)

	if err != nil {
		t.Errorf("failed to establish db connection: %v", err)
	}
}

func TestNewClickHouseStoreWrongDBName(t *testing.T) {
	_, err := storage.NewClickHouseStore(context.Background(), dbAddr, "wrongname", dbtablename, user, password)

	if err == nil {
		t.Error("Connection still established with wrong database name")
	}
}

func TestNewClickHouseStoreWrongUser(t *testing.T) {
	_, err := storage.NewClickHouseStore(context.Background(), dbAddr, dbname, dbtablename, "wronguser", password)

	if err == nil {
		t.Error("Connection still established with wrong username")
	}
}
func TestNewClickHouseStoreWrongPassword(t *testing.T) {
	_, err := storage.NewClickHouseStore(context.Background(), dbAddr, dbname, dbtablename, user, "wrongpassword")

	if err == nil {
		t.Error("Connection still established with wrong password")
	}
}

func TestBatchInsert(t *testing.T) {
	logStore, err := storage.NewClickHouseStore(context.Background(), dbAddr, dbname, dbtablename, user, password)
	ctx := context.Background()

	if err != nil {
		t.Fatalf("failed to establish db connection: %v", err)
	}

	testRecords := []core.FlatLogRecord{testRecord1}
	err = logStore.BatchInsert(ctx, testRecords)

	if err != nil {
		t.Fatalf("batch insert failed: %v", err)
	}

	conn, err := getRawDBConn()

	if err != nil {
		t.Fatalf("failed to get raw db connection: %v", err)
	}

	var count uint64
	err = conn.QueryRow(ctx, "SELECT Count() FROM logarithm.logs").Scan(&count)

	if err != nil {
		t.Fatalf("DB query failed: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 log got :%v", count)
	}
}

func TestBatchInsertMultipleLogs(t *testing.T) {
	logStore, err := storage.NewClickHouseStore(context.Background(), dbAddr, dbname, dbtablename, user, password)

	ctx := context.Background()
	if err != nil {
		t.Fatalf("failed to establish db connection: %v", err)
	}

	conn, err := getRawDBConn()

	if err != nil {
		t.Fatalf("failed to get raw db connection: %v", err)
	}

	err = conn.Exec(ctx, "TRUNCATE TABLE logarithm.logs")
	if err != nil {
		t.Fatalf("failed to truncate table: %v", err)
	}

	testRecords := []core.FlatLogRecord{testRecord1, testRecord2}
	err = logStore.BatchInsert(ctx, testRecords)
	if err != nil {
		t.Fatalf("batch insert failed: %v", err)
	}

	var finalCount uint64
	err = conn.QueryRow(ctx, "SELECT Count() FROM logarithm.logs").Scan(&finalCount)
	if err != nil {
		t.Errorf("DB query failed using raw conn: %v", err)
	}

	if finalCount != 2 {
		t.Errorf("Expected %v logs got :%v", 2, finalCount)
	}

}

func TestSearchLogs(t *testing.T) {
	logStore, err := storage.NewClickHouseStore(context.Background(), dbAddr, dbname, dbtablename, user, password)

	if err != nil {
		t.Fatalf("failed to establish db connection: %v", err)
	}

	ctx := context.Background()
	setupTestDB(t, ctx, logStore)

	testCases := []struct {
		testName        string
		filter          core.LogQueryFilter
		expectedCount   int
		expectedTraceId string
	}{
		{
			testName:      "Match exact serviceName",
			filter:        core.LogQueryFilter{ServiceName: "test-service"},
			expectedCount: 3,
		},
		{
			testName:      "Match prefix serviceName",
			filter:        core.LogQueryFilter{ServiceName: "test"},
			expectedCount: 3,
		},
		{
			testName:      "No match serviceName",
			filter:        core.LogQueryFilter{ServiceName: "none-match"},
			expectedCount: 0,
		},
		{
			testName:        "Severity filter WARNING",
			filter:          core.LogQueryFilter{SeverityText: "WARNING"},
			expectedCount:   1,
			expectedTraceId: testRecord2.TraceId,
		},
		{
			testName:      "Start time testRecord1 onwards",
			filter:        core.LogQueryFilter{StartTime: time.Date(2024, 5, 19, 0, 0, 0, 0, time.UTC)},
			expectedCount: 3,
		},
		{
			testName: "Start time testRecord1 onwards end time before testRecord3",
			filter: core.LogQueryFilter{
				StartTime: time.Date(2024, 5, 19, 0, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expectedCount: 2,
		},
		{
			testName:      "Start time testRecord1 onwards",
			filter:        core.LogQueryFilter{StartTime: time.Date(2024, 5, 19, 0, 0, 0, 0, time.UTC)},
			expectedCount: 3,
		},
		{
			testName:        "SearchTerm \"timeout\"",
			filter:          core.LogQueryFilter{SearchTerm: "timeout"},
			expectedCount:   1,
			expectedTraceId: testRecord1.TraceId,
		},
		{
			testName:        "Limit 1 orderby timestamp ascending",
			filter:          core.LogQueryFilter{Limit: 1, OrderBy: core.OrderByTimestamp},
			expectedCount:   1,
			expectedTraceId: testRecord1.TraceId,
		},
		{
			testName:        "Limit 1 orderby timestamp descending",
			filter:          core.LogQueryFilter{Limit: 1, OrderBy: core.OrderByTimestamp, Descending: true},
			expectedCount:   1,
			expectedTraceId: testRecord3.TraceId,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			result, err := logStore.SearchLogs(ctx, tc.filter)
			if err != nil {
				t.Fatalf("search logs failed: %v", err)
			}
			if len(result) != tc.expectedCount {
				t.Fatalf("search logs returned count: %v expected: %v", len(result), tc.expectedCount)
			}
			if tc.expectedTraceId != "" && tc.expectedTraceId != result[0].TraceId {
				t.Fatalf("search logs returned traceId: %v, expected: %v", result[0].TraceId, tc.expectedTraceId)
			}
		})
	}
}
