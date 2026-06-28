package worker

import (
	"testing"

	"github.com/Eddrick-23/Logarithm/internal/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecompressor(t *testing.T) {
	tests := []struct {
		name           string
		message        transport.Message
		expectedErr    bool
		expectedResult []byte
	}{
		{
			name:           "decompress zstd",
			message:        makeMessages(t, [][]byte{zstdCompress(t, []byte("data"))}, makeHeaders("", "zstd"))[0],
			expectedResult: []byte("data"),
		},
		{
			name:           "decompress gzip",
			message:        makeMessages(t, [][]byte{gzipCompress(t, []byte("data"))}, makeHeaders("", "gzip"))[0],
			expectedResult: []byte("data"),
		},
		{
			name:        "decompress zstd wrong encoding",
			message:     makeMessages(t, [][]byte{zstdCompress(t, []byte("data"))}, makeHeaders("", "gzip"))[0],
			expectedErr: true,
		},
		{
			name:        "decompress gzip wrong encoding",
			message:     makeMessages(t, [][]byte{gzipCompress(t, []byte("data"))}, makeHeaders("", "zstd"))[0],
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
		{
			name:        "unsupported encoding returns error",
			message:     makeMessages(t, [][]byte{gzipCompress(t, []byte("data"))}, makeHeaders("", "unsupported"))[0],
			expectedErr: true,
		},
	}

	decompressor, err := NewLogDecompressor()
	require.NoError(t, err)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, cleanup, err := decompressor.decompress(tc.message.Payload, tc.message.Headers)
			defer cleanup()

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResult, res)
			}
		})
	}
}
