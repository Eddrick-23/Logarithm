package dashboard

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Eddrick-23/OrbitalTest/internal/core"
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
	dummyLogs := []core.LogRecordDTO{
		{
			Timestamp:      time.Now().Add(-2 * time.Hour), // 2 hours ago
			TraceId:        "5b8aa5a2d2c872e8321cf3730591f855",
			SpanId:         "9248231e3ccb89ad",
			SeverityText:   "INFO",
			SeverityNumber: 9, // OTel standard for INFO
			Body:           "User authentication successful",
			LogAttributes: []core.KeyValue{
				{Key: "user.id", Value: "user_8932"},
				{Key: "http.method", Value: "POST"},
				{Key: "http.route", Value: "/api/login"},
			},
		},
		{
			Timestamp:      time.Now().Add(-5 * time.Minute), // 5 minutes ago
			TraceId:        "a4f812b18e7c10b240391c0192bd80aa",
			SpanId:         "d78a9c210bf23c45",
			SeverityText:   "WARN",
			SeverityNumber: 13, // OTel standard for WARN
			Body:           "API rate limit approaching",
			LogAttributes: []core.KeyValue{
				{Key: "tenant.id", Value: "org_112"},
				{Key: "rate.limit.remaining", Value: "5"},
			},
		},
		{
			Timestamp:      time.Now(), // Right now
			TraceId:        "f18b3d7a8e2c10b240391c0192bc55ef",
			SpanId:         "c32a9c110bf23e99",
			SeverityText:   "ERROR",
			SeverityNumber: 17, // OTel standard for ERROR
			Body:           "Database connection timeout",
			LogAttributes: []core.KeyValue{
				{Key: "db.system", Value: "postgresql"},
				{Key: "error.type", Value: "timeout"},
				{Key: "db.name", Value: "users_db"},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(dummyLogs)

	if err != nil {
		slog.Error("failed to write response", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// TODO: add search API which pings to database
