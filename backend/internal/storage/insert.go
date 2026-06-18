package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
)

const (
	small      = 100
	medium     = 10000
	large      = 100_000
	superLarge = 1_000_000
)

var (
	smallPool = sync.Pool{
		New: func() any {
			return newColumnBatch(small)
		},
	}
	mediumPool = sync.Pool{
		New: func() any {
			return newColumnBatch(medium)
		},
	}
	largePool = sync.Pool{
		New: func() any {
			return newColumnBatch(large)
		},
	}
	superLargePool = sync.Pool{
		New: func() any {
			return newColumnBatch(superLarge)
		},
	}
)

type columnBatch struct {
	Timestamps         []time.Time
	ScopeNames         []string
	ScopeVersions      []string
	TraceIds           []string
	SpanIds            []string
	ObservedTimestamps []time.Time
	SeverityTexts      []string
	SeverityNumbers    []uint8
	ServiceNames       []string
	Bodies             []string
	BodyTypes          []string
	LogAttrKeys        [][]string
	LogAttrValues      [][]string
	ResAttrKeys        [][]string
	ResAttrValues      [][]string
}

func newColumnBatch(n int) *columnBatch {
	return &columnBatch{
		Timestamps:         make([]time.Time, 0, n),
		ScopeNames:         make([]string, 0, n),
		ScopeVersions:      make([]string, 0, n),
		TraceIds:           make([]string, 0, n),
		SpanIds:            make([]string, 0, n),
		ObservedTimestamps: make([]time.Time, 0, n),
		SeverityTexts:      make([]string, 0, n),
		SeverityNumbers:    make([]uint8, 0, n),
		ServiceNames:       make([]string, 0, n),
		Bodies:             make([]string, 0, n),
		BodyTypes:          make([]string, 0, n),
		LogAttrKeys:        make([][]string, 0, n),
		LogAttrValues:      make([][]string, 0, n),
		ResAttrKeys:        make([][]string, 0, n),
		ResAttrValues:      make([][]string, 0, n),
	}
}

func (c *columnBatch) Reset() {
	c.Timestamps = c.Timestamps[:0]
	c.ScopeNames = c.ScopeNames[:0]
	c.ScopeVersions = c.ScopeVersions[:0]
	c.TraceIds = c.TraceIds[:0]
	c.SpanIds = c.SpanIds[:0]
	c.ObservedTimestamps = c.ObservedTimestamps[:0]
	c.SeverityTexts = c.SeverityTexts[:0]
	c.SeverityNumbers = c.SeverityNumbers[:0]
	c.ServiceNames = c.ServiceNames[:0]
	c.Bodies = c.Bodies[:0]
	c.BodyTypes = c.BodyTypes[:0]
	c.LogAttrKeys = c.LogAttrKeys[:0]
	c.LogAttrValues = c.LogAttrValues[:0]
	c.ResAttrKeys = c.ResAttrKeys[:0]
	c.ResAttrValues = c.ResAttrValues[:0]
}

// takes a size n and returns a pointer to a columnBatch with size of appropriate range
//
// Internally, the function decides which sync pool to get the batch from.
// n that is too large is created upfront and given an empty callback to avoid holding on to massive structs.
// Returns a pointer to the columnBatch and a callback which puts the struct back into the pool
// It is the caller's responsibility to put the struct back for reuse
func getBatch(n int) (*columnBatch, func()) {
	var pool *sync.Pool
	var batch *columnBatch

	switch {
	case n <= small:
		pool = &smallPool
	case n <= medium:
		pool = &mediumPool
	case n <= large:
		pool = &largePool
	case n <= superLarge:
		pool = &largePool
	default:
		// anyting larger than super large init once only then let gc cleanup
		b := newColumnBatch(n)
		return b, func() {}
	}

	batch = pool.Get().(*columnBatch)
	batch.Reset()

	return batch, func() {
		pool.Put(batch)
	}
}

