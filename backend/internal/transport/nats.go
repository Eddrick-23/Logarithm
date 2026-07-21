package transport

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const ( // infra constants
	LogStreamName           = "LOGS"
	LogStreamSubject        = "logs.>"
	LogStreamSubjectPrefix  = "logs."
	DLQStreamName           = "LOGS_DLQ"
	DLQSubject              = "dlq.logs"
	LiveTailSubject         = "tail.>"
	LiveTailSubjectPrefix   = "tail."
	LiveTailPresenceSubject = "presence.livetail"
)

var _ Producer = (*NatsBroker)(nil)
var _ Consumer = (*NatsJSConsumer)(nil)

type Message struct {
	Payload []byte
	Headers map[string][]string
}
type ProcessLogFunc func(messages []Message) error                   // callback to consume from stream
type DLQFunc func(payload []byte, headers map[string][]string) error // callback to move data to dlq stream
type DelayCalcFunc func(maxDeliver uint64) time.Duration             // callback to determine delay for NakWithDelay

type Producer interface { // for ingestion endpoint to push payload
	PublishLogs(context.Context, string, []byte, map[string][]string) error
	PublishLiveTail(string, []byte) error
}

type Consumer interface { // for worker to read logs from stream
	ConsumeLogs(ctx context.Context, logHandler ProcessLogFunc, dlqHandler DLQFunc, delayHandler DelayCalcFunc,
		maxBatch int, maxWait time.Duration) error
}

type NatsBroker struct {
	conn   *nats.Conn
	logger *slog.Logger
	js     jetstream.JetStream
}

type NatsJSConsumer struct {
	consumer   jetstream.Consumer
	logger     *slog.Logger
	maxDeliver uint64
}

type Option func(*NatsBroker)

func WithLogger(logger *slog.Logger) Option {
	return func(nb *NatsBroker) {
		nb.logger = logger
	}
}

func NewNatsBroker(ctx context.Context, natsUrl string, opts ...Option) (*NatsBroker, error) {
	broker := &NatsBroker{
		conn:   nil,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		js:     nil,
	}

	for _, opt := range opts {
		opt(broker)
	}

	broker.logger.Info("Connecting to NATS")
	// connect to server
	nc, err := nats.Connect(natsUrl, nats.DrainTimeout(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	broker.conn = nc

	// create JetStream management interface
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("failed to create to jetstream interface: %w", err)
	}
	broker.js = js

	broker.logger.Info("Connection to NATS established")
	return broker, nil
}

func (nb *NatsBroker) Close() {
	if nb.conn != nil {
		nb.logger.Info("Draining NATS connection")

		done := make(chan struct{})
		nb.conn.SetClosedHandler(func(_ *nats.Conn) {
			close(done)
		})
		nb.conn.Drain()

		<-done // block until connection fully closed
		nb.logger.Info("Connection closed")
	}
}

func (nb *NatsBroker) EnsureLogStream(ctx context.Context, streamName string, subject string, maxAge time.Duration, maxBytes int64) (jetstream.Stream, error) {
	nb.logger.Info("Ensuring stream exists", "subject", subject)
	streamConfig := jetstream.StreamConfig{
		Name:        streamName,
		Description: "unified stream containing raw log data",
		Subjects:    []string{subject},
		Storage:     jetstream.FileStorage,
		Discard:     jetstream.DiscardOld,
		MaxAge:      maxAge,
		MaxBytes:    maxBytes,
	}

	return nb.ensureStream(ctx, &streamConfig)
}

func (nb *NatsBroker) EnsureDLQStream(ctx context.Context, streamName string, subject string, maxAge time.Duration, maxBytes int64) (jetstream.Stream, error) {
	nb.logger.Info("Ensuring stream exists", "subject", subject)
	streamConfig := jetstream.StreamConfig{
		Name:        streamName,
		Description: "dead letter queue for failed log deliveries",
		Subjects:    []string{subject},
		Storage:     jetstream.FileStorage,
		Discard:     jetstream.DiscardOld,
		MaxAge:      maxAge,
		MaxBytes:    maxBytes,
	}

	return nb.ensureStream(ctx, &streamConfig)
}

func (nb *NatsBroker) ensureStream(ctx context.Context, config *jetstream.StreamConfig) (jetstream.Stream, error) {
	stream, err := nb.js.CreateOrUpdateStream(ctx, *config)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	nb.logger.Info("stream created", "name", config.Name, "subject", config.Subjects)
	return stream, nil
}

// Publish a log payload under a specified subject. This is a synchronous call.
//
// Headers can be attatched for routing in the worker layer. The worker layer checks for
// Content-Type and Content-Encoding.
func (nb *NatsBroker) PublishLogs(ctx context.Context, subject string, payload []byte, headers map[string][]string) error {
	msg := &nats.Msg{
		Subject: subject,
		Data:    payload,
		Header:  nats.Header(headers),
	}

	ack, err := nb.js.PublishMsg(ctx, msg)
	if err != nil {
		return fmt.Errorf("published failed: %w, subject: %v", err, subject)
	}

	nb.logger.Debug("logs published",
		"Stream", ack.Stream,
		"Sequence", ack.Sequence,
		"duplicate", ack.Duplicate)

	return nil
}

func (nb *NatsBroker) PublishLiveTail(subject string, data []byte) error {
	return nb.conn.Publish(subject, data) // core NATS, no ack just fire and forget
}

func (nb *NatsBroker) NewDurableConsumer(ctx context.Context, stream jetstream.Stream, consumerName string,
	maxDeliver int, backoff []time.Duration, maxAckPending int) (*NatsJSConsumer, error) {
	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    maxDeliver,
		BackOff:       backoff, // does not affect Nak, it defines how long nats waits for an Ack() before it times out
		MaxAckPending: maxAckPending,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create jetstream consumer: %w", err)
	}
	return &NatsJSConsumer{cons, nb.logger, uint64(maxDeliver)}, nil
}

