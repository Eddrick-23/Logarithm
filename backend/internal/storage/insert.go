package storage

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/proto"
	"github.com/Eddrick-23/Logarithm/internal/core"
)

type LogFields struct {
	ScopeName    string
	ScopeVersion string
	SeverityText string
	ServiceName  string
	Body         string
	BodyType     string
}

type LogAppender interface {
	Append(
		timestamp, observedTimestamp time.Time,
		severityNumber uint8,
		traceId [16]byte,
		spanId [8]byte,
		logAttrKeys, logAttrValues, resAttrKeys, resAttrValues []string,
		logFields LogFields,
	)

	Flush(context.Context) error // call back to insert + return underlying col batch to sync pool
}

type batchAppender struct {
	store *ClickHouseStore
	batch *columnBatch
}

func (b *batchAppender) Append(
	timestamp, observedTimestamp time.Time,
	severityNumber uint8,
	traceId [16]byte,
	spanId [8]byte,
	logAttrKeys, logAttrValues, resAttrKeys, resAttrValues []string,
	logFields LogFields,
) {
	b.batch.Timestamps.Append(timestamp)
	b.batch.ObservedTimestamps.Append(observedTimestamp)
	b.batch.SeverityNumbers.Append(severityNumber)
	var traceIdHex [32]byte
	hex.Encode(traceIdHex[:], traceId[:])
	b.batch.TraceIds.Append(traceIdHex)

	var spanIdHex [16]byte
	hex.Encode(spanIdHex[:], spanId[:])
	b.batch.SpanIds.Append(spanIdHex)

	b.batch.LogAttrKeys.Append(logAttrKeys)
	b.batch.LogAttrValues.Append(logAttrValues)
	b.batch.ResAttrKeys.Append(resAttrKeys)
	b.batch.ResAttrValues.Append(resAttrValues)

	b.batch.ScopeNames.Append(logFields.ScopeName)
	b.batch.ScopeVersions.Append(logFields.ScopeVersion)
	b.batch.SeverityTexts.Append(logFields.SeverityText)
	b.batch.ServiceNames.Append(logFields.ServiceName)
	b.batch.Bodies.Append(logFields.Body)
	b.batch.BodyTypes.Append(logFields.BodyType)
}

func (b *batchAppender) Flush(ctx context.Context) error {
	defer b.store.batchPool.Put(b.batch)
	return b.store.insertRecords(ctx, b.batch)
}

type columnBatch struct {
	Timestamps         proto.ColDateTime64
	ScopeNames         *proto.ColLowCardinality[string]
	ScopeVersions      *proto.ColLowCardinality[string]
	TraceIds           proto.ColFixedStr32
	SpanIds            proto.ColFixedStr16
	ObservedTimestamps proto.ColDateTime64
	SeverityTexts      *proto.ColLowCardinality[string]
	SeverityNumbers    proto.ColUInt8
	ServiceNames       *proto.ColLowCardinality[string]
	Bodies             proto.ColStr
	BodyTypes          *proto.ColLowCardinality[string]
	LogAttrKeys        *proto.ColArr[string]
	LogAttrValues      *proto.ColArr[string]
	ResAttrKeys        *proto.ColArr[string]
	ResAttrValues      *proto.ColArr[string]
}

func newColumnBatch() *columnBatch {
	return &columnBatch{
		Timestamps:         *new(proto.ColDateTime64).WithLocation(time.UTC).WithPrecision(proto.PrecisionNano),
		ScopeNames:         new(proto.ColStr).LowCardinality(),
		ScopeVersions:      new(proto.ColStr).LowCardinality(),
		TraceIds:           *new(proto.ColFixedStr32),
		SpanIds:            *new(proto.ColFixedStr16),
		ObservedTimestamps: *new(proto.ColDateTime64).WithLocation(time.UTC).WithPrecision(proto.PrecisionNano),
		SeverityTexts:      new(proto.ColStr).LowCardinality(),
		SeverityNumbers:    *new(proto.ColUInt8),
		ServiceNames:       new(proto.ColStr).LowCardinality(),
		Bodies:             *new(proto.ColStr),
		BodyTypes:          new(proto.ColStr).LowCardinality(),
		LogAttrKeys:        new(proto.ColStr).Array(),
		LogAttrValues:      new(proto.ColStr).Array(),
		ResAttrKeys:        new(proto.ColStr).Array(),
		ResAttrValues:      new(proto.ColStr).Array(),
	}
}

