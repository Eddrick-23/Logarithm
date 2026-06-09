package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"

	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

func ConsumeCallback(logger *slog.Logger, store storage.LogStore, producer transport.Producer) func([][]byte) error {
	// ConsumeCallback returns the message handler used by the log consumer.
	//
	// Incoming OTLP logs are flattened, grouped by service, published to the
	// live-tail stream on a file and forget basis, and bulk inserted into storage.
	// Storage insertion failures are returned; live-tail publish failures are
	// logged and ignored.
	return func(payloads [][]byte) error {
		if len(payloads) == 0 {
			return nil
		}

		flatLogsByServiceName := map[string][]core.FlatLogRecord{}
		numRecords := processPayloads(logger, payloads, flatLogsByServiceName)

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
	}
}

func processPayloads(logger *slog.Logger, payloads [][]byte, flatLogsByServiceName map[string][]core.FlatLogRecord) int {
	numRecords := 0
	for _, payload := range payloads {
		req := plogotlp.NewExportRequest()
		if err := req.UnmarshalJSON(payload); err != nil {
			logger.Error("dropped malformed log payload", "err", err, "payload_preview", string(payload))
			continue
		}

		logs := req.Logs()
		for i := 0; i < logs.ResourceLogs().Len(); i++ {
			numRecords += flattenLogs(logs.ResourceLogs().At(i), flatLogsByServiceName)
		}
	}
	return numRecords
}

func publishLiveTail(logger *slog.Logger, producer transport.Producer, serviceName string, logs []core.FlatLogRecord) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	data, err := json.Marshal(logs)
	if err != nil {
		logger.Error("json marshal failed before publish to live tail stream", "err", err)
		return
	}
	if err := producer.PublishLogs(ctx, transport.LiveTailSubjectPrefix+serviceName, data); err != nil {
		logger.Error("failed to publish flattened logs to live tail stream", "err", err)
		return
	}
}

func DLQCallback(producer transport.Producer, subject string) func([]byte) error {

	return func(payload []byte) error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return producer.PublishLogs(ctx, subject, payload)
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
