package swagger_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/swagger"
	"github.com/stretchr/testify/require"
)

func TestResponse_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "minimal_response",
			yml:  `description: Success`,
		},
		{
			name: "response_with_schema",
			yml: `description: User response
schema:
  type: object
  properties:
    id:
      type: integer
    name:
      type: string`,
		},
		{
			name: "response_with_headers",
			yml: `description: Success
headers:
  X-Rate-Limit:
    type: integer
    description: Rate limit`,
		},
		{
			name: "response_with_examples",
			yml: `description: Success
schema:
  type: string
examples:
  application/json: "example value"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var response swagger.Response

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &response)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := response.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestResponse_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name:     "missing_description",
			yml:      `schema: {type: object}`,
			wantErrs: []string{"response.description is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var response swagger.Response

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &response)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := response.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "Expected validation errors")

			for _, wantErr := range tt.wantErrs {
				found := false
				for _, gotErr := range allErrors {
					if gotErr != nil && strings.Contains(gotErr.Error(), wantErr) {
						found = true
						break
					}
				}
				require.True(t, found, "Expected error containing '%s' not found in: %v", wantErr, allErrors)
			}
		})
	}
}

func TestHeader_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_header_integer",
			yml: `type: integer
description: Rate limit`,
		},
		{
			name: "valid_header_string",
			yml: `type: string
description: Request ID`,
		},
		{
			name: "valid_header_array",
			yml: `type: array
items:
  type: string
description: Multiple values`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var header swagger.Header

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &header)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := header.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
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
			name:     "missing_type",
			yml:      `description: Some header`,
			wantErrs: []string{"header.type is required"},
		},
		{
			name: "invalid_type",
			yml: `type: object
description: Invalid type`,
			wantErrs: []string{"header.type must be one of"},
		},
		{
			name: "array_without_items",
			yml: `type: array
description: Array header`,
			wantErrs: []string{"header.items is required when type=array"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var header swagger.Header

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &header)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := header.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "Expected validation errors")

			for _, wantErr := range tt.wantErrs {
				found := false
				for _, gotErr := range allErrors {
					if gotErr != nil && strings.Contains(gotErr.Error(), wantErr) {
						found = true
						break
					}
				}
				require.True(t, found, "Expected error containing '%s' not found in: %v", wantErr, allErrors)
			}
		})
	}
}

func TestResponses_Getters_Success(t *testing.T) {
	t.Parallel()

	yml := `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        default:
          description: Default response
        "200":
          description: Success response
        x-custom: value
`
	doc, validationErrs, err := swagger.Unmarshal(t.Context(), strings.NewReader(yml))
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "should have no validation errors")

	pathItem, ok := doc.Paths.Get("/users")
	require.True(t, ok, "path should exist")
	resp := pathItem.Get().Responses

	require.NotNil(t, resp.GetDefault(), "GetDefault should return non-nil")
	require.NotNil(t, resp.GetExtensions(), "GetExtensions should return non-nil")
}

func TestResponses_Getters_Nil(t *testing.T) {
	t.Parallel()

	var resp *swagger.Responses

	require.Nil(t, resp.GetDefault(), "GetDefault should return nil for nil responses")
	require.NotNil(t, resp.GetExtensions(), "GetExtensions should return empty extensions for nil responses")
}

func TestResponses_NewResponses(t *testing.T) {
	t.Parallel()

	resp := swagger.NewResponses()
	require.NotNil(t, resp, "NewResponses should return non-nil")
}

func TestResponse_Getters_Success(t *testing.T) {
	t.Parallel()

	yml := `description: Success response
schema:
  type: object
headers:
  X-Rate-Limit:
    type: integer
examples:
  application/json: {"key": "value"}
x-custom: value
`
	var resp swagger.Response

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &resp)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "Success response", resp.GetDescription(), "GetDescription should return correct value")
	require.NotNil(t, resp.GetSchema(), "GetSchema should return non-nil")
	require.NotNil(t, resp.GetHeaders(), "GetHeaders should return non-nil")
	require.NotNil(t, resp.GetExamples(), "GetExamples should return non-nil")
	require.NotNil(t, resp.GetExtensions(), "GetExtensions should return non-nil")
}

func TestResponse_Getters_Nil(t *testing.T) {
	t.Parallel()

	var resp *swagger.Response

	require.Empty(t, resp.GetDescription(), "GetDescription should return empty string for nil")
	require.Nil(t, resp.GetSchema(), "GetSchema should return nil for nil response")
	require.Nil(t, resp.GetHeaders(), "GetHeaders should return nil for nil response")
	require.Nil(t, resp.GetExamples(), "GetExamples should return nil for nil response")
	require.NotNil(t, resp.GetExtensions(), "GetExtensions should return empty extensions for nil response")
}

func TestHeader_Getters_Success(t *testing.T) {
	t.Parallel()

	yml := `type: integer
description: Rate limit header
x-custom: value
`
	var header swagger.Header

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &header)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "Rate limit header", header.GetDescription(), "GetDescription should return correct value")
	require.Equal(t, "integer", header.GetType(), "GetType should return correct value")
	require.NotNil(t, header.GetExtensions(), "GetExtensions should return non-nil")
}

func TestHeader_Getters_Nil(t *testing.T) {
	t.Parallel()

	var header *swagger.Header

	require.Empty(t, header.GetDescription(), "GetDescription should return empty string for nil")
	require.Empty(t, header.GetType(), "GetType should return empty string for nil")
	require.NotNil(t, header.GetExtensions(), "GetExtensions should return empty extensions for nil header")
}
