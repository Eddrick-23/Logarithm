package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/pool"
	"github.com/nats-io/nats.go/jetstream"
)

type LogStore interface {
	FastInsert(int) LogAppender
	BatchInsert(context.Context, []core.FlatLogRecord) error
	SearchLogs(context.Context, core.LogQueryFilter) ([]core.FlatLogRecord, error)
}

type batchPool interface {
	Get() *columnBatch
	Put(*columnBatch) bool
}

type ClickHouseStore struct {
	conn       driver.Conn
	tables     map[string]string
	logger     *slog.Logger
	ingestMu   sync.Mutex // ch.Client not thread safe
	ingestConn *ch.Client // low level api for inserting
	batchPool  batchPool

	batchPoolSize int
	batchRowLimit int
}

var _ LogStore = (*ClickHouseStore)(nil)

type Config struct {
	Address  string // host:port
	Database string
	Username string
	Password string
}

type Option func(*ClickHouseStore)

func WithLogger(l *slog.Logger) Option {
	return func(s *ClickHouseStore) {
		s.logger = l
	}
}

func WithPoolSize(size int) Option {
	return func(s *ClickHouseStore) {
		s.batchPoolSize = size
	}
}

func WithRowLimit(limit int) Option {
	return func(s *ClickHouseStore) {
		s.batchRowLimit = limit
	}
}

const (
	defaultBatchPoolSize = 5
	defaultBatchRowLimit = 10000
)

