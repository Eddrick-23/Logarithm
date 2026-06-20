package worker

import (
	"log/slog"
	"sync"

	"github.com/Eddrick-23/Logarithm/internal/transport"
)

type tailJob struct {
	Subject string
	Payload *[]byte
}

type MsgMarshaler interface {
	MarshalMsg([]byte) ([]byte, error)
}

type Publisher interface {
	Enqueue(string, MsgMarshaler) bool
}

var _ Publisher = (*liveTailPublisher)(nil)

type liveTailPublisher struct {
	logger   *slog.Logger
	producer transport.Producer
	jobChan  chan tailJob
	bufPool  sync.Pool
}

// Creates a Live Tail publisher with an internal worker pool
//
// Clients will enqueue subjects and messages that support the MsgMarshaller interface.
// The publisher will handle efficient publishing to nats using workers and reusable buffers
func NewLiveTailPublisher(logger *slog.Logger, producer transport.Producer, workers int, queueSize int) *liveTailPublisher {
	p := &liveTailPublisher{
		logger:   logger,
		producer: producer,
		jobChan:  make(chan tailJob, queueSize),
		bufPool: sync.Pool{
			New: func() any {
				// prealloc reasonable size
				buf := make([]byte, 0, 1024)
				return &buf
			},
		},
	}

	for range workers {
		go p.worker()
	}
	p.logger.Info("Created live tail worker pool")
	return p
}

func (p *liveTailPublisher) Close() {
	p.logger.Info("Stopping live tail workers")
	close(p.jobChan)
}

// Enqueue aborts and returns false if queue is full
//
// Caller may retry or simply drop the message
func (p *liveTailPublisher) Enqueue(subject string, m MsgMarshaler) bool {
	bufPtr := p.bufPool.Get().(*[]byte)
	buf := (*bufPtr)[:0]

	data, err := m.MarshalMsg(buf)
	if err != nil {
		p.bufPool.Put(bufPtr)
		return false
	}

	*bufPtr = data // update underlying data incase buffer was reallocated

	job := tailJob{
		Subject: subject,
		Payload: bufPtr,
	}

	select {
	case p.jobChan <- job:
		return true
	default:
		// queue full, drop message
		// clear buffer and return to pool
		p.bufPool.Put(bufPtr)
		return false
	}
}

// worker that pulls from job channel and publishes to live tail
//
// publish errors simply logged in a fire and forget manner
// after publishing, buffer cleared and returned to pool for reuse
func (p *liveTailPublisher) worker() {
	for job := range p.jobChan {
		if err := p.producer.PublishLiveTail(job.Subject, *job.Payload); err != nil {
			p.logger.Error("failed to publish live tail", "err", err)
		}
		p.bufPool.Put(job.Payload)
	}
}
