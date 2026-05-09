package core

import (
	"time"
)

type LogRecord struct{
	Timestamp time.Time `json:"Timestamp" ch:"Timestamp"`   

	TraceId string `json:"TraceId" ch:"TraceId"`
	SpanId string `json:"SpanId" ch:"SpanId"`

	SeverityText string `json:"SeverityText" ch:"SeverityText"`
	SeverityNumber uint8 `json:"SeverityNumber" ch:"SeverityNumber"`

	ServiceName string `json:"ServiceName" ch:"ServiceName"`
	Body string `json:"Body" ch:"Body"`

	LogAttributes map[string]any `json:"LogAttributes" ch:"LogAttributes"`
	ResourceAttributes map[string]any `json:"ResourceAttributes" ch:"ResourceAttributes"`
}
	
/*
BatchLogs sent by clients will only include ServiceName and
ResourceAttributes once in the grouped json.
Before inserting to db, worker will insert the ServiceName
and ResourceAttributes to every individual record for consistency
*/
type BatchLogRequest struct {
	ServiceName string `json:"ServiceName"`
	ResourceAttributes map[string]any `json:"ResourceAttributes,omitempty"`
	Records []LogRecord `json:"Records"` 
}

type OrderByField string

const (
	OrderByTimestamp OrderByField = "Timestamp"
	OrderByServiceName OrderByField = "ServiceName"
)

type LogQueryFilter struct {
	StartTime time.Time
	EndTime time.Time
	ServiceName string
	SeverityText string
	TraceId string
	SpanId string
	SearchTerm string // search "Body" field
	Limit int
	OrderBy OrderByField
	Descending bool
}
