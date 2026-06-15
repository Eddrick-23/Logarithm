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
	"github.com/gorilla/websocket"
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
	mux.Handle("GET /api/error-metrics", handleErrorMetrics(logger, logStore))
	mux.Handle("GET /api/ingestion-metrics", handleIngestionMetrics(logger, logStore))
	mux.Handle("GET /api/ingestion-metrics/stream", handleIngestionMetricsStream(logger, logStore, appCtx))
	mux.Handle("GET /ws/logs/tail", handleLiveTail(logger, broker, config))
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

func handleDistinctServices(logger *slog.Logger, logStore *storage.ClickHouseStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx := context.Background()

		distinctServices, err := logStore.GetDistinctServices(ctx)
		if err != nil {
			logger.Error("failed to get distinct services", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		response := map[string]any{
			"services": distinctServices,
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

func handleIngestionMetrics(logger *slog.Logger, logStore *storage.ClickHouseStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx := context.Background()

		ingestionMetrics, err := logStore.GetIngestionMetrics(ctx)
		if err != nil {
			logger.Error("failed to get logging metrics", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(ingestionMetrics)
		if err != nil {
			logger.Error("failed to write response", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
	ingestionMetrics, err := logStore.GetIngestionMetrics(ctx)
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
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		// client likely disconnected
		return err
	}

	flusher.Flush()
	return nil
}

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

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		// send an initial payload immediately, don't wait for first tick
		if err := writeIngestionMetricsEvent(r.Context(), w, flusher, logger, logStore); err != nil {
			return
		}

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

			case <-ticker.C:
				if err := writeIngestionMetricsEvent(r.Context(), w, flusher, logger, logStore); err != nil {
					return
				}
			}
		}
	}
}

func handleErrorMetrics(logger *slog.Logger, logStore *storage.ClickHouseStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx := context.Background()

		errorMetrics, err := logStore.GetErrorMetrics(ctx)
		if err != nil {
			logger.Error("failed to get error metrics", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		response := core.ErrorMetricsResponse{
			Data: errorMetrics,
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

var upgrader = websocket.Upgrader{
	// CORS header
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleLiveTail(logger *slog.Logger, broker *transport.NatsBroker, config *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// upgrade HTTP to WebSocket
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("Failed to upgrade websocket", "error", err)
			return
		}
		defer ws.Close()

		// create a cancellable context tied to this WebSocket connection
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// listen for client disconnects to cancel the context
		go func() {
			for {
				// if ReadMessage fails, the client disconnected or the connection died
				if _, _, err := ws.ReadMessage(); err != nil {
					logger.Debug("Websocket client disconnected")
					cancel()
					return
				}
			}
		}()

		// start tailing NATS
		liveTailCh, cleanup, err := broker.TailLiveLogs(ctx, transport.LiveTailStreamName, transport.LiveTailSubject, config.LiveTailMaxBatch)
		if err != nil {
			logger.Error("Failed to start NATS tail", "error", err)
			ws.WriteMessage(websocket.CloseMessage, []byte("Internal Server Error"))
			return
		}
		defer cleanup() // ensure NATS consumer stops when the websocket closes

		ticker := time.NewTicker(time.Duration(config.LiveTailRefreshInterval) * time.Millisecond) // default flush interval: 500ms
		defer ticker.Stop()

		var batch []json.RawMessage // accumulate payloads between ticks

		// pump NATS messages to the WebSocket
		for {
			select {
			case <-ctx.Done():
				// context cancelled (client disconnected or server shutting down)
				return
			case payload, ok := <-liveTailCh:
				if !ok {
					// channel closed
					return
				}
				// each payload is an individual record, append directly
				batch = append(batch, json.RawMessage(payload))

			case <-ticker.C:
				if len(batch) == 0 {
					continue
				}
				out, err := json.Marshal(batch)
				if err != nil {
					logger.Error("Failed to marshal batch", "error", err)
					return
				}
				// write the log payload directly to the WebSocket
				err = ws.WriteMessage(websocket.TextMessage, out)
				if err != nil {
					logger.Error("Failed to write to websocket", "error", err)
					return // exiting the loop triggers defer cleanup() and cancel()
				}
				batch = batch[:0]
			}
		}
	}
}
