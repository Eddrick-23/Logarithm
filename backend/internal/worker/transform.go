package worker

import (
	"log/slog"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

func flattenLogs(logger *slog.Logger, resourceLogs plog.ResourceLogs, publisher Publisher, appender storage.LogAppender) {
	serviceName := "unknown"
	resAttrKeys := []string{}
	resAttrValues := []string{}
	logAttrKeys := []string{}
	logAttrValues := []string{}
	nowNano := uint64(time.Now().UnixNano())

	resourceLogs.Resource().Attributes().Range(func(k string, v pcommon.Value) bool {
		valStr := v.AsString()
		if k == "service.name" && valStr != "" {
			serviceName = valStr
		}

		resAttrKeys = append(resAttrKeys, k)
		resAttrValues = append(resAttrValues, valStr)
		return true
	})

	for i := 0; i < resourceLogs.ScopeLogs().Len(); i++ {
		scopeLogs := resourceLogs.ScopeLogs().At(i)
		scopeName := scopeLogs.Scope().Name()
		scopeVersion := scopeLogs.Scope().Version()

		for j := 0; j < scopeLogs.LogRecords().Len(); j++ {
			logRecord := scopeLogs.LogRecords().At(j)
			logAttrKeys = logAttrKeys[:0]
			logAttrValues = logAttrValues[:0]

			logRecord.Attributes().Range(func(k string, v pcommon.Value) bool {
				logAttrKeys = append(logAttrKeys, k)
				logAttrValues = append(logAttrValues, v.AsString())
				return true
			})

			observedTime := uint64(logRecord.ObservedTimestamp()) // want nano time
			if observedTime == 0 {
				observedTime = nowNano
			}

			eventTime := uint64(logRecord.Timestamp()) // want nanot time
			if eventTime == 0 {
				eventTime = nowNano
			}

			flatRecord := core.FlatLogRecord{
				Timestamp:         time.Unix(0, int64(eventTime)),
				ObservedTimestamp: time.Unix(0, int64(observedTime)),
				TraceId:           logRecord.TraceID().String(),
				SpanId:            logRecord.SpanID().String(),
				SeverityText:      logRecord.SeverityText(),
				SeverityNumber:    uint8(logRecord.SeverityNumber()),
				Body:              logRecord.Body().AsString(),
				BodyType:          "string", // or find a way to extract the original body type?

				ServiceName:  serviceName,
				ScopeName:    scopeName,
				ScopeVersion: scopeVersion,

				ResAttrKeys:   resAttrKeys,
				ResAttrValues: resAttrValues,
				LogAttrKeys:   logAttrKeys,
				LogAttrValues: logAttrValues,
			}

			if !publisher.Enqueue(transport.LiveTailSubjectPrefix+serviceName, &flatRecord) {
				logger.Warn("failed to enqueue log for live tail, queue full")
			}

			appender.Append(
				time.Unix(0, int64(eventTime)),
				time.Unix(0, int64(observedTime)),
				uint8(logRecord.SeverityNumber()),
				logRecord.TraceID(),
				logRecord.SpanID(),
				logAttrKeys,
				logAttrValues,
				resAttrKeys,
				resAttrValues,
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
