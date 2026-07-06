package worker

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/storage"
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

// used only here to isolate parallel processing of incoming payloads
type LockFreeAppender struct{}

func (l *LockFreeAppender) Append(
	timestamp, observedTimestamp time.Time,
	severityNumber uint8,
	traceId [16]byte,
	spanId [8]byte,
	logAttrKeys, logAttrValues, resAttrKeys, resAttrValues []string,
	logFields storage.LogFields,
) {
	// do nothing
}

func (l *LockFreeAppender) Flush(ctx context.Context) error {
	return nil
}

func BenchmarkConsumeCallback(b *testing.B) {
	validReq := newBaseRequest(b)
	validReqBytes, err := validReq.MarshalProto()
	require.NoError(b, err)
	compressedBytes := zstdCompress(b, validReqBytes)

	const numMessages = 300000
	payloads := make([][]byte, numMessages)
	for i := range payloads {
		payloads[i] = compressedBytes
	}

	messages := makeMessages(b, payloads, makeHeaders("application/x-protobuf", "zstd"))

	workerCounts := []int{1, 3, 5}

	for _, n := range workerCounts {
		b.Run(fmt.Sprintf("workers=%d", n), func(b *testing.B) {
			store := &MockLogStore{Appender: &LockFreeAppender{}}

			factory := func() (Decompressor, error) {
				if n == 1 {
					return NewLogDecompressor()
				} else {
					return NewLogDecompressor(WithZstdConcurrencyLimit(1))
				}
			}
			decoder := NewLogDecoder()

			transformer := NewLogTransformer(slog.New(slog.NewTextHandler(io.Discard, nil)))

			wp, err := NewWorkerPool(Config{
				Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
				Store:         store,
				DecompFactory: factory,
				Decoder:       decoder,
				Transformer:   transformer,
				Publisher:     &NoOpPublisher{},
				EstimatedRows: numMessages * 2,
				NumWorkers:    n,
			})
			defer wp.Close()

			callback, err := wp.ConsumeCallback()
			require.NoError(b, err)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				err := callback(messages)

				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
