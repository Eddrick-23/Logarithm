package worker

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
	"github.com/klauspost/compress/zstd"

	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

type decompressFunc func([]byte, map[string][]string) ([]byte, error)
type decoderFunc func([]byte, map[string][]string) (*plogotlp.ExportRequest, error)

// ConsumeCallback returns the message handler used by the log consumer.
//
// Incoming OTLP logs are flattened, grouped by service, published to the
// live-tail stream on a file and forget basis, and bulk inserted into storage.
// Storage insertion failures are returned; live-tail publish failures are
// logged and ignored.
func ConsumeCallback(logger *slog.Logger, store storage.LogStore, decompressor Decompressor, transformer Transformer, publisher Publisher) (func([]transport.Message) error, error) {
	decoder := makeDecoder()
	return func(messages []transport.Message) error {
		if len(messages) == 0 {
			return nil
		}

		appender := store.FastInsert()
		processMessages(logger, decompressor, decoder, messages, transformer, publisher, appender)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		return appender.Flush(ctx)
	}, nil
}

func processMessages(logger *slog.Logger, decompressor Decompressor, decoder decoderFunc, messages []transport.Message,
	transformer Transformer, publisher Publisher, appender storage.LogAppender) {

	for _, msg := range messages {
		decompressedPayload, cleanup, err := decompressor.decompress(msg.Payload, msg.Headers)
		if err != nil {
			logger.Error("dropped log payload due to decompress error", "err", err)
			continue
		}

		req, err := decoder(decompressedPayload, msg.Headers)
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

// TODO remove after integrating decompressor interface
func makeDecoder() func([]byte, map[string][]string) (*plogotlp.ExportRequest, error) {
	return func(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error) {
		req := plogotlp.NewExportRequest()

		contentType := ""
		if vals, ok := headers["Content-Type"]; ok && len(vals) > 0 {
			contentType = vals[0]
		}

		if strings.HasPrefix(contentType, "application/x-protobuf") {
			if err := req.UnmarshalProto(payload); err != nil {
				return nil, err
			}
			return &req, nil
		}

		if strings.HasPrefix(contentType, "application/json") {
			if err := req.UnmarshalJSON(payload); err != nil {
				return nil, err
			}
			return &req, nil
		}

		return nil, fmt.Errorf("unsupported or missing content type: %s", contentType)
	}
}

func makeDecompressor() (func([]byte, map[string][]string) ([]byte, error), error) {
	zstdDecoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to init zstd decoder: %w", err)
	}
	return func(payload []byte, headers map[string][]string) ([]byte, error) {
		encoding := ""
		if vals, ok := headers["Content-Encoding"]; ok && len(vals) > 0 {
			encoding = vals[0]
		}

		switch encoding {
		case "zstd":
			return zstdDecoder.DecodeAll(payload, nil)
		case "gzip":
			reader, err := gzip.NewReader(bytes.NewReader(payload))
			if err != nil {
				return nil, err
			}
			defer reader.Close()
			return io.ReadAll(reader)
		default:
			return payload, nil
		}
	}, nil
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
