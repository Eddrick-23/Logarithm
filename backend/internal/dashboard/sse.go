package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

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

		defer ticker5s.Stop()
		defer ticker15s.Stop()
		defer ticker30s.Stop()

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
				// ingestion graph and logs / second refreshes every 5s
				if err := writeIngestionMetricsEvent(r.Context(), w, flusher, logger, logStore); err != nil {
					return
				}

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
) error {
	ingestionMetrics, err := logStore.GetAllIngestionMetrics(ctx)
	if err != nil {
		logger.Error("failed to get ingestion metrics", "err", err)
		return err
	}

	data, err := json.Marshal(ingestionMetrics)
	if err != nil {
		logger.Error("failed to marshal ingestion metrics", "err", err)
		return err
	}

	// data: <payload>\n\n is the SSE wire protocol
	if _, err := fmt.Fprintf(w, "event: ingestion\ndata: %s\n\n", data); err != nil {
		// client likely disconnected
		return err
	}

	flusher.Flush()
	return nil
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
		logger.Error("failed to get ingestion metrics", "err", err)
		return err
	}

	data, err := json.Marshal(topServiceErrorsStats)
	if err != nil {
		logger.Error("failed to marshal ingestion metrics", "err", err)
		return err
	}

	// data: <payload>\n\n is the SSE wire protocol
	if _, err := fmt.Fprintf(w, "event: top-service-errors\ndata: %s\n\n", data); err != nil {
		// client likely disconnected
		return err
	}

	flusher.Flush()
	return nil
}
