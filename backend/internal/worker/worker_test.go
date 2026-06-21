package worker

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/transport"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

func TestDelayCalculator(t *testing.T) {
	tests := []struct {
		name           string
		deliveredCount uint64
		backoff        []time.Duration
		expectedDelay  time.Duration
	}{
		{
			"deliveredCount within backoff slice length",
			3,
			[]time.Duration{1 * time.Second, 2 * time.Second, 3 * time.Second, 4 * time.Second, 5 * time.Second},
			3 * time.Second,
		},
		{
			"deliveredCount 0 returns idx 0 duration",
			0,
			[]time.Duration{1 * time.Second, 2 * time.Second, 3 * time.Second, 4 * time.Second, 5 * time.Second},
			1 * time.Second,
		},
		{
			"deliveredCount greater than slice length returns last index duration",
			10,
			[]time.Duration{1 * time.Second, 2 * time.Second, 3 * time.Second, 4 * time.Second, 5 * time.Second},
			5 * time.Second,
		},
		{
			"empty backoff slice returns 0",
			3,
			[]time.Duration{},
			0 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := DelayCalculator(tc.backoff)
			delay := handler(tc.deliveredCount)

			assert.Equal(t, tc.expectedDelay, delay)
		})
	}

}

func newBaseRequest(t *testing.T) plogotlp.ExportRequest {
	inputJSON := `{
				"resourceLogs": [{
					"resource": {
						"attributes": [{"key": "service.name", "value": {"stringValue": "auth-service"}}]
					},
					"scopeLogs": [{
						"logRecords": [{
							"timeUnixNano": "1717732530000000000",
							"severityText": "INFO",
							"body": {"stringValue": "user logged in"}
						}]
					}]
				}]
			}`

	req := plogotlp.NewExportRequest()
	err := req.UnmarshalJSON([]byte(inputJSON))
	require.NoError(t, err, "invalid inputJSON provided")
	return req
}

func zstdCompress(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := zstd.NewWriter(&buf)
	require.NoError(t, err)
	_, err = w.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func gzipCompress(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	_, err := gz.Write(data)
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func TestDecode(t *testing.T) {
	req := newBaseRequest(t)
	jsonPayload, err := req.MarshalJSON()
	require.NoError(t, err)

	protobufPayload, err := req.MarshalProto()
	require.NoError(t, err)

	emptyHeaders := map[string][]string{}
	jsonHeaders := map[string][]string{"Content-Type": {"application/json"}}
	protobufHeaders := map[string][]string{"Content-Type": {"application/x-protobuf"}}

	tests := []struct {
		name        string
		payload     []byte
		headers     map[string][]string
		expectedErr bool
	}{
		{
			name:        "valid json payload with correct headers",
			payload:     jsonPayload,
			headers:     jsonHeaders,
			expectedErr: false,
		},
		{
			name:        "valid protobuf payload with correct headers",
			payload:     protobufPayload,
			headers:     protobufHeaders,
			expectedErr: false,
		},
		{
			name:        "invalid json correct headers",
			payload:     []byte("not json"),
			headers:     jsonHeaders,
			expectedErr: true,
		},
		{
			name:        "invalid protobuf correct headers",
			payload:     []byte("not protobuf"),
			headers:     protobufHeaders,
			expectedErr: true,
		},
		{
			name:        "valid json wrong headers",
			payload:     jsonPayload,
			headers:     protobufHeaders,
			expectedErr: true,
		},
		{
			name:        "valid protobuf wrong headers",
			payload:     protobufPayload,
			headers:     jsonHeaders,
			expectedErr: true,
		},
		{
			name:        "valid json no headers",
			payload:     jsonPayload,
			headers:     emptyHeaders,
			expectedErr: true,
		},
		{
			name:        "valid protobuf no headers",
			payload:     protobufPayload,
			headers:     emptyHeaders,
			expectedErr: true,
		},
	}

	decoder := makeDecoder()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decodedReq, err := decoder(tc.payload, tc.headers)

			if tc.expectedErr {
				require.Error(t, err)
				require.Nil(t, decodedReq)
			} else {
				require.NoError(t, err)
				require.NotNil(t, decodedReq)

				assert.Equal(
					t,
					1,
					decodedReq.Logs().ResourceLogs().Len(),
					"expected 1 resource log in decoded payload",
				)
			}
		})
	}
}

