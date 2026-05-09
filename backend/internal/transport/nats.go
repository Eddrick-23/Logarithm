package transport

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)


type Producer interface {

}

type Consumer interface {

}

type NatsBroker struct {
	conn *nats.Conn
	js jetstream.JetStream
}

func NewNatsBroker(ctx context.Context, natsUrl string) (*NatsBroker, error){
	slog.Info("Connecting to NATS")
	nc, err := nats.Connect(natsUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %v", err)
	}
	defer nc.Drain()

	js, err := jetstream.New(nc)	
	if err != nil {
		return nil, fmt.Errorf("failed to create to jetstream interface: %v", err)
	}
	
	slog.Info("Connection to NATS established")
	return &NatsBroker{conn: nc, js: js}, nil
}

func (nb *NatsBroker) Close() {
	if nb.conn != nil {
		slog.Info("Draining NATS connection...")
		nb.conn.Drain()
	}
}

func (nb *NatsBroker) CreateStream(ctx context.Context, streamName string, subject string) (jetstream.Stream, error) {
	slog.Info("creating stream...")
	stream, err := nb.js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{"jobs.>"}, // Listens for any subject starting with "jobs."
		Storage:  jetstream.FileStorage, // Persist to disk
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %v", err)
	}

	slog.Info("stream created", "name", streamName, "subject", subject)
	return stream, nil
}

// func (nb *NatsBroker) publishLogs(ctx context.Context, subject string, payload []byte) {
// 	ack, err := nb.js.Publish(ctx, subject, payload)
// 	if err != nil {

// 	}

// 	ack.
// }
