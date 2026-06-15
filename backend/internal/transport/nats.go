package transport

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const ( // infra constants
	LogStreamName          = "LOGS"
	LogStreamSubject       = "logs.>"
	LogStreamSubjectPrefix = "logs."
	DLQStreamName          = "LOGS_DLQ"
	DLQSubject             = "dlq.logs"
	LiveTailStreamName     = "TAIL"
	LiveTailSubject        = "tail.>"
	LiveTailSubjectPrefix  = "tail."
)

var _ Producer = (*NatsBroker)(nil)
var _ Consumer = (*NatsJSConsumer)(nil)

type ProcessLogFunc func(payload [][]byte) error                     // callback to consume from stream
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

func NewNatsBroker(ctx context.Context, logger *slog.Logger, natsUrl string) (*NatsBroker, error) {
	logger.Info("Connecting to NATS")
	// connect to server
	nc, err := nats.Connect(natsUrl, nats.DrainTimeout(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// create JetStream management interface
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("failed to create to jetstream interface: %w", err)
	}

	logger.Info("Connection to NATS established")
	return &NatsBroker{conn: nc, logger: logger, js: js}, nil
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

func (nb *NatsBroker) EnsureLiveTailStream(ctx context.Context, streamName string, subject string, maxBytes int64) (jetstream.Stream, error) {
	// acts as a small circular buffer, no max age, small storage limits, memory storage only
	nb.logger.Info("Ensuring stream exists", "subject", subject)
	streamConfig := jetstream.StreamConfig{
		Name:        streamName,
		Description: "live tail stream for flattened logs",
		Subjects:    []string{subject},
		Storage:     jetstream.MemoryStorage,
		Discard:     jetstream.DiscardOld,
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

		payloads := make([][]byte, len(batch))
		for i, msg := range batch {
			payloads[i] = msg.Data()
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

func (nb *NatsBroker) TailLiveLogs(ctx context.Context, streamName string, subject string, maxBatch int) (<-chan []byte, func(), error) {
	// Retrieve the stream
	stream, err := nb.js.Stream(ctx, streamName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get stream for tailing: %w", err)
	}

	cons, err := stream.OrderedConsumer(ctx, jetstream.OrderedConsumerConfig{
		FilterSubjects: []string{subject},
		DeliverPolicy:  jetstream.DeliverNewPolicy, // Start tailing from "now"
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ordered consumer: %w", err)
	}

	// Buffer the channel to handle slight backpressure from the websocket
	logCh := make(chan []byte, maxBatch)

	// Start consuming asynchronously
	cc, err := cons.Consume(func(msg jetstream.Msg) {
		select {
		case logCh <- msg.Data():
			// For a live tail, we don't necessarily need to Ack() since it's an ordered
			// consumer and we don't care about redelivery if the websocket drops.
		case <-ctx.Done():
			// Context cancelled, stop processing
		}
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start consuming for tail: %w", err)
	}

	// Provide a cleanup function to stop the NATS consumer and close the channel
	cleanup := func() {
		cc.Stop()
		close(logCh)
		nb.logger.Debug("stopped live tail consumer", "subject", subject)
	}

	return logCh, cleanup, nil
}
