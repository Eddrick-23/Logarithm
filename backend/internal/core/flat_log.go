package core

import "time"

/*
FlatLogRecord is optimized for storage in ClickHouse
*/
//go:generate go run github.com/tinylib/msgp@latest
type FlatLogRecord struct {
	Timestamp         time.Time `ch:"Timestamp" json:"timestamp" msg:"timestamp"`
	ObservedTimestamp time.Time `ch:"ObservedTimestamp" json:"observedTimestamp" msg:"observedTimestamp"`
	InsertedAt        time.Time `ch:"InsertedAt" json:"insertedAt" msg:"insertedAt"` // not mapped when unflattening, for now no need to expose to frontend
	TraceId           string    `ch:"TraceId" json:"traceId" msg:"traceId"`
	SpanId            string    `ch:"SpanId" json:"spanId" msg:"spanId"`
	SeverityText      string    `ch:"SeverityText" json:"severityText" msg:"severityText"`
	SeverityNumber    uint8     `ch:"SeverityNumber" json:"severityNumber" msg:"severityNumber"`
	ServiceName       string    `ch:"ServiceName" json:"serviceName" msg:"serviceName"`
	Body              string    `ch:"Body" json:"body" msg:"body"`
	BodyType          string    `ch:"BodyType" json:"bodyType" msg:"bodyType"` // can be "string"|"json"|"int"|"bool", but for now we convert to string
	ScopeName         string    `ch:"ScopeName" json:"scopeNmae" msg:"scopeName"`
	ScopeVersion      string    `ch:"ScopeVersion" json:"scopeVersion" msg:"scopeVersion"`
	LogAttrKeys       []string  `ch:"LogAttrKeys" json:"logAttrKeys" msg:"logAttrKeys"`
	LogAttrValues     []string  `ch:"LogAttrValues" json:"logAttrValues" msg:"logAttrValues"`
	ResAttrKeys       []string  `ch:"ResAttrKeys" json:"resAttrKeys" msg:"resAttrkeys"`
	ResAttrValues     []string  `ch:"ResAttrValues" json:"resAttrValues" msg:"resAttrValues"`
}
