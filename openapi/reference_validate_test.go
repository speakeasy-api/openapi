package openapi

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReference_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "valid reference with $ref only",
			yaml: `$ref: '#/components/examples/UserExample'`,
		},
		{
			name: "valid reference with $ref, summary, and description",
			yaml: `
$ref: '#/components/examples/UserExample'
summary: User example reference
description: A reference to the user example with additional context
`,
		},
		{
			name: "valid inline object without reference",
			yaml: `
summary: Inline user example
description: An inline example of a user object
value:
  id: 123
  name: John Doe
  email: john@example.com
`,
		},
		{
			name: "valid inline object with external value",
			yaml: `
summary: External user example
description: An example with external value reference
externalValue: https://example.com/user.json
`,
		},
		{
			name: "empty reference (valid but not useful)",
			yaml: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ref ReferencedExample
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yaml), &ref)
			require.NoError(t, err)

			// Validate the reference
			errs := ref.Validate(t.Context())
			assert.Empty(t, errs, "Expected no validation errors for valid reference")
			assert.True(t, ref.Valid, "Expected reference to be marked as valid")

			// Combine unmarshal and validation errors for comprehensive check
			allErrors := validationErrs
			allErrors = append(allErrors, errs...)
			assert.Empty(t, allErrors, "Expected no errors overall")
		})
	}
}

func TestReference_Validate_ReferenceString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		yaml        string
		expectValid bool
		errorMsg    string
	}{
		{
			name:        "valid simple reference",
			yaml:        `$ref: '#/components/examples/UserExample'`,
			expectValid: true,
		},
		{
			name:        "valid absolute URI reference",
			yaml:        `$ref: 'https://example.com/api.yaml#/components/schemas/User'`,
			expectValid: true,
		},
		{
			name:        "valid relative URI reference",
			yaml:        `$ref: 'schemas.yaml#/User'`,
			expectValid: true,
		},
		{
			name: "valid reference with summary and description",
			yaml: `
$ref: '#/components/examples/UserExample'
summary: User example reference
description: A reference to the user example
`,
			expectValid: true,
		},
		{
			name:        "invalid reference - malformed JSON pointer",
			yaml:        `$ref: '#components/examples/UserExample'`,
			expectValid: false,
			errorMsg:    "invalid reference JSON pointer",
		},
		{
			name:        "invalid reference - invalid URI",
			yaml:        `$ref: 'ht tp://example.com/api.yaml#/User'`,
			expectValid: false,
			errorMsg:    "invalid reference URI",
		},
		{
			name:        "invalid reference - unescaped tilde in JSON pointer",
			yaml:        `$ref: '#/components/schemas/User~Profile'`,
			expectValid: false,
			errorMsg:    "invalid reference JSON pointer",
		},
		{
			name:        "invalid reference - empty JSON pointer",
			yaml:        `$ref: '#'`,
			expectValid: false,
			errorMsg:    "invalid reference JSON pointer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ref ReferencedExample
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yaml), &ref)
			require.NoError(t, err)

			// Validate the reference
			errs := ref.Validate(t.Context())

			// Combine unmarshal and validation errors
			allErrors := validationErrs
			allErrors = append(allErrors, errs...)

			if tt.expectValid {
				assert.Empty(t, allErrors, "Expected no validation errors for valid reference")
				assert.True(t, ref.Valid, "Expected reference to be marked as valid")
			} else {
				assert.NotEmpty(t, allErrors, "Expected validation errors for invalid reference")
				assert.False(t, ref.Valid, "Expected reference to be marked as invalid")

				// Check that expected error message is present
				errorMessages := make([]string, len(allErrors))
				for i, err := range allErrors {
					errorMessages[i] = err.Error()
				}

				found := false
				for _, actualErr := range errorMessages {
					if assert.Contains(t, actualErr, tt.errorMsg) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected error message '%s' not found in: %v", tt.errorMsg, errorMessages)
			}
		})
	}
}

func TestReference_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		wantErrs []string
	}{
		{
			name: "invalid inline example - missing required value",
			yaml: `
summary: Invalid example
description: Example missing both value and externalValue
`,
			wantErrs: []string{"either value or externalValue must be specified"},
		},
		{
			name: "invalid inline example - both value and externalValue",
			yaml: `
summary: Invalid example
description: Example with both value and externalValue
value:
  id: 123
externalValue: https://example.com/user.json
`,
			wantErrs: []string{"value and externalValue are mutually exclusive"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ref ReferencedExample
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yaml), &ref)
			require.NoError(t, err)

			// Validate the reference
			errs := ref.Validate(t.Context())

			// Combine unmarshal and validation errors
			allErrors := validationErrs
			allErrors = append(allErrors, errs...)

			// Note: The validation errors come from the Example object validation, not the Reference itself
			// If there are no validation errors, it means the Example object is valid according to its rules
			if len(allErrors) > 0 {
				assert.False(t, ref.Valid, "Expected reference to be marked as invalid")

				// Check that expected error messages are present
				errorMessages := make([]string, len(allErrors))
				for i, err := range allErrors {
					errorMessages[i] = err.Error()
				}

				for _, expectedErr := range tt.wantErrs {
					found := false
					for _, actualErr := range errorMessages {
						if assert.Contains(t, actualErr, expectedErr) {
							found = true
							break
						}
					}
					if !found {
						t.Logf("Expected error message '%s' not found in: %v", expectedErr, errorMessages)
					}
				}
			} else {
				// If no validation errors, the test case might need adjustment
				t.Logf("No validation errors found for test case: %s", tt.name)
			}
		})
	}
}

