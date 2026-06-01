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
)

func AddRoutes(
	mux *http.ServeMux,
	logger *slog.Logger,
	config *config.Config,
	logStore *storage.ClickHouseStore,
) {
	mux.HandleFunc("GET /", handleRoot(logger))
	mux.HandleFunc("GET /health", handleHealth(logger))
	mux.Handle("GET /api/data", handleLogs(logger, logStore))
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
		bytes, err := json.Marshal(&response{Health: "ok"})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to marshall json: %v", err), http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(bytes)
		if err != nil {
			logger.Error("failed to write response", "err", err)
		}
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
				logger.Error("invalid start time type", "err", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
		}

		if s := query.Get("endTime"); s != "" {
			endTime, err = time.Parse(layout, s)
			if err != nil {
				logger.Error("invalid end time type", "err", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
		}

		orderBy := core.ParseOrderByField(query.Get("orderBy"))
		if err != nil {
			logger.Error("invalid order by", "err", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		descending, err := strconv.ParseBool(query.Get("descending"))
		if err != nil {
			logger.Error("invalid descending type", "err", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if s := query.Get("severityNumber"); s != "" {
			severityNumber, err = strconv.Atoi(query.Get("severityNumber"))
			if err != nil {
				logger.Error("invalid severityNumber type", "err", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
		}

		limit, err := strconv.Atoi(query.Get("limit"))
		if err != nil {
			logger.Error("invalid limit type", "err", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		offset, err := strconv.Atoi(query.Get("offset"))
		if err != nil {
			logger.Error("invalid offset type", "err", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
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
			logger.Error("failed to search logs", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		logsCount, err := logStore.GetFilteredLogsCount(ctx, filter)
		if err != nil {
			logger.Error("failed to get logs count", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			logger.Error("failed to write response", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}
