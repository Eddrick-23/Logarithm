package worker

import (
	"context"
	"sync"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
)

type MockLogStore struct {
	Appender *MockLogAppender
}

type MockLogAppender struct {
	AppendCount int
	FlushErr    error
}

func (a *MockLogAppender) Append(
	timestamp, observedTimestamp time.Time,
	severityNumber uint8,
	traceId [16]byte,
	spanId [8]byte,
	logAttrKeys, logAttrValues, resAttrKeys, resAttrValues []string,
	logFields storage.LogFields,
) {
	a.AppendCount++
}

func (a *MockLogAppender) Flush(ctx context.Context) error {
	return a.FlushErr
}

func (m *MockLogStore) FastInsert() storage.LogAppender {
	return m.Appender
}

func (m *MockLogStore) BatchInsert(ctx context.Context, records []core.FlatLogRecord) error {
	return nil // not needed for this test
}

func (m *MockLogStore) SearchLogs(ctx context.Context, filter core.LogQueryFilter) ([]core.FlatLogRecord, error) {
	return nil, nil // not needed for this test
}

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

type MockMsg struct {
	data []byte
}

func (m *MockMsg) MarshalMsg(dst []byte) ([]byte, error) {
	return append(dst, m.data...), nil
}

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

type NoOpPublisher struct{}

func (n *NoOpPublisher) Enqueue(subject string, msg MsgMarshaler) bool {
	return true
}
