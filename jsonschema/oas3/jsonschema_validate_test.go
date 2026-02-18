package oas3_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_TopLevel_Success(t *testing.T) {
	t.Parallel()

	t.Run("nil schema returns nil", func(t *testing.T) {
		t.Parallel()

		var schema *oas3.JSONSchema[oas3.Referenceable]
		errs := oas3.Validate(t.Context(), schema)
		require.Nil(t, errs, "nil schema should return nil errors")
	})

	t.Run("bool schema returns nil", func(t *testing.T) {
		t.Parallel()

		schema := oas3.NewJSONSchemaFromBool(true)
		errs := oas3.Validate(t.Context(), schema)
		require.Nil(t, errs, "bool schema should return nil errors")
	})

	t.Run("bool false schema returns nil", func(t *testing.T) {
		t.Parallel()

		schema := oas3.NewJSONSchemaFromBool(false)
		errs := oas3.Validate(t.Context(), schema)
		require.Nil(t, errs, "bool false schema should return nil errors")
	})

	t.Run("valid schema returns nil errors", func(t *testing.T) {
		t.Parallel()

		yml := `
type: string
title: Valid Schema
`
		var schema oas3.JSONSchema[oas3.Referenceable]
		_, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &schema)
		require.NoError(t, err)

		errs := oas3.Validate(t.Context(), &schema)
		require.Empty(t, errs, "valid schema should return no errors")
	})
}

func TestValidate_TopLevel_Error(t *testing.T) {
	t.Parallel()

	t.Run("invalid schema returns errors", func(t *testing.T) {
		t.Parallel()

		yml := `
type: invalid_type
`
		var schema oas3.JSONSchema[oas3.Referenceable]
		_, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &schema)
		require.NoError(t, err)

		errs := oas3.Validate(t.Context(), &schema)
		require.NotEmpty(t, errs, "invalid schema should return errors")
	})
}

func TestSchema_Validate_OpenAPIVersions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		yml     string
	}{
		{
			name:    "OpenAPI 3.0 version via context",
			version: "3.0.3",
			yml: `
type: string
`,
		},
		{
			name:    "OpenAPI 3.1 version via context",
			version: "3.1.0",
			yml: `
type: string
`,
		},
		{
			name:    "OpenAPI 3.2 version via context",
			version: "3.2.0",
			yml: `
type: string
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var schema oas3.Schema
			_, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &schema)
			require.NoError(t, err)

			dv := &oas3.ParentDocumentVersion{
				OpenAPI: &tt.version,
			}

			errs := schema.Validate(t.Context(), validation.WithContextObject(dv))
			require.Empty(t, errs, "valid schema should return no errors for version %s", tt.version)
		})
	}
}

func TestSchema_Validate_SchemaField_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "explicit 3.0 $schema field",
			yml: `
$schema: "https://spec.openapis.org/oas/3.0/dialect/2024-10-18"
type: string
`,
		},
		{
			name: "explicit 3.1 $schema field",
			yml: `
$schema: "https://spec.openapis.org/oas/3.1/meta/2024-11-10"
type: string
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var schema oas3.Schema
			_, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &schema)
			require.NoError(t, err)

			errs := schema.Validate(t.Context())
			require.Empty(t, errs, "valid schema should return no errors")
		})
	}
}

func TestSchema_Validate_UnsupportedVersion_Defaults(t *testing.T) {
	t.Parallel()

	t.Run("unsupported OpenAPI version defaults to 3.1", func(t *testing.T) {
		t.Parallel()

		yml := `
type: string
`
		var schema oas3.Schema
		_, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &schema)
		require.NoError(t, err)

		version := "2.0.0"
		dv := &oas3.ParentDocumentVersion{
			OpenAPI: &version,
		}

		errs := schema.Validate(t.Context(), validation.WithContextObject(dv))
		require.Empty(t, errs, "unsupported version should default to 3.1 and validate successfully")
	})

	t.Run("Arazzo version is unsupported and defaults to 3.1", func(t *testing.T) {
		t.Parallel()

		yml := `
type: string
`
		var schema oas3.Schema
		_, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &schema)
		require.NoError(t, err)

		version := "1.0.0"
		dv := &oas3.ParentDocumentVersion{
			Arazzo: &version,
		}

		errs := schema.Validate(t.Context(), validation.WithContextObject(dv))
		require.Empty(t, errs, "Arazzo version should default to 3.1 and validate successfully")
	})

	t.Run("unsupported $schema field defaults to 3.1", func(t *testing.T) {
		t.Parallel()

		yml := `
$schema: "https://json-schema.org/draft/2020-12/schema"
type: string
`
		var schema oas3.Schema
		_, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &schema)
		require.NoError(t, err)

		errs := schema.Validate(t.Context())
		require.Empty(t, errs, "unsupported $schema should default to 3.1")
	})
}

func TestJSONSchema_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "schema fails direct validation",
			yml: `
"test"`,
			wantErrs: []string{"[2:1] error validation-type-mismatch failed to validate either Schema [expected `object`, got `te...`] or bool [line 2: cannot construct !!str `test` into bool]"},
		},
		{
			name: "child schema fails validation",
			yml: `
type:
    - array
    - "null"
items:
    $ref: "#/components/schemas/ffmpeg-profile"
default:
    $ref: "#/components/schemas/stream/properties/profiles/default"
description:
    $ref: "#/components/schemas/stream/properties/profiles/description"
`,
			wantErrs: []string{
				"[2:1] error validation-type-mismatch schema.description expected `string`, got `object`",
				"[10:5] error validation-type-mismatch schema.description expected `string`, got `object`",
			},
		},
		{
			name: "incorrect type fails validation",
			yml: `
type: invalid_type
`,
			wantErrs: []string{
				"[2:7] error validation-invalid-schema schema.type value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string'",
				"[2:7] error validation-type-mismatch schema.type expected `array`, got `string`",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var schema oas3.JSONSchema[oas3.Referenceable]

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &schema)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := schema.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)
			validation.SortValidationErrors(allErrors)

			require.NotEmpty(t, allErrors, "expected validation errors")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range allErrors {
				if err != nil {
					errMessages = append(errMessages, err.Error())
				}
			}

			assert.Equal(t, tt.wantErrs, errMessages)
		})
	}
}

func TestJSONSchemaConcrete_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		schema      *oas3.JSONSchema[oas3.Concrete]
		expectEmpty bool
	}{
		{
			name:        "nil schema returns empty extensions",
			schema:      nil,
			expectEmpty: true,
		},
		{
			name:        "bool schema returns empty extensions",
			schema:      oas3.ReferenceableToConcrete(oas3.NewJSONSchemaFromBool(true)),
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.schema.GetExtensions()
			require.NotNil(t, result)
			if tt.expectEmpty {
				assert.Equal(t, 0, result.Len())
			}
		})
	}
}
