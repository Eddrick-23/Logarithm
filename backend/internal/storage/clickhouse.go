package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Eddrick-23/Logarithm/internal/core"
)

type LogStore interface {
	BatchInsert(context.Context, []core.FlatLogRecord) error
	SearchLogs(context.Context, core.LogQueryFilter) ([]core.FlatLogRecord, error)
}

type ClickHouseStore struct {
	conn       driver.Conn
	dbAndTable string
}

var _ LogStore = (*ClickHouseStore)(nil)

var testRecord1 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-1 * time.Hour),
	ObservedTimestamp: time.Now().Add(-1 * time.Hour),
	TraceId:           "4bf92f3577b34da6a3ce929d0e0e4736",
	SpanId:            "00f067aa0ba902b7",
	SeverityText:      "ERROR",
	SeverityNumber:    17,
	ServiceName:       "test-service1",
	Body:              "Failed to process transaction due to timeout",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"http.method", "http.status_code", "retry_count"},
	LogAttrValues:     []string{"POST", "504", "3"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-1", "linux", "1.2.0"},
}
var testRecord2 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-2 * time.Hour),
	ObservedTimestamp: time.Now().Add(-2 * time.Hour),
	TraceId:           "4bf92f3577b37da6a3ce929d0f0e4736",
	SpanId:            "01f067ef0ba402b7",
	SeverityText:      "WARNING",
	SeverityNumber:    13,
	ServiceName:       "test-service2",
	Body:              "extra information",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"http.method", "http.status_code", "retry_count"},
	LogAttrValues:     []string{"POST", "504", "3"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-2", "windows", "2.0.1"},
}
var testRecord3 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now(),
	ObservedTimestamp: time.Now(),
	TraceId:           "8bf92f3577b34da6d3ce921d0e0e4536",
	SpanId:            "02y067aa0ba902h3",
	SeverityText:      "INFO",
	SeverityNumber:    9,
	ServiceName:       "test-service3",
	Body:              "Just some test body",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"http.method", "http.status_code", "retry_count"},
	LogAttrValues:     []string{"GET", "500", "3"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-3", "linux", "3.1.0"},
}
var testRecord4 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-30 * time.Minute),
	ObservedTimestamp: time.Now().Add(-30 * time.Minute),
	TraceId:           "1cf92f3577b34da6a3ce929d0e0e1234",
	SpanId:            "10f067aa0ba902c1",
	SeverityText:      "INFO",
	SeverityNumber:    9,
	ServiceName:       "test-service1",
	Body:              "User authentication successful",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"http.method", "http.status_code", "user.id"},
	LogAttrValues:     []string{"POST", "200", "usr-991"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-1", "linux", "1.2.0"},
}
var testRecord5 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-45 * time.Minute),
	ObservedTimestamp: time.Now().Add(-45 * time.Minute),
	TraceId:           "2df92f3577b34da6a3ce929d0e0e5678",
	SpanId:            "11f067aa0ba902c2",
	SeverityText:      "ERROR",
	SeverityNumber:    17,
	ServiceName:       "test-service2",
	Body:              "Database connection pool exhausted",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"db.system", "db.name", "error.type"},
	LogAttrValues:     []string{"postgresql", "orders_db", "ConnectionPoolError"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-2", "windows", "2.0.1"},
}
var testRecord6 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-3 * time.Hour),
	ObservedTimestamp: time.Now().Add(-3 * time.Hour),
	TraceId:           "3ef92f3577b34da6a3ce929d0e0e9012",
	SpanId:            "12f067aa0ba902c3",
	SeverityText:      "WARNING",
	SeverityNumber:    13,
	ServiceName:       "test-service3",
	Body:              "Memory usage above 80% threshold",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"mem.used_mb", "mem.total_mb", "mem.percent"},
	LogAttrValues:     []string{"6554", "8192", "80.5"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-3", "linux", "3.1.0"},
}
var testRecord7 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-4 * time.Hour),
	ObservedTimestamp: time.Now().Add(-4 * time.Hour),
	TraceId:           "4ff92f3577b34da6a3ce929d0e0e3456",
	SpanId:            "13f067aa0ba902c4",
	SeverityText:      "INFO",
	SeverityNumber:    9,
	ServiceName:       "test-service1",
	Body:              "Cache invalidation completed",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"cache.keys_evicted", "cache.size_mb", "cache.hit_rate"},
	LogAttrValues:     []string{"142", "512", "0.87"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-1", "linux", "1.2.0"},
}
var testRecord8 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-5 * time.Hour),
	ObservedTimestamp: time.Now().Add(-5 * time.Hour),
	TraceId:           "5af92f3577b34da6a3ce929d0e0e7890",
	SpanId:            "14f067aa0ba902c5",
	SeverityText:      "ERROR",
	SeverityNumber:    17,
	ServiceName:       "test-service2",
	Body:              "Payment gateway returned unexpected response",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"http.method", "http.status_code", "payment.gateway"},
	LogAttrValues:     []string{"POST", "502", "stripe"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-2", "windows", "2.0.1"},
}
var testRecord9 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-6 * time.Hour),
	ObservedTimestamp: time.Now().Add(-6 * time.Hour),
	TraceId:           "6bf92f3577b34da6a3ce929d0e0e2345",
	SpanId:            "15f067aa0ba902c6",
	SeverityText:      "INFO",
	SeverityNumber:    9,
	ServiceName:       "test-service3",
	Body:              "Scheduled job completed successfully",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"job.name", "job.duration_ms", "job.records_processed"},
	LogAttrValues:     []string{"daily-report", "4521", "10200"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-3", "linux", "3.1.0"},
}
var testRecord10 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-7 * time.Hour),
	ObservedTimestamp: time.Now().Add(-7 * time.Hour),
	TraceId:           "7cf92f3577b34da6a3ce929d0e0e6789",
	SpanId:            "16f067aa0ba902c7",
	SeverityText:      "WARNING",
	SeverityNumber:    13,
	ServiceName:       "test-service1",
	Body:              "Retrying failed request to downstream service",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"http.method", "http.url", "retry.attempt"},
	LogAttrValues:     []string{"GET", "http://inventory-svc/api/stock", "2"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-1", "linux", "1.2.0"},
}
var testRecord11 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-8 * time.Hour),
	ObservedTimestamp: time.Now().Add(-8 * time.Hour),
	TraceId:           "8df92f3577b34da6a3ce929d0e0e1357",
	SpanId:            "17f067aa0ba902c8",
	SeverityText:      "ERROR",
	SeverityNumber:    17,
	ServiceName:       "test-service2",
	Body:              "Unhandled exception in order processing pipeline",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"exception.type", "exception.message", "order.id"},
	LogAttrValues:     []string{"NullPointerException", "order item is nil", "ord-77342"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-2", "windows", "2.0.1"},
}
var testRecord12 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-9 * time.Hour),
	ObservedTimestamp: time.Now().Add(-9 * time.Hour),
	TraceId:           "9ef92f3577b34da6a3ce929d0e0e2468",
	SpanId:            "18f067aa0ba902c9",
	SeverityText:      "INFO",
	SeverityNumber:    9,
	ServiceName:       "test-service3",
	Body:              "New user registered successfully",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"user.id", "user.email", "user.country"},
	LogAttrValues:     []string{"usr-1124", "newuser@example.com", "SG"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-3", "linux", "3.1.0"},
}
var testRecord13 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-10 * time.Hour),
	ObservedTimestamp: time.Now().Add(-10 * time.Hour),
	TraceId:           "aaf92f3577b34da6a3ce929d0e0e3579",
	SpanId:            "19f067aa0ba902d1",
	SeverityText:      "WARNING",
	SeverityNumber:    13,
	ServiceName:       "test-service1",
	Body:              "Disk usage approaching limit on data volume",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"disk.path", "disk.used_gb", "disk.total_gb"},
	LogAttrValues:     []string{"/data", "92", "100"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-1", "linux", "1.2.0"},
}
var testRecord14 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-11 * time.Hour),
	ObservedTimestamp: time.Now().Add(-11 * time.Hour),
	TraceId:           "bbf92f3577b34da6a3ce929d0e0e4680",
	SpanId:            "20f067aa0ba902d2",
	SeverityText:      "ERROR",
	SeverityNumber:    17,
	ServiceName:       "test-service2",
	Body:              "TLS certificate validation failed",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"tls.peer", "tls.error", "http.url"},
	LogAttrValues:     []string{"api.partner.com", "certificate expired", "https://api.partner.com/v2/sync"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-2", "windows", "2.0.1"},
}
var testRecord15 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-12 * time.Hour),
	ObservedTimestamp: time.Now().Add(-12 * time.Hour),
	TraceId:           "ccf92f3577b34da6a3ce929d0e0e5791",
	SpanId:            "21f067aa0ba902d3",
	SeverityText:      "INFO",
	SeverityNumber:    9,
	ServiceName:       "test-service3",
	Body:              "Feature flag evaluated successfully",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"feature.flag", "feature.enabled", "user.id"},
	LogAttrValues:     []string{"new-checkout-flow", "true", "usr-4421"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-3", "linux", "3.1.0"},
}

