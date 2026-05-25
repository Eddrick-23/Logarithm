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

	startTime, err := time.Parse(layout, query.Get("startTime"))
	if err != nil {
		slog.Error("invalid start time type", "err", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(layout, query.Get("endTime"))
	if err != nil {
		slog.Error("invalid end time type", "err", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	orderBy, err := core.ParseOrderByField(query.Get("orderBy"))
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

	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil {
		slog.Error("invalid limit type", "err", err)
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
		StartTime:    startTime,
		EndTime:      endTime,
		ServiceName:  query.Get("serviceName"),
		SeverityText: query.Get("severityText"),
		TraceId:      query.Get("traceId"),
		SpanId:       query.Get("spanId"),
		SearchTerm:   query.Get("searchTerm"),
		OrderBy:      orderBy,
		Descending:   descending,
		Limit:        limit,
	}

	records, err := logStore.SearchLogs(ctx, filter)
	if err != nil {
		slog.Error("failed to search logs", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	dtos := core.FlatLogRecordsToDTO(records)

	response := struct {
		Data []core.LogRecordDTO `json:"data"`
		Meta struct {
			TotalRowCount int `json:"totalRowCount"`
		} `json:"meta"`
	}{
		Data: dtos,
		Meta: struct {
			TotalRowCount int `json:"totalRowCount"`
		}{
			TotalRowCount: len(dtos),
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

// TODO: add search API which pings to database
