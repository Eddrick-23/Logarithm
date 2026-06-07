package main

import (
	"encoding/hex"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	v1 "go.opentelemetry.io/proto/otlp/logs/v1"
)

func flattenLogs(resourceLogs *v1.ResourceLogs, flatLogsByServiceName map[string][]core.FlatLogRecord) int {
	serviceName := "unknown"
	resAttrKeys := []string{}
	resAttrValues := []string{}
	nowNano := uint64(time.Now().UnixNano())

	count := 0

	for _, attr := range resourceLogs.Resource.GetAttributes() {
		valStr := attr.Value.GetStringValue()

		if attr.Key == "service.name" && valStr != "" {
			serviceName = valStr
		}

		resAttrKeys = append(resAttrKeys, attr.Key)
		resAttrValues = append(resAttrValues, valStr)
	}

	for _, scopeLogs := range resourceLogs.ScopeLogs {
		scopeName := scopeLogs.Scope.GetName()
		scopeVersion := scopeLogs.Scope.GetVersion()

		for _, logRecord := range scopeLogs.LogRecords {
			logAttrKeys := []string{}
			logAttrValues := []string{}

			for _, attr := range logRecord.GetAttributes() {
				logAttrKeys = append(logAttrKeys, attr.Key)
				logAttrValues = append(logAttrValues, attr.Value.GetStringValue())
			}

			// handle missing timestamps
			observedTime := logRecord.GetObservedTimeUnixNano()
			if observedTime == 0 {
				observedTime = nowNano
			}

			eventTime := logRecord.GetTimeUnixNano()
			if eventTime == 0 {
				eventTime = nowNano
			}

			flatRecord := core.FlatLogRecord{
				Timestamp:         time.Unix(0, int64(eventTime)),
				ObservedTimestamp: time.Unix(0, int64(observedTime)),
				TraceId:           hex.EncodeToString(logRecord.GetTraceId()),
				SpanId:            hex.EncodeToString(logRecord.GetSpanId()),
				SeverityText:      logRecord.SeverityText,
				SeverityNumber:    uint8(logRecord.SeverityNumber),
				Body:              logRecord.Body.GetStringValue(),
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
