package worker

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
	"github.com/klauspost/compress/zstd"

	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

type decompressFunc func([]byte, map[string][]string) ([]byte, error)

func ConsumeCallback(logger *slog.Logger, store storage.LogStore, producer transport.Producer) (func([]transport.Message) error, error) {
	// ConsumeCallback returns the message handler used by the log consumer.
	//
	// Incoming OTLP logs are flattened, grouped by service, published to the
	// live-tail stream on a file and forget basis, and bulk inserted into storage.
	// Storage insertion failures are returned; live-tail publish failures are
	// logged and ignored.
	decompress, err := makeDecompressor()
	if err != nil {
		return nil, err
	}
	return func(messages []transport.Message) error {
		if len(messages) == 0 {
			return nil
		}

		flatLogsByServiceName := map[string][]core.FlatLogRecord{}
		numRecords := processMessages(logger, decompress, messages, flatLogsByServiceName)

		if len(flatLogsByServiceName) == 0 { // no valid payloads to insert
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		batch := make([]core.FlatLogRecord, 0, numRecords) // prealloc
		for name, slice := range flatLogsByServiceName {
			go publishLiveTail(logger, producer, name, slice)
			batch = append(batch, slice...)
		}

		return store.BatchInsert(ctx, batch)
	}, nil
}

func processMessages(logger *slog.Logger, decompress decompressFunc, messages []transport.Message,
	flatLogsByServiceName map[string][]core.FlatLogRecord) int {
	numRecords := 0

	for _, msg := range messages {
		decompressedPayload, err := decompress(msg.Payload, msg.Headers)
		if err != nil {
			logger.Error("dropped log payload due to decompress error", "err", err)
			continue
		}

		req := plogotlp.NewExportRequest()
		if err := req.UnmarshalJSON(decompressedPayload); err != nil {
			logger.Error("dropped malformed log payload", "err", err)
			continue
		}

		logs := req.Logs()
		for i := 0; i < logs.ResourceLogs().Len(); i++ {
			numRecords += flattenLogs(logs.ResourceLogs().At(i), flatLogsByServiceName)
		}
	}
	return numRecords
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

func publishLiveTail(logger *slog.Logger, producer transport.Producer, serviceName string, logs []core.FlatLogRecord) {
	subject := transport.LiveTailSubjectPrefix + serviceName

	for _, record := range logs {
		data, err := json.Marshal(record)
		if err != nil {
			logger.Error("json marshal failed for live tail record", "err", err)
		}

		if err := producer.PublishLiveTail(subject, data); err != nil {
			logger.Error("failed to publish to live tail stream", "err", err)
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
