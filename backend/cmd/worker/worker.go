package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
)

func ConsumeCallback(logger *slog.Logger, store storage.LogStore) func([][]byte) error {
	return func(payloads [][]byte) error {
		if len(payloads) == 0 {
			return nil
		}

		// estimate around 500 logs per ingested Request to prealloc memory
		var records []core.FlatLogRecord = make([]core.FlatLogRecord, 0, len(payloads)*500)
		for _, payload := range payloads {
			var jsonPayload core.LogIngestRequest
			if err := json.Unmarshal(payload, &jsonPayload); err != nil {
				slog.Error("dropped malformed log payload", "err", err, "payload_preview", string(payload))
				continue
			}
			records = flattenLogs(jsonPayload, records)
		}

		if len(records) == 0 { // no valid payloads to insert
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := store.BatchInsert(ctx, records); err != nil {
			return err
		}

		return nil
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
