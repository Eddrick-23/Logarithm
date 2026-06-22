package worker

import (
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/transport"
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
			assert.Equal(t, tc.expectedFlattens, transformer.flattenCount)
		})
	}
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

			callback, err := ConsumeCallback(slog.Default(), store, &NoOpDecompressor{}, decoder, &NoOpTransformer{}, &NoOpPublisher{})

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
