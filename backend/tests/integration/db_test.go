//go:build integration

// Clickhouse helpers shared across test files
package integration

import (
	"context"
	"log/slog"
	"testing"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
)

var seedData = []core.FlatLogRecord{testRecord1, testRecord2, testRecord3, testRecord4}

func setupTestDB(t *testing.T, ctx context.Context, store storage.LogStore) {
	t.Helper()

	conn := getRawDBConn(t)
	defer conn.Close()

	err := conn.Exec(ctx, "TRUNCATE TABLE logarithm.logs")
	if err != nil {
		t.Fatalf("failed to truncate table: %v", err)
	}

	err = store.BatchInsert(ctx, seedData)
	if err != nil {
		t.Fatalf("failed to seed test data: %v", err)
	}
}

// Helper to create a raw db connection.
// Connection is closed automatically at the end of the test.
func getRawDBConn(t *testing.T) driver.Conn {
	t.Helper()
	ctx := context.Background()
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{dbAddr},
		Auth: clickhouse.Auth{
			Database: dbName,
			Username: user,
			Password: password,
		},
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})

	if err != nil {
		t.Fatalf("failed to configure clickhouse: %v", err)
	}

	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("failed to close db connection: %v", err)
		}
	})

	if err := conn.Ping(ctx); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			t.Logf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		t.Fatalf("failed to ping using established connection: %v", err)
	}

	return conn
}

// Helper to create a new ClickHouseStore.
// Store is closed automatically at the end of the test.
func getNewTestStore(t *testing.T) *storage.ClickHouseStore {
	t.Helper()
	chConfig := storage.Config{
		Address:  dbAddr,
		Database: dbName,
		Username: user,
		Password: password,
	}
	logStore, err := storage.NewClickHouseStore(context.Background(), chConfig, storage.WithLogger(slog.Default()))

	if err != nil {
		t.Fatalf("failed to establish db connection: %v", err)
	}

	t.Cleanup(func() {
		if err := logStore.Close(); err != nil {
			t.Errorf("failed to close store: %v", err)
		}
	})

	return logStore
}
