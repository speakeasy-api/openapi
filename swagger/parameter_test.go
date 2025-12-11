package swagger_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/swagger"
	"github.com/stretchr/testify/require"
)

func TestParameter_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
		in   swagger.ParameterIn
		typ  string
	}{
		{
			name: "path parameter",
			yaml: `
name: id
in: path
description: The ID
required: true
type: string
`,
			in:  swagger.ParameterInPath,
			typ: "string",
		},
		{
			name: "query parameter with array",
			yaml: `
name: ids
in: query
type: array
items:
  type: string
collectionFormat: csv
`,
			in:  swagger.ParameterInQuery,
			typ: "array",
		},
		{
			name: "body parameter",
			yaml: `
name: user
in: body
description: User object
required: true
schema:
  type: object
  properties:
    name:
      type: string
`,
			in: swagger.ParameterInBody,
		},
		{
			name: "header parameter",
			yaml: `
name: X-API-Key
in: header
type: string
required: true
`,
			in:  swagger.ParameterInHeader,
			typ: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var param swagger.Parameter
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yaml), &param)
			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should have no unmarshalling validation errors")

			require.Equal(t, tt.in, param.In, "should have correct location")
			if tt.typ != "" {
				require.NotNil(t, param.Type, "type should be set")
				require.Equal(t, tt.typ, *param.Type, "should have correct type")
			}
		})
	}
}

func TestParameter_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid path parameter",
			yml: `
name: id
in: path
required: true
type: string
`,
		},
		{
			name: "valid query parameter",
			yml: `
name: limit
in: query
type: integer
format: int32
`,
		},
		{
			name: "valid body parameter",
			yml: `
name: body
in: body
schema:
  type: object
`,
		},
		{
			name: "valid formData parameter",
			yml: `
name: file
in: formData
type: file
`,
		},
		{
			name: "valid header parameter",
			yml: `
name: X-API-Key
in: header
type: string
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var param swagger.Parameter
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &param)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := param.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, param.Valid, "expected parameter to be valid")
		})
	}
}

func TestParameter_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "missing name",
			yml: `
in: query
type: string
`,
			wantErrs: []string{"parameter.name is missing"},
		},
		{
			name: "empty name",
			yml: `
name: ""
in: query
type: string
`,
			wantErrs: []string{"parameter.name is required"},
		},
		{
			name: "missing in",
			yml: `
name: test
type: string
`,
			wantErrs: []string{"parameter.in is missing"},
		},
		{
			name: "path parameter not required",
			yml: `
name: userId
in: path
required: false
type: string
`,
			wantErrs: []string{"parameter.in=path requires required=true"},
		},
		{
			name: "path parameter missing required",
			yml: `
name: userId
in: path
type: string
`,
			wantErrs: []string{"parameter.in=path requires required=true"},
		},
		{
			name: "invalid parameter location",
			yml: `
name: test
in: invalid
type: string
`,
			wantErrs: []string{"parameter.in must be one of"},
		},
		{
			name: "body parameter without schema",
			yml: `
name: body
in: body
`,
			wantErrs: []string{"parameter.schema is required for in=body"},
		},
		{
			name: "non-body parameter without type",
			yml: `
name: id
in: query
`,
			wantErrs: []string{"parameter.type is required for non-body parameters"},
		},
		{
			name: "array parameter without items",
			yml: `
name: ids
in: query
type: array
`,
			wantErrs: []string{"parameter.items is required when type=array"},
		},
		{
			name: "file type not in formData",
			yml: `
name: file
in: query
type: file
`,
			wantErrs: []string{"parameter.type=file requires in=formData"},
		},
		{
			name: "invalid parameter type",
			yml: `
name: test
in: query
type: object
`,
			wantErrs: []string{"parameter.type must be one of"},
		},
		{
			name: "multiple validation errors",
			yml: `
name: ""
in: path
required: false
`,
			wantErrs: []string{
				"parameter.name is required",
				"parameter.in=path requires required=true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var param swagger.Parameter
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &param)
			require.NoError(t, err)

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			allErrors = append(allErrors, validationErrs...)

			validateErrs := param.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "expected validation errors")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range allErrors {
				errMessages = append(errMessages, err.Error())
			}

			for _, expectedErr := range tt.wantErrs {
				found := false
				for _, errMsg := range errMessages {
					if strings.Contains(errMsg, expectedErr) {
						found = true
						break
					}
				}
				require.True(t, found, "expected error message '%s' not found in: %v", expectedErr, errMessages)
			}
		})
	}
}

func TestItems_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "simple string items",
			yml: `
type: string
`,
		},
		{
			name: "nested array items",
			yml: `
type: array
items:
  type: integer
`,
		},
		{
			name: "items with format",
			yml: `
type: integer
format: int64
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var items swagger.Items
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &items)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := items.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, items.Valid, "expected items to be valid")
		})
	}
}

