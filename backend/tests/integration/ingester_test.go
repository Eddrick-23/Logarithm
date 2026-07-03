//go:build integration

package integration

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/ingester"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// helper to set contentType and contentEncoding in headers
//
// If empty string is given, the key-value pair is not set at all
func makeHeaders(contentType string, contentEncoding string) map[string][]string {
	headers := map[string][]string{}
	if contentType != "" {
		headers["Content-Type"] = []string{contentType}
	}

	if contentEncoding != "" {
		headers["Content-Encoding"] = []string{contentEncoding}
	}

	return headers
}

// helper to compress payloads using zstd
func zstdCompress(tb testing.TB, data []byte) []byte {
	tb.Helper()
	var buf bytes.Buffer
	w, err := zstd.NewWriter(&buf)
	require.NoError(tb, err)
	_, err = w.Write(data)
	require.NoError(tb, err)
	require.NoError(tb, w.Close())
	return buf.Bytes()
}

// helper to compress payloads using gzip
func gzipCompress(tb testing.TB, data []byte) []byte {
	tb.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	_, err := gz.Write(data)
	require.NoError(tb, err)
	require.NoError(tb, gz.Close())
	return buf.Bytes()
}

type MockProducer struct {
	Err              error
	PublishLogsCount int
	PublishHeaders   map[string][]string
}

func (m *MockProducer) PublishLogs(ctx context.Context, subject string, payload []byte, headers map[string][]string) error {
	m.PublishLogsCount++
	m.PublishHeaders = headers
	return m.Err
}

func (m *MockProducer) PublishLiveTail(subject string, data []byte) error {
	return nil // not used
}

func setupTestApp(producerErr error) (http.Handler, *MockProducer) {
	mockProducer := &MockProducer{Err: producerErr}
	mux := ingester.NewHTTPServer(slog.Default(), mockProducer)
	return mux, mockProducer
}

func TestIngestEndpoint(t *testing.T) {
	dummyBody := []byte("data")
	tests := []struct {
		name                   string
		contentType            string
		encoding               string
		producerErr            error
		expectedPublishCount   int
		expectedPublishHeaders map[string][]string
		expectedStatus         int
		expectedBody           string
	}{
		{
			name:                   "json no encoding",
			contentType:            "application/json",
			encoding:               "",
			producerErr:            nil,
			expectedPublishCount:   1,
			expectedPublishHeaders: makeHeaders("application/json", ""),
			expectedStatus:         http.StatusAccepted,
			expectedBody:           "Log ingested successfully",
		},
		{
			name:                   "json gzip encoding",
			contentType:            "application/json",
			encoding:               "gzip",
			producerErr:            nil,
			expectedPublishCount:   1,
			expectedPublishHeaders: makeHeaders("application/json", "gzip"),
			expectedStatus:         http.StatusAccepted,
			expectedBody:           "Log ingested successfully",
		},
		{
			name:                   "json with zstd encoding",
			contentType:            "application/json",
			encoding:               "zstd",
			producerErr:            nil,
			expectedPublishCount:   1,
			expectedPublishHeaders: makeHeaders("application/json", "zstd"),
			expectedStatus:         http.StatusAccepted,
			expectedBody:           "Log ingested successfully",
		},
		{
			name:                   "protobuf no encoding",
			contentType:            "application/x-protobuf",
			encoding:               "",
			producerErr:            nil,
			expectedPublishCount:   1,
			expectedPublishHeaders: makeHeaders("application/x-protobuf", ""),
			expectedStatus:         http.StatusAccepted,
			expectedBody:           "Log ingested successfully",
		},
		{
			name:                   "protobuf gzip encoding",
			contentType:            "application/x-protobuf",
			encoding:               "gzip",
			producerErr:            nil,
			expectedPublishCount:   1,
			expectedPublishHeaders: makeHeaders("application/x-protobuf", "gzip"),
			expectedStatus:         http.StatusAccepted,
			expectedBody:           "Log ingested successfully",
		},
		{
			name:                   "protobuf with zstd encoding",
			contentType:            "application/x-protobuf",
			encoding:               "zstd",
			producerErr:            nil,
			expectedPublishCount:   1,
			expectedPublishHeaders: makeHeaders("application/x-protobuf", "zstd"),
			expectedStatus:         http.StatusAccepted,
			expectedBody:           "Log ingested successfully",
		},
		{
			name:                   "unsupported content type",
			contentType:            "text/plain",
			encoding:               "",
			producerErr:            nil,
			expectedPublishCount:   0,
			expectedPublishHeaders: nil,
			expectedStatus:         http.StatusUnsupportedMediaType,
			expectedBody:           "Unsupported or Missing Content-Type: text/plain",
		},
		{
			name:                   "missing content type",
			contentType:            "",
			encoding:               "",
			producerErr:            nil,
			expectedPublishCount:   0,
			expectedPublishHeaders: nil,
			expectedStatus:         http.StatusUnsupportedMediaType,
			expectedBody:           "Unsupported or Missing Content-Type:",
		},
		{
			name:                   "json publish failed",
			contentType:            "application/json",
			encoding:               "",
			producerErr:            fmt.Errorf("publish to nats failed"),
			expectedPublishCount:   1,
			expectedPublishHeaders: makeHeaders("application/json", ""),
			expectedStatus:         http.StatusServiceUnavailable,
			expectedBody:           "Message broker unavailable",
		},
		{
			name:                   "json unknown encoding",
			contentType:            "application/json",
			encoding:               "br",
			producerErr:            nil,
			expectedPublishCount:   0,
			expectedPublishHeaders: nil,
			expectedStatus:         http.StatusUnsupportedMediaType,
			expectedBody:           "Unsupported Content-Encoding: br",
		},
		{
			name:                   "protobuf publish failed",
			contentType:            "application/x-protobuf",
			encoding:               "",
			producerErr:            fmt.Errorf("publish to nats failed"),
			expectedPublishCount:   1,
			expectedPublishHeaders: makeHeaders("application/x-protobuf", ""),
			expectedStatus:         http.StatusServiceUnavailable,
			expectedBody:           "Message broker unavailable",
		},
		{
			name:                   "protobuf unknown encoding",
			contentType:            "application/x-protobuf",
			encoding:               "br",
			producerErr:            nil,
			expectedPublishCount:   0,
			expectedPublishHeaders: nil,
			expectedStatus:         http.StatusUnsupportedMediaType,
			expectedBody:           "Unsupported Content-Encoding: br",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			app, mockProducer := setupTestApp(tc.producerErr)
			req := httptest.NewRequest("POST", "/v1/logs", bytes.NewReader(dummyBody))
			req.Header.Set("Content-type", tc.contentType)
			req.Header.Set("Content-Encoding", tc.encoding)

			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedPublishCount, mockProducer.PublishLogsCount)
			// verify headers if we did publish
			if tc.producerErr == nil {
				assert.Equal(t, tc.expectedPublishHeaders, mockProducer.PublishHeaders)
			}

			assert.Equal(t, tc.expectedStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tc.expectedBody)
		})
	}
}

