package sequencedmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HTTPMethod mirrors the type from openapi package
type HTTPMethod string

const (
	GET    HTTPMethod = "get"
	POST   HTTPMethod = "post"
	PUT    HTTPMethod = "put"
	DELETE HTTPMethod = "delete"
)

func TestNavigateWithKey_HTTPMethodConversion_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		setupKey HTTPMethod
		getKey   string
		expected string
	}{
		{
			name:     "HTTPMethod setup, string get",
			setupKey: GET,
			getKey:   "get",
			expected: "get_value",
		},
		{
			name:     "POST method",
			setupKey: POST,
			getKey:   "post",
			expected: "post_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := New[HTTPMethod, string]()
			m.Set(tt.setupKey, tt.expected)

			value, err := m.NavigateWithKey(tt.getKey)
			require.NoError(t, err, "NavigateWithKey should not fail")
			assert.Equal(t, tt.expected, value, "should return correct value")
		})
	}
}

func TestNavigateWithKey_InvalidKeyType_Error(t *testing.T) {
	t.Parallel()
	// Test with map that has non-string key type
	m := New[int, string]()
	m.Set(42, "value")

	_, err := m.NavigateWithKey("42")
	assert.Error(t, err, "should fail with non-string key type")
	assert.Contains(t, err.Error(), "key type must be string", "should contain appropriate error message")
}
