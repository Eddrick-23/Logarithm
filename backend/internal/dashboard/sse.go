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
	"golang.org/x/sync/errgroup"
)

func handleDashboardStream(logger *slog.Logger, logStore *storage.ClickHouseStore, appCtx context.Context) http.HandlerFunc {
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

		for {
			select {
			case <-r.Context().Done():
				// Client closed the browser tab
				logger.Debug("client disconnected from dashboard stream")
				return

			case <-appCtx.Done():
				// Server is shutting down (ctrl-c)
				logger.Debug("server is shutting down, closing SSE stream gracefully")
				return

			case <-ticker5s.C:
				// ingestion graph and log rate metrics refreshes every 5s
				if err := writeIngestionGraphAndLogRatesEvent(r.Context(), w, flusher, logger, logStore); err != nil {
					return
				}

			case <-ticker15s.C:
				// error rate and nats queue depth metrics refreshes every 15s
				if err := writeErrorRateAndNatsQueueDepthMetricsEvent(r.Context(), w, flusher, logger, logStore); err != nil {
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

// writeSSEEvent marshals data as JSON and writes it as a single SSE event
// with the given event name, then flushes it to the client immediately.
func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, event string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal %q: %w", event, err)
	}

	// SSE format: "event: <name>\ndata: <payload>\n\n"
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		// client likely disconnected
		return fmt.Errorf("failed to format %q: %w", event, err)
	}

	flusher.Flush()
	return nil
}

func writeIngestionGraphAndLogRatesEvent(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	logger *slog.Logger,
	logStore *storage.ClickHouseStore,
) error {
	var ingestionGraphMetrics core.IngestionGraphMetrics
	var logRateStats core.LogRateStatistics

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		var err error
		// retrieve metrics from the past minute
		ingestionGraphMetrics, err = logStore.GetIngestionGraphMetrics(egCtx)
		return err
	})
	eg.Go(func() error {
		var err error
		logRateStats, err = logStore.GetLogRateStatistics(egCtx)
		return err
	})

	if err := eg.Wait(); err != nil {
		logger.Error("failed to fetch ingestion graph / log rate stats", "error", err)
		return err
	}

	if err := writeSSEEvent(w, flusher, "ingestion-graph-metrics", ingestionGraphMetrics); err != nil {
		return err
	}

	if err := writeSSEEvent(w, flusher, "log-rate-stats", logRateStats); err != nil {
		return err
	}

	return nil
}

func writeErrorRateAndNatsQueueDepthMetricsEvent(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	logger *slog.Logger,
	logStore *storage.ClickHouseStore,
) error {
	var errorRateMetrics core.ErrorRateMetrics
	var natsQueueDepthMetrics core.NatsQueueDepthGraphMetrics

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		var err error
		// retrieve error rate metrics
		errorRateMetrics, err = logStore.GetErrorRateMetrics(egCtx)
		return err
	})
	eg.Go(func() error {
		var err error
		// retrieve nats queue depth graph metrics from the past 15 minutes
		natsQueueDepthMetrics, err = logStore.GetNatsQueueDepthMetrics(egCtx, 15)
		return err
	})

	if err := eg.Wait(); err != nil {
		logger.Error("failed to fetch error rate metrics / nats queue depth metrics", "error", err)
		return err
	}

	if err := writeSSEEvent(w, flusher, "error-rate", errorRateMetrics); err != nil {
		return err
	}

	if err := writeSSEEvent(w, flusher, "nats-queue-depth", natsQueueDepthMetrics); err != nil {
		return err
	}

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

	return writeSSEEvent(w, flusher, "top-service-errors", topServiceErrorsStats)
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

	return writeSSEEvent(w, flusher, "storage-info", card)
}