func (c *columnBatch) Reset() {
	c.Timestamps.Reset()
	c.ScopeNames.Reset()
	c.ScopeVersions.Reset()
	c.TraceIds.Reset()
	c.SpanIds.Reset()
	c.ObservedTimestamps.Reset()
	c.SeverityTexts.Reset()
	c.SeverityNumbers.Reset()
	c.ServiceNames.Reset()
	c.Bodies.Reset()
	c.BodyTypes.Reset()
	c.LogAttrKeys.Reset()
	c.LogAttrValues.Reset()
	c.ResAttrKeys.Reset()
	c.ResAttrValues.Reset()
}

// keep private to call internally as of now.
// May expose to clients for better dynamic sizing in the future.
func (c *columnBatch) ensureSize(targetRows int) {
	const estimatedBodyBytesPerRow = 50
	const estimatedKeyBytes = 50
	const estimatedValBytes = 100
	const estimatedAttrPerRow = 5

	existingRows := cap(c.Timestamps.Data)
	if existingRows >= targetRows {
		return
	}

	// resize only if existing rows is smaller
	c.Timestamps.Data = make([]proto.DateTime64, 0, targetRows)
	c.ScopeNames.Values = make([]string, 0, targetRows)
	c.ScopeVersions.Values = make([]string, 0, targetRows)

	c.TraceIds = make(proto.ColFixedStr32, 0, targetRows)
	c.SpanIds = make(proto.ColFixedStr16, 0, targetRows)
	c.ObservedTimestamps.Data = make([]proto.DateTime64, 0, targetRows)
	c.SeverityTexts.Values = make([]string, 0, targetRows)
	c.SeverityNumbers = make(proto.ColUInt8, 0, targetRows)
	c.ServiceNames.Values = make([]string, 0, targetRows)
	c.Bodies.Buf = make([]byte, 0, targetRows*estimatedBodyBytesPerRow)
	c.Bodies.Pos = make([]proto.Position, 0, targetRows)
	c.BodyTypes.Values = make([]string, 0, targetRows)

	// Preallocating the ColArr[T] type
	// ch-go's ColArr works like so:
	// It is a generic with a Data field that holds columnOf interface.
	// We hold T of type string, so underlying its a proto.Colstr that satisfies the interface
	// ColStr internally holds a Buf:[]byte and a Pos:[]Position. Position tells us []byte[start:end] is a word.
	// Buf is preallocated with targetRows * estimatedKeyBytes(total bytes per []string appended)
	// Pos is preallocated with targetRows * estimatedAttrPerRow(length of []string appended)
	// ColArr now wraps ColStr and does the row tracking with its Offsets slice(Pos[start:end] is one row).
	// Overall structure:
	// Row 0 (LogAttrKeys for record 0): ["env", "region"]
	// Row 1 (LogAttrKeys for record 1): ["pod"]
	// Row 2 (LogAttrKeys for record 2): ["env", "pod", "zone"]

	// Data (a *ColStr, flattened):
	//   Buf: [e n v r e g i o n p o d e n v p o d z o n e]
	//   Pos: [{0,3}, {3,9}, {9,12}, {12,15}, {15,18}, {18,22}]
	//           "env"  "region" "pod"   "env"   "pod"   "zone"

	// Offsets (ColUInt64, cumulative count of elements per row):
	//   [2, 3, 6]
	//   row 0 = Data rows [0:2]   = "env", "region"
	//   row 1 = Data rows [2:3]   = "pod"
	//   row 2 = Data rows [3:6]   = "env", "pod", "zone"
	if k, ok := c.LogAttrKeys.Data.(*proto.ColStr); ok {
		k.Buf = make([]byte, 0, targetRows*estimatedKeyBytes)
		k.Pos = make([]proto.Position, 0, targetRows*estimatedAttrPerRow)
	}
	c.LogAttrKeys.Offsets = make(proto.ColUInt64, 0, targetRows)

	if k, ok := c.LogAttrValues.Data.(*proto.ColStr); ok {
		k.Buf = make([]byte, 0, targetRows*estimatedKeyBytes)
		k.Pos = make([]proto.Position, 0, targetRows*estimatedAttrPerRow)
	}
	c.LogAttrValues.Offsets = make(proto.ColUInt64, 0, targetRows)

	if k, ok := c.ResAttrKeys.Data.(*proto.ColStr); ok {
		k.Buf = make([]byte, 0, targetRows*estimatedKeyBytes)
		k.Pos = make([]proto.Position, 0, targetRows*estimatedAttrPerRow)
	}
	c.ResAttrKeys.Offsets = make(proto.ColUInt64, 0, targetRows)

	if k, ok := c.ResAttrValues.Data.(*proto.ColStr); ok {
		k.Buf = make([]byte, 0, targetRows*estimatedKeyBytes)
		k.Pos = make([]proto.Position, 0, targetRows*estimatedAttrPerRow)
	}
	c.ResAttrValues.Offsets = make(proto.ColUInt64, 0, targetRows)
}

