package worker

import (
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/plog"
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

func TestProcessMessages(t *testing.T) {
	validReq := newBaseRequest(t)
	validReqBytes, err := validReq.MarshalProto()
	require.NoError(t, err)

	decompressSuccess := DecompressFunc(func(payload []byte, headers map[string][]string) ([]byte, func(), error) {
		return payload, func() {}, nil
	})

	decompressFail := DecompressFunc(func(payload []byte, headers map[string][]string) ([]byte, func(), error) {
		return nil, func() {}, fmt.Errorf("decompress failed")
	})

	decoderSuccess := DecoderFunc(func(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error) {
		return &validReq, nil
	})

	decoderFail := DecoderFunc(func(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error) {
		return nil, fmt.Errorf("decode failed")
	})

	headers := makeHeaders("application/x-protobuf", "")
	tests := []struct {
		name             string
		messages         []transport.Message
		decompressor     Decompressor
		decoder          Decoder
		expectedFlattens int
	}{
		{
			name:             "successful process single message",
			messages:         makeMessages(t, [][]byte{validReqBytes}, headers),
			decompressor:     decompressSuccess,
			decoder:          decoderSuccess,
			expectedFlattens: 1,
		},
		{
			name:             "successful process multiple message",
			messages:         makeMessages(t, [][]byte{validReqBytes, validReqBytes}, headers),
			decompressor:     decompressSuccess,
			decoder:          decoderSuccess,
			expectedFlattens: 2,
		},
		{
			name:         "mixed batch one failure one success",
			messages:     makeMessages(t, [][]byte{validReqBytes, []byte("bad-data")}, headers),
			decompressor: decompressSuccess,
			decoder: DecoderFunc(func(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error) {
				if string(payload) == "bad-data" {
					return nil, fmt.Errorf("decode failed")
				}
				return &validReq, nil
			}),
			expectedFlattens: 1,
		},
		{
			name:             "empty messages",
			messages:         makeMessages(t, [][]byte{}, headers),
			decompressor:     decompressSuccess,
			decoder:          decoderSuccess,
			expectedFlattens: 0,
		},
		{
			name:             "decompress failure",
			messages:         makeMessages(t, [][]byte{validReqBytes}, headers),
			decompressor:     decompressFail,
			decoder:          decoderSuccess,
			expectedFlattens: 0,
		},
		{
			name:             "decode failure",
			messages:         makeMessages(t, [][]byte{validReqBytes}, headers),
			decompressor:     decompressSuccess,
			decoder:          decoderFail,
			expectedFlattens: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			appender := &MockLogAppender{}
			transformer := &MockTransformer{}
			processMessages(slog.Default(), tc.decompressor, tc.decoder, tc.messages,
				transformer, &NoOpPublisher{}, appender)
			assert.Equal(t, tc.expectedFlattens, transformer.GetFlattenCount())
		})
	}
}

func TestNewWorkerPool(t *testing.T) {
	tests := []struct {
		name               string
		config             Config
		expectedRows       int
		expectedNumWorkers int
	}{
		{
			name: "Positive EstimatedRows and NumWorkers",
			config: Config{
				Logger:        slog.Default(),
				EstimatedRows: 10000,
				NumWorkers:    3,
			},
			expectedRows:       10000,
			expectedNumWorkers: 3,
		},
		{
			name: "Zero EstimatedRows and NumWorkers fall back to defaults",
			config: Config{
				Logger:        slog.Default(),
				EstimatedRows: 0,
				NumWorkers:    0,
			},
			expectedRows:       defaultEstimatedRows,
			expectedNumWorkers: defaultNumWorkers,
		},
		{
			name: "nil logger falls back to default logger",
			config: Config{
				EstimatedRows: 10000,
				NumWorkers:    3,
			},
			expectedRows:       10000,
			expectedNumWorkers: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wp := NewWorkerPool(tc.config)
			defer wp.Close()

			assert.Equal(t, tc.expectedRows, wp.estimatedRows)
			assert.Equal(t, tc.expectedNumWorkers, wp.numWorkers)
		})
	}
}

func TestSplitMessages(t *testing.T) {
	tests := []struct {
		name        string
		numMessages int
		numWorkers  int
		expected    []interval
	}{
		{
			name:        "even split",
			numMessages: 6,
			numWorkers:  3,
			expected:    []interval{{0, 2}, {2, 4}, {4, 6}},
		},
		{
			name:        "remainder distributed equally to front regions",
			numMessages: 5,
			numWorkers:  3,
			expected:    []interval{{0, 2}, {2, 4}, {4, 5}},
		},
		{
			name:        "single message singel worker",
			numMessages: 1,
			numWorkers:  1,
			expected:    []interval{{0, 1}},
		},
		{
			name:        "zero message returns nil",
			numMessages: 0,
			numWorkers:  3,
			expected:    nil,
		},
		{
			name:        "zero workers returns nil",
			numMessages: 5,
			numWorkers:  0,
			expected:    nil,
		},
		{
			name:        "single worker takes everything",
			numMessages: 7,
			numWorkers:  1,
			expected:    []interval{{0, 7}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := splitMessages(tc.numMessages, tc.numWorkers)

			assert.Equal(t, tc.expected, result)

			if result != nil {
				// test invariants
				// interval.start <= interval.end
				// should be contiguous [start, end)
				// must cover entire region 0...len(messages)
				lastEnd := 0
				for i, iv := range result {
					assert.Equal(t, lastEnd, iv.start, "regiond %d should start where previous ended", i)
					assert.LessOrEqual(t, iv.start, iv.end)
					lastEnd = iv.end
				}

				assert.Equal(t, tc.numMessages, lastEnd, "region should cover all messages")
			}
		})
	}
}