var testData []core.FlatLogRecord = []core.FlatLogRecord{
	testRecord1, testRecord2, testRecord3, testRecord4, testRecord5,
	testRecord6, testRecord7, testRecord8, testRecord9, testRecord10,
	testRecord11, testRecord12, testRecord13, testRecord14, testRecord15,
}

// addr should be full host:port e.g. localhost:9000 or clickhouse:9000
func NewClickHouseStore(ctx context.Context, addr string, dbName string, tableName string, username string, password string) (*ClickHouseStore, error) {
	slog.Info("Connecting to database...")
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: dbName,
			Username: username,
			Password: password,
		},
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to configure clickhouse: %v", err)
	}

	// TODO: add backoff and retry logic in case of connection instability
	if err := conn.Ping(ctx); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return nil, err
	}
	slog.Info("connection to database established")
	return &ClickHouseStore{conn: conn, dbAndTable: dbName + "." + tableName}, nil
}

func (s *ClickHouseStore) InitDB(ctx context.Context) error {
	fmt.Println(">>> InitDB called, table:", s.dbAndTable)

	var count uint64
	if err := s.conn.QueryRow(ctx, "SELECT count() FROM "+s.dbAndTable).Scan(&count); err != nil {
		fmt.Println(">>> count query failed:", err)
		return fmt.Errorf("failed to check existing data: %w", err)
	}

	if count > 0 {
		fmt.Println(">>> skipping seed")
		return nil
	}

	err := s.BatchInsert(ctx, testData)
	if err != nil {
		fmt.Println(">>> BatchInsert failed:", err)
		return fmt.Errorf("failed to init db: %w", err)
	}

	fmt.Println(">>> seeded successfully")
	return nil
}

