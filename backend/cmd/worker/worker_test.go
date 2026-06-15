package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"log/slog"
	"sync"
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
		name           string
		inputJSON      string
		expectedLength int
		check          func(t *testing.T, actualLogs []core.FlatLogRecord)
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
			expectedLength: 1,
			check: func(t *testing.T, actualLogs []core.FlatLogRecord) {
				require.Len(t, actualLogs, 1)
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
			expectedLength: 1,
			check: func(t *testing.T, actualLogs []core.FlatLogRecord) {
				require.Len(t, actualLogs, 1)
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
			expectedLength: 2,
			check: func(t *testing.T, actualLogs []core.FlatLogRecord) {
				assert.Len(t, actualLogs, 2)
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
			expectedLength: 2,
			check: func(t *testing.T, actualLogs []core.FlatLogRecord) {
				assert.Len(t, actualLogs, 2)
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
			expectedLength: 1,
			check: func(t *testing.T, actualLogs []core.FlatLogRecord) {
				require.Len(t, actualLogs, 1)
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
			expectedLength: 1,
			check: func(t *testing.T, actualLogs []core.FlatLogRecord) {
				require.Len(t, actualLogs, 1)
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
			totalExtracted := 0

			logs := req.Logs()

			for i := 0; i < logs.ResourceLogs().Len(); i++ {
				totalExtracted += flattenLogs(logs.ResourceLogs().At(i), flatLogsByServiceName)
			}

			var actualLogs []core.FlatLogRecord
			for _, logs := range flatLogsByServiceName {
				actualLogs = append(actualLogs, logs...)
			}

			assert.Equal(t, tc.expectedLength, totalExtracted, "extracted count mismatch")
			assert.Equal(t, tc.expectedLength, len(actualLogs), "slice length mismatch")

			if tc.expectedLength > 0 {
				tc.check(t, actualLogs)
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

type MockLogStore struct {
	InsertedRecords []core.FlatLogRecord
	InsertErr       error
	InsertCount     int
}

func (m *MockLogStore) BatchInsert(ctx context.Context, records []core.FlatLogRecord) error {
	m.InsertCount++
	m.InsertedRecords = records
	return m.InsertErr
}

func (m *MockLogStore) SearchLogs(ctx context.Context, filter core.LogQueryFilter) ([]core.FlatLogRecord, error) {
	return nil, nil // not needed for this test
}

type MockProducer struct {
	PublishedRecords map[string][][]byte
	PublishErr       error
	PublishCount     int
	mu               sync.Mutex
	PublishCh        chan struct{}
}

func (m *MockProducer) PublishLogs(ctx context.Context, subject string, payload []byte, headers map[string][]string) error {
	return nil // not used
}

func (m *MockProducer) PublishLiveTail(subject string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.PublishedRecords == nil {
		m.PublishedRecords = map[string][][]byte{}
	}

	m.PublishedRecords[subject] = append(m.PublishedRecords[subject], data)

	select {
	case m.PublishCh <- struct{}{}: // signal test thread a publish occured
	default:
	}

	return m.PublishErr
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

func assertPublishes(t *testing.T, mp *MockProducer, expectedCount int) {
	t.Helper()
	for i := 0; i < expectedCount; i++ {
		select {
		case <-mp.PublishCh:
		case <-time.After(1 * time.Second):
			t.Fatalf("timeout waiting for background nats publish (got %d of %d)", i, expectedCount)
		}
	}
	if expectedCount > 0 {
		mp.mu.Lock()
		defer mp.mu.Unlock()
		subject := transport.LiveTailSubjectPrefix + "auth-service"
		assert.Contains(t, mp.PublishedRecords, subject)
		assert.NotEmpty(t, mp.PublishedRecords[subject])
	}
}

func assertInsertedRecords(t *testing.T, ms *MockLogStore, expected core.FlatLogRecord) {
	t.Helper()
	for _, record := range ms.InsertedRecords {
		assert.Equal(t, expected.ServiceName, record.ServiceName)
		assert.Equal(t, expected.Body, record.Body)
		assert.Equal(t, expected.SeverityText, record.SeverityText)
	}
}

func TestConsumeCallback(t *testing.T) {
	req := newBaseRequest(t)

	reqBytes, err := req.MarshalJSON()
	require.NoError(t, err, "failed to marshal request")

	flatLogsByServiceName := map[string][]core.FlatLogRecord{}
	logs := req.Logs()
	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		flattenLogs(logs.ResourceLogs().At(i), flatLogsByServiceName)
	}
	expectedRecord := flatLogsByServiceName["auth-service"][0]

	tests := []struct {
		name                string
		messages            []transport.Message
		mockDBError         error
		mockProducerError   error
		expectedErr         bool
		expectedInsertCount int
		expectedRecordCount int
		expectedPublishes   int
	}{
		{
			name:     "empty payloads slice",
			messages: []transport.Message{},
		},
		{
			name:                "successful batch insert",
			messages:            makeMessages(t, [][]byte{reqBytes, reqBytes}, nil),
			expectedInsertCount: 1,
			expectedRecordCount: 2,
			expectedPublishes:   1,
		},
		{
			name: "skip malformed json but insert valid ones",
			messages: makeMessages(t, [][]byte{
				reqBytes,
				[]byte(`{malformed payload]`),
				reqBytes,
			}, nil),
			expectedInsertCount: 1,
			expectedRecordCount: 2,
			expectedPublishes:   1,
		},
		{
			name: "all malformed json returns no error and no insert",
			messages: makeMessages(t, [][]byte{
				[]byte(`{malformed payload]`),
				[]byte(`{malformed payload]`),
				[]byte(`{malformed payload]`),
			}, nil),
		},
		{
			name: "database insert failure returns error, live tail still publishes",
			messages: makeMessages(t, [][]byte{
				reqBytes,
			}, nil),
			mockDBError:         fmt.Errorf("test insert error"),
			expectedErr:         true,
			expectedInsertCount: 1,
			expectedRecordCount: 1,
			expectedPublishes:   1,
		},
		{
			name: "database insert sucess, live tail publish failure",
			messages: makeMessages(t, [][]byte{
				reqBytes,
			}, nil),
			mockProducerError:   fmt.Errorf("test publish error"),
			expectedInsertCount: 1,
			expectedRecordCount: 1,
			expectedPublishes:   1,
		},
		{
			name:                "zstd compressed payload is decompressed and consumed correctly",
			messages:            makeMessages(t, [][]byte{zstdCompress(t, reqBytes)}, map[string][]string{"Content-Encoding": {"zstd"}}),
			expectedInsertCount: 1,
			expectedRecordCount: 1,
			expectedPublishes:   1,
		},
		{
			name:     "zstd compressed payload with wrong header is dropped",
			messages: makeMessages(t, [][]byte{zstdCompress(t, reqBytes)}, map[string][]string{"Content-Encoding": {"gzip"}}),
		},
		{
			name:     "zstd compressed payload with no header is dropped",
			messages: makeMessages(t, [][]byte{zstdCompress(t, reqBytes)}, nil),
		},
		{
			name:                "gzip compressed payload is decompressed and consumed correctly",
			messages:            makeMessages(t, [][]byte{gzipCompress(t, reqBytes)}, map[string][]string{"Content-Encoding": {"gzip"}}),
			expectedInsertCount: 1,
			expectedRecordCount: 1,
			expectedPublishes:   1,
		},
		{
			name:     "gzip compressed payload with wrong header is dropped",
			messages: makeMessages(t, [][]byte{gzipCompress(t, reqBytes)}, map[string][]string{"Content-Encoding": {"zstd"}}),
		},
		{
			name:     "gzip compressed payload with no header is dropped",
			messages: makeMessages(t, [][]byte{gzipCompress(t, reqBytes)}, nil),
		},
		{
			name:     "failed decompression returns no error and no insert",
			messages: makeMessages(t, [][]byte{[]byte("invalid")}, map[string][]string{"Content-Encoding": {"zstd"}}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := &MockLogStore{
				InsertErr: tc.mockDBError,
			}

			mockProducer := &MockProducer{
				PublishedRecords: map[string][][]byte{},
				PublishCh:        make(chan struct{}, 10),
				PublishErr:       tc.mockProducerError,
			}

			callback, err := ConsumeCallback(slog.Default(), mockStore, mockProducer)
			require.NoError(t, err)
			err = callback(tc.messages)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assertPublishes(t, mockProducer, tc.expectedPublishes)
			assert.Equal(t, tc.expectedInsertCount, mockStore.InsertCount)
			assert.Len(t, mockStore.InsertedRecords, tc.expectedRecordCount)
			if tc.expectedInsertCount > 0 && !tc.expectedErr {
				assertInsertedRecords(t, mockStore, expectedRecord)
			}

		})
	}
}
