package worker

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/transport"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

func TestFlattenLogs(t *testing.T) {
	tests := []struct {
		name            string
		inputJSON       string
		expectedLength  int
		expectedAppends int
		check           func(t *testing.T, expectedLength, expectedAppends int, actualLogs []core.FlatLogRecord, mockAppender *MockLogAppender)
	}{
		{
			name: "Single Log",
			inputJSON: `{
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
			}`,
			expectedLength:  1,
			expectedAppends: 1,
			check: func(t *testing.T, expectedLength, expectedAppends int, actualLogs []core.FlatLogRecord, mockAppender *MockLogAppender) {
				require.Len(t, actualLogs, expectedLength)
				assert.Equal(t, mockAppender.AppendCount, expectedAppends)
				assert.Equal(t, time.Unix(0, 1717732530000000000), actualLogs[0].Timestamp)
				assert.Equal(t, "auth-service", actualLogs[0].ServiceName)
				assert.Equal(t, "INFO", actualLogs[0].SeverityText)
				assert.Equal(t, "user logged in", actualLogs[0].Body)
				assert.Equal(t, "string", actualLogs[0].BodyType)
			},
		},
		{
			name: "Single Log with Log and Resource Attributes",
			inputJSON: `{
				"resourceLogs": [{
					"resource": {
						"attributes": [{"key": "service.name", "value": {"stringValue": "auth-service"}}]
					},
					"scopeLogs": [{
						"logRecords": [{
							"timeUnixNano": "1717732530000000000",
							"severityText": "INFO",
							"body": {"stringValue": "user logged in"},
							"attributes": [
								{
								"key": "test.environment",
								"value": { "stringValue": "local" }
								}
							]
						}]
					}]
				}]
			}`,
			expectedLength:  1,
			expectedAppends: 1,
			check: func(t *testing.T, expectedLength, expectedAppends int, actualLogs []core.FlatLogRecord, mockAppender *MockLogAppender) {
				require.Len(t, actualLogs, expectedLength)
				assert.Equal(t, expectedAppends, mockAppender.AppendCount)
				assert.Equal(t, []string{"service.name"}, actualLogs[0].ResAttrKeys)
				assert.Equal(t, []string{"auth-service"}, actualLogs[0].ResAttrValues)
				assert.Equal(t, []string{"test.environment"}, actualLogs[0].LogAttrKeys)
				assert.Equal(t, []string{"local"}, actualLogs[0].LogAttrValues)
			},
		},
		{
			name: "Single Log Multiple Services",
			inputJSON: `{
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
				},
				{
					"resource": {
						"attributes": [{"key": "service.name", "value": {"stringValue": "payment-service"}}]
					},
					"scopeLogs": [{
						"logRecords": [{
							"timeUnixNano": "1717732530000000000",
							"severityText": "INFO",
							"body": {"stringValue": "user logged in"}
						}]
					}]
				}
				]
			}`,
			expectedLength:  2,
			expectedAppends: 2,
			check: func(t *testing.T, expectedLength, expectedAppends int, actualLogs []core.FlatLogRecord, mockAppender *MockLogAppender) {
				require.Len(t, actualLogs, expectedLength)
				assert.Equal(t, expectedAppends, mockAppender.AppendCount)
				serviceNames := []string{}
				for _, record := range actualLogs {
					serviceNames = append(serviceNames, record.ServiceName)
				}

				assert.Contains(t, serviceNames, "auth-service")
				assert.Contains(t, serviceNames, "payment-service")
			},
		},
		{
			name: "Multiple Logs Single Service",
			inputJSON: `{
				"resourceLogs": [{
					"resource": {
						"attributes": [{"key": "service.name", "value": {"stringValue": "auth-service"}}]
					},
					"scopeLogs": [{
						"logRecords": [
						{
							"timeUnixNano": "1717732530000000000",
							"severityText": "INFO",
							"body": {"stringValue": "user logged in"}
						},
						{
							"timeUnixNano": "1717732530000000000",
							"severityText": "ERROR",
							"body": {"stringValue": "login timed out"}
						}
						]
					}]
				}]
			}`,
			expectedLength:  2,
			expectedAppends: 2,
			check: func(t *testing.T, expectedLength, expectedAppends int, actualLogs []core.FlatLogRecord, mockAppender *MockLogAppender) {
				require.Len(t, actualLogs, expectedLength)
				assert.Equal(t, expectedAppends, mockAppender.AppendCount)
				serviceNames := make(map[string]struct{})
				for _, record := range actualLogs {
					serviceNames[record.ServiceName] = struct{}{}
				}

				assert.Len(t, serviceNames, 1)
			},
		},
		{
			name: "Missing Service Name",
			inputJSON: `{
				"resourceLogs": [{
					"resource": {}, 
					"scopeLogs": [{
						"logRecords": [{
							"severityText": "ERROR",
							"body": {"stringValue": "crash"}
						}]
					}]
				}]
			}`,
			expectedLength:  1,
			expectedAppends: 1,
			check: func(t *testing.T, expectedLength, expectedAppends int, actualLogs []core.FlatLogRecord, mockAppender *MockLogAppender) {
				require.Len(t, actualLogs, expectedLength)
				assert.Equal(t, expectedAppends, mockAppender.AppendCount)
				assert.Equal(t, "unknown", actualLogs[0].ServiceName)
				assert.Equal(t, "ERROR", actualLogs[0].SeverityText)
			},
		},
		{
			name: "Missing Timestamps",
			inputJSON: `{
				"resourceLogs": [{
					"resource": {
						"attributes": [{"key": "service.name", "value": {"stringValue": "auth-service"}}]
					},
					"scopeLogs": [{
						"logRecords": [{
							"severityText": "ERROR",
							"body": {"stringValue": "crash"}
						}]
					}]
				}]
			}`,
			expectedLength:  1,
			expectedAppends: 1,
			check: func(t *testing.T, expectedLength, expectedAppends int, actualLogs []core.FlatLogRecord, mockAppender *MockLogAppender) {
				require.Len(t, actualLogs, expectedLength)
				assert.Equal(t, expectedAppends, mockAppender.AppendCount)
				assert.WithinDuration(t, time.Now(), actualLogs[0].Timestamp, 2*time.Second)
				assert.WithinDuration(t, time.Now(), actualLogs[0].ObservedTimestamp, 2*time.Second)
			},
		},
		{
			name:           "Empty Payload",
			inputJSON:      `{"resourceLogs": []}`,
			expectedLength: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := plogotlp.NewExportRequest()
			err := req.UnmarshalJSON([]byte(tc.inputJSON))
			require.NoError(t, err, "invalid test JSON provided")

			flatLogsByServiceName := make(map[string][]core.FlatLogRecord)

			logs := req.Logs()

			mockAppender := MockLogAppender{}
			for i := 0; i < logs.ResourceLogs().Len(); i++ {
				flattenLogs(logs.ResourceLogs().At(i), flatLogsByServiceName, &mockAppender)
			}

			var actualLogs []core.FlatLogRecord
			for _, logs := range flatLogsByServiceName {
				actualLogs = append(actualLogs, logs...)
			}

			assert.Equal(t, tc.expectedLength, len(actualLogs), "slice length mismatch")

			if tc.expectedLength > 0 {
				tc.check(t, tc.expectedLength, tc.expectedAppends, actualLogs, &mockAppender)
			}
		})
	}
}
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

