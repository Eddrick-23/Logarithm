package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"

	collectorlogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func ConsumeCallback(logger *slog.Logger, store storage.LogStore, producer transport.Producer) func([][]byte) error {
	return func(payloads [][]byte) error {
		if len(payloads) == 0 {
			return nil
		}

		numRecords := 0
		flatLogsByServiceName := map[string][]core.FlatLogRecord{}
		for _, payload := range payloads {
			var req collectorlogspb.ExportLogsServiceRequest
			if err := protojson.Unmarshal(payload, &req); err != nil {
				slog.Error("dropped malformed log payload", "err", err, "payload_preview", string(payload))
				continue
			}

			for _, resource := range req.ResourceLogs {
				numRecords += flattenLogs(resource, flatLogsByServiceName) //appends to the required slice in the map
			}
		}

		if len(flatLogsByServiceName) == 0 { // no valid payloads to insert
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		batch := make([]core.FlatLogRecord, 0, numRecords) // prealloc
		for name, slice := range flatLogsByServiceName {
			go func(svcName string, logs []core.FlatLogRecord) {
				pubCtx, pubCancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer pubCancel()
				data, err := json.Marshal(slice)
				if err != nil {
					logger.Error("json marshal failed before publish to live tail stream", "err", err)
					return
				}
				if err := producer.PublishLogs(pubCtx, transport.LiveTailSubjectTemplate+name, data); err != nil {
					logger.Error("failed to publish flattened logs to live tail stream", "err", err)
					return
				}
			}(name, slice)
			batch = append(batch, slice...)
		}

		if err := store.BatchInsert(ctx, batch); err != nil {
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
