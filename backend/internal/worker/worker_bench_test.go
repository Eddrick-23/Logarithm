package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/transport"
)

var testRecord1 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-1 * time.Hour),
	ObservedTimestamp: time.Now().Add(-1 * time.Hour),
	TraceId:           "4bf92f3577b34da6a3ce929d0e0e4736",
	SpanId:            "00f067aa0ba902b7",
	SeverityText:      "ERROR",
	SeverityNumber:    17,
	ServiceName:       "test-service1",
	Body:              "Failed to process transaction due to timeout",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"http.method", "http.status_code", "retry_count"},
	LogAttrValues:     []string{"POST", "504", "3"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-1", "linux", "1.2.0"},
}
var testRecord2 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-2 * time.Hour),
	ObservedTimestamp: time.Now().Add(-2 * time.Hour),
	TraceId:           "4bf92f3577b37da6a3ce929d0f0e4736",
	SpanId:            "01f067ef0ba402b7",
	SeverityText:      "WARNING",
	SeverityNumber:    13,
	ServiceName:       "test-service2",
	Body:              "extra information",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"http.method", "http.status_code", "retry_count"},
	LogAttrValues:     []string{"POST", "504", "3"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-2", "windows", "2.0.1"},
}
var testRecord3 core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now(),
	ObservedTimestamp: time.Now(),
	TraceId:           "8bf92f3577b34da6d3ce921d0e0e4536",
	SpanId:            "02y067aa0ba902h3",
	SeverityText:      "INFO",
	SeverityNumber:    9,
	ServiceName:       "test-service3",
	Body:              "Just some test body",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"http.method", "http.status_code", "retry_count"},
	LogAttrValues:     []string{"GET", "500", "3"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-3", "linux", "3.1.0"},
}

type noOpProducer struct{}

func (n *noOpProducer) PublishLogs(_ context.Context, _ string, _ []byte, _ map[string][]string) error {
	return nil
}

func (n *noOpProducer) PublishLiveTail(_ string, _ []byte) error {
	return nil
}

func publishLiveTailJSON(logger *slog.Logger, producer transport.Producer, serviceName string, logs []core.FlatLogRecord) {
	subject := transport.LiveTailSubjectPrefix + serviceName

	for _, record := range logs {
		data, err := json.Marshal(record)
		if err != nil {
			logger.Error("json marshal failed for live tail record", "err", err)
		}

		if err := producer.PublishLiveTail(subject, data); err != nil {
			logger.Error("failed to publish to live tail stream", "err", err)
		}
	}
}

func BenchmarkPublishLiveTail_JSON(b *testing.B) {
	testLogs := []core.FlatLogRecord{
		testRecord1,
		testRecord2,
		testRecord3,
	}
	producer := noOpProducer{}
	b.ResetTimer()
	for b.Loop() {
		publishLiveTailJSON(slog.Default(), &producer, "", testLogs)
	}
}

func BenchmarkPublishLiveTail_MsgPack(b *testing.B) {
	testLogs := []core.FlatLogRecord{
		testRecord1,
		testRecord2,
		testRecord3,
	}
	producer := noOpProducer{}
	b.ResetTimer()
	for b.Loop() {
		publishLiveTail(slog.Default(), &producer, "", testLogs)
	}
}