// Returns an interface for optimised inserts.
//
// It is the callers responsibility to append required fields.
// Then call the Flush() interface method to perform the insert.
// This avoids intermediate allocations and fills out ch-go's internal
// column buffers directly.
func (s *ClickHouseStore) FastInsert(preSize int) LogAppender {
	colBatch := s.batchPool.Get().(*columnBatch)
	colBatch.Reset()
	colBatch.ensureSize(preSize)
	return &batchAppender{
		store: s,
		batch: colBatch,
	}
}

func traceIdToBytes(s string) ([32]byte, error) {
	var v [32]byte
	if len(s) != 32 {
		return v, fmt.Errorf("invalid traceId")
	}
	copy(v[:], s)
	return v, nil
}

func spanIdToBytes(s string) ([16]byte, error) {
	var v [16]byte
	if len(s) != 16 {
		return v, fmt.Errorf("invalid spanId")
	}
	copy(v[:], s)
	return v, nil
}

// Batch Insert by passing in a slice of FlatLogRecords.
//
// Performs transformation to fill up ch-go column buffers under the hood.
// If batches are large consider using the FastInsert interface.
func (s *ClickHouseStore) BatchInsert(ctx context.Context, records []core.FlatLogRecord) error {
	if len(records) == 0 {
		return nil
	}

	colBatch := s.batchPool.Get().(*columnBatch)
	colBatch.Reset()
	colBatch.ensureSize(len(records))
	defer s.batchPool.Put(colBatch)

	for _, record := range records {
		traceIdBytes, err := traceIdToBytes(record.TraceId)
		if err != nil {
			return err
		}
		colBatch.TraceIds.Append(traceIdBytes)

		spanIdBytes, err := spanIdToBytes(record.SpanId)
		if err != nil {
			return err
		}
		colBatch.SpanIds.Append(spanIdBytes)

		colBatch.Timestamps.Append(record.Timestamp)
		colBatch.ScopeNames.Append(record.ScopeName)
		colBatch.ScopeVersions.Append(record.ScopeVersion)
		colBatch.ObservedTimestamps.Append(record.ObservedTimestamp)
		colBatch.SeverityTexts.Append(record.SeverityText)
		colBatch.SeverityNumbers.Append(record.SeverityNumber)
		colBatch.ServiceNames.Append(record.ServiceName)
		colBatch.Bodies.Append(record.Body)
		colBatch.BodyTypes.Append(record.BodyType)
		colBatch.LogAttrKeys.Append(record.LogAttrKeys)
		colBatch.LogAttrValues.Append(record.LogAttrValues)
		colBatch.ResAttrKeys.Append(record.ResAttrKeys)
		colBatch.ResAttrValues.Append(record.ResAttrValues)
	}

	return s.insertRecords(ctx, colBatch)
}

func (s *ClickHouseStore) insertRecords(ctx context.Context, colBatch *columnBatch) error {
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()

	tbl, err := s.table(TableLogs)
	if err != nil {
		return fmt.Errorf("failed to get table: %v", err)
	}

	// must explicitly state all cols since we have an extra insertAt column
	// that clickhouse will fill in itself
	insertStatement := "INSERT INTO " + tbl +
		` (Timestamp, ScopeName, ScopeVersion, TraceId, SpanId, ObservedTimestamp, SeverityText, SeverityNumber,
         ServiceName, Body, BodyType, LogAttrKeys, LogAttrValues, ResAttrKeys, ResAttrValues) VALUES`

	input := proto.Input{
		{Name: "Timestamp", Data: colBatch.Timestamps},
		{Name: "ScopeName", Data: colBatch.ScopeNames},
		{Name: "ScopeVersion", Data: colBatch.ScopeVersions},
		{Name: "TraceId", Data: colBatch.TraceIds},
		{Name: "SpanId", Data: colBatch.SpanIds},
		{Name: "ObservedTimestamp", Data: colBatch.ObservedTimestamps},
		{Name: "SeverityText", Data: colBatch.SeverityTexts},
		{Name: "SeverityNumber", Data: colBatch.SeverityNumbers},
		{Name: "ServiceName", Data: colBatch.ServiceNames},
		{Name: "Body", Data: colBatch.Bodies},
		{Name: "BodyType", Data: colBatch.BodyTypes},
		{Name: "LogAttrKeys", Data: colBatch.LogAttrKeys},
		{Name: "LogAttrValues", Data: colBatch.LogAttrValues},
		{Name: "ResAttrKeys", Data: colBatch.ResAttrKeys},
		{Name: "ResAttrValues", Data: colBatch.ResAttrValues},
	}

	if err := s.ingestConn.Do(ctx, ch.Query{
		Body:  insertStatement,
		Input: input,
	}); err != nil {
		return err
	}

	return nil
}
