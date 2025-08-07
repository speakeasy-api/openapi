package jsonpointer

import (
	"testing"

	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test for the bug fix where embedded sequencedmap navigation was duplicating navigation parts
func TestNavigateModel_EmbeddedMapComplexPath(t *testing.T) {
	t.Parallel()

	// Create a nested structure that mimics OpenAPI's paths structure
	// This tests the specific bug where embedded sequencedmap navigation
	// was incorrectly appending currentPart to the navigation stack

	// Create inner sequenced map (like operations in a PathItem)
	operations := sequencedmap.New[string, string]()
	operations.Set("get", "GET operation")
	operations.Set("post", "POST operation")

	// Create outer sequenced map (like paths in OpenAPI)
	paths := sequencedmap.New[string, *sequencedmap.Map[string, string]]()
	paths.Set("/users/{userId}", operations)
	paths.Set("/users", operations)

	// Test complex JSON pointer that should navigate through both levels
	tests := []struct {
		name        string
		pointer     JSONPointer
		expected    string
		expectError bool
	}{
		{
			name:     "escaped path to nested operation",
			pointer:  "/~1users~1{userId}/get",
			expected: "GET operation",
		},
		{
			name:     "escaped path to different operation",
			pointer:  "/~1users~1{userId}/post",
			expected: "POST operation",
		},
		{
			name:     "escaped path with simple key",
			pointer:  "/~1users/get",
			expected: "GET operation",
		},
		{
			name:        "invalid operation",
			pointer:     "/~1users~1{userId}/delete",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := GetTarget(paths, tt.pointer)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test that verifies the navigation stack is not corrupted during embedded map navigation
func TestNavigateModel_NavigationStackIntegrity(t *testing.T) {
	t.Parallel()

	// Create a deep nested structure to test stack management
	level3 := sequencedmap.New[string, string]()
	level3.Set("param1", "parameter 1")
	level3.Set("param2", "parameter 2")

	level2 := sequencedmap.New[string, *sequencedmap.Map[string, string]]()
	level2.Set("parameters", level3)
	level2.Set("responses", level3) // reuse for simplicity

	level1 := sequencedmap.New[string, *sequencedmap.Map[string, *sequencedmap.Map[string, string]]]()
	level1.Set("get", level2)
	level1.Set("post", level2)

	root := sequencedmap.New[string, *sequencedmap.Map[string, *sequencedmap.Map[string, *sequencedmap.Map[string, string]]]]()
	root.Set("/users/{userId}", level1)

	// Test deep navigation that would have failed with the bug
	pointer := JSONPointer("/~1users~1{userId}/get/parameters/param1")
	result, err := GetTarget(root, pointer)

	require.NoError(t, err)
	assert.Equal(t, "parameter 1", result)
}