func TestDecompress(t *testing.T) {
	tests := []struct {
		name           string
		message        transport.Message
		expectedErr    bool
		expectedResult []byte
	}{
		{
			name:           "decompress zstd",
			message:        makeMessages(t, [][]byte{zstdCompress(t, []byte("data"))}, map[string][]string{"Content-Encoding": {"zstd"}})[0],
			expectedResult: []byte("data"),
		},
		{
			name:           "decompress gzip",
			message:        makeMessages(t, [][]byte{gzipCompress(t, []byte("data"))}, map[string][]string{"Content-Encoding": {"gzip"}})[0],
			expectedResult: []byte("data"),
		},
		{
			name:        "decompress zstd wrong encoding",
			message:     makeMessages(t, [][]byte{zstdCompress(t, []byte("data"))}, map[string][]string{"Content-Encoding": {"gzip"}})[0],
			expectedErr: true,
		},
		{
			name:        "decompress gzip wrong encoding",
			message:     makeMessages(t, [][]byte{gzipCompress(t, []byte("data"))}, map[string][]string{"Content-Encoding": {"zstd"}})[0],
			expectedErr: true,
		},
		{
			name:           "no header returns raw zstd data",
			message:        makeMessages(t, [][]byte{zstdCompress(t, []byte("data"))}, nil)[0],
			expectedResult: zstdCompress(t, []byte("data")),
		},
		{
			name:           "no header returns raw gzip data",
			message:        makeMessages(t, [][]byte{gzipCompress(t, []byte("data"))}, nil)[0],
			expectedResult: gzipCompress(t, []byte("data")),
		},
	}

	decompress, err := makeDecompressor()
	require.NoError(t, err)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := decompress(tc.message.Payload, tc.message.Headers)
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResult, res)
			}
		})
	}
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
	validReqbytes, err := validReq.MarshalProto()
	require.NoError(t, err)

	mockDecompressor := func(payload []byte, headers map[string][]string) ([]byte, error) {
		return payload, nil
	}

	mockDecoder := func(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error) {
		return &validReq, nil
	}

	mockDecompressorErr := func(payload []byte, headers map[string][]string) ([]byte, error) {
		return payload, fmt.Errorf("decompress failed")
	}

	mockDecoderErr := func(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error) {
		return nil, fmt.Errorf("decode failed")
	}

	headers := makeHeaders("application/x-protobuf", "")
	tests := []struct {
		name            string
		messages        []transport.Message
		decompressor    decompressFunc
		decoder         decoderFunc
		expectedAppends int
	}{
		{
			name:            "successful process single message",
			messages:        makeMessages(t, [][]byte{validReqbytes}, headers),
			decompressor:    mockDecompressor,
			decoder:         mockDecoder,
			expectedAppends: 1,
		},
		{
			name:            "successful process multiple message",
			messages:        makeMessages(t, [][]byte{validReqbytes, validReqbytes}, headers),
			decompressor:    mockDecompressor,
			decoder:         mockDecoder,
			expectedAppends: 2,
		},
		{
			name:         "mixed batch one failure one success",
			messages:     makeMessages(t, [][]byte{validReqbytes, []byte("bad-data")}, headers),
			decompressor: mockDecompressor,
			decoder: func(b []byte, m map[string][]string) (*plogotlp.ExportRequest, error) {
				if string(b) == "bad-data" {
					return nil, fmt.Errorf("decode failed")
				}
				return &validReq, nil
			},
			expectedAppends: 1,
		},
		{
			name:            "empty messages",
			messages:        makeMessages(t, [][]byte{}, headers),
			decompressor:    mockDecompressor,
			decoder:         mockDecoder,
			expectedAppends: 0,
		},
		{
			name:            "decompress failure",
			messages:        makeMessages(t, [][]byte{validReqbytes}, headers),
			decompressor:    mockDecompressorErr,
			decoder:         mockDecoder,
			expectedAppends: 0,
		},
		{
			name:            "decode failure",
			messages:        makeMessages(t, [][]byte{validReqbytes}, headers),
			decompressor:    mockDecompressor,
			decoder:         mockDecoderErr,
			expectedAppends: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flatLogsMap := map[string][]core.FlatLogRecord{}
			appender := MockLogAppender{}
			processMessages(slog.Default(), tc.decompressor, tc.decoder, tc.messages, flatLogsMap, &appender)

			assert.Equal(t, tc.expectedAppends, appender.AppendCount)
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
		name                string
		message             []transport.Message
		flushErr            error
		expectedAppends     int
		expectedTailSubject string // if empty, skip live tail assertion
	}{
		{
			name:                "Successful Processing and Live Tail Publish",
			message:             message,
			flushErr:            nil,
			expectedAppends:     1,
			expectedTailSubject: transport.LiveTailSubjectPrefix + "auth-service",
		},
		{
			name:                "Empty messages short circuit",
			message:             []transport.Message{},
			flushErr:            nil,
			expectedAppends:     0,
			expectedTailSubject: "",
		},
		{
			name:                "Flush error is returned",
			message:             message,
			flushErr:            flushErr,
			expectedAppends:     1,
			expectedTailSubject: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			appender := MockLogAppender{FlushErr: tc.flushErr}
			store := &MockLogStore{Appender: &appender}
			producer := &MockProducer{
				PublishCh: make(chan struct{}, 1),
			}
			callback, err := ConsumeCallback(slog.Default(), store, producer)

			require.NoError(t, err, "error creating consume callback")

			err = callback(tc.message)

			if tc.flushErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expectedAppends, appender.AppendCount)

			// assert tail publishing if subject given
			if tc.expectedTailSubject != "" {
				select {
				case <-producer.PublishCh:
				case <-time.After(2 * time.Second):
					t.Fatalf("timed out waiting for live tail publish go routine")
				}

				producer.mu.Lock()
				defer producer.mu.Unlock()

				published := producer.PublishedRecords[tc.expectedTailSubject]
				require.Len(t, published, 1, "should have published 1 record to subject: %v", tc.expectedTailSubject)
				assert.NotEmpty(t, published[0], "published payload should not be empty")
			}

		})
	}
}
