package schemas

import "time"

// KeyValue represents the standard OTel key-value pair.
type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// LogRecordDTO represents the individual record payload from the frontend/client.
type LogRecordDTO struct { //TODO remove after migration
	Timestamp      time.Time  `json:"timestamp"`
	TraceId        string     `json:"traceId"`
	SpanId         string     `json:"spanId"`
	SeverityText   string     `json:"severityText"`
	SeverityNumber uint8      `json:"severityNumber"`
	Body           string     `json:"body"`
	LogAttributes  []KeyValue `json:"logAttributes"`
}

/*
LogIngestRequest handles batch logs sent by clients.
Includes ServiceName and ResourceAttributes once in the grouped JSON.
*/
type LogIngestRequest struct { //TODO remove after migration
	ServiceName        string         `json:"serviceName"`
	ResourceAttributes []KeyValue     `json:"resourceAttributes,omitempty"`
	Records            []LogRecordDTO `json:"records"`
}
