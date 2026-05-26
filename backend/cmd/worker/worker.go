package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
)

func ConsumeCallback(store storage.LogStore) func([][]byte) error {
	return func(payloads [][]byte) error {
		if len(payloads) == 0 {
			return nil
		}

		// estimate around 500 logs per ingested Request to prealloc memory
		var records []core.FlatLogRecord = make([]core.FlatLogRecord, 0, len(payloads)*500)
		for _, payload := range payloads {
			var jsonPayload core.LogIngestRequest
			if err := json.Unmarshal(payload, &jsonPayload); err != nil {
				return fmt.Errorf("error unmarshalling to json: %w", err)
			}
			records = flattenLogs(jsonPayload, records)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := store.BatchInsert(ctx, records); err != nil {
			return err
		}

		return nil
	}
}
