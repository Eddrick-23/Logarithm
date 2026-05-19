package core

import (
	"time"
)

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type LogRecordDTO struct {
	Timestamp      time.Time  `json:"timestamp"`
	TraceId        string     `json:"traceId"`
	SpanId         string     `json:"spanId"`
	SeverityText   string     `json:"severityText"`
	SeverityNumber uint8      `json:"severityNumber"`
	Body           string     `json:"body"`
	LogAttributes  []KeyValue `json:"logAttributes"`
}

/*
Batch logs sent by clients will only include ServiceName and
ResourceAttributes once in the grouped json.
Before inserting to db, worker will insert the ServiceName
and ResourceAttributes to every individual record for consistency
*/
type LogIngestRequest struct {
	ServiceName        string         `json:"serviceName"`
	ResourceAttributes []KeyValue     `json:"resourceAttributes,omitempty"`
	Records            []LogRecordDTO `json:"records"`
}

/*
Store in a flattened structure for efficient storage and transer
for NATS and Clickhouse
*/
type FlatLogRecord struct {
	Timestamp      time.Time `ch:"Timestamp"`
	TraceId        string    `ch:"TraceId"`
	SpanId         string    `ch:"SpanId"`
	SeverityText   string    `ch:"SeverityText"`
	SeverityNumber uint8     `ch:"SeverityNumber"`
	ServiceName    string    `ch:"ServiceName"`
	Body           string    `ch:"Body"`
	LogAttrKeys    []string  `ch:"LogAttrKeys"`
	LogAttrValues  []string  `ch:"LogAttrValues"`
	ResAttrKeys    []string  `ch:"ResAttrKeys"`
	ResAttrValues  []string  `ch:"ResAttrValues"`
}

type OrderByField string

const (
	OrderByTimestamp   OrderByField = "Timestamp"
	OrderByServiceName OrderByField = "ServiceName"
)

type LogQueryFilter struct {
	StartTime    time.Time
	EndTime      time.Time
	ServiceName  string
	SeverityText string
	TraceId      string
	SpanId       string
	SearchTerm   string // search "Body" field
	Limit        int
	OrderBy      OrderByField
	Descending   bool
}
