package ingester

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// create helper for uncompressed, gzip and zstd compressed bytes.
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

func TestDetectCompression(t *testing.T) {
	tests := []struct {
		name           string
		payload        []byte
		expectedResult string
	}{
		{
			name:           "zstd payload",
			payload:        zstdCompress(t, []byte("test")),
			expectedResult: "zstd",
		},
		{
			name:           "gzip payload",
			payload:        gzipCompress(t, []byte("test")),
			expectedResult: "gzip",
		},
		{
			name:           "no compression",
			payload:        []byte("test"),
			expectedResult: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := detectCompression(tc.payload)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}
