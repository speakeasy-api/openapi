package openapi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestServer_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid server with URL only",
			yml: `
url: https://api.example.com
`,
		},
		{
			name: "valid server with URL and description",
			yml: `
url: https://api.example.com/v1
description: Production server
`,
		},
		{
			name: "valid server with variables",
			yml: `
url: https://{environment}.example.com/{version}
description: Server with variables
variables:
  environment:
    default: api
    enum:
      - api
      - staging
    description: Environment name
  version:
    default: v1
    description: API version
x-test: some-value
`,
		},
		{
			name: "valid server with localhost URL",
			yml: `
url: http://localhost:8080
description: Local development server
`,
		},
		{
			name: "valid server with relative URL",
			yml: `
url: /api/v1
description: Relative URL server
`,
		},
		{
			name: "valid server with complex variables",
			yml: `
url: https://{subdomain}.{domain}.com:{port}/{basePath}
variables:
  subdomain:
    default: api
  domain:
    default: example
  port:
    default: "443"
    enum: ["443", "8443"]
  basePath:
    default: v1
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var server openapi.Server
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &server)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := server.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, server.Valid, "expected server to be valid")
		})
	}
}

func TestServer_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "missing URL",
			yml: `
description: Server without URL
`,
			wantErrs: []string{"[2:1] error validation-required-field `server.url` is required"},
		},
		{
			name: "empty URL",
			yml: `
url: ""
description: Server with empty URL
`,
			wantErrs: []string{"[2:6] error validation-required-field `server.url` is required"},
		},
		{
			name: "variable without default value",
			yml: `
url: https://{environment}.example.com
variables:
  environment:
    description: Environment name
`,
			wantErrs: []string{"[5:5] error validation-required-field `serverVariable.default` is required"},
		},
		{
			name: "variable with empty default",
			yml: `
url: https://{environment}.example.com
variables:
  environment:
    default: ""
    description: Environment name
`,
			wantErrs: []string{"[5:14] error validation-required-field `serverVariable.default` is required"},
		},
		{
			name: "variable with invalid enum value",
			yml: `
url: https://{environment}.example.com
variables:
  environment:
    default: production
    enum:
      - staging
      - development
    description: Environment name
`,
			wantErrs: []string{"[5:14] error validation-allowed-values serverVariable.default must be one of [`staging, development`]"},
		},
		{
			name: "multiple validation errors",
			yml: `
url: ""
variables:
  environment:
    default: ""
    description: Environment name
`,
			wantErrs: []string{
				"[2:6] error validation-required-field `server.url` is required",
				"[5:14] error validation-required-field `serverVariable.default` is required",
			},
		},
		{
			name: "double curly braces variable",
			yml: `
url: http://{{hostname}}:8080
variables:
  hostname:
    default: api
`,
			wantErrs: []string{
				"error validation-invalid-syntax server variable `{hostname}` is not defined. Use single curly braces for variable substitution",
			},
		},
		{
			name: "double curly braces multiple variables",
			yml: `
url: http://{{hostname}}{{port}}
variables:
  hostname:
    default: api
  port:
    default: "8080"
`,
			wantErrs: []string{
				"error validation-invalid-syntax server variable `{hostname}` is not defined. Use single curly braces for variable substitution",
				"error validation-invalid-syntax server variable `{port}` is not defined. Use single curly braces for variable substitution",
			},
		},
		{
			name: "missing variable with single braces",
			yml: `
url: http://{hostname}:8080
variables:
  port:
    default: "8080"
`,
			wantErrs: []string{
				"error validation-invalid-syntax server variable `hostname` is not defined",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var server openapi.Server

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &server)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := server.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "expected validation errors")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range allErrors {
				if err != nil {
					errMessages = append(errMessages, err.Error())
				}
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

func TestServerVariable_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid server variable with default only",
			yml: `
default: api
`,
		},
		{
			name: "valid server variable with default and description",
			yml: `
default: v1
description: API version
`,
		},
		{
			name: "valid server variable with enum",
			yml: `
default: production
enum:
  - production
  - staging
  - development
description: Environment name
`,
		},
		{
			name: "valid server variable with single enum value",
			yml: `
default: v1
enum:
  - v1
description: Fixed version
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var variable openapi.ServerVariable
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &variable)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := variable.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, variable.Valid, "expected server variable to be valid")
		})
	}
}

func TestServerVariable_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "missing default",
			yml: `
description: Variable without default
`,
			wantErrs: []string{"[2:1] error validation-required-field `serverVariable.default` is required"},
		},
		{
			name: "empty default",
			yml: `
default: ""
description: Variable with empty default
`,
			wantErrs: []string{"[2:10] error validation-required-field `serverVariable.default` is required"},
		},
		{
			name: "default not in enum",
			yml: `
default: invalid
enum:
  - valid1
  - valid2
description: Variable with invalid default
`,
			wantErrs: []string{"[2:10] error validation-allowed-values serverVariable.default must be one of [`valid1, valid2`]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var variable openapi.ServerVariable

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &variable)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := variable.Validate(t.Context())
			allErrors = append(allErrors, validateErrs...)

			require.NotEmpty(t, allErrors, "expected validation errors")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range allErrors {
				if err != nil {
					errMessages = append(errMessages, err.Error())
				}
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
