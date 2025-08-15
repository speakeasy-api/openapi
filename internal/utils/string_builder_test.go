package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildAbsoluteReference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		absRef   string
		jsonPtr  string
		expected string
	}{
		{
			name:     "empty json pointer",
			absRef:   "https://example.com/schema.json",
			jsonPtr:  "",
			expected: "https://example.com/schema.json",
		},
		{
			name:     "with json pointer",
			absRef:   "https://example.com/schema.json",
			jsonPtr:  "/definitions/User",
			expected: "https://example.com/schema.json#/definitions/User",
		},
		{
			name:     "file path with json pointer",
			absRef:   "/path/to/schema.json",
			jsonPtr:  "/properties/name",
			expected: "/path/to/schema.json#/properties/name",
		},
		{
			name:     "already has fragment",
			absRef:   "https://example.com/schema.json#existing",
			jsonPtr:  "/definitions/User",
			expected: "https://example.com/schema.json#existing#/definitions/User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := BuildAbsoluteReference(tt.absRef, tt.jsonPtr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJoinWithSeparator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		separator string
		parts     []string
		expected  string
	}{
		{
			name:      "empty parts",
			separator: " -> ",
			parts:     []string{},
			expected:  "",
		},
		{
			name:      "single part",
			separator: " -> ",
			parts:     []string{"first"},
			expected:  "first",
		},
		{
			name:      "multiple parts",
			separator: " -> ",
			parts:     []string{"first", "second", "third"},
			expected:  "first -> second -> third",
		},
		{
			name:      "comma separator",
			separator: ", ",
			parts:     []string{"a", "b", "c"},
			expected:  "a, b, c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := JoinWithSeparator(tt.separator, tt.parts...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkBuildAbsoluteReference(b *testing.B) {
	absRef := "https://example.com/very/long/path/to/schema.json"
	jsonPtr := "/definitions/User/properties/name"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildAbsoluteReference(absRef, jsonPtr)
	}
}

func BenchmarkBuildAbsoluteReferenceEmpty(b *testing.B) {
	absRef := "https://example.com/very/long/path/to/schema.json"
	jsonPtr := ""

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildAbsoluteReference(absRef, jsonPtr)
	}
}

func BenchmarkStringConcatenation(b *testing.B) {
	absRef := "https://example.com/very/long/path/to/schema.json"
	jsonPtr := "/definitions/User/properties/name"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result string
		if jsonPtr != "" {
			result = absRef + "#" + jsonPtr
		} else {
			result = absRef
		}
		_ = result
	}
}
