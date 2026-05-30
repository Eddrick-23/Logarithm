package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestExtractAttributes(t *testing.T) {
	tests := []struct {
		name           string
		attributes     []core.KeyValue
		expectedKeys   []string
		expectedValues []string
	}{
		{"slice with full KeyValue pairs",
			[]core.KeyValue{{Key: "key1", Value: "val1"}, {Key: "key2", Value: "val2"}, {Key: "key3", Value: "val3"}},
			[]string{"key1", "key2", "key3"},
			[]string{"val1", "val2", "val3"},
		},
		{"KeyValue pair empty value",
			[]core.KeyValue{{Key: "key1", Value: ""}},
			[]string{"key1"},
			[]string{""},
		},
		{"empty KeyValue slice", []core.KeyValue{}, []string{}, []string{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			keys, values := extractAttributes(tc.attributes)

			assert.Equal(t, tc.expectedKeys, keys)
			assert.Equal(t, tc.expectedValues, values)
		})
	}
}

func newBaseRequest() core.LogIngestRequest {
	mockTime := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	return core.LogIngestRequest{
		ServiceName: "default-service",
		Records: []core.LogRecordDTO{
			{
				Timestamp:      mockTime,
				TraceId:        "default-trace",
				SpanId:         "default-span",
				SeverityText:   "INFO",
				SeverityNumber: 9,
				Body:           "default body",
			},
		},
	}
}

func buildTestRequest(template core.LogIngestRequest, resAttr []core.KeyValue, logAttr []core.KeyValue) core.LogIngestRequest {
	template.ResourceAttributes = resAttr

	for i := range template.Records { // reference by index so we modify the actual underlying slice
		template.Records[i].LogAttributes = logAttr
	}

	return template
}
func TestFlattenLogs(t *testing.T) {
	tests := []struct {
		name                     string
		customResourceAttributes []core.KeyValue
		customLogAttributes      []core.KeyValue
		expectedResKeys          []string
		expectedResValues        []string
		expectedLogKeys          []string
		expectedLogValues        []string
	}{
		{
			"map attributes correctly",
			[]core.KeyValue{{Key: "res-key1", Value: "res-val1"}},
			[]core.KeyValue{{Key: "log-key1", Value: "log-val1"}},
			[]string{"res-key1"},
			[]string{"res-val1"},
			[]string{"log-key1"},
			[]string{"log-val1"},
		},
		{
			"map multiple attributes correctly",
			[]core.KeyValue{{Key: "res-key1", Value: "res-val1"}, {Key: "res-key2", Value: "res-val2"}},
			[]core.KeyValue{{Key: "log-key1", Value: "log-val1"}, {Key: "log-key2", Value: "log-val2"}},
			[]string{"res-key1", "res-key2"},
			[]string{"res-val1", "res-val2"},
			[]string{"log-key1", "log-key2"},
			[]string{"log-val1", "log-val2"},
		},
		{
			"empty attributes",
			[]core.KeyValue{},
			[]core.KeyValue{},
			[]string{},
			[]string{},
			[]string{},
			[]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			completeTemplate := buildTestRequest(newBaseRequest(), tc.customResourceAttributes, tc.customLogAttributes)
			buffer := make([]core.FlatLogRecord, 0)
			buffer = flattenLogs(completeTemplate, buffer)
			expected := []core.FlatLogRecord{
				{Timestamp: completeTemplate.Records[0].Timestamp,
					TraceId:        completeTemplate.Records[0].TraceId,
					SpanId:         completeTemplate.Records[0].SpanId,
					SeverityText:   completeTemplate.Records[0].SeverityText,
					SeverityNumber: completeTemplate.Records[0].SeverityNumber,
					ServiceName:    completeTemplate.ServiceName,
					Body:           completeTemplate.Records[0].Body,
					LogAttrKeys:    tc.expectedLogKeys,
					LogAttrValues:  tc.expectedLogValues,
					ResAttrKeys:    tc.expectedResKeys,
					ResAttrValues:  tc.expectedResValues,
				}}

			assert.Equal(t, expected, buffer)
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
func TestConsumeCallback(t *testing.T) {
	validReq := newBaseRequest()
	validJSON, err := json.Marshal(validReq)
	if err != nil {
		t.Fatalf("failed to marshal valid test request: %v", err)
	}

	expectedRecord := flattenLogs(validReq, []core.FlatLogRecord{})[0]

	tests := []struct {
		name                string
		payloads            [][]byte
		mockDBError         error
		expectedErr         bool
		expectedInsertCount int
	}{
		{
			"empty payloads slice",
			[][]byte{},
			nil,
			false,
			0,
		},
		{
			"successful batch insert",
			[][]byte{
				validJSON,
				validJSON,
			},
			nil,
			false,
			1,
		},
		{
			"skip malformed json but insert valid ones",
			[][]byte{
				validJSON,
				[]byte(`{malformed payload]`),
				validJSON,
			},
			nil,
			false,
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
			false,
			0,
		},
		{
			"database insert failure returns error",
			[][]byte{
				validJSON,
			},
			fmt.Errorf("test insert error"),
			true,
			1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := &MockLogStore{
				InsertErr: tc.mockDBError,
			}
			callback := ConsumeCallback(slog.Default(), mockStore)
			err := callback(tc.payloads)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedInsertCount, mockStore.InsertCount)

			if tc.expectedInsertCount > 0 && !tc.expectedErr {
				expectedCount := 0
				for _, p := range tc.payloads {
					var data core.LogIngestRequest
					if err := json.Unmarshal(p, &data); err != nil {
						continue
					}
					expectedCount++
				}

				assert.Len(t, mockStore.InsertedRecords, expectedCount)

				for _, record := range mockStore.InsertedRecords {
					assert.Equal(t, expectedRecord, record)
				}
			}

		})
	}
}