func TestItems_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "missing type",
			yml: `
format: int32
`,
			wantErrs: []string{"items.type is missing"},
		},
		{
			name: "array items without nested items",
			yml: `
type: array
`,
			wantErrs: []string{"items.items is required when type=array"},
		},
		{
			name: "invalid items type",
			yml: `
type: object
`,
			wantErrs: []string{"items.type must be one of"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var items swagger.Items
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &items)
			require.NoError(t, err)

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			allErrors = append(allErrors, validationErrs...)

			validateErrs := items.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "expected validation errors")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range allErrors {
				errMessages = append(errMessages, err.Error())
			}

			for _, expectedErr := range tt.wantErrs {
				found := false
				for _, errMsg := range errMessages {
					if strings.Contains(errMsg, expectedErr) {
						found = true
						break
					}
				}
				require.True(t, found, "expected error message '%s' not found in: %v", expectedErr, errMessages)
			}
		})
	}
}

func TestParameter_Getters_Success(t *testing.T) {
	t.Parallel()

	yml := `
name: userId
in: path
description: User identifier
required: true
type: string
schema:
  type: object
x-custom: value
`
	var param swagger.Parameter

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &param)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "userId", param.GetName(), "GetName should return correct value")
	require.Equal(t, swagger.ParameterInPath, param.GetIn(), "GetIn should return correct value")
	require.Equal(t, "User identifier", param.GetDescription(), "GetDescription should return correct value")
	require.True(t, param.GetRequired(), "GetRequired should return true")
	require.Equal(t, "string", param.GetType(), "GetType should return correct value")
	require.NotNil(t, param.GetSchema(), "GetSchema should return non-nil")
	require.NotNil(t, param.GetExtensions(), "GetExtensions should return non-nil")
}

func TestParameter_Getters_Nil(t *testing.T) {
	t.Parallel()

	var param *swagger.Parameter

	require.Empty(t, param.GetName(), "GetName should return empty string for nil")
	require.Empty(t, param.GetIn(), "GetIn should return empty for nil")
	require.Empty(t, param.GetDescription(), "GetDescription should return empty string for nil")
	require.False(t, param.GetRequired(), "GetRequired should return false for nil")
	require.Empty(t, param.GetType(), "GetType should return empty string for nil")
	require.Nil(t, param.GetSchema(), "GetSchema should return nil for nil param")
	require.NotNil(t, param.GetExtensions(), "GetExtensions should return empty extensions for nil param")
}

func TestItems_Getters_Success(t *testing.T) {
	t.Parallel()

	yml := `
type: string
x-custom: value
`
	var items swagger.Items

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &items)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "string", items.GetType(), "GetType should return correct value")
	require.NotNil(t, items.GetExtensions(), "GetExtensions should return non-nil")
}

func TestItems_Getters_Nil(t *testing.T) {
	t.Parallel()

	var items *swagger.Items

	require.Empty(t, items.GetType(), "GetType should return empty string for nil")
	require.NotNil(t, items.GetExtensions(), "GetExtensions should return empty extensions for nil items")
}
