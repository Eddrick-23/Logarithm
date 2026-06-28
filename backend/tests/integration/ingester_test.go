//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/ingester"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type MockProducer struct {
	Err error
}

func (m *MockProducer) PublishLogs(context.Context, string, []byte, map[string][]string) error {
	return m.Err
}

func (m *MockProducer) PublishLiveTail(subject string, data []byte) error {
	return nil // not used
}

func setupTestApp(producerErr error) http.Handler {
	mockProducer := &MockProducer{producerErr}
	mux := http.NewServeMux()
	ingester.AddRoutes(mux, slog.Default(), mockProducer, "logs.")
	return mux
}

func TestIngestEndpoint(t *testing.T) {
	dummyBody := []byte("data")
	tests := []struct {
		name           string
		contentType    string
		encoding       string
		producerErr    error
		expectedStatus int
		expectedBody   string
	}{
		{"json no encoding", "application/json", "", nil, http.StatusAccepted, "Log ingested successfully"},
		{"json gzip encoding", "application/json", "gzip", nil, http.StatusAccepted, "Log ingested successfully"},
		{"json with zstd encoding", "application/json", "zstd", nil, http.StatusAccepted, "Log ingested successfully"},
		{"protobuf no encoding", "application/x-protobuf", "", nil, http.StatusAccepted, "Log ingested successfully"},
		{"protobuf gzip encoding", "application/x-protobuf", "gzip", nil, http.StatusAccepted, "Log ingested successfully"},
		{"protobuf with zstd encoding", "application/x-protobuf", "zstd", nil, http.StatusAccepted, "Log ingested successfully"},
		{"unsupported content type", "text/plain", "", nil, http.StatusUnsupportedMediaType, "Unsupported or Missing Content-Type: text/plain"},
		{"missing content type", "", "", nil, http.StatusUnsupportedMediaType, "Unsupported or Missing Content-Type:"},
		{"json publish failed", "application/json", "", fmt.Errorf("publish to nats failed"), http.StatusServiceUnavailable, "Message broker unavailable"},
		{"json unknown encoding", "application/json", "br", nil, http.StatusUnsupportedMediaType, "Unsupported Content-Encoding: br"},
		{"protobuf publish failed", "application/x-protobuf", "", fmt.Errorf("publish to nats failed"), http.StatusServiceUnavailable, "Message broker unavailable"},
		{"protobuf unknown encoding", "application/x-protobuf", "br", nil, http.StatusUnsupportedMediaType, "Unsupported Content-Encoding: br"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			app := setupTestApp(tc.producerErr)
			req := httptest.NewRequest("POST", "/v1/logs", bytes.NewReader(dummyBody))
			req.Header.Set("Content-type", tc.contentType)
			req.Header.Set("Content-Encoding", tc.encoding)

			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tc.expectedBody)
		})
	}
}

const bufSize = 1024 * 1024

func setupGRPCTestApp(t *testing.T, producerErr error) (*grpc.Server, *bufconn.Listener) {
	t.Helper()
	mockProducer := MockProducer{Err: producerErr}
	lis := bufconn.Listen(bufSize)

	proxyHandler := ingester.NewProxyHandler(slog.Default(), &mockProducer, "logs.")

	server := grpc.NewServer(
		grpc.ForceServerCodecV2(encoding.GetCodecV2(ingester.CodecName)),
		grpc.UnknownServiceHandler(proxyHandler.StreamHandler),
	)

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Errorf("gRPC server exited with error: %v", err)
		}
	}()

	return server, lis
}

func TestGRPCIngestEndpoint(t *testing.T) {
	dummyBody := []byte("grpc-test-data")

	tests := []struct {
		name           string
		encoding       string
		producerErr    error
		expectedCode   codes.Code
		expectEncoding string
	}{
		// We trust the gRPC framework to reject unsupported encodings
		// Only test the logic our proxy handles directly
		{"no encoding", "", nil, codes.OK, ""},
		{"gzip encoding", "gzip", nil, codes.OK, "gzip"},
		{"zstd encoding", "zstd", nil, codes.OK, "zstd"},
		{"publish failed", "", fmt.Errorf("publish to nats failed"), codes.Internal, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server, lis := setupGRPCTestApp(t, tc.producerErr)
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

			reqData := ingester.RawFrame{
				RawBytes: dummyBody,
			}
			resData := ingester.RawFrame{}

			// make actual request
			err = conn.Invoke(ctx, "/OpenTelemetry.Logs/Export", &reqData, &resData, callOpts...)

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
