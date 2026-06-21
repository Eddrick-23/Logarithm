package worker

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/stretchr/testify/require"
)

var testRecord core.FlatLogRecord = core.FlatLogRecord{
	Timestamp:         time.Now().Add(-1 * time.Hour),
	ObservedTimestamp: time.Now().Add(-1 * time.Hour),
	TraceId:           "4bf92f3577b34da6a3ce929d0e0e4736",
	SpanId:            "00f067aa0ba902b7",
	SeverityText:      "ERROR",
	SeverityNumber:    17,
	ServiceName:       "test-service1",
	Body:              "Failed to process transaction due to timeout",
	BodyType:          "string",
	ScopeName:         "1.0.0",
	ScopeVersion:      "manual-tester",
	LogAttrKeys:       []string{"http.method", "http.status_code", "retry_count"},
	LogAttrValues:     []string{"POST", "504", "3"},
	ResAttrKeys:       []string{"host.name", "os.type", "service.version"},
	ResAttrValues:     []string{"prod-server-1", "linux", "1.2.0"},
}

type noOpProducer struct{}

func (n *noOpProducer) PublishLogs(_ context.Context, _ string, _ []byte, _ map[string][]string) error {
	return nil
}

func (n *noOpProducer) PublishLiveTail(_ string, _ []byte) error {
	return nil
}

func BenchmarkLiveTailPublisher(b *testing.B) {
	// 2 Workers, Queue size of 1000
	pub := NewLiveTailPublisher(slog.Default(), &noOpProducer{}, 2, 1000)
	defer pub.Close()

	b.ReportAllocs()

	for b.Loop() {
		pub.Enqueue("test-subject", &testRecord)
	}
}

func BenchmarkDecompressorZstd(b *testing.B) {
	decompressor, err := NewLogDecompressor()
	require.NoError(b, err)
	zstdPayload := zstdCompress(b, []byte("sample payload"))
	headers := makeHeaders("", "zstd")

	b.ReportAllocs()
	for b.Loop() {
		_, cleanup, err := decompressor.decompress(zstdPayload, headers)
		if err != nil {
			b.Fatal(err)
		}
		cleanup()
	}
}

func BenchmarkDecompressorGzip(b *testing.B) {
	decompressor, err := NewLogDecompressor()
	require.NoError(b, err)
	gzipPayload := gzipCompress(b, []byte("sample payload"))
	headers := makeHeaders("", "gzip")

	b.ReportAllocs()
	for b.Loop() {
		_, cleanup, err := decompressor.decompress(gzipPayload, headers)
		if err != nil {
			b.Fatal(err)
		}
		cleanup()
	}
}
