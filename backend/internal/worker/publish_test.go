package worker

import (
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLiveTailPublisherConcurrentEnqueue(t *testing.T) {
	producer := &MockProducer{
		publishCh: make(chan struct{}, 10),
	}

	publisher := NewLiveTailPublisher(slog.Default(), producer, 2, 10)
	defer publisher.Close()

	msg := &MockMsg{
		data: []byte("log data"),
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := publisher.Enqueue("test-subject", msg)
			assert.True(t, success, "enqueue should succeed")
		}()
	}

	wg.Wait()

	for i := 0; i < 10; i++ {
		<-producer.publishCh
	}

	producer.mu.Lock()
	defer producer.mu.Unlock()
	assert.Equal(t, 10, producer.publishCount, "All 10 messages should be published successfully")
}

func TestLiveTailPublisherBackPressure(t *testing.T) {
	producer := &MockProducer{}

	pub := NewLiveTailPublisher(slog.Default(), producer, 0, 1) // 0 worker so never drains
	defer pub.Close()

	msg := &MockMsg{data: []byte("log data")}

	success1 := pub.Enqueue("test-subject", msg)
	assert.True(t, success1, "First enqueue should succeed")

	success2 := pub.Enqueue("test-subject", msg)
	assert.False(t, success2, "Second enqueue should fail")
}
