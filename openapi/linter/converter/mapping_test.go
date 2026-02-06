package converter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookupNativeRule_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		spectralName string
		expectedID   string
	}{
		{
			name:         "operation-tags maps to style-operation-tags",
			spectralName: "operation-tags",
			expectedID:   "style-operation-tags",
		},
		{
			name:         "operation-operationId maps to semantic-operation-operation-id",
			spectralName: "operation-operationId",
			expectedID:   "semantic-operation-operation-id",
		},
		{
			name:         "path-params maps to semantic-path-params",
			spectralName: "path-params",
			expectedID:   "semantic-path-params",
		},
		{
			name:         "oas3-server-not-example.com maps to style-oas3-host-not-example",
			spectralName: "oas3-server-not-example.com",
			expectedID:   "style-oas3-host-not-example",
		},
		{
			name:         "no-$ref-siblings variant maps to style-no-ref-siblings",
			spectralName: "no-$ref-siblings",
			expectedID:   "style-no-ref-siblings",
		},
		{
			name:         "no-ref-siblings variant maps to style-no-ref-siblings",
			spectralName: "no-ref-siblings",
			expectedID:   "style-no-ref-siblings",
		},
		{
			name:         "oas3-unused-component maps to semantic-unused-component",
			spectralName: "oas3-unused-component",
			expectedID:   "semantic-unused-component",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			nativeID, ok := LookupNativeRule(tt.spectralName)
			assert.True(t, ok, "should find native rule")
			assert.Equal(t, tt.expectedID, nativeID, "native rule ID")
		})
	}
}

func TestLookupNativeRule_NotFound(t *testing.T) {
	t.Parallel()

	nativeID, ok := LookupNativeRule("some-unknown-rule")
	assert.False(t, ok, "should not find native rule")
	assert.Empty(t, nativeID, "native rule ID should be empty")
}

func TestMapSeverityToNative(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "error stays error", input: "error", expected: "error"},
		{name: "warn becomes warning", input: "warn", expected: "warning"},
		{name: "warning stays warning", input: "warning", expected: "warning"},
		{name: "info becomes hint", input: "info", expected: "hint"},
		{name: "hint stays hint", input: "hint", expected: "hint"},
		{name: "unknown defaults to warning", input: "something", expected: "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, mapSeverityToNative(tt.input), "mapped severity")
		})
	}
}
