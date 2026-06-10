package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestByteSizeDecode(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expected      int64
		expectedError bool
	}{
		{"parse GB", "50GB", 50 * 1024 * 1024 * 1024, false},
		{"parse MB", "10MB", 10 * 1024 * 1024, false},
		{"parse KB", "10KB", 10 * 1024, false},
		{"empty string", "", 0, true},
		{"invalid string", "invalidMB", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var b ByteSize
			err := b.EnvDecode(tc.input)

			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, b, ByteSize(tc.expected))
			}
		})
	}

}
