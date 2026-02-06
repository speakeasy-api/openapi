package openapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeader_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid header with schema",
			yml: `
schema:
  type: string
description: API version header
`,
		},
		{
			name: "valid required header",
			yml: `
required: true
schema:
  type: string
  pattern: "^v[0-9]+$"
description: Version header
`,
		},
		{
			name: "valid header with content",
			yml: `
content:
  application/json:
    schema:
      type: object
      properties:
        version:
          type: string
description: Complex header content
`,
		},
		{
			name: "valid header with examples",
			yml: `
schema:
  type: string
examples:
  v1:
    value: "v1.0"
    summary: Version 1
  v2:
    value: "v2.0"
    summary: Version 2
description: Version header with examples
`,
		},
		{
			name: "valid header with style and explode",
			yml: `
schema:
  type: array
  items:
    type: string
style: simple
explode: false
description: Array header
`,
		},
		{
			name: "valid deprecated header",
			yml: `
deprecated: true
schema:
  type: string
description: Deprecated header
`,
		},
		{
			name: "valid header with extensions",
			yml: `
schema:
  type: string
description: Header with extensions
x-test: some-value
x-custom: custom-data
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var header openapi.Header
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &header)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := header.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, header.Valid, "expected header to be valid")
		})
	}
}

func TestHeader_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "invalid schema type",
			yml: `
schema:
  type: invalid-type
description: Header with invalid schema
`,
			wantErrs: []string{
				"[3:9] error validation-invalid-schema schema.type value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string'",
				"[3:9] error validation-type-mismatch schema.type expected `array`, got `string`",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var header openapi.Header
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &header)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := header.Validate(t.Context())
			require.NotEmpty(t, errs, "expected validation errors")
			require.False(t, header.Valid, "expected header to be invalid")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range errs {
				errMessages = append(errMessages, err.Error())
			}

			assert.Equal(t, tt.wantErrs, errMessages)
		})
	}
}

func TestHeader_Getters_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yml          string
		wantRequired bool
		wantDeprec   bool
		wantStyle    openapi.SerializationStyle
		wantExplode  bool
		wantDescr    string
	}{
		{
			name: "all fields set",
			yml: `
required: true
deprecated: true
style: simple
explode: true
description: Test header
schema:
  type: string
`,
			wantRequired: true,
			wantDeprec:   true,
			wantStyle:    openapi.SerializationStyleSimple,
			wantExplode:  true,
			wantDescr:    "Test header",
		},
		{
			name: "default values",
			yml: `
schema:
  type: string
`,
			wantRequired: false,
			wantDeprec:   false,
			wantStyle:    openapi.SerializationStyleSimple,
			wantExplode:  false,
			wantDescr:    "",
		},
		{
			name: "only required set",
			yml: `
required: true
schema:
  type: string
`,
			wantRequired: true,
			wantDeprec:   false,
			wantStyle:    openapi.SerializationStyleSimple,
			wantExplode:  false,
			wantDescr:    "",
		},
		{
			name: "only deprecated set",
			yml: `
deprecated: true
schema:
  type: string
`,
			wantRequired: false,
			wantDeprec:   true,
			wantStyle:    openapi.SerializationStyleSimple,
			wantExplode:  false,
			wantDescr:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var header openapi.Header
			_, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &header)
			require.NoError(t, err)

			assert.Equal(t, tt.wantRequired, header.GetRequired(), "GetRequired mismatch")
			assert.Equal(t, tt.wantDeprec, header.GetDeprecated(), "GetDeprecated mismatch")
			assert.Equal(t, tt.wantStyle, header.GetStyle(), "GetStyle mismatch")
			assert.Equal(t, tt.wantExplode, header.GetExplode(), "GetExplode mismatch")
			assert.Equal(t, tt.wantDescr, header.GetDescription(), "GetDescription mismatch")
			assert.NotNil(t, header.GetSchema(), "GetSchema should not be nil")
			assert.NotNil(t, header.GetExtensions(), "GetExtensions should never be nil")
		})
	}
}

func TestHeader_Getters_Nil(t *testing.T) {
	t.Parallel()

	var header *openapi.Header = nil

	assert.False(t, header.GetRequired(), "nil header GetRequired should return false")
	assert.False(t, header.GetDeprecated(), "nil header GetDeprecated should return false")
	assert.Equal(t, openapi.SerializationStyleSimple, header.GetStyle(), "nil header GetStyle should return simple")
	assert.False(t, header.GetExplode(), "nil header GetExplode should return false")
	assert.Empty(t, header.GetDescription(), "nil header GetDescription should return empty")
	assert.Nil(t, header.GetSchema(), "nil header GetSchema should return nil")
	assert.Nil(t, header.GetContent(), "nil header GetContent should return nil")
	assert.Nil(t, header.GetExample(), "nil header GetExample should return nil")
	assert.Nil(t, header.GetExamples(), "nil header GetExamples should return nil")
	assert.NotNil(t, header.GetExtensions(), "nil header GetExtensions should return empty")
}

func TestHeader_GetContent_Success(t *testing.T) {
	t.Parallel()

	yml := `
content:
  application/json:
    schema:
      type: object
description: Header with content
`

	var header openapi.Header
	_, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &header)
	require.NoError(t, err)

	content := header.GetContent()
	require.NotNil(t, content, "GetContent should not be nil")
	assert.Equal(t, 1, content.Len(), "content should have one entry")
}

func TestHeader_GetExamples_Success(t *testing.T) {
	t.Parallel()

	yml := `
schema:
  type: string
examples:
  example1:
    value: "test1"
  example2:
    value: "test2"
`

	var header openapi.Header
	_, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &header)
	require.NoError(t, err)

	examples := header.GetExamples()
	require.NotNil(t, examples, "GetExamples should not be nil")
	assert.Equal(t, 2, examples.Len(), "examples should have two entries")
}
