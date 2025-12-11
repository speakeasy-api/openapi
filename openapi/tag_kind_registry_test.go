package openapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
)

func TestTagKind_String_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		kind     openapi.TagKind
		expected string
	}{
		{
			name:     "nav tag kind",
			kind:     openapi.TagKindNav,
			expected: "nav",
		},
		{
			name:     "badge tag kind",
			kind:     openapi.TagKindBadge,
			expected: "badge",
		},
		{
			name:     "audience tag kind",
			kind:     openapi.TagKindAudience,
			expected: "audience",
		},
		{
			name:     "custom tag kind",
			kind:     openapi.TagKind("custom"),
			expected: "custom",
		},
		{
			name:     "empty tag kind",
			kind:     openapi.TagKind(""),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.kind.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
