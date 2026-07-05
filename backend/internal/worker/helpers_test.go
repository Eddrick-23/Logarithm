package worker

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/Eddrick-23/Logarithm/internal/transport"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

func zstdCompress(tb testing.TB, data []byte) []byte {
	tb.Helper()
	var buf bytes.Buffer
	w, err := zstd.NewWriter(&buf)
	require.NoError(tb, err)
	_, err = w.Write(data)
	require.NoError(tb, err)
	require.NoError(tb, w.Close())
	return buf.Bytes()
}

func gzipCompress(tb testing.TB, data []byte) []byte {
	tb.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	_, err := gz.Write(data)
	require.NoError(tb, err)
	require.NoError(tb, gz.Close())
	return buf.Bytes()
}

func makeMessages(tb testing.TB, payloads [][]byte, headers map[string][]string) []transport.Message {
	tb.Helper()
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

func newBaseRequest(tb testing.TB) plogotlp.ExportRequest {
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
	require.NoError(tb, err, "invalid inputJSON provided")
	return req
}
