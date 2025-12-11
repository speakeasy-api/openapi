package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReference_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ref  Reference
	}{
		{
			name: "empty reference",
			ref:  "",
		},
		{
			name: "simple fragment reference",
			ref:  "#/components/schemas/User",
		},
		{
			name: "relative URI with fragment",
			ref:  "schemas.yaml#/User",
		},
		{
			name: "absolute URI with fragment",
			ref:  "https://example.com/api.yaml#/components/schemas/User",
		},
		{
			name: "absolute URI without fragment",
			ref:  "https://example.com/api.yaml",
		},
		{
			name: "relative URI without fragment",
			ref:  "schemas.yaml",
		},
		{
			name: "complex JSON pointer",
			ref:  "#/components/schemas/User/properties/address/properties/street",
		},
		{
			name: "JSON pointer with array index",
			ref:  "#/paths/~1users~1{id}/get/responses/200/content/application~1json/examples/0",
		},
		{
			name: "file URI",
			ref:  "file:///path/to/schema.yaml#/User",
		},
		{
			name: "URI with query parameters",
			ref:  "https://example.com/api.yaml?version=1.0#/components/schemas/User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.ref.Validate()
			require.NoError(t, err, "Expected reference to be valid: %s", tt.ref)
		})
	}
}

func TestReference_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		ref         Reference
		expectError string
	}{
		{
			name:        "invalid URI scheme",
			ref:         "ht tp://example.com/api.yaml#/User",
			expectError: "invalid reference URI",
		},
		{
			name:        "invalid JSON pointer - missing leading slash",
			ref:         "#components/schemas/User",
			expectError: "invalid reference JSON pointer",
		},
		{
			name:        "invalid JSON pointer - unescaped tilde",
			ref:         "#/components/schemas/User~Profile",
			expectError: "invalid reference JSON pointer",
		},
		{
			name:        "invalid JSON pointer - invalid escape sequence",
			ref:         "#/components/schemas/User~2",
			expectError: "invalid reference JSON pointer",
		},
		{
			name:        "malformed URI with invalid characters",
			ref:         "https://example .com/api.yaml#/User",
			expectError: "invalid reference URI",
		},
		{
			name:        "empty component name - schemas",
			ref:         "#/components/schemas/",
			expectError: "component name cannot be empty",
		},
		{
			name:        "empty component name - parameters",
			ref:         "#/components/parameters/",
			expectError: "component name cannot be empty",
		},
		{
			name:        "empty component name - responses",
			ref:         "#/components/responses/",
			expectError: "component name cannot be empty",
		},
		{
			name:        "missing component name - schemas",
			ref:         "#/components/schemas",
			expectError: "component name cannot be empty",
		},
		{
			name:        "component name with space",
			ref:         "#/components/schemas/User Schema",
			expectError: "must match pattern",
		},
		{
			name:        "component name with special characters",
			ref:         "#/components/schemas/User@Schema",
			expectError: "must match pattern",
		},
		{
			name:        "component name starting with slash",
			ref:         "#/components/schemas//UserSchema",
			expectError: "component name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.ref.Validate()
			require.Error(t, err, "Expected reference to be invalid: %s", tt.ref)
			assert.Contains(t, err.Error(), tt.expectError, "Error message should contain expected text")
		})
	}
}

func TestReference_Validate_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("reference with only fragment separator", func(t *testing.T) {
		t.Parallel()
		ref := Reference("#")
		err := ref.Validate()
		// An empty JSON pointer is actually invalid according to the JSON Pointer spec
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid reference JSON pointer")
	})

	t.Run("reference with multiple fragment separators", func(t *testing.T) {
		t.Parallel()

		ref := Reference("https://example.com/api.yaml#/User#invalid")
		err := ref.Validate()
		// This should be valid as we only split on the first #
		require.NoError(t, err, "Reference with multiple # should be valid (only first # is used)")
	})

	t.Run("reference with empty URI and valid pointer", func(t *testing.T) {
		t.Parallel()

		ref := Reference("#/components/schemas/User")
		err := ref.Validate()
		require.NoError(t, err, "Reference with empty URI and valid pointer should be valid")
	})
}

func TestReference_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ref      Reference
		expected string
	}{
		{
			name:     "simple reference",
			ref:      "#/components/schemas/User",
			expected: "#/components/schemas/User",
		},
		{
			name:     "empty reference",
			ref:      "",
			expected: "",
		},
		{
			name:     "complex reference",
			ref:      "https://example.com/api.yaml#/components/schemas/User",
			expected: "https://example.com/api.yaml#/components/schemas/User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := string(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReference_TypeConversion(t *testing.T) {
	t.Parallel()

	t.Run("string to Reference", func(t *testing.T) {
		t.Parallel()
		str := "#/components/schemas/User"
		ref := Reference(str)
		assert.Equal(t, str, string(ref))
	})

	t.Run("Reference to string", func(t *testing.T) {
		t.Parallel()

		ref := Reference("#/components/schemas/User")
		str := string(ref)
		assert.Equal(t, "#/components/schemas/User", str)
	})
}
