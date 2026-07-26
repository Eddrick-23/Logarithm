package worker

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

// MockLogAppender
type MockLogAppender struct {
	appendCount atomic.Int64
	flushErr    error
}

// Safe to call concurrently
// Does not store the row being appended. Only tracks call count.
func (m *MockLogAppender) Append(
	timestamp, observedTimestamp time.Time,
	severityNumber uint8,
	traceId [16]byte,
	spanId [8]byte,
	logAttrKeys, logAttrValues, resAttrKeys, resAttrValues []string,
	logFields storage.LogFields,
) {
	m.appendCount.Add(1)
}

func (m *MockLogAppender) Flush(ctx context.Context) error {
	return m.flushErr
}

func (m *MockLogAppender) GetAppendCount() int {
	return int(m.appendCount.Load())
}

// MockLogStore
type MockLogStore struct {
	Appender storage.LogAppender
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
	publishCount atomic.Int64
	publishCh    chan struct{}
}

func (m *MockProducer) PublishLogs(ctx context.Context, subject string, payload []byte, headers map[string][]string) error {
	return nil // Not used here
}

// Safe to call concurrently
func (m *MockProducer) PublishLiveTail(subject string, data []byte) error {
	m.publishCount.Add(1)

	// Non-blocking send to notify the test runner
	select {
	case m.publishCh <- struct{}{}:
	default:
	}
	return nil
}

func (m *MockProducer) GetPublishLiveTailCount() int {
	return int(m.publishCount.Load())
}

// MockDecompressor: Use a functional adapter since single method interface
type DecompressFunc func(payload []byte, headers map[string][]string) ([]byte, func(), error)

func (f DecompressFunc) decompress(payload []byte, headers map[string][]string) ([]byte, func(), error) {
	return f(payload, headers)
}

// MockDecoder: Use a functional adapter since single method interface
type DecoderFunc func(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error)

func (f DecoderFunc) decode(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error) {
	return f(payload, headers)
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
	enqueueCount   atomic.Int64
	enqueueSuccess bool
	records        []core.FlatLogRecord
	hasSubscribers bool
}

// Safe to call concurrently.
// stores messages in an internal []core.FlatLogRecord slice
// Success is controlled by the enqueueSuccess field
func (m *MockPublisher) Enqueue(subject string, msg MsgMarshaler) bool {
	m.enqueueCount.Add(1)
	if record, ok := msg.(*core.FlatLogRecord); ok {
		m.records = append(m.records, *record)
	}

	return m.enqueueSuccess
}

func (m *MockPublisher) GetEnqueueCount() int {
	return int(m.enqueueCount.Load())
}

func (m *MockPublisher) HasSubscribers() bool {
	return m.hasSubscribers
}

// MockTransformer
type MockTransformer struct {
	flattenCount atomic.Int64
}

// Safe to call concurrently
// Does not perform append operation, only tracks number of times this function is called
func (m *MockTransformer) Flatten(resourceLogs plog.ResourceLogs, publisher Publisher, appender storage.LogAppender) {
	m.flattenCount.Add(1)
}

func (m *MockTransformer) GetFlattenCount() int {
	return int(m.flattenCount.Load())
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

func (n *NoOpPublisher) HasSubscribers() bool {
	return true
}

type NoOpTransformer struct {
}

func (n *NoOpTransformer) Flatten(resourceLogs plog.ResourceLogs, publisher Publisher, appender storage.LogAppender) {
	// do nothing
}
