package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/config"
	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
)

func AddRoutes(
	mux *http.ServeMux,
	logger *slog.Logger,
	config *config.Config,
	logStore *storage.ClickHouseStore,
	broker *transport.NatsBroker,
	appCtx context.Context,
) {
	mux.HandleFunc("GET /", handleRoot(logger))
	mux.HandleFunc("GET /health", handleHealth(logger))
	mux.Handle("GET /api/search", handleLogs(logger, logStore))
	mux.Handle("GET /api/services", handleDistinctServices(logger, logStore))
	mux.Handle("GET /api/top-service-errors", handleTopServiceErrors(logger, logStore))
	mux.Handle("GET /api/ingestion-graph-metrics", handleIngestionGraphMetrics(logger, logStore))
	mux.Handle("GET /api/dashboard/stream", handleDashboardStream(logger, logStore, broker, appCtx))
	mux.Handle("GET /api/log-rate-stats", handleLogRateStats(logger, logStore))
	mux.Handle("GET /api/error-rate-metrics", handleErrorRateMetrics(logger, logStore))
	mux.Handle("GET /api/storage-info", handleStorageInfo(logger, logStore))
	mux.Handle("GET /api/nats-dlq-info", handleNatsDLQInfo(logger, broker))
	mux.Handle("GET /api/nats-queue-depth-metrics", handleNatsQueueDepthMetrics(logger, logStore))
	mux.Handle("GET /api/config", handleConfig(logger, config))
	mux.Handle("GET /ws/logs/tail", handleLiveTail(logger, broker, config))
}

