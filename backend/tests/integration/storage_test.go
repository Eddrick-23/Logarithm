//go:build integration

package integration

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClickHouseStore(t *testing.T) {
	chConfig := storage.Config{
		Address:  dbAddr,
		Database: dbName,
		Username: user,
		Password: password,
	}
	_, err := storage.NewClickHouseStore(context.Background(), chConfig, storage.WithLogger(slog.Default()))

	if err != nil {
		t.Errorf("failed to establish db connection: %v", err)
	}
}

func TestNewClickHouseStoreWrongDBName(t *testing.T) {
	chConfig := storage.Config{
		Address:  dbAddr,
		Database: "wrongname",
		Username: user,
		Password: password,
	}
	_, err := storage.NewClickHouseStore(context.Background(), chConfig, storage.WithLogger(slog.Default()))

	if err == nil {
		t.Error("Connection still established with wrong database name")
	}
}

func TestNewClickHouseStoreWrongUser(t *testing.T) {
	chConfig := storage.Config{
		Address:  dbAddr,
		Database: dbName,
		Username: "wronguser",
		Password: password,
	}
	_, err := storage.NewClickHouseStore(context.Background(), chConfig, storage.WithLogger(slog.Default()))

	if err == nil {
		t.Error("Connection still established with wrong username")
	}
}
func TestNewClickHouseStoreWrongPassword(t *testing.T) {
	chConfig := storage.Config{
		Address:  dbAddr,
		Database: dbName,
		Username: user,
		Password: "wrongpassword",
	}
	_, err := storage.NewClickHouseStore(context.Background(), chConfig, storage.WithLogger(slog.Default()))

	if err == nil {
		t.Error("Connection still established with wrong password")
	}
}

func TestBatchInsert(t *testing.T) {
	logStore := getNewTestStore(t)
	conn := getRawDBConn(t)

	ctx := context.Background()

	err := conn.Exec(ctx, "TRUNCATE TABLE logarithm.logs")
	if err != nil {
		t.Fatalf("failed to truncate table: %v", err)
	}

	testRecords := []core.FlatLogRecord{testRecordEveryField}
	err = logStore.BatchInsert(ctx, testRecords)

	if err != nil {
		t.Fatalf("batch insert failed: %v", err)
	}

	var count uint64
	err = conn.QueryRow(ctx, "SELECT Count() FROM logarithm.logs").Scan(&count)

	if err != nil {
		t.Fatalf("DB query failed: %v", err)
	}

	if count != 1 {
		t.Fatalf("Expected 1 log got :%v", count)
	}

	var records []core.FlatLogRecord
	err = conn.Select(context.Background(), &records, "SELECT * FROM logarithm.logs")
	require.NoError(t, err, "error reading from clickhouse")

	// ignore insertedAtField since that is managed by clickhouse
	records[0].InsertedAt = time.Time{}
	testRecordEveryField.InsertedAt = time.Time{}
	assert.Equal(t, testRecordEveryField, records[0])
}

func TestBatchInsertMultipleLogs(t *testing.T) {
	logStore := getNewTestStore(t)
	conn := getRawDBConn(t)

	ctx := context.Background()

	err := conn.Exec(ctx, "TRUNCATE TABLE logarithm.logs")
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

func TestFastInsert(t *testing.T) {
	tests := []struct {
		name         string
		records      []core.FlatLogRecord
		expectedRows uint64
	}{
		{
			name:         "0 input records",
			records:      []core.FlatLogRecord{},
			expectedRows: 0,
		},
		{
			name:         "1 input records",
			records:      []core.FlatLogRecord{testRecord1},
			expectedRows: 1,
		},
		{
			name:         "2 input records",
			records:      []core.FlatLogRecord{testRecord1, testRecord2},
			expectedRows: 2,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logStore := getNewTestStore(t)
			conn := getRawDBConn(t)

			ctx := context.Background()

			err := conn.Exec(ctx, "TRUNCATE TABLE logarithm.logs")
			if err != nil {
				t.Fatalf("failed to truncate table: %v", err)
			}

			appender := logStore.FastInsert(len(tc.records))

			for _, record := range tc.records {
				var traceBytes [16]byte
				copy(traceBytes[:], record.TraceId)

				var spanBytes [8]byte
				copy(spanBytes[:], record.SpanId)

				appender.Append(
					record.Timestamp,
					record.ObservedTimestamp,
					record.SeverityNumber,
					traceBytes,
					spanBytes,
					record.LogAttrKeys,
					record.LogAttrValues,
					record.ResAttrKeys,
					record.ResAttrValues,
					storage.LogFields{
						ScopeName:    record.ScopeName,
						ScopeVersion: record.ScopeVersion,
						SeverityText: record.SeverityText,
						ServiceName:  record.ServiceName,
						Body:         record.Body,
						BodyType:     record.BodyType,
					},
				)
			}

			err = appender.Flush(ctx)
			if err != nil {
				t.Fatalf("batch insert failed: %v", err)
			}

			var finalCount uint64
			err = conn.QueryRow(ctx, "SELECT Count() FROM logarithm.logs").Scan(&finalCount)
			if err != nil {
				t.Errorf("DB query failed using raw conn: %v", err)
			}

			assert.Equal(t, tc.expectedRows, finalCount)

		})
	}
}

func TestSearchLogs(t *testing.T) {
	logStore := getNewTestStore(t)

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
			expectedCount: len(seedData),
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
			testName:        "SearchTerm \"timeout\"",
			filter:          core.LogQueryFilter{Body: "timeout"},
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
			expectedTraceId: testRecord4.TraceId,
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