func (nc *NatsJSConsumer) ConsumeLogs(ctx context.Context, logHandler ProcessLogFunc, dlqHandler DLQFunc, delayHandler DelayCalcFunc,
	maxBatch int, maxWait time.Duration) error {

	msgCh := make(chan jetstream.Msg, maxBatch*2) // buffered channel between NATS and batching loop
	nc.logger.Info("starting streaming from jetstream to database")

	cons, err := nc.consumer.Consume(func(msg jetstream.Msg) {
		// buffer messages straight channel
		select {
		case msgCh <- msg:
		case <-ctx.Done():
			msg.Nak() // context cancelled so mark as Nak, don't process message
		}

	})
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}
	defer cons.Drain()

	timer := time.NewTimer(maxWait)
	defer timer.Stop()

	var batch []jetstream.Msg
	flush := func() {
		defer func() { // resets timer fully every flush call
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(maxWait)
		}()

		if len(batch) == 0 {
			return
		}

		payloads := make([]Message, len(batch))
		for i, msg := range batch {
			payloads[i] = Message{
				Payload: msg.Data(),
				Headers: msg.Headers(),
			}
		}

		if err := logHandler(payloads); err != nil {
			nc.logger.Error("batch insert failed, NAKing messages", "err", err, "batchsize", len(batch))

			for _, msg := range batch {
				var delay time.Duration
				metadata, err := msg.Metadata()

				if err != nil {
					nc.logger.Error("cannot extract message metadata defaulting delay duration to 30s")
					delay = 30 * time.Second
					msg.NakWithDelay(delay)
					continue
				}
				if metadata.NumDelivered >= nc.maxDeliver {
					if err := dlqHandler(msg.Data(), msg.Headers()); err != nil {
						nc.logger.Error("Failed to publish to DLQ, message will be lost",
							"err", err,
							"payload", string(msg.Data()),
							"num_delivered", metadata.NumDelivered,
						)
					}
					msg.Ack()
				} else {
					msg.NakWithDelay(delayHandler(metadata.NumDelivered))
				}

			}
		} else {
			for _, msg := range batch {
				msg.Ack()
			}
		}

		batch = batch[:0] // clear batch buffer
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return nil
		case <-timer.C:
			nc.logger.Debug("timer flush")
			flush()
		case msg := <-msgCh:
			batch = append(batch, msg)
			if len(batch) >= maxBatch {
				nc.logger.Debug("size flush", "batchSize", len(batch))
				flush()
			}
		}

	}
}

