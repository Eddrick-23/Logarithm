package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/Eddrick-23/Logarithm/internal/transport"
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

func (m *MockProducer) PublishLogs(ctx context.Context, subject string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.PublishedRecords == nil {
		m.PublishedRecords = map[string][][]byte{}
	}

	m.PublishedRecords[subject] = append(m.PublishedRecords[subject], payload)

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
		payloads            [][]byte
		mockDBError         error
		mockProducerError   error
		expectedErr         bool
		expectedInsertCount int
		expectedRecordCount int
		expectedPublishes   int
	}{
		{
			"empty payloads slice",
			[][]byte{},
			nil,
			nil,
			false,
			0,
			0,
			0,
		},
		{
			"successful batch insert",
			[][]byte{
				reqBytes,
				reqBytes,
			},
			nil,
			nil,
			false,
			1,
			2,
			1,
		},
		{
			"skip malformed json but insert valid ones",
			[][]byte{
				reqBytes,
				[]byte(`{malformed payload]`),
				reqBytes,
			},
			nil,
			nil,
			false,
			1,
			2,
			1,
		},
		{
			"all malformed json returns no error and no insert",
			[][]byte{
				[]byte(`{malformed payload]`),
				[]byte(`{malformed payload]`),
				[]byte(`{malformed payload]`),
			},
			nil,
			nil,
			false,
			0,
			0,
			0,
		},
		{
			"database insert failure returns error, live tail still publishes",
			[][]byte{
				reqBytes,
			},
			fmt.Errorf("test insert error"),
			nil,
			true,
			1,
			1,
			1,
		},
		{
			"database insert sucess, live tail publish failure",
			[][]byte{
				reqBytes,
			},
			nil,
			fmt.Errorf("test publish error"),
			false,
			1,
			1,
			1,
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

			callback := ConsumeCallback(slog.Default(), mockStore, mockProducer)
			err := callback(tc.payloads)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// wait for background go routines to finish publishing
			// check that we receive exact number of publishes
			for i := 0; i < tc.expectedPublishes; i++ {
				select {
				case <-mockProducer.PublishCh:
					// received publish
				case <-time.After(1 * time.Second):
					t.Fatalf("timeout waiting for background nats publish")
				}
			}

			assert.Equal(t, tc.expectedInsertCount, mockStore.InsertCount)
			assert.Len(t, mockStore.InsertedRecords, tc.expectedRecordCount)

			if tc.expectedInsertCount > 0 && !tc.expectedErr {

				for _, record := range mockStore.InsertedRecords {
					assert.Equal(t, expectedRecord.ServiceName, record.ServiceName)
					assert.Equal(t, expectedRecord.Body, record.Body)
					assert.Equal(t, expectedRecord.SeverityText, record.SeverityText)
				}
			}

			if tc.expectedPublishes > 0 {
				mockProducer.mu.Lock()

				subject := transport.LiveTailSubjectPrefix + "auth-service"
				assert.Contains(t, mockProducer.PublishedRecords, subject)

				assert.NotEmpty(t, mockProducer.PublishedRecords[subject])

				mockProducer.mu.Unlock()
			}

		})
	}
}
