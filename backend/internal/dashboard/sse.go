package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
)

func handleIngestionMetricsStream(logger *slog.Logger, logStore *storage.ClickHouseStore, appCtx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		// write headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ticker5s := time.NewTicker(5 * time.Second)
		ticker15s := time.NewTicker(15 * time.Second)
		ticker30s := time.NewTicker(30 * time.Second)
		ticker15m := time.NewTicker(15 * time.Minute)

		defer ticker5s.Stop()
		defer ticker15s.Stop()
		defer ticker30s.Stop()
		defer ticker15m.Stop()

		lastSent := time.Now().UTC().Truncate(time.Second)

		for {
			select {
			case <-r.Context().Done():
				// Client closed the browser tab
				logger.Debug("client disconnected from ingestion metrics stream")
				return

			case <-appCtx.Done():
				// Server is shutting down (ctrl-c)
				logger.Debug("server is shutting down, closing SSE stream gracefully")
				return

			case <-ticker5s.C:
				// ingestion graph refreshes every 5s
				newLastSent, err := writeIngestionMetricsEvent(r.Context(), w, flusher, logger, logStore, lastSent)
				if err != nil {
					return
				}
				lastSent = newLastSent

			// TODO: logs / second

			case <-ticker15s.C:
				// error rate metrics refreshes every 15s
				if err := writeErrorRateMetricsEvent(r.Context(), w, flusher, logger, logStore); err != nil {
					return
				}

			case <-ticker30s.C:
				// top service errors refreshes every 30s
				if err := writeTopServiceErrorsStatsEvent(r.Context(), w, flusher, logger, logStore); err != nil {
					return
				}

			case <-ticker15m.C:
				// storage info refreshes every 15 minutes
				if err := writeStorageInfoEvent(r.Context(), w, flusher, logger, logStore); err != nil {
					return
				}
			}
		}
	}
}

func writeIngestionMetricsEvent(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	logger *slog.Logger,
	logStore *storage.ClickHouseStore,
	lastSent time.Time,
) (time.Time, error) {
	// retrieve metrics from last seen timing
	ingestionMetrics, err := logStore.GetIngestionMetricsSince(ctx, lastSent)
	if err != nil {
		logger.Error("failed to get ingestion metrics", "err", err)
		return lastSent, err
	}

	if len(ingestionMetrics.Timestamps) == 0 {
		return lastSent, nil
	}

	data, err := json.Marshal(ingestionMetrics)
	if err != nil {
		logger.Error("failed to marshal ingestion metrics", "err", err)
		return lastSent, err
	}

	// event: ingestion
	// data: <payload>\n\n is the SSE wire protocol
	if _, err := fmt.Fprintf(w, "event: ingestion\ndata: %s\n\n", data); err != nil {
		// client likely disconnected
		return lastSent, err
	}

	// update last sent to be 1 second after the last timestamp recorded
	newLastSent := time.UnixMilli(ingestionMetrics.Timestamps[len(ingestionMetrics.Timestamps)-1]).Add(time.Second)
	flusher.Flush()
	return newLastSent, nil
}

func writeErrorRateMetricsEvent(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	logger *slog.Logger,
	logStore *storage.ClickHouseStore,
) error {
	errorRateMetrics, err := logStore.GetErrorRateMetrics(ctx)
	if err != nil {
		logger.Error("failed to get error rate metrics", "err", err)
		return err
	}

	data, err := json.Marshal(errorRateMetrics)
	if err != nil {
		logger.Error("failed to marshal error rate metrics", "err", err)
		return err
	}

	// event: error-rate
	// data: <payload>\n\n is the SSE wire protocol
	if _, err := fmt.Fprintf(w, "event: error-rate\ndata: %s\n\n", data); err != nil {
		// client likely disconnected
		return err
	}

	flusher.Flush()
	return nil
}

func writeTopServiceErrorsStatsEvent(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	logger *slog.Logger,
	logStore *storage.ClickHouseStore,
) error {
	topServiceErrorsStats, err := logStore.GetTopServiceErrorsStats(ctx)
	if err != nil {
		logger.Error("failed to get top service error metrics", "err", err)
		return err
	}

	data, err := json.Marshal(topServiceErrorsStats)
	if err != nil {
		logger.Error("failed to marshal top service error metrics", "err", err)
		return err
	}

	// event: top-service-errors
	// data: <payload>\n\n is the SSE wire protocol
	if _, err := fmt.Fprintf(w, "event: top-service-errors\ndata: %s\n\n", data); err != nil {
		// client likely disconnected
		return err
	}

	flusher.Flush()
	return nil
}

func writeStorageInfoEvent(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	logger *slog.Logger,
	logStore *storage.ClickHouseStore,
) error {
	stats, err := logStore.GetStorageStats(ctx)
	if err != nil {
		logger.Error("failed to get storage stats", "error", err)
		return err
	}
	if len(stats) == 0 {
		logger.Error("no disk stats found")
		return fmt.Errorf("no disk stats found")
	}

	disk := stats[0]
	usedBytes := disk.TotalBytes - disk.FreeBytes
	usedPercent := float64(usedBytes) / float64(disk.TotalBytes) * 100

	outlook, err := logStore.GetLogsStorageOutlook(ctx)
	if err != nil {
		logger.Error("failed to get logs storage outlook", "error", err)
		return err
	}

	card := core.StorageCardData{
		Value: fmt.Sprintf("%.0f", usedPercent),
		Unit:  "%",
	}

	projectedTotal := outlook.ProjectedSteadyStateBytes
	if projectedTotal <= usedBytes {
		card.Delta = "Stable"
		card.DeltaColour = "text.secondary"
	} else {
		// Steady-state would exceed current capacity
		growthPerDay := float64(projectedTotal-usedBytes) / 30
		daysRemaining := float64(disk.FreeBytes) / growthPerDay

		switch {
		case daysRemaining > 30:
			card.Delta = fmt.Sprintf("~%.0f days at current rate", daysRemaining)
			card.DeltaColour = "warning.main"
		default:
			card.Delta = fmt.Sprintf("~%.0f days — action needed", daysRemaining)
			card.DeltaColour = "error.main"
		}
	}

	data, err := json.Marshal(card)
	if err != nil {
		logger.Error("failed to marshal storage info metrics", "err", err)
		return err
	}

	// event: storage-info
	// data: <payload>\n\n is the SSE wire protocol
	if _, err := fmt.Fprintf(w, "event: storage-info\ndata: %s\n\n", data); err != nil {
		// client likely disconnected
		return err
	}

	flusher.Flush()
	return nil
}
