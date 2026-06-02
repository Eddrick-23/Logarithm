package main

import (
	"github.com/Eddrick-23/Logarithm/api/schemas"
	"github.com/Eddrick-23/Logarithm/internal/core"
)

func extractAttributes(attributes []schemas.KeyValue) ([]string, []string) {
	numAttributes := len(attributes)

	keys := make([]string, 0, numAttributes)
	values := make([]string, 0, numAttributes)

	for _, pair := range attributes {
		keys = append(keys, pair.Key)
		values = append(values, pair.Value)
	}

	return keys, values
}

func flattenLogs(ingestedReq schemas.LogIngestRequest, buffer []core.FlatLogRecord) []core.FlatLogRecord {
	resKeys, resValues := extractAttributes(ingestedReq.ResourceAttributes)

	for _, logDTO := range ingestedReq.Records {
		logKeys, logValues := extractAttributes(logDTO.LogAttributes)

		flatLog := core.FlatLogRecord{
			Timestamp:      logDTO.Timestamp,
			TraceId:        logDTO.TraceId,
			SpanId:         logDTO.SpanId,
			SeverityText:   logDTO.SeverityText,
			SeverityNumber: logDTO.SeverityNumber,
			ServiceName:    ingestedReq.ServiceName,
			Body:           logDTO.Body,
			LogAttrKeys:    logKeys,
			LogAttrValues:  logValues,
			ResAttrKeys:    resKeys,
			ResAttrValues:  resValues,
		}

		buffer = append(buffer, flatLog)
	}

	return buffer
}
