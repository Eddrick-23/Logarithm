package worker

import (
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

func flattenLogs(resourceLogs plog.ResourceLogs, flatLogsByServiceName map[string][]core.FlatLogRecord) int {
	serviceName := "unknown"
	resAttrKeys := []string{}
	resAttrValues := []string{}
	nowNano := uint64(time.Now().UnixNano())

	count := 0

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
			logAttrKeys := []string{}
			logAttrValues := []string{}

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

			flatLogsByServiceName[serviceName] = append(flatLogsByServiceName[serviceName], flatRecord)
			count++
		}
	}

	return count
}