func (s *ClickHouseStore) Close() error {
	slog.Info("Closing clickhouse connection")
	return s.conn.Close()
}

func (s *ClickHouseStore) BatchInsert(ctx context.Context, records []core.FlatLogRecord) error {
	// must explicitly state all cols since we have an extra insertAt column
	// that clickhouse will fill in itself
	insertStatement := "INSERT INTO " + s.dbAndTable +
		` (Timestamp, ScopeName, ScopeVersion, TraceId, SpanId, SeverityText, SeverityNumber,
         ServiceName, Body, BodyType, LogAttrKeys, LogAttrValues, ResAttrKeys, ResAttrValues)`
	batch, err := s.conn.PrepareBatch(ctx, insertStatement)

	if err != nil {
		slog.Error("Failed to prepare batch: %v", "err", err)
		return err
	}

	for _, record := range records {
		err = batch.Append(
			record.Timestamp,
			record.ScopeName,
			record.ScopeVersion,
			record.TraceId,
			record.SpanId,
			record.SeverityText,
			record.SeverityNumber,
			record.ServiceName,
			record.Body,
			record.BodyType,
			record.LogAttrKeys,
			record.LogAttrValues,
			record.ResAttrKeys,
			record.ResAttrValues,
		)
		if err != nil {
			return fmt.Errorf("failed to append row: %v", err)
		}
	}

	if err := batch.Send(); err != nil {
		return err
	}
	return nil
}

