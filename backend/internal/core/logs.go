package core

import (
	"time"

	"github.com/Eddrick-23/Logarithm/api/schemas"
)

/*
FlatLogRecord is optimized for storage in ClickHouse
*/
type FlatLogRecord struct {
	Timestamp         time.Time `ch:"Timestamp" json:"timestamp"`
	ObservedTimestamp time.Time `ch:"ObservedTimestamp" json:"observedTimestamp"` // NEW
	InsertedAt        time.Time `ch:"InsertedAt" json:"insertedAt"`               // not mapped when unflattening, for now no need to expose to frontend
	TraceId           string    `ch:"TraceId" json:"traceId"`
	SpanId            string    `ch:"SpanId" json:"spanId"`
	SeverityText      string    `ch:"SeverityText" json:"severityText"`
	SeverityNumber    uint8     `ch:"SeverityNumber" json:"severityNumber"`
	ServiceName       string    `ch:"ServiceName" json:"serviceName"`
	Body              string    `ch:"Body" json:"body"`
	BodyType          string    `ch:"BodyType" json:"bodyType"`         // new: e.g. "string"|"json"|"int"|"bool"
	ScopeName         string    `ch:"ScopeName" json:"scopeNmae"`       // NEW: e.g. "go.opentelemetry.io/contrib"
	ScopeVersion      string    `ch:"ScopeVersion" json:"scopeVersion"` // NEW: e.g. "v0.46.0"
	LogAttrKeys       []string  `ch:"LogAttrKeys" json:"logAttrKeys"`
	LogAttrValues     []string  `ch:"LogAttrValues" json:"logAttrValues"`
	ResAttrKeys       []string  `ch:"ResAttrKeys" json:"resAttrKeys"`
	ResAttrValues     []string  `ch:"ResAttrValues" json:"resAttrValues"`
}

/*
LogRecord represents the complete structure when reading back
from ClickHouse to display on the frontend.
*/
type LogRecord struct {
	Timestamp          time.Time          `json:"timestamp"`
	ObservedTimestamp  time.Time          `json:"observedTimestamp"` // NEW
	TraceId            string             `json:"traceId"`
	SpanId             string             `json:"spanId"`
	SeverityText       string             `json:"severityText"`
	SeverityNumber     uint8              `json:"severityNumber"`
	ServiceName        string             `json:"serviceName"`
	Body               string             `json:"body"`
	BodyType           string             `json:"bodyType"`     // NEW
	ScopeName          string             `json:"scopeName"`    // NEW
	ScopeVersion       string             `json:"scopeVersion"` // NEW
	LogAttributes      []schemas.KeyValue `json:"logAttributes"`
	ResourceAttributes []schemas.KeyValue `json:"resourceAttributes"`
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
	logAttributes := make([]schemas.KeyValue, 0, len(flat.LogAttrKeys))
	for i, key := range flat.LogAttrKeys {
		value := ""
		if i < len(flat.LogAttrValues) {
			value = flat.LogAttrValues[i]
		}
		logAttributes = append(logAttributes, schemas.KeyValue{Key: key, Value: value})
	}

	resourceAttributes := make([]schemas.KeyValue, 0, len(flat.ResAttrKeys))
	for i, key := range flat.ResAttrKeys {
		value := ""
		if i < len(flat.ResAttrValues) {
			value = flat.ResAttrValues[i]
		}
		resourceAttributes = append(resourceAttributes, schemas.KeyValue{Key: key, Value: value})
	}

	return LogRecord{
		Timestamp:          flat.Timestamp,
		ObservedTimestamp:  flat.ObservedTimestamp,
		TraceId:            flat.TraceId,
		SpanId:             flat.SpanId,
		SeverityText:       flat.SeverityText,
		SeverityNumber:     flat.SeverityNumber,
		ServiceName:        flat.ServiceName,
		Body:               flat.Body,
		BodyType:           flat.BodyType,
		ScopeName:          flat.ScopeName,
		ScopeVersion:       flat.ScopeVersion,
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
