package worker

import (
	"log/slog"
	"sync"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

// create Transformer interface
type Transformer interface {
	Flatten(resourceLogs plog.ResourceLogs, publisher Publisher, appender storage.LogAppender)
}

var _ Transformer = (*LogTransformer)(nil)

type LogTransformer struct {
	logger  *slog.Logger
	bufPool sync.Pool
}

func NewLogTransformer(logger *slog.Logger) *LogTransformer {
	return &LogTransformer{
		logger: logger,
		bufPool: sync.Pool{
			New: newTransformBuffer,
		},
	}
}

// Flatten logs and dispatch to publisher for live tail and fills up appender during decoding
//
// Transformer will use its internal buffers to minimise memory usage if Flatten is called many times.
func (t *LogTransformer) Flatten(resourceLogs plog.ResourceLogs, publisher Publisher, appender storage.LogAppender) {
	transBuf := t.bufPool.Get().(*transformBuffer)
	defer func() { // prevent bloat if large log comes in
		if cap(transBuf.logAttrKeys) > 200 || cap(transBuf.resAttrKeys) > 200 {
			return
		}
		t.bufPool.Put(transBuf)
	}()

	transBuf.Reset()

	serviceName := "unknown"
	nowNano := uint64(time.Now().UnixNano())

	resourceLogs.Resource().Attributes().Range(func(k string, v pcommon.Value) bool {
		valStr := v.AsString()
		if k == "service.name" && valStr != "" {
			serviceName = valStr
		}

		transBuf.resAttrKeys = append(transBuf.resAttrKeys, k)
		transBuf.resAttrValues = append(transBuf.resAttrValues, valStr)
		return true
	})

	for i := 0; i < resourceLogs.ScopeLogs().Len(); i++ {
		scopeLogs := resourceLogs.ScopeLogs().At(i)
		scopeName := scopeLogs.Scope().Name()
		scopeVersion := scopeLogs.Scope().Version()

		for j := 0; j < scopeLogs.LogRecords().Len(); j++ {
			logRecord := scopeLogs.LogRecords().At(j)

			// reset for every log record
			transBuf.logAttrKeys = transBuf.logAttrKeys[:0]
			transBuf.logAttrValues = transBuf.logAttrValues[:0]

			logRecord.Attributes().Range(func(k string, v pcommon.Value) bool {
				transBuf.logAttrKeys = append(transBuf.logAttrKeys, k)
				transBuf.logAttrValues = append(transBuf.logAttrValues, v.AsString())
				return true
			})

			observedTime := uint64(logRecord.ObservedTimestamp()) // want nano time
			if observedTime == 0 {
				observedTime = nowNano
			}

			eventTime := uint64(logRecord.Timestamp()) // want nano time
			if eventTime == 0 {
				eventTime = nowNano
			}

			transBuf.flatLogrecord.Timestamp = time.Unix(0, int64(eventTime))
			transBuf.flatLogrecord.ObservedTimestamp = time.Unix(0, int64(observedTime))
			transBuf.flatLogrecord.TraceId = logRecord.TraceID().String()
			transBuf.flatLogrecord.SpanId = logRecord.SpanID().String()
			transBuf.flatLogrecord.SeverityText = logRecord.SeverityText()
			transBuf.flatLogrecord.SeverityNumber = uint8(logRecord.SeverityNumber())
			transBuf.flatLogrecord.Body = logRecord.Body().AsString()
			transBuf.flatLogrecord.BodyType = "string" // TODO see if there is need to extract original body type
			transBuf.flatLogrecord.ServiceName = serviceName
			transBuf.flatLogrecord.ScopeName = scopeName
			transBuf.flatLogrecord.ScopeVersion = scopeVersion
			transBuf.flatLogrecord.ResAttrKeys = transBuf.resAttrKeys
			transBuf.flatLogrecord.ResAttrValues = transBuf.resAttrValues
			transBuf.flatLogrecord.LogAttrKeys = transBuf.logAttrKeys
			transBuf.flatLogrecord.LogAttrValues = transBuf.logAttrValues

			if !publisher.Enqueue(transport.LiveTailSubjectPrefix+serviceName, &transBuf.flatLogrecord) {
				t.logger.Warn("failed to enqueue log for live tail, queue full")
			}

			appender.Append(
				time.Unix(0, int64(eventTime)),
				time.Unix(0, int64(observedTime)),
				uint8(logRecord.SeverityNumber()),
				logRecord.TraceID(),
				logRecord.SpanID(),
				transBuf.logAttrKeys,
				transBuf.logAttrValues,
				transBuf.resAttrKeys,
				transBuf.resAttrValues,
				storage.LogFields{
					ScopeName:    scopeName,
					ScopeVersion: scopeVersion,
					SeverityText: logRecord.SeverityText(),
					ServiceName:  serviceName,
					Body:         logRecord.Body().AsString(),
					BodyType:     "string",
				},
			)
		}
	}
}

type transformBuffer struct {
	flatLogrecord core.FlatLogRecord
	resAttrKeys   []string
	resAttrValues []string
	logAttrKeys   []string
	logAttrValues []string
}

func newTransformBuffer() any {
	return &transformBuffer{
		flatLogrecord: core.FlatLogRecord{},
		resAttrKeys:   make([]string, 0, 10),
		resAttrValues: make([]string, 0, 10),
		logAttrKeys:   make([]string, 0, 10),
		logAttrValues: make([]string, 0, 10),
	}
}

func (t *transformBuffer) Reset() {
	t.flatLogrecord = core.FlatLogRecord{}
	t.resAttrKeys = t.resAttrKeys[:0]
	t.resAttrValues = t.resAttrValues[:0]
	t.logAttrKeys = t.logAttrKeys[:0]
	t.logAttrValues = t.logAttrValues[:0]
}