// writeJSON attempts to marshal v to json
// if successful, it will attempt to write to the response
func writeJSON(w http.ResponseWriter, logger *slog.Logger, status int, v any) {
	// Attempt to marshal the data first
	data, err := json.Marshal(v)
	if err != nil {
		logger.Error("failed to marshal json response", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set headers only after we know encoding succeeded
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Write the actual data to the response
	if _, err := w.Write(data); err != nil {
		logger.Error("failed to write response body", "err", err)
	}
}

// writeError logs an error and sends an HTTP error response with the given
// status code and message.
func writeError(w http.ResponseWriter, logger *slog.Logger, err error, logMsg string, clientMsg string, status int) {
	logger.Error(logMsg, "err", err)
	http.Error(w, clientMsg, status)
}

func handleRoot(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintln(w, "Dashboard API root!")

		if err != nil {
			logger.Error("failed to write response", "err", err)
		}
	}
}

func handleHealth(logger *slog.Logger) http.HandlerFunc {
	type response struct {
		Health string `json:"health"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, logger, http.StatusOK, response{Health: "ok"})
	}
}

func handleLogs(logger *slog.Logger, logStore *storage.ClickHouseStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		layout := "2006-01-02T15:04:05" // reference layout for time
		var startTime, endTime time.Time
		var err error
		var severityNumber int = -1

		if s := query.Get("startTime"); s != "" {
			startTime, err = time.Parse(layout, s)
			if err != nil {
				writeError(w, logger, err, "invalid start time type", "Bad Request", http.StatusBadRequest)
				return
			}
		}

		if s := query.Get("endTime"); s != "" {
			endTime, err = time.Parse(layout, s)
			if err != nil {
				writeError(w, logger, err, "invalid end time type", "Bad Request", http.StatusBadRequest)
				return
			}
		}

		orderBy := core.ParseOrderByField(query.Get("orderBy"))
		descending, err := strconv.ParseBool(query.Get("descending"))
		if err != nil {
			writeError(w, logger, err, "invalid descending type", "Bad Request", http.StatusBadRequest)
			return
		}

		if s := query.Get("severityNumber"); s != "" {
			severityNumber, err = strconv.Atoi(query.Get("severityNumber"))
			if err != nil {
				writeError(w, logger, err, "invalid severityNumber type", "Bad Request", http.StatusBadRequest)
				return
			}
		}

		limit, err := strconv.Atoi(query.Get("limit"))
		if err != nil {
			writeError(w, logger, err, "invalid limit type", "Bad Request", http.StatusBadRequest)
			return
		}

		offset, err := strconv.Atoi(query.Get("offset"))
		if err != nil {
			writeError(w, logger, err, "invalid offset type", "Bad Request", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		filter := core.LogQueryFilter{
			StartTime:      startTime,
			EndTime:        endTime,
			ServiceName:    query.Get("serviceName"),
			SeverityNumber: severityNumber,
			SeverityText:   query.Get("severityText"),
			TraceId:        query.Get("traceId"),
			SpanId:         query.Get("spanId"),
			Body:           query.Get("body"),
			OrderBy:        orderBy,
			Descending:     descending,
			Limit:          limit,
			Offset:         offset,
		}

		flatLogRecords, err := logStore.SearchLogs(ctx, filter)
		if err != nil {
			writeError(w, logger, err, "failed to search logs", "Internal Server Error", http.StatusInternalServerError)
			return
		}

		logsCount, err := logStore.GetFilteredLogsCount(ctx, filter)
		if err != nil {
			writeError(w, logger, err, "failed to get logs count", "Internal Server Error", http.StatusInternalServerError)
			return
		}

		logRecords := core.UnflattenLogRecords(flatLogRecords)

		response := struct {
			Data []core.LogRecord `json:"data"`
			Meta struct {
				TotalRowCount int `json:"totalRowCount"`
			} `json:"meta"`
		}{
			Data: logRecords,
			Meta: struct {
				TotalRowCount int `json:"totalRowCount"`
			}{
				TotalRowCount: logsCount,
			},
		}

		writeJSON(w, logger, http.StatusOK, response)
	}
}

func handleDistinctServices(logger *slog.Logger, logStore *storage.ClickHouseStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx := r.Context()

		distinctServices, err := logStore.GetDistinctServices(ctx)
		if err != nil {
			writeError(w, logger, err, "failed to get distinct services", "Internal Server Error", http.StatusInternalServerError)
			return
		}

		response := map[string]any{
			"services": distinctServices,
		}

		writeJSON(w, logger, http.StatusOK, response)
	}
}

func handleLogRateStats(logger *slog.Logger, logStore *storage.ClickHouseStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx := r.Context()

		logRateStats, err := logStore.GetLogRateStatistics(ctx)
		if err != nil {
			writeError(w, logger, err, "failed to get log rate stats", "Internal Server Error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, logger, http.StatusOK, logRateStats)
	}
}

func handleErrorRateMetrics(logger *slog.Logger, logStore *storage.ClickHouseStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx := r.Context()

		errorRateMetrics, err := logStore.GetErrorRateMetrics(ctx)
		if err != nil {
			writeError(w, logger, err, "failed to get error metrics", "Internal Server Error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, logger, http.StatusOK, errorRateMetrics)
	}
}

func handleIngestionGraphMetrics(logger *slog.Logger, logStore *storage.ClickHouseStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx := r.Context()

		ingestionGraphMetrics, err := logStore.GetIngestionGraphMetrics(ctx)
		if err != nil {
			writeError(w, logger, err, "failed to get ingestion graph metrics", "Internal Server Error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, logger, http.StatusOK, ingestionGraphMetrics)
	}
}

func handleTopServiceErrors(logger *slog.Logger, logStore *storage.ClickHouseStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx := r.Context()

		topServiceErrorsStats, err := logStore.GetTopServiceErrorsStats(ctx)
		if err != nil {
			writeError(w, logger, err, "failed to get top service errors stats", "Internal Server Error", http.StatusInternalServerError)
			return
		}

		response := core.TopServiceErrorsStatsResponse{
			Data: topServiceErrorsStats,
		}

		writeJSON(w, logger, http.StatusOK, response)
	}
}

func handleStorageInfo(logger *slog.Logger, logStore *storage.ClickHouseStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		stats, err := logStore.GetStorageStats(ctx)
		if err != nil {
			writeError(w, logger, err, "failed to get storage stats", "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if len(stats) == 0 {
			writeError(w, logger, fmt.Errorf("no disk stats found"), "no disk stats found", "Internal Server Error", http.StatusInternalServerError)
			return
		}

		disk := stats[0]
		usedBytes := disk.TotalBytes - disk.FreeBytes
		usedPercent := float64(usedBytes) / float64(disk.TotalBytes) * 100

		outlook, err := logStore.GetLogsStorageOutlook(ctx)
		if err != nil {
			writeError(w, logger, err, "failed to get logs storage outlook", "Internal Server Error", http.StatusInternalServerError)
			return
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
			// Steady-state would exceed current capacity — compute a real runway.
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

		writeJSON(w, logger, http.StatusOK, card)
	}
}

func handleNatsDLQInfo(logger *slog.Logger, natsBroker *transport.NatsBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		natsDLQInfo, err := natsBroker.GetDLQStreamInfo(ctx, transport.DLQStreamName)
		if err != nil {
			writeError(w, logger, err, "failed to get nats dlq info", "Internal Server Error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, logger, http.StatusOK, natsDLQInfo)
	}

}

func handleNatsQueueDepthMetrics(logger *slog.Logger, logStore *storage.ClickHouseStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// retrieve nats queue depth metrics in the past 15 minutes
		natsQueueDepthMetrics, err := logStore.GetNatsQueueDepthMetrics(ctx, 15)
		if err != nil {
			writeError(w, logger, err, "failed to get nats queue depth metrics", "Internal Server Error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, logger, http.StatusOK, natsQueueDepthMetrics)
	}
}

func handleConfig(logger *slog.Logger, config *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, logger, http.StatusOK, config.Public())
	}
}