const bufSize = 1024 * 1024

func setupGRPCTestApp(t *testing.T, producerErr error) (*grpc.Server, *bufconn.Listener, *MockProducer) {
	t.Helper()
	mockProducer := &MockProducer{Err: producerErr}
	lis := bufconn.Listen(bufSize)

	server := ingester.NewGRPCServer(slog.Default(), mockProducer)

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Errorf("gRPC server exited with error: %v", err)
		}
	}()

	return server, lis, mockProducer
}

func TestGRPCIngestEndpoint(t *testing.T) {
	dummyBody := []byte("grpc-test-data")

	tests := []struct {
		name                   string
		encoding               string
		producerErr            error
		expectedPublishCount   int
		expectedPublishHeaders map[string][]string
		expectedCode           codes.Code
		expectedEncodingHeader string
	}{
		// We trust the gRPC framework to reject unsupported encodings
		// Only test the logic our proxy handles directly
		{
			name:                   "no encoding",
			encoding:               "",
			producerErr:            nil,
			expectedPublishCount:   1,
			expectedPublishHeaders: makeHeaders("application/x-protobuf", ""),
			expectedCode:           codes.OK,
			expectedEncodingHeader: "",
		},
		{
			name:                   "gzip encoding",
			encoding:               "gzip",
			producerErr:            nil,
			expectedPublishCount:   1,
			expectedPublishHeaders: makeHeaders("application/x-protobuf", "gzip"),
			expectedCode:           codes.OK,
			expectedEncodingHeader: "gzip",
		},
		{
			name:                   "zstd encoding",
			encoding:               "zstd",
			producerErr:            nil,
			expectedPublishCount:   1,
			expectedPublishHeaders: makeHeaders("application/x-protobuf", "zstd"),
			expectedCode:           codes.OK,
			expectedEncodingHeader: "zstd",
		},
		{
			name:                   "publish failed",
			encoding:               "",
			producerErr:            fmt.Errorf("publish to nats failed"),
			expectedPublishCount:   1,
			expectedPublishHeaders: makeHeaders("application/x-protobuf", ""),
			expectedCode:           codes.Internal,
			expectedEncodingHeader: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server, lis, mockProducer := setupGRPCTestApp(t, tc.producerErr)
			defer server.Stop()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			// setup client
			conn, err := grpc.NewClient("passthrough://bufnet",
				grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
					return lis.Dial()
				}),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithDefaultCallOptions(grpc.CallContentSubtype(ingester.CodecName)), // call our custom codec
			)
			assert.NoError(t, err)
			defer conn.Close()

			// setup internal calls
			var callOpts []grpc.CallOption
			if tc.encoding != "" { // call specific compressor
				callOpts = append(callOpts, grpc.UseCompressor(tc.encoding))
			}

			var payload []byte
			switch tc.encoding {
			case "zstd":
				payload = zstdCompress(t, dummyBody)
			case "gzip":
				payload = gzipCompress(t, dummyBody)
			default:
				payload = dummyBody
			}

			reqData := ingester.RawFrame{
				RawBytes: payload,
			}
			resData := ingester.RawFrame{}

			// make actual request
			err = conn.Invoke(ctx, "/OpenTelemetry.Logs/Export", &reqData, &resData, callOpts...)

			assert.Equal(t, tc.expectedPublishCount, mockProducer.PublishLogsCount)
			assert.Equal(t, tc.expectedPublishHeaders, mockProducer.PublishHeaders)

			if tc.expectedCode != codes.OK {
				assert.Error(t, err)
				st, _ := status.FromError(err)
				assert.Equal(t, tc.expectedCode, st.Code())
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestGRPCHealthCheck(t *testing.T) {
	server, lis, _ := setupGRPCTestApp(t, nil)
	defer server.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	assert.NoError(t, err)
	defer conn.Close()

	healthClient := healthpb.NewHealthClient(conn)
	resp, err := healthClient.Check(ctx, &healthpb.HealthCheckRequest{})
	assert.NoError(t, err)
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.Status)
}
