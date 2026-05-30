package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
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