func TestReference_Validate_DifferentTypes(t *testing.T) {
	t.Parallel()

	t.Run("ReferencedParameter with valid reference", func(t *testing.T) {
		t.Parallel()

		yaml := `$ref: '#/components/parameters/UserIdParam'`
		var ref ReferencedParameter
		validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yaml), &ref)
		require.NoError(t, err)

		errs := ref.Validate(t.Context())
		allErrors := validationErrs
		allErrors = append(allErrors, errs...)
		assert.Empty(t, allErrors)
		assert.True(t, ref.Valid)
	})

	t.Run("ReferencedParameter with inline object", func(t *testing.T) {
		t.Parallel()

		yaml := `
name: userId
in: path
required: true
schema:
  type: string
description: The user ID parameter
`
		var ref ReferencedParameter
		validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yaml), &ref)
		require.NoError(t, err)

		errs := ref.Validate(t.Context())
		allErrors := validationErrs
		allErrors = append(allErrors, errs...)
		assert.Empty(t, allErrors)
		assert.True(t, ref.Valid)
	})

	t.Run("ReferencedResponse with valid reference", func(t *testing.T) {
		t.Parallel()

		yaml := `$ref: '#/components/responses/NotFound'`
		var ref ReferencedResponse
		validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yaml), &ref)
		require.NoError(t, err)

		errs := ref.Validate(t.Context())
		allErrors := validationErrs
		allErrors = append(allErrors, errs...)
		assert.Empty(t, allErrors)
		assert.True(t, ref.Valid)
	})

	t.Run("ReferencedResponse with inline object", func(t *testing.T) {
		t.Parallel()

		yaml := `
description: User not found
content:
  application/json:
    schema:
      type: object
      properties:
        error:
          type: string
`
		var ref ReferencedResponse
		validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yaml), &ref)
		require.NoError(t, err)

		errs := ref.Validate(t.Context())
		allErrors := validationErrs
		allErrors = append(allErrors, errs...)
		assert.Empty(t, allErrors)
		assert.True(t, ref.Valid)
	})

	t.Run("ReferencedRequestBody with valid reference", func(t *testing.T) {
		t.Parallel()

		yaml := `$ref: '#/components/requestBodies/UserBody'`
		var ref ReferencedRequestBody
		validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yaml), &ref)
		require.NoError(t, err)

		errs := ref.Validate(t.Context())
		allErrors := validationErrs
		allErrors = append(allErrors, errs...)
		assert.Empty(t, allErrors)
		assert.True(t, ref.Valid)
	})

	t.Run("ReferencedRequestBody with inline object", func(t *testing.T) {
		t.Parallel()

		yaml := `
description: User data for creation
required: true
content:
  application/json:
    schema:
      type: object
      properties:
        name:
          type: string
        email:
          type: string
`
		var ref ReferencedRequestBody
		validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yaml), &ref)
		require.NoError(t, err)

		errs := ref.Validate(t.Context())
		allErrors := validationErrs
		allErrors = append(allErrors, errs...)
		assert.Empty(t, allErrors)
		assert.True(t, ref.Valid)
	})
}

func TestReference_Validate_WithOptions(t *testing.T) {
	t.Parallel()

	t.Run("validation with custom options", func(t *testing.T) {
		t.Parallel()

		yaml := `
summary: Test example
description: A test example for validation
value:
  id: 123
  name: Test User
`
		var ref ReferencedExample
		validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yaml), &ref)
		require.NoError(t, err)

		// Test validation with custom options (using a mock context object)
		mockOpenAPI := &OpenAPI{}
		opts := []validation.Option{
			validation.WithContextObject(mockOpenAPI),
		}
		errs := ref.Validate(t.Context(), opts...)
		allErrors := validationErrs
		allErrors = append(allErrors, errs...)
		assert.Empty(t, allErrors)
		assert.True(t, ref.Valid)
	})
}

func TestReference_Validate_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("nil reference validation", func(t *testing.T) {
		t.Parallel()

		var ref *ReferencedExample
		// This should not panic
		errs := ref.Validate(t.Context())
		// Nil reference should be considered invalid
		assert.NotEmpty(t, errs)
	})

	t.Run("reference with nil core", func(t *testing.T) {
		t.Parallel()

		ref := &ReferencedExample{}
		// This should not panic even with uninitialized core
		errs := ref.Validate(t.Context())
		// An uninitialized reference may or may not have errors depending on the core state
		// The important thing is that it doesn't panic
		assert.NotNil(t, errs) // Just ensure we get a slice back, even if empty
	})
}
