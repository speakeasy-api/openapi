package oas3_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchema_ExclusiveMinimumMaximum_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                     string
		yaml                     string
		openAPIVersion           *string     // Optional OpenAPI document version
		expectedExclusiveMinimum interface{} // bool or float64
		expectedExclusiveMaximum interface{} // bool or float64
		shouldValidate           bool
	}{
		// Boolean values with OpenAPI 3.0 context
		{
			name: "boolean exclusiveMinimum and exclusiveMaximum with OpenAPI 3.0 document version",
			yaml: `
type: number
minimum: 0
maximum: 100
exclusiveMinimum: true
exclusiveMaximum: false
`,
			openAPIVersion:           pointer.From("3.0.3"),
			expectedExclusiveMinimum: true,
			expectedExclusiveMaximum: false,
			shouldValidate:           true,
		},
		{
			name: "both boolean values true with OpenAPI 3.0 document version",
			yaml: `
type: number
minimum: 0
maximum: 100
exclusiveMinimum: true
exclusiveMaximum: true
`,
			openAPIVersion:           pointer.From("3.0.3"),
			expectedExclusiveMinimum: true,
			expectedExclusiveMaximum: true,
			shouldValidate:           true,
		},
		{
			name: "both boolean values false with OpenAPI 3.0 document version",
			yaml: `
type: number
minimum: 0
maximum: 100
exclusiveMinimum: false
exclusiveMaximum: false
`,
			openAPIVersion:           pointer.From("3.0.3"),
			expectedExclusiveMinimum: false,
			expectedExclusiveMaximum: false,
			shouldValidate:           true,
		},
		// Boolean values with explicit 3.0 $schema
		{
			name: "boolean exclusiveMinimum and exclusiveMaximum with 3.0 $schema",
			yaml: `
$schema: "https://spec.openapis.org/oas/3.0/dialect/2024-10-18"
type: number
minimum: 0
maximum: 100
exclusiveMinimum: true
exclusiveMaximum: false
`,
			expectedExclusiveMinimum: true,
			expectedExclusiveMaximum: false,
			shouldValidate:           true,
		},
		{
			name: "both boolean values true with 3.0 $schema",
			yaml: `
$schema: "https://spec.openapis.org/oas/3.0/dialect/2024-10-18"
type: number
minimum: 0
maximum: 100
exclusiveMinimum: true
exclusiveMaximum: true
`,
			expectedExclusiveMinimum: true,
			expectedExclusiveMaximum: true,
			shouldValidate:           true,
		},
		// Numeric values (should work with any version)
		{
			name: "numeric exclusiveMinimum and exclusiveMaximum",
			yaml: `
type: number
exclusiveMinimum: 0.5
exclusiveMaximum: 99.5
`,
			expectedExclusiveMinimum: 0.5,
			expectedExclusiveMaximum: 99.5,
			shouldValidate:           true,
		},
		{
			name: "numeric exclusiveMinimum and exclusiveMaximum as integers",
			yaml: `
type: number
exclusiveMinimum: 1
exclusiveMaximum: 99
`,
			expectedExclusiveMinimum: 1.0,
			expectedExclusiveMaximum: 99.0,
			shouldValidate:           true,
		},
		{
			name: "numeric exclusiveMinimum and exclusiveMaximum with OpenAPI 3.1",
			yaml: `
type: number
exclusiveMinimum: 0.5
exclusiveMaximum: 99.5
`,
			openAPIVersion:           pointer.From("3.1.0"),
			expectedExclusiveMinimum: 0.5,
			expectedExclusiveMaximum: 99.5,
			shouldValidate:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var schema oas3.Schema

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yaml), &schema)
			require.NoError(t, err, "Unmarshaling should succeed")
			require.Empty(t, validationErrs, "Should have no validation errors during unmarshaling")

			// Test schema validation with optional document version context
			var validationErrors []error
			if tt.openAPIVersion != nil {
				docVersion := &oas3.ParentDocumentVersion{
					OpenAPI: tt.openAPIVersion,
				}
				validationErrors = schema.Validate(t.Context(), validation.WithContextObject(docVersion))
			} else {
				validationErrors = schema.Validate(t.Context())
			}

			if tt.shouldValidate {
				assert.Empty(t, validationErrors, "Schema validation should pass for: %s", tt.name)
			} else {
				assert.NotEmpty(t, validationErrors, "Schema validation should fail for: %s", tt.name)
			}

			// Verify parsed values
			if tt.expectedExclusiveMinimum != nil {
				require.NotNil(t, schema.ExclusiveMinimum, "ExclusiveMinimum should not be nil")

				switch expected := tt.expectedExclusiveMinimum.(type) {
				case bool:
					assert.True(t, schema.ExclusiveMinimum.IsLeft(), "ExclusiveMinimum should be boolean (Left side)")
					if schema.ExclusiveMinimum.IsLeft() {
						actual := *schema.ExclusiveMinimum.Left
						assert.Equal(t, expected, actual, "ExclusiveMinimum boolean value should match expected")
					}
				case float64:
					assert.True(t, schema.ExclusiveMinimum.IsRight(), "ExclusiveMinimum should be number (Right side)")
					if schema.ExclusiveMinimum.IsRight() {
						actual := *schema.ExclusiveMinimum.Right
						assert.InDelta(t, expected, actual, 0.001, "ExclusiveMinimum number value should match expected")
					}
				}
			}

			if tt.expectedExclusiveMaximum != nil {
				require.NotNil(t, schema.ExclusiveMaximum, "ExclusiveMaximum should not be nil")

				switch expected := tt.expectedExclusiveMaximum.(type) {
				case bool:
					assert.True(t, schema.ExclusiveMaximum.IsLeft(), "ExclusiveMaximum should be boolean (Left side)")
					if schema.ExclusiveMaximum.IsLeft() {
						actual := *schema.ExclusiveMaximum.Left
						assert.Equal(t, expected, actual, "ExclusiveMaximum boolean value should match expected")
					}
				case float64:
					assert.True(t, schema.ExclusiveMaximum.IsRight(), "ExclusiveMaximum should be number (Right side)")
					if schema.ExclusiveMaximum.IsRight() {
						actual := *schema.ExclusiveMaximum.Right
						assert.InDelta(t, expected, actual, 0.001, "ExclusiveMaximum number value should match expected")
					}
				}
			}

			// Verify $schema property if present
			if tt.yaml != "" && strings.Contains(tt.yaml, "$schema:") {
				require.NotNil(t, schema.Schema, "Schema property should not be nil when $schema is specified")
			}
		})
	}
}

