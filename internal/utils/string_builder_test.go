package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnyToString_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{name: "string value", input: "hello", expected: "hello"},
		{name: "empty string", input: "", expected: ""},
		{name: "int value", input: 42, expected: "42"},
		{name: "int zero", input: 0, expected: "0"},
		{name: "int negative", input: -7, expected: "-7"},
		{name: "int64 value", input: int64(9999999999), expected: "9999999999"},
		{name: "int32 value", input: int32(123), expected: "123"},
		{name: "float64 integer", input: float64(3), expected: "3"},
		{name: "float64 decimal", input: 3.14, expected: "3.14"},
		{name: "float64 large", input: 1e18, expected: "1E+18"},
		{name: "float64 small", input: 0.000123, expected: "0.000123"},
		{name: "float32 value", input: float32(2.5), expected: "2.5"},
		{name: "bool true", input: true, expected: "true"},
		{name: "bool false", input: false, expected: "false"},
		{name: "uint64 value", input: uint64(100), expected: "100"},
		{name: "uint value", input: uint(50), expected: "50"},
		{name: "fallback type", input: []int{1, 2}, expected: fmt.Sprintf("%v", []int{1, 2})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, AnyToString(tt.input), "AnyToString should match expected output")
		})
	}
}

// TestAnyToString_MatchesFmtSprintf verifies that AnyToString matches
// fmt.Sprintf("%v") for common types. Note: float scientific notation uses
// uppercase 'E' (from 'G' format) vs fmt.Sprintf's lowercase 'e', so floats
// that produce scientific notation are excluded from this exact-match test.
func TestAnyToString_MatchesFmtSprintf(t *testing.T) {
	t.Parallel()

	values := []any{
		"text", 42, int64(100), int32(10),
		3.14, float64(0.5),
		float32(1.5), true, false,
		uint64(99), uint(7),
	}

	for _, v := range values {
		t.Run(fmt.Sprintf("%T(%v)", v, v), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, fmt.Sprintf("%v", v), AnyToString(v),
				"AnyToString should match fmt.Sprintf for %T", v)
		})
	}
}

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

func TestBuildString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		parts    []string
		expected string
	}{
		{
			name:     "empty parts",
			parts:    []string{},
			expected: "",
		},
		{
			name:     "single part",
			parts:    []string{"hello"},
			expected: "hello",
		},
		{
			name:     "multiple parts",
			parts:    []string{"hello", " ", "world"},
			expected: "hello world",
		},
		{
			name:     "three parts with empty",
			parts:    []string{"a", "", "b"},
			expected: "ab",
		},
		{
			name:     "four parts",
			parts:    []string{"one", "two", "three", "four"},
			expected: "onetwothreefour",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := BuildString(tt.parts...)
			assert.Equal(t, tt.expected, result)
		})
	}
}
