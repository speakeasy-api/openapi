package specversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{name: "release version", version: "1.1.0", expected: "1.1.0"},
		{name: "patch version", version: "1.0.1", expected: "1.0.1"},
		{name: "prerelease suffix is ignored", version: "1.1.0-rc1", expected: "1.1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v, err := Parse(tt.version)
			require.NoError(t, err, "parse should succeed")
			assert.Equal(t, tt.expected, v.String(), "parsed version should match")
		})
	}
}

func TestParse_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
	}{
		{name: "empty prerelease suffix", version: "1.1.0-"},
		{name: "not a version", version: "one"},
		{name: "empty", version: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Parse(tt.version)
			require.Error(t, err, "parse should fail")
		})
	}
}
