package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
)

// ConsumeCallback returns the message handler used by the log consumer.
//
// Incoming OTLP logs are flattened, grouped by service, published to the
// live-tail stream on a file and forget basis, and bulk inserted into storage.
// Storage insertion failures are returned; live-tail publish failures are
// logged and ignored.
func ConsumeCallback(logger *slog.Logger, store storage.LogStore, decompressor Decompressor,
	decoder Decoder, transformer Transformer, publisher Publisher, estRows int) (func([]transport.Message) error, error) {
	return func(messages []transport.Message) error {
		if len(messages) == 0 {
			return nil
		}

		appender := store.FastInsert(estRows)
		processMessages(logger, decompressor, decoder, messages, transformer, publisher, appender)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		return appender.Flush(ctx)
	}, nil
}

func processMessages(logger *slog.Logger, decompressor Decompressor, decoder Decoder, messages []transport.Message,
	transformer Transformer, publisher Publisher, appender storage.LogAppender) {

	for _, msg := range messages {
		decompressedPayload, cleanup, err := decompressor.decompress(msg.Payload, msg.Headers)
		if err != nil {
			logger.Error("dropped log payload due to decompress error", "err", err)
			continue
		}

		req, err := decoder.decode(decompressedPayload, msg.Headers)
		if err != nil {
			logger.Error("dropped malformed log payload", "err", err)
			cleanup()
			continue
		}
		cleanup()

		logs := req.Logs()
		for i := 0; i < logs.ResourceLogs().Len(); i++ {
			transformer.Flatten(logs.ResourceLogs().At(i), publisher, appender)
		}
	}
}

func DLQCallback(producer transport.Producer, subject string) func([]byte, map[string][]string) error {

	return func(payload []byte, headers map[string][]string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return producer.PublishLogs(ctx, subject, payload, headers)
	}
}

func DelayCalculator(backoff []time.Duration) func(uint64) time.Duration {
	return func(deliveredCount uint64) time.Duration {
		if len(backoff) == 0 {
			return 0
		}
		if int(deliveredCount) >= len(backoff) {
			return backoff[len(backoff)-1]
		}
		idx := max(int(deliveredCount)-1, 0)
		return backoff[idx]
	}
}