// TailLiveLogs subscribes to a core NATS subject and pipes incoming log data into a channel.
// It utilizes an ephemeral, pure pub-sub connection with no underlying persistence.
//
// To protect the NATS connection from slow consumers (like a lagging WebSocket),
// the returned channel acts as a ring buffer of capacity maxBatch. If the caller
// falls behind, the oldest unread logs are dropped to make room for live data.
//
// The caller receives a channel for the data and a cleanup closure. The caller MUST
// either execute the cleanup closure when finished, or cancel the provided context
// to unsubscribe from NATS and safely close the channel.
func (nb *NatsBroker) TailLiveLogs(ctx context.Context, subject string, maxBatch int) (<-chan []byte, func(), error) {
	// Buffer the channel to handle slight backpressure from the websocket
	nb.logger.Info("setting up live tail subscription")
	logCh := make(chan []byte, maxBatch)

	sub, err := nb.conn.Subscribe(subject, func(msg *nats.Msg) {
		// handle closed connection
		select {
		case <-ctx.Done():
			return
		default:
		}

		// use channel as ring buffer
		select {
		case logCh <- msg.Data: // successful insert
			return
		default: // drop oldest and insert
			select {
			case <-logCh:
			default:
			}

			logCh <- msg.Data
		}

	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to start live tail subscription")
	}

	// Provide a cleanup function to stop the NATS consumer and close the channel
	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			if err := sub.Unsubscribe(); err != nil {
				nb.logger.Error("failed to unsubscribe core nats sub", "err", err)
			}
			close(logCh)
			nb.logger.Info("stopped live tail consumer", "subject", subject)
		})
	}

	// in case context dies before caller cleans up
	go func() {
		<-ctx.Done()
		if sub.IsValid() {
			cleanup()
		}
	}()

	return logCh, cleanup, nil
}

func (nb *NatsBroker) GetJetstreamConsumer(ctx context.Context, streamName string, consumerName string) (jetstream.Consumer, error) {
	const (
		retryAttempts = 5
		baseDelay     = time.Second
		maxDelay      = 15 * time.Second
	)

	var consumer jetstream.Consumer
	var err error

	for i := range retryAttempts {
		consumer, err = nb.js.Consumer(ctx, streamName, consumerName)
		if err == nil {
			return consumer, nil
		}

		nb.logger.Warn(
			fmt.Sprintf("failed to get jetstream consumer, attempt %d/%d", i+1, retryAttempts),
			"jetstream consumer", err,
		)
		delay := calculateExponentialBackoff(i, baseDelay, maxDelay)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled while retrying consumer fetch: %w", ctx.Err())
		case <-time.After(delay): // wait for delay duration to elapse
		}
	}

	return nil, fmt.Errorf("failed to get jetstream consumer %q on stream %q after %d attempts: %w",
		consumerName, streamName, retryAttempts, err)
}

func calculateExponentialBackoff(attempt int, baseDelay time.Duration, maxDelay time.Duration) time.Duration {
	delay := baseDelay * time.Duration(1<<attempt) // 1x, 2x, 4x, 8x ...
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

// Starts an internal goroutine and publishes a simple ping message to the
// specified subject using core NATS.
//
// This is used with StartPresenceListener for inter-container communication.
func (nb *NatsBroker) StartPresencePublisher(ctx context.Context, subject string, interval time.Duration) {
	nb.logger.Info("starting presence publisher", "subject", subject, "interval", interval)

	if interval <= 0 {
		nb.logger.Warn("invalid interval passed in, defaulting to 1s")
		interval = 1 * time.Second
	}

	go func() {

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		pingPayload := []byte("ping")

		for {
			select {
			case <-ctx.Done():
				nb.logger.Info("stopping presence publisher", "subject", subject)
				return
			case <-ticker.C:
				if err := nb.conn.Publish(subject, pingPayload); err != nil {
					nb.logger.Error("failed to publish presence heartbeat", "err", err)
				}
			}
		}
	}()
}

// StartPresenceListener listens for heartbeats and returns a lock-free function
// that checks if the last heartbeat was received within the timeout period.
//
// This is used with StartPresencePublisher for inter-container communication.
func (nb *NatsBroker) StartPresenceListener(ctx context.Context, subject string, timeout time.Duration) (func() bool, error) {
	nb.logger.Info("starting presence listener", "subject", subject, "timeout", timeout)

	var lastSeenNano atomic.Int64
	lastSeenNano.Store(0)

	sub, err := nb.conn.Subscribe(subject, func(msg *nats.Msg) {
		lastSeenNano.Store(time.Now().UnixNano())
	})

	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		if err := sub.Unsubscribe(); err != nil {
			nb.logger.Error("failed to unsubscribe presence listener", "err", err)
		}
		nb.logger.Info("stopped presence listener", "subject", subject)
	}()

	isActive := func() bool {
		lastSeen := lastSeenNano.Load()
		if lastSeen == 0 {
			return false
		}
		// return if last ping is newer than now - timeout
		return time.Since(time.Unix(0, lastSeen)) < timeout
	}

	return isActive, nil
}
