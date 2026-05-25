package transport

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)


type Producer interface { // for ingestion endpoint to push payload
	publishLogs(context.Context, string, []byte) error
}

type Consumer interface { // for worker to read logs from stream
	consumeLogs()
}

type NatsBroker struct {
	conn *nats.Conn
	logger *slog.Logger
	js jetstream.JetStream
}

type NatsJSConsumer struct {
	consumer jetstream.Consumer
	logger *slog.Logger
}

func NewNatsBroker(ctx context.Context, logger *slog.Logger, natsUrl string) (*NatsBroker, error){
	logger.Info("Connecting to NATS")
	nc, err := nats.Connect(natsUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

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
		nb.conn.Drain()
		nb.logger.Info("Connection closed")
	}
}

func (nb *NatsBroker) CreateStream(ctx context.Context, streamName string, subject string) (jetstream.Stream, error) {
	nb.logger.Info("creating stream...")
	stream, err := nb.js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject}, // Listens for any subject starting with "jobs."
		Storage:  jetstream.FileStorage, // Persist to disk
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	nb.logger.Info("stream created", "name", streamName, "subject", subject)
	return stream, nil
}

func (nb *NatsBroker) publishLogs(ctx context.Context, subject string, payload []byte) error {
	// ack, err := nb.js.Publish(ctx, subject, payload)
	// if err != nil {

	// }

	// // ack.
	
	return nil
}

func (nb *NatsBroker) newConsumer(ctx context.Context, streamName string, consumer string) (*NatsJSConsumer, error) {
	cons, err := nb.js.Consumer(ctx, streamName, "test")
	if err != nil {
		return nil, fmt.Errorf("failed to create jetstream consumer: %w", err)
	}
	return &NatsJSConsumer{cons, nb.logger}, nil
}

func (nc *NatsJSConsumer) consumeLogs(ctx context.Context, streamName string) ([]core.LogIngestRequest, error) {
	// need to check use Consume -> pass in callback
	// or use Messages
	// Fetch is worse for throughput
	// nc.consumer.Consume()
	nc.consumer.Messages()
	return nil, nil
}
