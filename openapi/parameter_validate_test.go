package openapi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestParameter_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid path parameter",
			yml: `
name: userId
in: path
required: true
schema:
  type: string
description: The user ID
`,
		},
		{
			name: "valid query parameter",
			yml: `
name: limit
in: query
schema:
  type: integer
  minimum: 1
  maximum: 100
description: Number of items to return
`,
		},
		{
			name: "valid header parameter",
			yml: `
name: X-API-Key
in: header
required: true
schema:
  type: string
description: API key for authentication
`,
		},
		{
			name: "valid cookie parameter",
			yml: `
name: sessionId
in: cookie
schema:
  type: string
description: Session identifier
`,
		},
		{
			name: "parameter with content",
			yml: `
name: filter
in: query
content:
  application/json:
    schema:
      type: object
description: Complex filter object
`,
		},
		{
			name: "parameter with examples",
			yml: `
name: status
in: query
schema:
  type: string
  enum: [active, inactive]
examples:
  active:
    value: active
    summary: Active status
  inactive:
    value: inactive
    summary: Inactive status
`,
		},
		{
			name: "parameter with style and explode",
			yml: `
name: tags
in: query
style: form
explode: true
schema:
  type: array
  items:
    type: string
`,
		},
		{
			name: "deprecated parameter",
			yml: `
name: oldParam
in: query
deprecated: true
schema:
  type: string
description: This parameter is deprecated
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var param openapi.Parameter
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
schema:
  type: string
`,
			wantErrs: []string{"[2:1] parameter.name is missing"},
		},
		{
			name: "empty name",
			yml: `
name: ""
in: query
schema:
  type: string
`,
			wantErrs: []string{"[2:7] parameter.name is required"},
		},
		{
			name: "missing in",
			yml: `
name: test
schema:
  type: string
`,
			wantErrs: []string{"[2:1] parameter.in is missing"},
		},
		{
			name: "path parameter not required",
			yml: `
name: userId
in: path
required: false
schema:
  type: string
`,
			wantErrs: []string{"[4:11] parameter.in=path requires required=true"},
		},
		{
			name: "invalid parameter location",
			yml: `
name: test
in: invalid
schema:
  type: string
`,
			wantErrs: []string{"[3:5] parameter.in must be one of [query, header, path, cookie]"},
		},
		{
			name: "multiple validation errors",
			yml: `
name: ""
in: path
required: false
`,
			wantErrs: []string{
				"[2:7] parameter.name is required",
				"[4:11] parameter.in=path requires required=true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var param openapi.Parameter
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
