package oas3_test

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInline_ContextTimeout_Error(t *testing.T) {
	t.Parallel()

	// Create a schema with a simple reference
	schema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
		Type: oas3.NewTypeFromString("string"),
	})

	// Create a context that is already cancelled to ensure deterministic behavior
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately to ensure context is cancelled before Inline is called

	opts := oas3.InlineOptions{
		ResolveOptions: oas3.ResolveOptions{
			TargetLocation: "test.json",
			RootDocument:   schema,
		},
	}

	// Try to inline - should fail with timeout error
	_, err := oas3.Inline(ctx, schema, opts)
	require.Error(t, err, "should fail with timeout error")

	// Check that it's the expected timeout error
	require.ErrorIs(t, err, oas3.ErrInlineTimeout, "should be timeout error")
	assert.Contains(t, err.Error(), "inline operation timed out", "should contain timeout message")
}

func TestInline_MaxCycles_Error(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Create a schema with a simple reference
	schema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
		Type: oas3.NewTypeFromString("string"),
	})

	opts := oas3.InlineOptions{
		ResolveOptions: oas3.ResolveOptions{
			TargetLocation: "test.json",
			RootDocument:   schema,
		},
		MaxCycles: 1, // Very low limit to trigger the error quickly
	}

	// Try to inline - should fail with max cycles error
	_, err := oas3.Inline(ctx, schema, opts)
	require.Error(t, err, "should fail with max cycles error")

	// Check that it's the expected timeout error (cycles are reported as timeout)
	require.ErrorIs(t, err, oas3.ErrInlineTimeout, "should be timeout error")
	assert.Contains(t, err.Error(), "exceeded limit", "should contain exceeded limit message")
}

func TestInline_DefaultMaxCycles_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Create a simple schema without references
	schema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
		Type: oas3.NewTypeFromString("string"),
	})

	opts := oas3.InlineOptions{
		ResolveOptions: oas3.ResolveOptions{
			TargetLocation: "test.json",
			RootDocument:   schema,
		},
		// MaxCycles not set, should use default of 500000
	}

	// Should succeed with default max cycles
	result, err := oas3.Inline(ctx, schema, opts)
	require.NoError(t, err, "should succeed with default max cycles")
	require.NotNil(t, result, "result should not be nil")
}

func TestInline_CustomMaxCycles_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Create a simple schema without references
	schema := oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
		Type: oas3.NewTypeFromString("string"),
	})

	opts := oas3.InlineOptions{
		ResolveOptions: oas3.ResolveOptions{
			TargetLocation: "test.json",
			RootDocument:   schema,
		},
		MaxCycles: 1000, // Custom limit
	}

	// Should succeed with custom max cycles
	result, err := oas3.Inline(ctx, schema, opts)
	require.NoError(t, err, "should succeed with custom max cycles")
	require.NotNil(t, result, "result should not be nil")
}

func TestInline_NilSchema_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	opts := oas3.InlineOptions{
		MaxCycles: 100,
	}

	// Should handle nil schema gracefully
	result, err := oas3.Inline(ctx, nil, opts)
	require.NoError(t, err, "should handle nil schema gracefully")
	assert.Nil(t, result, "result should be nil for nil input")
}