func (s *ClickHouseStore) BatchInsert(ctx context.Context, records []core.FlatLogRecord) error {
	if len(records) == 0 {
		return nil
	}

	colBatch, release := getBatch(len(records))
	defer release()

	for _, record := range records {
		colBatch.Timestamps = append(colBatch.Timestamps, record.Timestamp)
		colBatch.ScopeNames = append(colBatch.ScopeNames, record.ScopeName)
		colBatch.ScopeVersions = append(colBatch.ScopeVersions, record.ScopeVersion)
		colBatch.TraceIds = append(colBatch.TraceIds, record.TraceId)
		colBatch.SpanIds = append(colBatch.SpanIds, record.SpanId)
		colBatch.ObservedTimestamps = append(colBatch.ObservedTimestamps, record.ObservedTimestamp)
		colBatch.SeverityTexts = append(colBatch.SeverityTexts, record.SeverityText)
		colBatch.SeverityNumbers = append(colBatch.SeverityNumbers, record.SeverityNumber)
		colBatch.ServiceNames = append(colBatch.ServiceNames, record.ServiceName)
		colBatch.Bodies = append(colBatch.Bodies, record.Body)
		colBatch.BodyTypes = append(colBatch.BodyTypes, record.BodyType)
		colBatch.LogAttrKeys = append(colBatch.LogAttrKeys, record.LogAttrKeys)
		colBatch.LogAttrValues = append(colBatch.LogAttrValues, record.LogAttrValues)
		colBatch.ResAttrKeys = append(colBatch.ResAttrKeys, record.ResAttrKeys)
		colBatch.ResAttrValues = append(colBatch.ResAttrValues, record.ResAttrValues)
	}

	tbl, err := s.table(TableLogs)
	if err != nil {
		return fmt.Errorf("failed to get table: %v", err)
	}

	// must explicitly state all cols since we have an extra insertAt column
	// that clickhouse will fill in itself
	insertStatement := "INSERT INTO " + tbl +
		` (Timestamp, ScopeName, ScopeVersion, TraceId, SpanId, ObservedTimestamp, SeverityText, SeverityNumber,
         ServiceName, Body, BodyType, LogAttrKeys, LogAttrValues, ResAttrKeys, ResAttrValues)`
	batch, err := s.conn.PrepareBatch(ctx, insertStatement)

	if err != nil {
		s.logger.Error("Failed to prepare batch: %v", "err", err)
		return err
	}

	// zero reflection extraction, append fields directly to columns
	if err := batch.Column(0).Append(colBatch.Timestamps); err != nil {
		return fmt.Errorf("failed to append Timestamps: %w", err)
	}
	if err := batch.Column(1).Append(colBatch.ScopeNames); err != nil {
		return fmt.Errorf("failed to append ScopeNames: %w", err)
	}
	if err := batch.Column(2).Append(colBatch.ScopeVersions); err != nil {
		return fmt.Errorf("failed to append ScopeVersions: %w", err)
	}
	if err := batch.Column(3).Append(colBatch.TraceIds); err != nil {
		return fmt.Errorf("failed to append TraceIds: %w", err)
	}
	if err := batch.Column(4).Append(colBatch.SpanIds); err != nil {
		return fmt.Errorf("failed to append SpanIds: %w", err)
	}
	if err := batch.Column(5).Append(colBatch.ObservedTimestamps); err != nil {
		return fmt.Errorf("failed to append ObservedTimestamps: %w", err)
	}
	if err := batch.Column(6).Append(colBatch.SeverityTexts); err != nil {
		return fmt.Errorf("failed to append SeverityTexts: %w", err)
	}
	if err := batch.Column(7).Append(colBatch.SeverityNumbers); err != nil {
		return fmt.Errorf("failed to append SeverityNumbers: %w", err)
	}
	if err := batch.Column(8).Append(colBatch.ServiceNames); err != nil {
		return fmt.Errorf("failed to append ServiceNames: %w", err)
	}
	if err := batch.Column(9).Append(colBatch.Bodies); err != nil {
		return fmt.Errorf("failed to append Bodies: %w", err)
	}
	if err := batch.Column(10).Append(colBatch.BodyTypes); err != nil {
		return fmt.Errorf("failed to append BodyTypes: %w", err)
	}
	if err := batch.Column(11).Append(colBatch.LogAttrKeys); err != nil {
		return fmt.Errorf("failed to append LogAttrKeys: %w", err)
	}
	if err := batch.Column(12).Append(colBatch.LogAttrValues); err != nil {
		return fmt.Errorf("failed to append LogAttrValues: %w", err)
	}
	if err := batch.Column(13).Append(colBatch.ResAttrKeys); err != nil {
		return fmt.Errorf("failed to append ResAttrKeys: %w", err)
	}
	if err := batch.Column(14).Append(colBatch.ResAttrValues); err != nil {
		return fmt.Errorf("failed to append ResAttrValues: %w", err)
	}

	if err := batch.Send(); err != nil {
		return err
	}

	return nil
}
