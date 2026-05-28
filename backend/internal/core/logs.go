package core

import (
	"time"
)

// KeyValue represents the standard OTel key-value pair.
type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// LogRecordDTO represents the individual record payload from the frontend/client.
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
LogIngestRequest handles batch logs sent by clients.
Includes ServiceName and ResourceAttributes once in the grouped JSON.
*/
type LogIngestRequest struct {
	ServiceName        string         `json:"serviceName"`
	ResourceAttributes []KeyValue     `json:"resourceAttributes,omitempty"`
	Records            []LogRecordDTO `json:"records"`
}

/*
FlatLogRecord is optimized for storage in ClickHouse
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

/*
LogRecord represents the complete structure when reading back
from ClickHouse to display on the frontend.
*/
type LogRecord struct {
	Timestamp          time.Time  `json:"timestamp"`
	TraceId            string     `json:"traceId"`
	SpanId             string     `json:"spanId"`
	SeverityText       string     `json:"severityText"`
	SeverityNumber     uint8      `json:"severityNumber"`
	ServiceName        string     `json:"serviceName"`
	Body               string     `json:"body"`
	LogAttributes      []KeyValue `json:"logAttributes"`
	ResourceAttributes []KeyValue `json:"resourceAttributes"`
}

type OrderByField string

const (
	OrderByTimestamp   OrderByField = "Timestamp"
	OrderByServiceName OrderByField = "ServiceName"
)

type LogQueryFilter struct {
	StartTime      time.Time
	EndTime        time.Time
	ServiceName    string
	SeverityNumber int
	SeverityText   string
	TraceId        string
	SpanId         string
	Body           string
	Limit          int
	Offset         int
	OrderBy        OrderByField
	Descending     bool
}

func ParseOrderByField(s string) OrderByField {
	switch OrderByField(s) {
	case OrderByTimestamp, OrderByServiceName:
		return OrderByField(s)
	default:
		return ""
	}
}

func unflattenLogRecord(flat FlatLogRecord) LogRecord {
	logAttributes := make([]KeyValue, 0, len(flat.LogAttrKeys))
	for i, key := range flat.LogAttrKeys {
		value := ""
		if i < len(flat.LogAttrValues) {
			value = flat.LogAttrValues[i]
		}
		logAttributes = append(logAttributes, KeyValue{Key: key, Value: value})
	}

	resourceAttributes := make([]KeyValue, 0, len(flat.ResAttrKeys))
	for i, key := range flat.ResAttrKeys {
		value := ""
		if i < len(flat.ResAttrValues) {
			value = flat.ResAttrValues[i]
		}
		resourceAttributes = append(resourceAttributes, KeyValue{Key: key, Value: value})
	}

	return LogRecord{
		Timestamp:          flat.Timestamp,
		TraceId:            flat.TraceId,
		SpanId:             flat.SpanId,
		SeverityText:       flat.SeverityText,
		SeverityNumber:     flat.SeverityNumber,
		ServiceName:        flat.ServiceName,
		Body:               flat.Body,
		LogAttributes:      logAttributes,
		ResourceAttributes: resourceAttributes,
	}
}

func UnflattenLogRecords(flats []FlatLogRecord) []LogRecord {
	dtos := make([]LogRecord, len(flats))
	for i, flat := range flats {
		dtos[i] = unflattenLogRecord(flat)
	}
	return dtos
}
