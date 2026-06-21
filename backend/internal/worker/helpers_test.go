package worker

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/Eddrick-23/Logarithm/internal/transport"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
)

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
