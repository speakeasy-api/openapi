package walk_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/walk"
	"github.com/stretchr/testify/assert"
)

func TestLocations_IsParent_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		locations walk.Locations[string]
		field     string
		expected  bool
	}{
		{
			name:      "empty locations returns false",
			locations: walk.Locations[string]{},
			field:     "anything",
			expected:  false,
		},
		{
			name: "last entry matches field directly",
			locations: walk.Locations[string]{
				{ParentField: "paths"},
				{ParentField: "responses"},
			},
			field:    "responses",
			expected: true,
		},
		{
			name: "last entry does not match field",
			locations: walk.Locations[string]{
				{ParentField: "paths"},
				{ParentField: "responses"},
			},
			field:    "schemas",
			expected: false,
		},
		{
			name: "last entry has parent key so checks second to last",
			locations: walk.Locations[string]{
				{ParentField: "paths"},
				{ParentField: "responses"},
				{ParentKey: pointer.From("200")},
			},
			field:    "responses",
			expected: true,
		},
		{
			name: "last entry has parent index so checks second to last",
			locations: walk.Locations[string]{
				{ParentField: "paths"},
				{ParentField: "tags"},
				{ParentIndex: pointer.From(0)},
			},
			field:    "tags",
			expected: true,
		},
		{
			name: "last entry has parent key but second to last does not match",
			locations: walk.Locations[string]{
				{ParentField: "paths"},
				{ParentField: "responses"},
				{ParentKey: pointer.From("200")},
			},
			field:    "schemas",
			expected: false,
		},
		{
			name: "single entry with parent key returns false (no second to last)",
			locations: walk.Locations[string]{
				{ParentKey: pointer.From("key")},
			},
			field:    "anything",
			expected: false,
		},
		{
			name: "single entry with parent index returns false (no second to last)",
			locations: walk.Locations[string]{
				{ParentIndex: pointer.From(0)},
			},
			field:    "anything",
			expected: false,
		},
		{
			name: "single entry matches field directly",
			locations: walk.Locations[string]{
				{ParentField: "paths"},
			},
			field:    "paths",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.locations.IsParent(tt.field)
			assert.Equal(t, tt.expected, result, "IsParent result should match expected")
		})
	}
}

func TestLocations_ParentKey_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		locations walk.Locations[string]
		expected  string
	}{
		{
			name:      "empty locations returns empty string",
			locations: walk.Locations[string]{},
			expected:  "",
		},
		{
			name: "last entry has parent key",
			locations: walk.Locations[string]{
				{ParentField: "paths"},
				{ParentKey: pointer.From("/users")},
			},
			expected: "/users",
		},
		{
			name: "last entry has no parent key",
			locations: walk.Locations[string]{
				{ParentField: "paths"},
				{ParentField: "responses"},
			},
			expected: "",
		},
		{
			name: "single entry with parent key",
			locations: walk.Locations[string]{
				{ParentKey: pointer.From("myKey")},
			},
			expected: "myKey",
		},
		{
			name: "last entry has parent index but no key",
			locations: walk.Locations[string]{
				{ParentField: "tags"},
				{ParentIndex: pointer.From(3)},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.locations.ParentKey()
			assert.Equal(t, tt.expected, result, "ParentKey result should match expected")
		})
	}
}
