package worker

import (
	"io"
	"log/slog"
	"sync"
	"time"

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
	HasSubscribers() bool
}

var _ Publisher = (*LiveTailPublisher)(nil)

type LiveTailPublisher struct {
	logger          *slog.Logger
	producer        transport.Producer
	jobChan         chan tailJob
	bufPool         sync.Pool
	subscriberCheck func() bool
	workers         int
}
type Option func(p *LiveTailPublisher)

func WithLogger(l *slog.Logger) Option {
	return func(p *LiveTailPublisher) {
		p.logger = l
	}
}

// control number of workers that pull from queue and publish
func WithWorkerCount(count int) Option {
	return func(p *LiveTailPublisher) {
		p.workers = count
	}
}

// control internal channel size which acts as a queue
func WithQueueSize(size int) Option {
	return func(p *LiveTailPublisher) {
		p.jobChan = make(chan tailJob, size)
	}
}

// Creates a Live Tail publisher with an internal worker pool
//
// Clients will enqueue subjects and messages that support the MsgMarshaller interface.
// The publisher will handle efficient publishing to nats using workers and reusable buffers
func NewLiveTailPublisher(producer transport.Producer, subscriberCheck func() bool, opts ...Option) *LiveTailPublisher {
	const defaultWorkers = 3
	const defaultQueueSize = 10000
	const defaultPresenceTimeout = 1 * time.Second
	defaultLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	p := &LiveTailPublisher{
		logger:   defaultLogger,
		producer: producer,
		jobChan:  make(chan tailJob, defaultQueueSize),
		bufPool: sync.Pool{
			New: func() any {
				// prealloc reasonable size
				buf := make([]byte, 0, 1024)
				return &buf
			},
		},
		subscriberCheck: subscriberCheck,
		workers:         defaultWorkers,
	}

	for _, opt := range opts {
		opt(p)
	}

	for range p.workers {
		go p.worker()
	}
	p.logger.Info("Created live tail worker pool")
	return p
}

func (p *LiveTailPublisher) Close() {
	p.logger.Info("Stopping live tail workers")
	close(p.jobChan)
}

// Enqueue aborts and returns false if queue is full
//
// Caller may retry or simply drop the message
func (p *LiveTailPublisher) Enqueue(subject string, m MsgMarshaler) bool {
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
func (p *LiveTailPublisher) worker() {
	for job := range p.jobChan {
		if err := p.producer.PublishLiveTail(job.Subject, *job.Payload); err != nil {
			p.logger.Error("failed to publish live tail", "err", err)
		}
		p.bufPool.Put(job.Payload)
	}
}

// Check for active subscribers to live tail
//
// If there no active subscribers, client can skip live
// tail generation work and not enqueue at all.
func (p *LiveTailPublisher) HasSubscribers() bool {
	return p.subscriberCheck()
}
