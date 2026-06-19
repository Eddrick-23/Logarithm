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
	PublishedRecords map[string][][]byte
	PublishErr       error
	PublishCount     int
	mu               sync.Mutex
	PublishCh        chan struct{}
}

func (m *MockProducer) PublishLogs(ctx context.Context, subject string, payload []byte, headers map[string][]string) error {
	return nil // not used
}

func (m *MockProducer) PublishLiveTail(subject string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.PublishedRecords == nil {
		m.PublishedRecords = map[string][][]byte{}
	}

	m.PublishedRecords[subject] = append(m.PublishedRecords[subject], data)

	select {
	case m.PublishCh <- struct{}{}: // signal test thread a publish occured
	default:
	}

	return m.PublishErr
}
