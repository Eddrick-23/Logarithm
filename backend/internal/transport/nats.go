package transport

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	LogStreamName = "LOGS"
	DLQStreamName = "LOGS_DLQ"
	DLQSubject    = "dlq.logs"
)

var _ Producer = (*NatsBroker)(nil)
var _ Consumer = (*NatsJSConsumer)(nil)

type ProcessLogFunc func(payload [][]byte) error         // callback to consume from stream
type DLQFunc func(payload []byte) error                  // callback to move data to dlq stream
type DelayCalcFunc func(maxDeliver uint64) time.Duration // callback to determine delay for NakWithDelay

type Producer interface { // for ingestion endpoint to push payload
	PublishLogs(context.Context, string, []byte) error
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

func (nb *NatsBroker) EnsureLogStream(ctx context.Context, streamName string, subject string, maxAge time.Duration) (jetstream.Stream, error) {
	return nb.ensureStream(ctx, streamName, "unified log stream for all client services", subject, maxAge)
}

func (nb *NatsBroker) EnsureDLQStream(ctx context.Context, streamName string, subject string, maxAge time.Duration) (jetstream.Stream, error) {
	return nb.ensureStream(ctx, streamName, "dead letter queue for failed log deliveries", subject, maxAge)
}

func (nb *NatsBroker) ensureStream(ctx context.Context, streamName string, description string,
	subject string, NatsStreamMaxAge time.Duration) (jetstream.Stream, error) {
	nb.logger.Info("Ensuring stream exists", "subject", subject)

	stream, err := nb.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        streamName,
		Description: description,
		Subjects:    []string{subject},
		Storage:     jetstream.FileStorage, // Persist to disk for durable queue
		MaxAge:      NatsStreamMaxAge,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	nb.logger.Info("stream created", "name", streamName, "subject", subject)
	return stream, nil
}

func (nb *NatsBroker) PublishLogs(ctx context.Context, subject string, payload []byte) error {
	ack, err := nb.js.Publish(ctx, subject, payload)
	if err != nil {
		return fmt.Errorf("published failed: %w, subject: %v", err, subject)
	}

	nb.logger.Debug("logs published",
		"Stream", ack.Stream,
		"Sequence", ack.Sequence,
		"duplicate", ack.Duplicate)

	return nil
}

func (nb *NatsBroker) NewDurableConsumer(ctx context.Context, stream jetstream.Stream, consumerName string, maxDeliver int, backoff []time.Duration) (*NatsJSConsumer, error) {
	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:       consumerName,
		Durable:    consumerName,
		AckPolicy:  jetstream.AckExplicitPolicy,
		MaxDeliver: maxDeliver,
		BackOff:    backoff, // does not affect Nak, it defines how long nats waits for an Ack() before it times out
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
					if err := dlqHandler(msg.Data()); err != nil {
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
