package storage

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Eddrick-23/Logarithm/internal/core"
	"golang.org/x/sync/errgroup"
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

const INGESTION_METRICS_DURATION = 1 // it is in minutes

// ingestion graph only, TODO: remove this in favour of SSE
func (s *ClickHouseStore) GetIngestionMetrics(ctx context.Context) (core.IngestionMetricsResponse, error) {
	tbl, err := s.table(TableMetrics)
	if err != nil {
		return core.IngestionMetricsResponse{}, fmt.Errorf("failed to get table: %v", err)
	}

	whereClause := `WHERE Timestamp >= @start AND Timestamp < @end
					GROUP BY ServiceName, Timestamp
					ORDER BY ServiceName, Timestamp ASC
					WITH FILL
						FROM @start
    					TO @end
						STEP toIntervalSecond(1)`
	// back fills timestamps with no values
	queryString := fmt.Sprintf("SELECT Timestamp, ServiceName, sum(LogsCount) AS LogsCount FROM %v %v ", tbl, whereClause)
	end := time.Now().UTC().Truncate(time.Second)
	start := end.Add(-time.Duration(INGESTION_METRICS_DURATION) * time.Minute)

	var rows []core.IngestionMetrics
	// for now, im taking the metrics in the past 1 min, can be adjusted based on specifications
	if err := s.conn.Select(ctx, &rows, queryString, clickhouse.Named("start", start), clickhouse.Named("end", end)); err != nil {
		return core.IngestionMetricsResponse{}, err
	}

	return core.NewIngestionMetricsResponse(rows, INGESTION_METRICS_DURATION*60), nil
}

// all ingestion metrics
func (s *ClickHouseStore) GetAllIngestionMetrics(ctx context.Context) (core.IngestionMetricsEvent, error) {
	tbl, err := s.table(TableMetrics)
	if err != nil {
		return core.IngestionMetricsEvent{}, err
	}

	var (
		result      core.IngestionMetricsEvent
		ingestionMu sync.Mutex
		eg, egCtx   = errgroup.WithContext(ctx)
	)

	// query 1: ingestion graph points (last 60 ticks for the chart)
	eg.Go(func() error {
		end := time.Now().UTC().Truncate(time.Second)
		start := end.Add(-time.Duration(INGESTION_METRICS_DURATION) * time.Minute)

		whereClause := `WHERE Timestamp >= @start AND Timestamp < @end
					GROUP BY ServiceName, Timestamp
					ORDER BY ServiceName, Timestamp ASC
					WITH FILL
						FROM @start
    					TO @end
						STEP toIntervalSecond(1)`
		queryString := fmt.Sprintf("SELECT Timestamp, ServiceName, sum(LogsCount) AS LogsCount FROM %v %v ", tbl, whereClause)

		var rows []core.IngestionMetrics
		if err := s.conn.Select(egCtx, &rows, queryString, clickhouse.Named("start", start), clickhouse.Named("end", end)); err != nil {
			return fmt.Errorf("ingestion graph: %w", err)
		}

		ingestionMu.Lock()
		result.Graph = core.NewIngestionMetricsResponse(rows, INGESTION_METRICS_DURATION*60)
		ingestionMu.Unlock()
		return nil
	})

	// query 2: log rate stats
	eg.Go(func() error {
		now := time.Now().UTC().Truncate(time.Second)

		queryString := fmt.Sprintf(`
		WITH
			current AS (
				SELECT sum(LogsCount) / 5 AS rate
				FROM %v
				WHERE Timestamp >= @now - INTERVAL 5 SECOND
			),
			baseline AS (
				SELECT sum(LogsCount) / 60 AS rate
				FROM %v
				WHERE Timestamp >= @now - INTERVAL 60 SECOND
			)
		SELECT
			current.rate AS CurrentRate,
			baseline.rate AS AvgRate,
			current.rate / nullIf(baseline.rate, 0) AS Ratio
		FROM current, baseline
	`, tbl, tbl)

		var row core.LogRateStatistics
		if err := s.conn.QueryRow(egCtx, queryString, clickhouse.Named("now", now)).ScanStruct(&row); err != nil {
			return fmt.Errorf("log rate stats: %w", err)
		}

		ingestionMu.Lock()
		result.LogStats = row
		ingestionMu.Unlock()
		return nil
	})

	if err := eg.Wait(); err != nil {
		return core.IngestionMetricsEvent{}, err
	}

	return result, nil

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

func (s *ClickHouseStore) GetLogRateStatistics(ctx context.Context) (core.LogRateStatistics, error) {
	tbl, err := s.table(TableMetrics)
	if err != nil {
		return core.LogRateStatistics{}, fmt.Errorf("failed to get table: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)

	// current rate: logs / second in past 5 seconds
	// average rate: logs / second in past 1 minute
	queryString := fmt.Sprintf(`
		WITH
			current AS (
				SELECT sum(LogsCount) / 5 AS rate
				FROM %v
				WHERE Timestamp >= @now - INTERVAL 5 SECOND
			),
			baseline AS (
				SELECT sum(LogsCount) / 60 AS rate
				FROM %v
				WHERE Timestamp >= @now - INTERVAL 60 SECOND
			)
		SELECT
			current.rate AS CurrentRate,
			baseline.rate AS AvgRate,
			current.rate / nullIf(baseline.rate, 0) AS Ratio
		FROM current, baseline
	`, tbl, tbl)

	var result core.LogRateStatistics
	if err := s.conn.Select(ctx, &result, queryString, clickhouse.Named("now", now)); err != nil {
		return core.LogRateStatistics{}, err
	}

	return result, nil
}
