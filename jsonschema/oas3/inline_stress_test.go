package oas3_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInline_CombinatorialLongLoop_Timeout_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		contextTimeout   time.Duration
		maxCycles        int64
		expectedErrorMsg string
		description      string
	}{
		{
			name:             "max cycles exceeded set limit",
			contextTimeout:   0, // No context timeout
			maxCycles:        1000000,
			expectedErrorMsg: "exceeded limit",
			description:      "should fail with max cycles timeout for complex combinatorial schema",
		},
		// Commented out as its too slow for a test but here to allow manual testing
		// {
		// 	name:             "max cycles exceeded default",
		// 	contextTimeout:   0, // No context timeout
		// 	maxCycles:        0,
		// 	expectedErrorMsg: "exceeded limit",
		// 	description:      "should fail with max cycles timeout for complex combinatorial schema",
		// },
		{
			name:             "context timeout",
			contextTimeout:   5 * time.Second,
			maxCycles:        10000000000, // High limit so we test time timeout instead
			expectedErrorMsg: "context deadline exceeded",
			description:      "should fail with context timeout for complex combinatorial schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create context based on test case
			var ctx context.Context
			var cancel context.CancelFunc
			if tt.contextTimeout > 0 {
				ctx, cancel = context.WithTimeout(context.Background(), tt.contextTimeout)
				defer cancel()
			} else {
				ctx = context.Background()
			}

			// Load the combinatorial.json file once for all test cases
			combinatorialPath := "testdata/stresstest/combinatorial.json"
			combinatorialFile, err := os.Open(combinatorialPath)
			require.NoError(t, err, "failed to read combinatorial.json")

			// Parse the OpenAPI document
			openAPIDoc, _, err := openapi.Unmarshal(context.Background(), combinatorialFile)
			require.NoError(t, err, "failed to parse combinatorial.json as OpenAPI document")

			// Extract the schema from the post operation at /api/rest/shops
			schemaPointer := "/paths/~1api~1rest~1shops/post/requestBody/content/application~1json/schema/properties/object"

			schema, err := extractSchemaFromOpenAPI(openAPIDoc, schemaPointer)
			require.NoError(t, err, "failed to extract schema from OpenAPI document at %s", schemaPointer)

			// Verify this schema references shops_insert_input!
			require.True(t, schema.IsReference(), "expected schema to be a reference")
			ref := schema.GetRef()
			assert.Contains(t, ref.String(), "shops_insert_input!", "expected reference to contain shops_insert_input!")

			// Create resolve options
			opts := oas3.InlineOptions{
				ResolveOptions: oas3.ResolveOptions{
					TargetLocation: "combinatorial.json",
					RootDocument:   openAPIDoc,
				},
				RemoveUnusedDefs: true,
				MaxCycles:        tt.maxCycles,
			}

			// Try to inline the schema - this should fail with a timeout error due to complexity
			// This prevents infinite loops and provides a proper error instead of hanging
			_, err = oas3.Inline(ctx, schema, opts)
			require.Error(t, err, tt.description)

			// Check that it's the expected timeout error
			assert.True(t, errors.Is(err, oas3.ErrInlineTimeout), "should be timeout error")
			assert.Contains(t, err.Error(), tt.expectedErrorMsg, "should contain expected error message")
		})
	}
}
