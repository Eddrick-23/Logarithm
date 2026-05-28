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

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/api/data", apiDataHandler)

	return mux
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintln(w, "Dashboard API root!")

	if err != nil {
		slog.Error("failed to write response", "err", err)
	}
}

func apiDataHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	layout := "2006-01-02T15:04:05" // reference layout for time
	var startTime, endTime time.Time
	var err error
	var severityNumber int = -1

	if s := query.Get("startTime"); s != "" {
		startTime, err = time.Parse(layout, s)
		if err != nil {
			slog.Error("invalid start time type", "err", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
	}

	if s := query.Get("endTime"); s != "" {
		endTime, err = time.Parse(layout, s)
		if err != nil {
			slog.Error("invalid end time type", "err", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
	}

	orderBy := core.ParseOrderByField(query.Get("orderBy"))
	if err != nil {
		slog.Error("invalid order by", "err", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	descending, err := strconv.ParseBool(query.Get("descending"))
	if err != nil {
		slog.Error("invalid descending type", "err", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if s := query.Get("severityNumber"); s != "" {
		severityNumber, err = strconv.Atoi(query.Get("severityNumber"))
		if err != nil {
			slog.Error("invalid severityNumber type", "err", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
	}

	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil {
		slog.Error("invalid limit type", "err", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	offset, err := strconv.Atoi(query.Get("offset"))
	if err != nil {
		slog.Error("invalid offset type", "err", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	cfg := config.LoadConfig()
	ctx := context.Background()

	logStore, err := storage.NewClickHouseStore(ctx, cfg.DBAddress, cfg.DBName, cfg.DBTableName, cfg.DBUser, cfg.DBPassword)
	if err != nil {
		slog.Error("failed to search logs", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

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
		slog.Error("failed to search logs", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	logsCount, err := logStore.GetFilteredLogsCount(ctx, filter)
	if err != nil {
		slog.Error("failed to get logs count", "err", err)
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
		slog.Error("failed to write response", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
