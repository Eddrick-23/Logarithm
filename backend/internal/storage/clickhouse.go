package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Eddrick-23/Logarithm/internal/core"
)

type LogStore interface {
	BatchInsert(context.Context, []core.FlatLogRecord) error
	SearchLogs(context.Context, core.LogQueryFilter) ([]core.FlatLogRecord, error)
}

type ClickHouseStore struct {
	conn   driver.Conn
	tables map[string]string
	logger *slog.Logger
}

var _ LogStore = (*ClickHouseStore)(nil)

// addr should be full host:port e.g. localhost:9000 or clickhouse:9000
func NewClickHouseStore(ctx context.Context, logger *slog.Logger, addr string, dbName string, username string, password string) (*ClickHouseStore, error) {
	logger.Info("Connecting to database...")
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

	tables, err := initTables(dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise table mapping: %v", err)
	}

	// TODO: add backoff and retry logic in case of connection instability
	if err := conn.Ping(ctx); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return nil, err
	}
	logger.Info("connection to database established")
	return &ClickHouseStore{
		conn:   conn,
		tables: tables,
		logger: logger,
	}, nil
}

func (s *ClickHouseStore) table(name string) (string, error) {
	t, ok := s.tables[name]
	if !ok {
		return "", fmt.Errorf("unknown table: %s", name)
	}
	return t, nil
}

func (s *ClickHouseStore) InitDB(ctx context.Context) error {
	fmt.Println(">>> InitDB called, table:", s.tables[TableLogs])

	tbl, err := s.table(TableLogs)
	if err != nil {
		return fmt.Errorf("failed to get table: %v", err)
	}

	var count uint64
	if err := s.conn.QueryRow(ctx, "SELECT count() FROM "+tbl).Scan(&count); err != nil {
		fmt.Println(">>> count query failed:", err)
		return fmt.Errorf("failed to check existing data: %w", err)
	}

	if count > 0 {
		fmt.Println(">>> skipping seed")
		return nil
	}

	err = s.BatchInsert(ctx, testData)
	if err != nil {
		fmt.Println(">>> BatchInsert failed:", err)
		return fmt.Errorf("failed to init db: %w", err)
	}

	fmt.Println(">>> seeded successfully")
	return nil
}

func (s *ClickHouseStore) Close() error {
	s.logger.Info("Closing clickhouse connection")
	return s.conn.Close()
}

func (s *ClickHouseStore) BatchInsert(ctx context.Context, records []core.FlatLogRecord) error {
	// must explicitly state all cols since we have an extra insertAt column
	// that clickhouse will fill in itself
	tbl, err := s.table(TableLogs)
	if err != nil {
		return fmt.Errorf("failed to get table: %v", err)
	}

	insertStatement := "INSERT INTO " + tbl +
		` (Timestamp, ScopeName, ScopeVersion, TraceId, SpanId, ObservedTimestamp, SeverityText, SeverityNumber,
         ServiceName, Body, BodyType, LogAttrKeys, LogAttrValues, ResAttrKeys, ResAttrValues)`
	batch, err := s.conn.PrepareBatch(ctx, insertStatement)

	if err != nil {
		s.logger.Error("Failed to prepare batch: %v", "err", err)
		return err
	}

	for _, record := range records {
		err = batch.Append(
			record.Timestamp,
			record.ScopeName,
			record.ScopeVersion,
			record.TraceId,
			record.SpanId,
			record.ObservedTimestamp,
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
	tbl, err := s.table(TableLogs)
	if err != nil {
		return nil, fmt.Errorf("failed to get table: %v", err)
	}

	whereClause, args := s.buildFilterQueryString(filter)
	queryString := fmt.Sprintf("Select * FROM %v %v", tbl, whereClause)

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
	tbl, err := s.table(TableLogs)
	if err != nil {
		return 0, fmt.Errorf("failed to get table: %v", err)
	}

	whereClause, args := s.buildFilterQueryString(filter)
	queryString := fmt.Sprintf("SELECT COUNT(*) FROM %v %v", tbl, whereClause)

	var count uint64
	if err := s.conn.QueryRow(ctx, queryString, args...).Scan(&count); err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *ClickHouseStore) GetDistinctServices(ctx context.Context) ([]string, error) {
	tbl, err := s.table(TableServiceRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to get table: %v", err)
	}

	// ReplacingMergeTree removes duplicates asynchronously so we need FINAL
	queryString := "SELECT ServiceName FROM " + tbl + " FINAL"

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

func (s *ClickHouseStore) GetIngestionMetrics(ctx context.Context) (core.IngestionMetricsMap, error) {
	tbl, err := s.table(TableMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to get table: %v", err)
	}

	whereClause := `WHERE Timestamp >= now() - toIntervalMinute(@mins)
					GROUP BY ServiceName, Timestamp
					ORDER BY ServiceName, Timestamp ASC
					WITH FILL
						FROM toDateTime(now() - toIntervalMinute(@mins))
    					TO toDateTime(now())
						STEP toIntervalSecond(1)`
	// back fills timestamps with no values
	queryString := fmt.Sprintf("SELECT Timestamp, ServiceName, sum(LogsCount) AS LogsCount FROM %v %v ", tbl, whereClause)

	var rows []core.IngestionMetrics
	// for now, im taking the metrics in the past 1 min, can be adjusted based on specifications
	if err := s.conn.Select(ctx, &rows, queryString, clickhouse.Named("mins", 1)); err != nil {
		return nil, err
	}

	return core.NewIngestionMetricsMap(rows), nil
}

func (s *ClickHouseStore) GetErrorMetrics(ctx context.Context) ([]core.ErrorMetrics, error) {
	// error metrics will return error rates within the past 1 hour
	tbl, err := s.table(TableMetrics1m)
	if err != nil {
		return nil, fmt.Errorf("failed to get table: %v", err)
	}

	queryString := fmt.Sprintf(`
        SELECT 
            ServiceName, 
            sum(ErrorsCount) AS TotalErrors, 
            round(sum(ErrorsCount) / sum(LogsCount) * 100, 2) AS ErrorRate
        FROM %v
        WHERE Timestamp >= now() - toIntervalHour(@hour)
        GROUP BY ServiceName
        ORDER BY ErrorRate DESC
        LIMIT 5
    `, tbl)

	var result []core.ErrorMetrics
	if err := s.conn.Select(ctx, &result, queryString, clickhouse.Named("hour", 1)); err != nil {
		return nil, err
	}

	return result, nil
}