func makeMessages(t *testing.T, payloads [][]byte, headers map[string][]string) []transport.Message {
	t.Helper()
	msgs := make([]transport.Message, len(payloads))

	for i, p := range payloads {
		msgs[i] = transport.Message{
			Payload: p,
			Headers: headers,
		}
	}

	return msgs
}

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

func TestProcessMessages(t *testing.T) {
	validReq := newBaseRequest(t)
	validReqBytes, err := validReq.MarshalProto()
	require.NoError(t, err)

	mockDecoder := func(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error) {
		return &validReq, nil
	}

	mockDecoderErr := func(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error) {
		return nil, fmt.Errorf("decode failed")
	}

	headers := makeHeaders("application/x-protobuf", "")
	tests := []struct {
		name             string
		messages         []transport.Message
		decompressor     Decompressor
		decoder          decoderFunc
		expectedFlattens int
	}{
		{
			name:             "successful process single message",
			messages:         makeMessages(t, [][]byte{validReqBytes}, headers),
			decompressor:     &MockDecompressor{decompressError: nil},
			decoder:          mockDecoder,
			expectedFlattens: 1,
		},
		{
			name:             "successful process multiple message",
			messages:         makeMessages(t, [][]byte{validReqBytes, validReqBytes}, headers),
			decompressor:     &MockDecompressor{decompressError: nil},
			decoder:          mockDecoder,
			expectedFlattens: 2,
		},
		{
			name:         "mixed batch one failure one success",
			messages:     makeMessages(t, [][]byte{validReqBytes, []byte("bad-data")}, headers),
			decompressor: &MockDecompressor{decompressError: nil},
			decoder: func(b []byte, m map[string][]string) (*plogotlp.ExportRequest, error) {
				if string(b) == "bad-data" {
					return nil, fmt.Errorf("decode failed")
				}
				return &validReq, nil
			},
			expectedFlattens: 1,
		},
		{
			name:             "empty messages",
			messages:         makeMessages(t, [][]byte{}, headers),
			decompressor:     &MockDecompressor{decompressError: nil},
			decoder:          mockDecoder,
			expectedFlattens: 0,
		},
		{
			name:             "decompress failure",
			messages:         makeMessages(t, [][]byte{validReqBytes}, headers),
			decompressor:     &MockDecompressor{decompressError: fmt.Errorf("decompress error")},
			decoder:          mockDecoder,
			expectedFlattens: 0,
		},
		{
			name:             "decode failure",
			messages:         makeMessages(t, [][]byte{validReqBytes}, headers),
			decompressor:     &MockDecompressor{decompressError: nil},
			decoder:          mockDecoderErr,
			expectedFlattens: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			appender := &MockLogAppender{}
			transformer := &MockTransformer{}
			processMessages(slog.Default(), tc.decompressor, tc.decoder, tc.messages,
				transformer, &NoOpPublisher{}, appender)
			assert.Equal(t, tc.expectedFlattens, transformer.flattenCount)
		})
	}
}

func TestConsumeCallback(t *testing.T) {
	valiReq := newBaseRequest(t)
	validReqbytes, err := valiReq.MarshalProto()
	require.NoError(t, err)

	message := makeMessages(t, [][]byte{validReqbytes}, makeHeaders("application/x-protobuf", ""))
	flushErr := fmt.Errorf("database insert error")

	tests := []struct {
		name     string
		message  []transport.Message
		flushErr error
	}{
		{
			name:     "Successful Processing and Live Tail Publish",
			message:  message,
			flushErr: nil,
		},
		{
			name:     "Empty messages short circuit",
			message:  []transport.Message{},
			flushErr: nil,
		},
		{
			name:     "Flush error is returned",
			message:  message,
			flushErr: flushErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			appender := MockLogAppender{flushErr: tc.flushErr}
			store := &MockLogStore{Appender: &appender}
			callback, err := ConsumeCallback(slog.Default(), store, &NoOpDecompressor{}, &NoOpTransformer{}, &NoOpPublisher{})

			require.NoError(t, err, "error creating consume callback")

			err = callback(tc.message)

			if tc.flushErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