func NewClickHouseStore(ctx context.Context, config Config, opts ...Option) (*ClickHouseStore, error) {
	s := &ClickHouseStore{
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		batchPoolSize: defaultBatchPoolSize,
		batchRowLimit: defaultBatchRowLimit,
	}

	for _, opt := range opts {
		opt(s)
	}

	s.logger.Info("Connecting to database...")

	// high level driver
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{config.Address},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to configure clickhouse: %v", err)
	}

	// low level driver
	ingestConn, err := ch.Dial(ctx, ch.Options{
		Address:  config.Address,
		Database: config.Database,
		User:     config.Username,
		Password: config.Password,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to dial ch-go ingest conn: %w", err)
	}

	tables, err := initTables(config.Database)
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

	batchPool := pool.New(s.batchPoolSize, newColumnBatch, pool.WithReuseCheck(func(c *columnBatch) bool {
		return c.Capacity() <= s.batchRowLimit
	}))

	s.conn = conn
	s.ingestConn = ingestConn
	s.tables = tables
	s.batchPool = batchPool

	s.logger.Info("connection to database established")
	return s, nil
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

	var errs []error
	errs = append(errs, s.ingestConn.Close())
	errs = append(errs, s.conn.Close())

	return errors.Join(errs...)
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

// return ingestion graph metrics in the past minute
func (s *ClickHouseStore) GetIngestionGraphMetrics(ctx context.Context) (core.IngestionGraphMetrics, error) {
	tbl, err := s.table(TableMetrics)
	if err != nil {
		return core.IngestionGraphMetrics{}, fmt.Errorf("failed to get table: %v", err)
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
		return core.IngestionGraphMetrics{}, err
	}

	return core.NewIngestionGraphMetrics(rows, INGESTION_METRICS_DURATION*60), nil
}

// returns ingestion graph metrics from a certain time,
// function returns an empty struct if since is newer than the newest data in the database
func (s *ClickHouseStore) GetIngestionGraphMetricsSince(ctx context.Context, since time.Time) (core.IngestionGraphMetrics, error) {
	tbl, err := s.table(TableMetrics)
	if err != nil {
		return core.IngestionGraphMetrics{}, err
	}

	var result core.IngestionGraphMetrics
	end := time.Now().UTC().Truncate(time.Second)
	start := since
	if !start.Before(end) {
		// error handling in case start timing is after end
		return core.IngestionGraphMetrics{}, nil
	}

	whereClause := `WHERE Timestamp >= @start AND Timestamp < @end
					GROUP BY ServiceName, Timestamp
					ORDER BY ServiceName, Timestamp ASC
					WITH FILL
						FROM @start
    					TO @end
						STEP toIntervalSecond(1)`
	queryString := fmt.Sprintf("SELECT Timestamp, ServiceName, sum(LogsCount) AS LogsCount FROM %v %v ", tbl, whereClause)

	var rows []core.IngestionMetrics
	if err := s.conn.Select(ctx, &rows, queryString, clickhouse.Named("start", start), clickhouse.Named("end", end)); err != nil {
		return core.IngestionGraphMetrics{}, err
	}

	numSeconds := int(end.Sub(start).Seconds())
	result = core.NewIngestionGraphMetrics(rows, numSeconds)

	return result, nil
}

func (s *ClickHouseStore) GetErrorRateMetrics(ctx context.Context) (core.ErrorRateMetrics, error) {
	tbl, err := s.table(TableMetrics)
	if err != nil {
		return core.ErrorRateMetrics{}, fmt.Errorf("failed to get table: %v", err)
	}

	queryString := fmt.Sprintf(`
		SELECT
			sum(ErrorsCount) / nullIf(sum(LogsCount), 0) AS CurrentRate
		FROM %v
		WHERE Timestamp >= now() - toIntervalMinute(@minute)
	`, tbl)

	// calculate error rate metrics for the past 5 minutes
	var result core.ErrorRateMetrics
	if err := s.conn.QueryRow(ctx, queryString, clickhouse.Named("minute", 5)).ScanStruct(&result); err != nil {
		return core.ErrorRateMetrics{}, err
	}

	// convert from fraction to percentage to be passed to frontend
	result.CurrentRate = result.CurrentRate * 100
	return result, nil
}

func (s *ClickHouseStore) GetTopServiceErrorsStats(ctx context.Context) ([]core.TopServiceErrorsStats, error) {
	// top service errors stats will return error rates within the past 1 hour
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

	var result []core.TopServiceErrorsStats
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
	if err := s.conn.QueryRow(ctx, queryString, clickhouse.Named("now", now)).ScanStruct(&result); err != nil {
		return core.LogRateStatistics{}, err
	}

	return result, nil
}

func (s *ClickHouseStore) GetStorageStats(ctx context.Context) ([]core.StorageStats, error) {
	queryString := `
        SELECT
            name AS DiskName,
            free_space AS FreeBytes,
            total_space AS TotalBytes,
            round((total_space - free_space) / total_space * 100, 2) AS UsedPercent
        FROM system.disks
    `

	var result []core.StorageStats
	if err := s.conn.Select(ctx, &result, queryString); err != nil {
		return nil, fmt.Errorf("failed to get storage stats: %v", err)
	}
	return result, nil
}

const LOGS_TTL_DAYS = 30

// GetLogsStorageOutlook reports the logs table's current size and, if it
// hasn't yet reached its 30-day TTL steady-state, projects what that will be.
func (s *ClickHouseStore) GetLogsStorageOutlook(ctx context.Context) (*core.StorageOutlook, error) {
	queryString := `
        SELECT
            partition,
            sum(bytes_on_disk) AS PartitionBytes
        FROM system.parts
        WHERE database = 'logarithm' AND table = 'logs' AND active
        GROUP BY partition
        ORDER BY partition
    `
	var partitions []struct {
		Partition      string `ch:"partition"`
		PartitionBytes uint64 `ch:"PartitionBytes"`
	}
	if err := s.conn.Select(ctx, &partitions, queryString); err != nil {
		return nil, fmt.Errorf("failed to get partition sizes: %v", err)
	}

	var currentTotal uint64
	for _, p := range partitions {
		currentTotal += p.PartitionBytes
	}

	result := &core.StorageOutlook{
		CurrentTotalBytes: currentTotal,
		DaysOfHistory:     len(partitions),
	}

	if len(partitions) >= LOGS_TTL_DAYS {
		// Steady-state already reached
		result.IsSteadyState = true
		result.ProjectedSteadyStateBytes = currentTotal
		return result, nil
	}

	var avgDailyBytes uint64
	if len(partitions) == 0 {
		// if there is currently no history, set average daily bytes to be 0
		avgDailyBytes = 0
	} else {
		// Not enough history yet — project forward using recent daily average.
		// Exclude the most recent (still-filling) partition for a fairer average.
		usable := partitions
		if len(usable) > 1 {
			usable = usable[:len(usable)-1]
		}
		var sum uint64
		for _, p := range usable {
			sum += p.PartitionBytes
		}
		avgDailyBytes = sum / uint64(len(usable))
	}

	result.IsSteadyState = false
	result.ProjectedSteadyStateBytes = avgDailyBytes * LOGS_TTL_DAYS
	return result, nil
}

func (s *ClickHouseStore) GetNatsQueueDepthMetrics(ctx context.Context, durationMinutes int) (core.NatsQueueDepthGraphMetrics, error) {
	tbl, err := s.table(TableJetstreamConsumerMetrics)
	if err != nil {
		return core.NatsQueueDepthGraphMetrics{}, fmt.Errorf("failed to get table: %v", err)
	}

	queryString := fmt.Sprintf(`
		SELECT Timestamp, NumPending, NumAckPending
		FROM %s
		WHERE Timestamp >= now() - toIntervalMinute(@duration)
		ORDER BY Timestamp ASC
	`, tbl)

	rows, err := s.conn.Query(ctx, queryString, clickhouse.Named("duration", durationMinutes))
	if err != nil {
		return core.NatsQueueDepthGraphMetrics{}, fmt.Errorf("failed to query consumer info: %v", err)
	}
	defer rows.Close()

	response := core.NatsQueueDepthGraphMetrics{
		Timestamps:    []int64{},
		NumPending:    []uint64{},
		NumAckPending: []uint64{},
	}

	for rows.Next() {
		var timestamp time.Time
		var numPending uint64
		var numAckPending uint64

		if err := rows.Scan(&timestamp, &numPending, &numAckPending); err != nil {
			return core.NatsQueueDepthGraphMetrics{}, fmt.Errorf("failed to scan row: %v", err)
		}
		response.Timestamps = append(response.Timestamps, timestamp.UnixMilli())
		response.NumPending = append(response.NumPending, numPending)
		response.NumAckPending = append(response.NumAckPending, numAckPending)
	}

	if err := rows.Err(); err != nil {
		return core.NatsQueueDepthGraphMetrics{}, fmt.Errorf("row iteration error: %v", err)
	}

	return response, nil
}

func (s *ClickHouseStore) SaveConsumerInfo(ctx context.Context, info *jetstream.ConsumerInfo) error {
	tbl, err := s.table(TableJetstreamConsumerMetrics)
	if err != nil {
		return fmt.Errorf("failed to get table: %v", err)
	}

	queryString := fmt.Sprintf(`
		INSERT INTO %s (
			Timestamp, ConsumerName, StreamName, 
			NumAckPending, NumRedelivered, NumPending
		) VALUES (
			?, ?, ?, ?, ?, ?
		)
	`, tbl)

	err = s.conn.Exec(ctx, queryString,
		time.Now().UTC(),
		info.Name,
		info.Stream,
		info.NumAckPending,
		info.NumRedelivered,
		info.NumPending,
	)
	return err
}