func (s *ClickHouseStore) buildFilterQueryString(filter core.LogQueryFilter) (string, []any) {
	filterQueryString := "WHERE 1=1"
	var args []any

	if !filter.StartTime.IsZero() {
		filterQueryString += " AND Timestamp >= ?"
		args = append(args, filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		filterQueryString += " AND Timestamp <= ?"
		args = append(args, filter.EndTime)
	}
	if filter.ServiceName != "" {
		filterQueryString += " AND ServiceName ILIKE ?"
		args = append(args, filter.ServiceName+"%")
	}
	if filter.SeverityText != "" {
		filterQueryString += " AND SeverityText ILIKE ?"
		args = append(args, filter.SeverityText+"%")
	}
	if filter.TraceId != "" {
		filterQueryString += " AND TraceId ILIKE ?"
		args = append(args, filter.TraceId+"%")
	}
	if filter.SpanId != "" {
		filterQueryString += " AND SpanId ILIKE ?"
		args = append(args, filter.SpanId+"%")
	}
	if filter.SeverityNumber > 0 { // OTel severity number ranges 1 - 24
		filterQueryString += " AND SeverityNumber = ?"
		args = append(args, filter.SeverityNumber)
	}
	if filter.Body != "" {
		filterQueryString += " AND Body ILIKE ?"
		args = append(args, "%"+filter.Body+"%")
	}

	return filterQueryString, args
}

func (s *ClickHouseStore) SearchLogs(ctx context.Context, filter core.LogQueryFilter) ([]core.FlatLogRecord, error) {
	whereClause, args := s.buildFilterQueryString(filter)
	queryString := fmt.Sprintf("Select * FROM %v %v", s.dbAndTable, whereClause)

	if filter.OrderBy != "" {
		queryString += fmt.Sprintf(" ORDER BY %s", filter.OrderBy)
	} else {
		queryString += fmt.Sprintf(" ORDER BY %s", core.OrderByTimestamp)
	}

	if filter.Descending {
		queryString += " DESC"
	} else {
		queryString += " ASC"
	}

	// set default limit to 100 records
	limit := 100
	if filter.Limit != 0 {
		limit = filter.Limit
	}
	queryString += fmt.Sprintf(" LIMIT %v", limit)

	offset := max(filter.Offset, 0)
	queryString += fmt.Sprintf(" OFFSET %v", offset)

	var result []core.FlatLogRecord
	if err := s.conn.Select(ctx, &result, queryString, args...); err != nil {
		return result, err
	}

	return result, nil
}

func (s *ClickHouseStore) GetFilteredLogsCount(ctx context.Context, filter core.LogQueryFilter) (int, error) {
	whereClause, args := s.buildFilterQueryString(filter)
	queryString := fmt.Sprintf("SELECT COUNT(*) FROM %v %v", s.dbAndTable, whereClause)

	var count uint64
	if err := s.conn.QueryRow(ctx, queryString, args...).Scan(&count); err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *ClickHouseStore) CountInsertedWithin(ctx context.Context, minutes uint64) (uint64, error) {
	whereClause := "WHERE InsertedAt >= now() - toIntervalMinute(@mins)"
	queryString := "SELECT count() FROM " + s.dbAndTable + " " + whereClause

	var count uint64
	if err := s.conn.QueryRow(ctx, queryString, clickhouse.Named("mins", minutes)).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to fetch row count: %v", err)
	}

	return count, nil
}

func (s *ClickHouseStore) GetDistinctServices(ctx context.Context) ([]string, error) {
	queryString := "SELECT DISTINCT ServiceName FROM " + s.dbAndTable

	rows, err := s.conn.Query(ctx, queryString)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query for distinct services: %v", err)
	}
	defer rows.Close()

	var services []string

	for rows.Next() {
		var service string
		if err := rows.Scan(&service); err != nil {
			return nil, fmt.Errorf("failed to scan service name: %v", err)
		}
		services = append(services, service)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over service rows: %v", err)
	}

	return services, nil
}
