package ingester

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Eddrick-23/Logarithm/internal/transport"
	"google.golang.org/grpc"
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

type mockServerStream struct {
	grpc.ServerStream // Embed to satisfy unused methods
	payload           []byte
}

func (m *mockServerStream) Context() context.Context {
	return context.Background()
}

func (m *mockServerStream) RecvMsg(v any) error {
	frame, ok := v.(*RawFrame)
	if !ok {
		return fmt.Errorf("expected *RawFrame, got %T", v)
	}

	frame.RawBytes = append(frame.RawBytes[:0], m.payload...)
	return nil
}

func (m *mockServerStream) SendMsg(v any) error {
	return nil
}

func BenchmarkIngestEndpointGrpc(b *testing.B) {
	noOpLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	producer := &noOpProducer{}

	const bufSize = 1024 * 1024

	proxy := newProxyHandler(noOpLogger, producer, transport.LogStreamSubject)
	handler := proxy.NewStreamHandler(bufSize, bufSize*2)

	dummyPayload := []byte("data")

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		stream := &mockServerStream{
			payload: dummyPayload,
		}

		err := handler(nil, stream)
		if err != nil {
			b.Fatalf("error sending grpc request: %v", err)
		}
	}
}
