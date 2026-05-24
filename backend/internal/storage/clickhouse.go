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
	BatchInsert(context.Context, []core.FlatLogRecord) (error)
	SearchLogs(context.Context, core.LogQueryFilter) ([]core.FlatLogRecord, error) 
}

type ClickHouseStore struct {
	conn driver.Conn
	dbAndTable string	
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

func (s *ClickHouseStore) BatchInsert(ctx context.Context, records []core.FlatLogRecord) error {
	batch , err := s.conn.PrepareBatch(ctx, "INSERT INTO " + s.dbAndTable)

	if err != nil {
		slog.Error("Failed to prepare batch: %v", "err", err)
		return err
	}

	for _, record := range records{
		err = batch.Append(
			record.Timestamp,
			record.TraceId,
			record.SpanId,
			record.SeverityText,
			record.SeverityNumber,
			record.ServiceName,
			record.Body,
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

func (s *ClickHouseStore) SearchLogs(ctx context.Context, filter core.LogQueryFilter) ([]core.FlatLogRecord, error){
	queryString := fmt.Sprintf("Select * FROM %v WHERE 1=1", s.dbAndTable)
	var args []any

	if !filter.StartTime.IsZero() {
		queryString += " AND Timestamp >= ?"
		args = append(args, filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		queryString += " AND Timestamp <= ?"
		args = append(args, filter.EndTime)
	}
	if filter.ServiceName != "" {
		queryString += " AND ServiceName ILIKE ?"
		args = append(args, filter.ServiceName + "%")
	}
	if filter.SeverityText != "" {
		queryString += " AND SeverityText ILIKE ?"
		args = append(args, filter.SeverityText + "%")
	}
	if filter.TraceId != "" {
		queryString += " AND TraceId ILIKE ?"
		args = append(args, filter.TraceId + "%")
	}
	if filter.SpanId != "" {
		queryString += " AND SpanId ILIKE ?"
		args = append(args, filter.SpanId + "%")
	}

	if filter.SearchTerm != "" {
		queryString += " AND Body ILIKE ?"
		args = append(args, "%" + filter.SearchTerm + "%")
	}

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

	var result []core.FlatLogRecord
	if err := s.conn.Select(ctx, &result, queryString, args...); err != nil {
		return result, err
	}

	return result, nil
}
