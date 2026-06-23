package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecode(t *testing.T) {
	req := newBaseRequest(t)
	jsonPayload, err := req.MarshalJSON()
	require.NoError(t, err)

	protobufPayload, err := req.MarshalProto()
	require.NoError(t, err)

	emptyHeaders := map[string][]string{}
	jsonHeaders := map[string][]string{"Content-Type": {"application/json"}}
	protobufHeaders := map[string][]string{"Content-Type": {"application/x-protobuf"}}

	tests := []struct {
		name        string
		payload     []byte
		headers     map[string][]string
		expectedErr bool
	}{
		{
			name:        "valid json payload with correct headers",
			payload:     jsonPayload,
			headers:     jsonHeaders,
			expectedErr: false,
		},
		{
			name:        "valid protobuf payload with correct headers",
			payload:     protobufPayload,
			headers:     protobufHeaders,
			expectedErr: false,
		},
		{
			name:        "invalid json correct headers",
			payload:     []byte("not json"),
			headers:     jsonHeaders,
			expectedErr: true,
		},
		{
			name:        "invalid protobuf correct headers",
			payload:     []byte("not protobuf"),
			headers:     protobufHeaders,
			expectedErr: true,
		},
		{
			name:        "valid json wrong headers",
			payload:     jsonPayload,
			headers:     protobufHeaders,
			expectedErr: true,
		},
		{
			name:        "valid protobuf wrong headers",
			payload:     protobufPayload,
			headers:     jsonHeaders,
			expectedErr: true,
		},
		{
			name:        "valid json no headers",
			payload:     jsonPayload,
			headers:     emptyHeaders,
			expectedErr: true,
		},
		{
			name:        "valid protobuf no headers",
			payload:     protobufPayload,
			headers:     emptyHeaders,
			expectedErr: true,
		},
	}

	decoder := NewLogDecoder()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decodedReq, err := decoder.decode(tc.payload, tc.headers)

			if tc.expectedErr {
				require.Error(t, err)
				require.Nil(t, decodedReq)
			} else {
				require.NoError(t, err)
				require.NotNil(t, decodedReq)

				assert.Equal(
					t,
					1,
					decodedReq.Logs().ResourceLogs().Len(),
					"expected 1 resource log in decoded payload",
				)
			}
		})
	}
}
