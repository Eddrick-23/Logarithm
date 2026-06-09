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
	conn       driver.Conn
	dbAndTable string
	logger     *slog.Logger
}

var _ LogStore = (*ClickHouseStore)(nil)

// addr should be full host:port e.g. localhost:9000 or clickhouse:9000
func NewClickHouseStore(ctx context.Context, logger *slog.Logger, addr string, dbName string, tableName string, username string, password string) (*ClickHouseStore, error) {
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

	// TODO: add backoff and retry logic in case of connection instability
	if err := conn.Ping(ctx); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return nil, err
	}
	logger.Info("connection to database established")
	return &ClickHouseStore{
		conn:       conn,
		dbAndTable: dbName + "." + tableName,
		logger:     logger}, nil
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
	s.logger.Info("Closing clickhouse connection")
	return s.conn.Close()
}

func (s *ClickHouseStore) BatchInsert(ctx context.Context, records []core.FlatLogRecord) error {
	// must explicitly state all cols since we have an extra insertAt column
	// that clickhouse will fill in itself
	insertStatement := "INSERT INTO " + s.dbAndTable +
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
