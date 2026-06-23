package worker

import (
	"context"
	"sync"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"go.opentelemetry.io/collector/pdata/plog"
)

// MockLogAppender
type MockLogAppender struct {
	appendCount int
	flushErr    error
}

func (a *MockLogAppender) Append(
	timestamp, observedTimestamp time.Time,
	severityNumber uint8,
	traceId [16]byte,
	spanId [8]byte,
	logAttrKeys, logAttrValues, resAttrKeys, resAttrValues []string,
	logFields storage.LogFields,
) {
	a.appendCount++
}

func (a *MockLogAppender) Flush(ctx context.Context) error {
	return a.flushErr
}

// MockLogStore
type MockLogStore struct {
	Appender *MockLogAppender
}

func (m *MockLogStore) FastInsert(preSize int) storage.LogAppender {
	return m.Appender
}

func (m *MockLogStore) BatchInsert(ctx context.Context, records []core.FlatLogRecord) error {
	return nil // not needed for this test
}

func (m *MockLogStore) SearchLogs(ctx context.Context, filter core.LogQueryFilter) ([]core.FlatLogRecord, error) {
	return nil, nil // not needed for this test
}

// MockProducer
type MockProducer struct {
	publishCount int
	mu           sync.Mutex
	publishCh    chan struct{}
}

func (m *MockProducer) PublishLogs(ctx context.Context, subject string, payload []byte, headers map[string][]string) error {
	return nil // Not used here
}

func (m *MockProducer) PublishLiveTail(subject string, data []byte) error {
	m.mu.Lock()
	m.publishCount++
	m.mu.Unlock()

	// Non-blocking send to notify the test runner
	select {
	case m.publishCh <- struct{}{}:
	default:
	}
	return nil
}

// MockDecompressor
type MockDecompressor struct {
	decompressError error
}

func (m *MockDecompressor) decompress(payload []byte, headers map[string][]string) ([]byte, func(), error) {
	return payload, func() {}, m.decompressError
}

// MockMsg
type MockMsg struct {
	data []byte
}

func (m *MockMsg) MarshalMsg(dst []byte) ([]byte, error) {
	return append(dst, m.data...), nil
}

// MockPublisher
type MockPublisher struct {
	enqueueCount   int
	enqueueSuccess bool
	records        []core.FlatLogRecord
}

func (m *MockPublisher) Enqueue(subject string, msg MsgMarshaler) bool {
	m.enqueueCount++
	if record, ok := msg.(*core.FlatLogRecord); ok {
		m.records = append(m.records, *record)
	}

	return m.enqueueSuccess
}

// MockTransformer
type MockTransformer struct {
	flattenCount int
}

func (m *MockTransformer) Flatten(resourceLogs plog.ResourceLogs, publisher Publisher, appender storage.LogAppender) {
	m.flattenCount++
}

// NoOp Mocks
type NoOpDecompressor struct {
}

func (n *NoOpDecompressor) decompress(payload []byte, headers map[string][]string) ([]byte, func(), error) {
	return nil, func() {}, nil
}

type NoOpAppender struct {
}

func (n *NoOpAppender) Append(
	timestamp, observedTimestamp time.Time,
	severityNumber uint8,
	traceId [16]byte,
	spanId [8]byte,
	logAttrKeys, logAttrValues, resAttrKeys, resAttrValues []string,
	logFields storage.LogFields) {
	// do nothing
}

func (n *NoOpAppender) Flush() error {
	return nil
}

type NoOpPublisher struct{}

func (n *NoOpPublisher) Enqueue(subject string, msg MsgMarshaler) bool {
	return true
}

type NoOpTransformer struct {
}

func (n *NoOpTransformer) Flatten(resourceLogs plog.ResourceLogs, publisher Publisher, appender storage.LogAppender) {
	// do nothing
}
