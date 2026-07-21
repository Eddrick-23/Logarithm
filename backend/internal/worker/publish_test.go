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

	publisher := NewLiveTailPublisher(
		producer,
		func() bool { return true },
		WithLogger(slog.Default()),
		WithWorkerCount(2),
		WithQueueSize(10),
	)
	defer publisher.Close()

	msg := &MockMsg{
		data: []byte("log data"),
	}

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			success := publisher.Enqueue("test-subject", msg)
			assert.True(t, success, "enqueue should succeed")
		})
	}

	wg.Wait()

	for range 10 {
		<-producer.publishCh
	}

	assert.Equal(t, 10, producer.GetPublishLiveTailCount(), "All 10 messages should be published successfully")
}

func TestLiveTailPublisherBackPressure(t *testing.T) {
	producer := &MockProducer{}

	publisher := NewLiveTailPublisher(
		producer,
		func() bool { return true },
		WithLogger(slog.Default()),
		WithWorkerCount(0),
		WithQueueSize(1),
	)
	defer publisher.Close()

	msg := &MockMsg{data: []byte("log data")}

	success1 := publisher.Enqueue("test-subject", msg)
	assert.True(t, success1, "First enqueue should succeed")

	success2 := publisher.Enqueue("test-subject", msg)
	assert.False(t, success2, "Second enqueue should fail")
}
