package overlay_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/overlay"
	"github.com/stretchr/testify/assert"
)

func TestUsesRFC9535(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		version         string
		jsonPathVersion string
		expected        bool
	}{
		// Version 1.0.0 tests - default is legacy, opt-IN to RFC 9535
		{
			name:            "1.0.0 default uses legacy",
			version:         "1.0.0",
			jsonPathVersion: "",
			expected:        false,
		},
		{
			name:            "1.0.0 with rfc9535 opt-in",
			version:         "1.0.0",
			jsonPathVersion: "rfc9535",
			expected:        true,
		},
		{
			name:            "1.0.0 with legacy explicit",
			version:         "1.0.0",
			jsonPathVersion: "legacy",
			expected:        false,
		},
		// Version 1.1.0 tests - default is RFC 9535, opt-OUT to legacy
		{
			name:            "1.1.0 default uses rfc9535",
			version:         "1.1.0",
			jsonPathVersion: "",
			expected:        true,
		},
		{
			name:            "1.1.0 with legacy opt-out",
			version:         "1.1.0",
			jsonPathVersion: "legacy",
			expected:        false,
		},
		{
			name:            "1.1.0 with rfc9535 explicit (redundant but valid)",
			version:         "1.1.0",
			jsonPathVersion: "rfc9535",
			expected:        true,
		},
		// Invalid version tests
		{
			name:            "invalid version uses legacy for safety",
			version:         "invalid",
			jsonPathVersion: "",
			expected:        false,
		},
		{
			name:            "invalid version with rfc9535 still uses rfc9535",
			version:         "invalid",
			jsonPathVersion: "rfc9535",
			expected:        true,
		},
		{
			name:            "empty version uses legacy",
			version:         "",
			jsonPathVersion: "",
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			o := &overlay.Overlay{
				Version:         tt.version,
				JSONPathVersion: tt.jsonPathVersion,
			}

			assert.Equal(t, tt.expected, o.UsesRFC9535(), "UsesRFC9535 mismatch")
		})
	}
}

func TestJSONPathConstants(t *testing.T) {
	t.Parallel()

	// Verify the constants are correctly defined
	assert.Equal(t, "rfc9535", overlay.JSONPathRFC9535, "JSONPathRFC9535 constant")
	assert.Equal(t, "legacy", overlay.JSONPathLegacy, "JSONPathLegacy constant")
}

func TestVersionConstants(t *testing.T) {
	t.Parallel()

	// Verify version constants are correctly defined
	assert.Equal(t, "1.1.0", overlay.LatestVersion, "LatestVersion constant")
	assert.Equal(t, "1.0.0", overlay.Version100, "Version100 constant")
	assert.Equal(t, "1.1.0", overlay.Version110, "Version110 constant")
}