func TestRunSafely(t *testing.T) {
	t.Run("panic is recovered, does not propagate", func(t *testing.T) {
		assert.NotPanics(t, func() {
			runSafely(slog.Default(), func() {
				panic("panic in job")
			})
		})
	})

	t.Run("normal job executes", func(t *testing.T) {
		ran := false
		runSafely(slog.Default(), func() {
			ran = true
		})

		assert.True(t, ran)
	})
}

func TestConsumeCallback(t *testing.T) {
	validReq := newBaseRequest(t)
	validReqbytes, err := validReq.MarshalProto()
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
			decoder := DecoderFunc(func(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error) {
				return &validReq, nil
			})

			workPool := NewWorkerPool(Config{
				Logger:        slog.Default(),
				Store:         store,
				Decompressor:  &NoOpDecompressor{},
				Decoder:       decoder,
				Transformer:   &NoOpTransformer{},
				Publisher:     &NoOpPublisher{},
				EstimatedRows: 10,
			})

			callback, err := workPool.ConsumeCallback()

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

func TestConsumeCallback_AllMessagesProcessedAcrossRegions(t *testing.T) {
	validReq := newBaseRequest(t)
	validReqBytes, err := validReq.MarshalProto()
	require.NoError(t, err)

	const numMessages = 37 // force imperfect division to test distribution correctness
	payloads := make([][]byte, numMessages)

	for i := range payloads {
		payloads[i] = validReqBytes
	}

	messages := makeMessages(t, payloads, makeHeaders("application/x-protobuf", ""))

	appender := &MockLogAppender{flushErr: nil}
	store := &MockLogStore{Appender: appender}
	decoder := DecoderFunc(func(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error) {
		return &validReq, nil
	})
	transformer := &MockTransformer{}

	wp := NewWorkerPool(Config{
		Logger:        slog.Default(),
		Store:         store,
		Decompressor:  &NoOpDecompressor{},
		Decoder:       decoder,
		Transformer:   transformer,
		Publisher:     &NoOpPublisher{},
		EstimatedRows: numMessages,
		NumWorkers:    3,
	})

	defer wp.Close()

	callback, err := wp.ConsumeCallback()
	require.NoError(t, err)

	require.NoError(t, callback(messages))
	assert.Equal(t, numMessages, transformer.GetFlattenCount(),
		"every message should be flattended exactly once, regardless of splitting")
}

// custom mock that controls panics using a boolean field
// Used only here
type PanicTransformer struct {
	panicDuringFlatten bool
}

func (p *PanicTransformer) Flatten(resourceLogs plog.ResourceLogs, publisher Publisher, appender storage.LogAppender) {
	if p.panicDuringFlatten {
		panic("panic during flatten")
	}
}

func TestConsumeCallback_SurvivesTransformerPanic(t *testing.T) {
	validReq := newBaseRequest(t)
	validReqBytes, err := validReq.MarshalProto()
	require.NoError(t, err)

	const numMessages = 10
	payloads := make([][]byte, numMessages)
	for i := range payloads {
		payloads[i] = validReqBytes
	}
	messages := makeMessages(t, payloads, makeHeaders("application/x-protobuf", ""))

	appender := &MockLogAppender{}
	store := &MockLogStore{Appender: appender}
	decoder := DecoderFunc(func(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error) {
		return &validReq, nil
	})

	panicTransformer := &PanicTransformer{panicDuringFlatten: true}

	wp := NewWorkerPool(Config{
		Logger:        slog.Default(),
		Store:         store,
		Decompressor:  &NoOpDecompressor{},
		Decoder:       decoder,
		Transformer:   panicTransformer,
		Publisher:     &NoOpPublisher{},
		EstimatedRows: numMessages,
		NumWorkers:    1,
	})
	defer wp.Close()

	callback, err := wp.ConsumeCallback()
	require.NoError(t, err)

	// Test that a panic does not crash the callback
	done := make(chan error, 1)
	go func() { done <- callback(messages) }()

	select {
	case <-done:
		// callback returned, internal wg.Done() fired correctly even though the job panicked
	case <-time.After(2 * time.Second):
		t.Fatal("ConsumeCallback did not return in time, likely issue with panic recovery")
	}

	// Test that worker pool still available despite a job panicking
	panicTransformer.panicDuringFlatten = false
	go func() { done <- callback(messages) }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ConsumeCallback did not return in time, go routine might have closed due to previous panic")
	}
}
