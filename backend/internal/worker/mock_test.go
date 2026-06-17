package worker

import (
	"context"
	"sync"

	"github.com/Eddrick-23/Logarithm/internal/core"
)

type MockLogStore struct {
	InsertedRecords []core.FlatLogRecord
	InsertErr       error
	InsertCount     int
}

func (m *MockLogStore) BatchInsert(ctx context.Context, records []core.FlatLogRecord) error {
	m.InsertCount++
	m.InsertedRecords = records
	return m.InsertErr
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
