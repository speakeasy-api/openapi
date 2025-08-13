package openapi

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReference_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		testFunc func(t *testing.T, ref *ReferencedExample)
	}{
		{
			name: "reference with $ref only",
			yaml: `$ref: '#/components/examples/UserExample'`,
			testFunc: func(t *testing.T, ref *ReferencedExample) {
				t.Helper()
				assert.Equal(t, "#/components/examples/UserExample", string(ref.GetReference()))
				assert.Empty(t, ref.GetSummary())
				assert.Empty(t, ref.GetDescription())
				assert.True(t, ref.IsReference())
				assert.Nil(t, ref.Object)
			},
		},
		{
			name: "reference with $ref, summary, and description",
			yaml: `
$ref: '#/components/examples/UserExample'
summary: User example reference
description: A reference to the user example with additional context
`,
			testFunc: func(t *testing.T, ref *ReferencedExample) {
				t.Helper()
				assert.Equal(t, "#/components/examples/UserExample", string(ref.GetReference()))
				assert.Equal(t, "User example reference", ref.GetSummary())
				assert.Equal(t, "A reference to the user example with additional context", ref.GetDescription())
				assert.True(t, ref.IsReference())
				assert.Nil(t, ref.Object)
			},
		},
		{
			name: "inline object without reference",
			yaml: `
summary: Inline user example
description: An inline example of a user object
value:
  id: 123
  name: John Doe
  email: john@example.com
`,
			testFunc: func(t *testing.T, ref *ReferencedExample) {
				t.Helper()
				assert.Empty(t, string(ref.GetReference()))
				assert.Empty(t, ref.GetSummary()) // Summary/Description are on the object, not the reference
				assert.Empty(t, ref.GetDescription())
				assert.False(t, ref.IsReference())
				assert.NotNil(t, ref.Object)
				assert.Equal(t, "Inline user example", ref.Object.GetSummary())
				assert.Equal(t, "An inline example of a user object", ref.Object.GetDescription())
			},
		},
		{
			name: "empty reference",
			yaml: `{}`,
			testFunc: func(t *testing.T, ref *ReferencedExample) {
				t.Helper()
				assert.Empty(t, string(ref.GetReference()))
				assert.Empty(t, ref.GetSummary())
				assert.Empty(t, ref.GetDescription())
				assert.False(t, ref.IsReference())
				// Empty reference creates an empty object, not nil
				assert.NotNil(t, ref.Object)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ref ReferencedExample
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yaml), &ref)
			require.NoError(t, err)
			assert.Empty(t, validationErrs)

			tt.testFunc(t, &ref)
		})
	}
}

func TestReference_Unmarshal_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		yaml        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid YAML syntax",
			yaml:        `$ref: '#/components/examples/UserExample'\ninvalid: [`,
			expectError: true,
			errorMsg:    "mapping values are not allowed in this context",
		},
		{
			name:        "non-mapping node",
			yaml:        `- item1\n- item2`,
			expectError: false, // Should be validation error, not unmarshal error
		},
		{
			name:        "scalar value",
			yaml:        `"just a string"`,
			expectError: false, // Should be validation error, not unmarshal error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ref ReferencedExample
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yaml), &ref)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				// For non-mapping nodes, we should get validation errors
				if tt.yaml != `{}` {
					assert.NotEmpty(t, validationErrs)
				}
			}
		})
	}
}

func TestReference_GetterMethods_NilSafety(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ref      *ReferencedExample
		testFunc func(t *testing.T, ref *ReferencedExample)
	}{
		{
			name: "nil reference",
			ref:  nil,
			testFunc: func(t *testing.T, ref *ReferencedExample) {
				t.Helper()
				assert.Empty(t, string(ref.GetReference()))
				assert.Empty(t, ref.GetSummary())
				assert.Empty(t, ref.GetDescription())
				assert.False(t, ref.IsReference())
			},
		},
		{
			name: "reference with nil fields",
			ref:  &ReferencedExample{},
			testFunc: func(t *testing.T, ref *ReferencedExample) {
				t.Helper()
				assert.Empty(t, string(ref.GetReference()))
				assert.Empty(t, ref.GetSummary())
				assert.Empty(t, ref.GetDescription())
				assert.False(t, ref.IsReference())
			},
		},
		{
			name: "reference with populated fields",
			ref: &ReferencedExample{
				Reference:   pointer.From(references.Reference("#/components/examples/UserExample")),
				Summary:     pointer.From("Test summary"),
				Description: pointer.From("Test description"),
			},
			testFunc: func(t *testing.T, ref *ReferencedExample) {
				t.Helper()
				assert.Equal(t, "#/components/examples/UserExample", string(ref.GetReference()))
				assert.Equal(t, "Test summary", ref.GetSummary())
				assert.Equal(t, "Test description", ref.GetDescription())
				assert.True(t, ref.IsReference())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.testFunc(t, tt.ref)
		})
	}
}

func TestReference_DifferentTypes(t *testing.T) {
	t.Parallel()

	t.Run("ReferencedParameter", func(t *testing.T) {
		t.Parallel()

		yaml := `
$ref: '#/components/parameters/UserIdParam'
summary: User ID parameter reference
description: Reference to the user ID parameter
`
		var ref ReferencedParameter
		validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yaml), &ref)
		require.NoError(t, err)
		assert.Empty(t, validationErrs)

		assert.Equal(t, "#/components/parameters/UserIdParam", string(ref.GetReference()))
		assert.Equal(t, "User ID parameter reference", ref.GetSummary())
		assert.Equal(t, "Reference to the user ID parameter", ref.GetDescription())
		assert.True(t, ref.IsReference())
	})

	t.Run("ReferencedResponse", func(t *testing.T) {
		t.Parallel()

		yaml := `
$ref: '#/components/responses/NotFound'
summary: Not found response reference
description: Reference to the standard not found response
`
		var ref ReferencedResponse
		validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yaml), &ref)
		require.NoError(t, err)
		assert.Empty(t, validationErrs)

		assert.Equal(t, "#/components/responses/NotFound", string(ref.GetReference()))
		assert.Equal(t, "Not found response reference", ref.GetSummary())
		assert.Equal(t, "Reference to the standard not found response", ref.GetDescription())
		assert.True(t, ref.IsReference())
	})

	t.Run("ReferencedRequestBody", func(t *testing.T) {
		t.Parallel()

		yaml := `
$ref: '#/components/requestBodies/UserBody'
summary: User request body reference
description: Reference to the user request body schema
`
		var ref ReferencedRequestBody
		validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yaml), &ref)
		require.NoError(t, err)
		assert.Empty(t, validationErrs)

		assert.Equal(t, "#/components/requestBodies/UserBody", string(ref.GetReference()))
		assert.Equal(t, "User request body reference", ref.GetSummary())
		assert.Equal(t, "Reference to the user request body schema", ref.GetDescription())
		assert.True(t, ref.IsReference())
	})
}
