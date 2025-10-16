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
			wantErrs: []string{"response.description is missing"},
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
			wantErrs: []string{"header.type is missing"},
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
