package ingester

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type noOpProducer struct{}

func (n *noOpProducer) PublishLogs(context.Context, string, []byte, map[string][]string) error {
	return nil
}

func (n *noOpProducer) PublishLiveTail(string, []byte) error {
	return nil
}

func setupTestApp(tb testing.TB) http.Handler {
	tb.Helper()
	noOpLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mux := NewHTTPServer(noOpLogger, &noOpProducer{}, 4*1024, 8*1024)
	return mux
}

func BenchmarkIngestEndpointHttp(b *testing.B) {
	dummyBody := []byte("data")
	app := setupTestApp(b)
	req := httptest.NewRequest("POST", "/v1/logs", nil)
	req.Header.Set("Content-type", "application/json")
	rec := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		// reuse the body per iteration
		req.Body = io.NopCloser(bytes.NewReader(dummyBody))
		rec.Body.Reset()
		rec.Code = 0

		app.ServeHTTP(rec, req)
	}
}
