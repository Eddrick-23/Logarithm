package worker

import (
	"log/slog"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

func createTestLog(t *testing.T) plog.ResourceLogs {
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
	require.NoError(t, err, "invalid test JSON provided")

	return req.Logs().ResourceLogs().At(0)
}

func TestFlattenRouting(t *testing.T) {
	tests := []struct {
		name            string
		enqueueSuccess  bool
		expectedAppends int
		expectedEnqueue int
	}{
		{
			name:            "successful enqueue and append",
			enqueueSuccess:  true,
			expectedAppends: 1,
			expectedEnqueue: 1,
		},
		{
			name:            "enqueue fail does not block append",
			enqueueSuccess:  false,
			expectedAppends: 1,
			expectedEnqueue: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			publisher := &MockPublisher{enqueueSuccess: tc.enqueueSuccess}
			appender := &MockLogAppender{}

			logTransformer := NewLogTransformer(slog.Default())

			logTransformer.Flatten(createTestLog(t), publisher, appender)

			assert.Equal(t, tc.expectedEnqueue, publisher.GetEnqueueCount())
			assert.Equal(t, tc.expectedAppends, appender.GetAppendCount())
		})
	}
}

func TestFlattenLogic(t *testing.T) {
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
				assert.Equal(t, expectedAppends, mockAppender.GetAppendCount())
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
				assert.Equal(t, expectedAppends, mockAppender.GetAppendCount())
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
				assert.Equal(t, expectedAppends, mockAppender.GetAppendCount())
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
				assert.Equal(t, expectedAppends, mockAppender.GetAppendCount())
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
				assert.Equal(t, expectedAppends, mockAppender.GetAppendCount())
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
				assert.Equal(t, expectedAppends, mockAppender.GetAppendCount())
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

			logs := req.Logs()

			mockAppender := &MockLogAppender{}
			mockPublisher := &MockPublisher{enqueueSuccess: true}

			logTransformer := NewLogTransformer(slog.Default())
			for i := 0; i < logs.ResourceLogs().Len(); i++ {
				logTransformer.Flatten(logs.ResourceLogs().At(i), mockPublisher, mockAppender)
			}

			actualLogs := mockPublisher.records

			assert.Equal(t, tc.expectedLength, len(actualLogs), "slice length mismatch")

			if tc.expectedLength > 0 {
				tc.check(t, tc.expectedLength, tc.expectedAppends, actualLogs, mockAppender)
			}
		})
	}
}