func TestSchema_ExclusiveMinimumMaximum_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yaml           string
		openAPIVersion *string
		wantErrs       []string
	}{
		// Boolean values should fail with OpenAPI 3.1
		{
			name: "boolean exclusiveMinimum with OpenAPI 3.1 document version should fail",
			yaml: `
type: number
minimum: 0
maximum: 100
exclusiveMinimum: true
exclusiveMaximum: false
`,
			openAPIVersion: pointer.From("3.1.0"),
			wantErrs:       []string{"[5:19] error validation-type-mismatch schema.exclusiveMinimum expected number, got boolean", "[6:19] error validation-type-mismatch schema.exclusiveMaximum expected number, got boolean"},
		},
		{
			name: "boolean exclusiveMinimum with 3.1 $schema should fail",
			yaml: `
$schema: "https://spec.openapis.org/oas/3.1/dialect/2024-11-10"
type: number
minimum: 0
maximum: 100
exclusiveMinimum: true
exclusiveMaximum: false
`,
			wantErrs: []string{"[6:19] error validation-type-mismatch schema.exclusiveMinimum expected number, got boolean", "[7:19] error validation-type-mismatch schema.exclusiveMaximum expected number, got boolean"},
		},
		// Invalid types should always fail
		{
			name: "invalid string type for exclusiveMinimum",
			yaml: `
type: number
exclusiveMinimum: "invalid"
`,
			wantErrs: []string{"[2:1] error validation-type-mismatch schema.exclusiveMinimum expected number, got string", "[3:19] error validation-type-mismatch schema.exclusiveMinimum failed to validate either bool [schema.exclusiveMinimum line 3: cannot unmarshal !!str `invalid` into bool] or float64 [schema.exclusiveMinimum line 3: cannot unmarshal !!str `invalid` into float64]"},
		},
		{
			name: "invalid string type for exclusiveMaximum",
			yaml: `
type: number
exclusiveMaximum: "invalid"
`,
			wantErrs: []string{"[2:1] error validation-type-mismatch schema.exclusiveMaximum expected number, got string", "[3:19] error validation-type-mismatch schema.exclusiveMaximum failed to validate either bool [schema.exclusiveMaximum line 3: cannot unmarshal !!str `invalid` into bool] or float64 [schema.exclusiveMaximum line 3: cannot unmarshal !!str `invalid` into float64]"},
		},
		{
			name: "invalid array type for exclusiveMinimum",
			yaml: `
type: number
exclusiveMinimum: [1, 2, 3]
`,
			wantErrs: []string{"[2:1] error validation-type-mismatch schema.exclusiveMinimum expected number, got array", "[3:19] error validation-type-mismatch schema.exclusiveMinimum failed to validate either bool [schema.exclusiveMinimum expected bool, got sequence] or float64 [schema.exclusiveMinimum expected float64, got sequence]"},
		},
		// Mixed boolean and numeric should fail with OpenAPI 3.0 (only supports boolean)
		{
			name: "mixed boolean exclusiveMinimum and numeric exclusiveMaximum with OpenAPI 3.0 should fail",
			yaml: `
type: number
minimum: 0
exclusiveMinimum: true
exclusiveMaximum: 50.5
`,
			openAPIVersion: pointer.From("3.0.3"),
			wantErrs:       []string{"[5:19] error validation-type-mismatch schema.exclusiveMaximum expected boolean, got number"},
		},
		{
			name: "mixed numeric exclusiveMinimum and boolean exclusiveMaximum with OpenAPI 3.0 should fail",
			yaml: `
type: number
maximum: 100
exclusiveMinimum: 0.5
exclusiveMaximum: true
`,
			openAPIVersion: pointer.From("3.0.3"),
			wantErrs:       []string{"[4:19] error validation-type-mismatch schema.exclusiveMinimum expected boolean, got number"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var schema oas3.Schema

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yaml), &schema)
			if err == nil {
				if tt.openAPIVersion != nil {
					docVersion := &oas3.ParentDocumentVersion{
						OpenAPI: tt.openAPIVersion,
					}
					validationErrs = append(validationErrs, schema.Validate(t.Context(), validation.WithContextObject(docVersion))...)
				} else {
					validationErrs = append(validationErrs, schema.Validate(t.Context())...)
				}
			}

			validation.SortValidationErrors(validationErrs)

			// Check if any error contains the expected string
			gotErrs := make([]string, len(validationErrs))
			for i, validationErr := range validationErrs {
				gotErrs[i] = validationErr.Error()
			}
			assert.Equal(t, tt.wantErrs, gotErrs)
		})
	}
}
