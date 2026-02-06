package swagger_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/swagger"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/require"
)

func TestSwagger_Validate_BasePath_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_basePath_with_leading_slash",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
basePath: /v1
paths: {}`,
		},
		{
			name: "valid_basePath_with_just_slash",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
basePath: /
paths: {}`,
		},
		{
			name: "valid_basePath_with_multiple_segments",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
basePath: /api/v1
paths: {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var doc swagger.Swagger

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &doc)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := doc.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestSwagger_Validate_BasePath_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "basePath_without_leading_slash",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
basePath: v1
paths: {}`,
			wantErrs: []string{"basePath must start with a leading slash"},
		},
		{
			name: "basePath_with_only_text",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
basePath: api
paths: {}`,
			wantErrs: []string{"basePath must start with a leading slash"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var doc swagger.Swagger

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &doc)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := doc.Validate(t.Context())
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

func TestPaths_Validate_PathKeys_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_path_with_leading_slash",
			yml: `/users:
  get:
    responses:
      200:
        description: Success`,
		},
		{
			name: "valid_path_with_parameters",
			yml: `/users/{id}:
  get:
    responses:
      200:
        description: Success`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var paths swagger.Paths

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &paths)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := paths.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestPaths_Validate_PathKeys_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "path_without_leading_slash",
			yml: `users:
  get:
    responses:
      200:
        description: Success`,
			wantErrs: []string{"must begin with a slash"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var paths swagger.Paths

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &paths)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := paths.Validate(t.Context())
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

func TestSwagger_Validate_Schemes_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_http_scheme",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
schemes:
  - http
paths: {}`,
		},
		{
			name: "valid_https_scheme",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
schemes:
  - https
paths: {}`,
		},
		{
			name: "valid_multiple_schemes",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
schemes:
  - http
  - https
  - ws
  - wss
paths: {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var doc swagger.Swagger

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &doc)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := doc.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestSwagger_Validate_Schemes_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "invalid_scheme_ftp",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
schemes:
  - ftp
paths: {}`,
			wantErrs: []string{"scheme must be one of"},
		},
		{
			name: "invalid_scheme_mixed",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
schemes:
  - https
  - invalid
paths: {}`,
			wantErrs: []string{"scheme must be one of"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var doc swagger.Swagger

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &doc)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := doc.Validate(t.Context())
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

func TestOperation_Validate_Schemes_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_operation_schemes",
			yml: `schemes:
  - https
responses:
  200:
    description: Success`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var op swagger.Operation

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &op)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := op.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestOperation_Validate_Schemes_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "invalid_operation_scheme",
			yml: `schemes:
  - invalid
responses:
  200:
    description: Success`,
			wantErrs: []string{"operation.scheme must be one of"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var op swagger.Operation

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &op)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := op.Validate(t.Context())
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

func TestSwagger_Validate_MIMETypes_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_consumes",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
consumes:
  - application/json
  - application/xml
paths: {}`,
		},
		{
			name: "valid_produces",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
produces:
  - application/json
  - text/plain
paths: {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var doc swagger.Swagger

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &doc)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := doc.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestSwagger_Validate_MIMETypes_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "invalid_consumes_MIME_type",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
consumes:
  - "invalid mime"
paths: {}`,
			wantErrs: []string{"consumes contains invalid MIME type"},
		},
		{
			name: "invalid_produces_MIME_type",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
produces:
  - invalid//mime
paths: {}`,
			wantErrs: []string{"produces contains invalid MIME type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var doc swagger.Swagger

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &doc)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := doc.Validate(t.Context())
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

func TestSwagger_Validate_TagNameUniqueness_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "unique_tag_names",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
tags:
  - name: users
    description: User operations
  - name: posts
    description: Post operations
paths: {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var doc swagger.Swagger

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &doc)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := doc.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestSwagger_Validate_TagNameUniqueness_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "duplicate_tag_names",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
tags:
  - name: users
    description: User operations
  - name: users
    description: Duplicate tag
paths: {}`,
			wantErrs: []string{"tag name `users` must be unique"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var doc swagger.Swagger

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &doc)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := doc.Validate(t.Context())
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

func TestSwagger_Validate_OperationIdUniqueness_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "unique_operation_IDs",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        200:
          description: Success
  /posts:
    get:
      operationId: getPosts
      responses:
        200:
          description: Success`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var doc swagger.Swagger

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &doc)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := doc.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestSwagger_Validate_OperationIdUniqueness_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "duplicate_operation_IDs",
			yml: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getItems
      responses:
        200:
          description: Success
  /posts:
    get:
      operationId: getItems
      responses:
        200:
          description: Success`,
			wantErrs: []string{"operationId `getItems` must be unique"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var doc swagger.Swagger

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &doc)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := doc.Validate(t.Context())
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

func TestResponses_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "responses_with_200",
			yml: `200:
  description: Success`,
		},
		{
			name: "responses_with_default",
			yml: `default:
  description: Default response`,
		},
		{
			name: "responses_with_multiple",
			yml: `200:
  description: Success
404:
  description: Not found`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var responses swagger.Responses

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &responses)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := responses.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestResponses_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name:     "empty_responses",
			yml:      `{}`,
			wantErrs: []string{"responses must contain at least one response code"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var responses swagger.Responses

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &responses)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := responses.Validate(t.Context())
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

func TestParameter_Validate_CollectionFormat_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "collectionFormat_multi_with_query",
			yml: `name: ids
in: query
type: array
items:
  type: string
collectionFormat: multi`,
		},
		{
			name: "collectionFormat_multi_with_formData",
			yml: `name: tags
in: formData
type: array
items:
  type: string
collectionFormat: multi`,
		},
		{
			name: "collectionFormat_csv_with_path",
			yml: `name: ids
in: path
required: true
type: array
items:
  type: string
collectionFormat: csv`,
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
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestParameter_Validate_CollectionFormat_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "collectionFormat_multi_with_path",
			yml: `name: ids
in: path
required: true
type: array
items:
  type: string
collectionFormat: multi`,
			wantErrs: []string{"collectionFormat='multi' is only valid for in=query or in=formData"},
		},
		{
			name: "collectionFormat_multi_with_header",
			yml: `name: ids
in: header
type: array
items:
  type: string
collectionFormat: multi`,
			wantErrs: []string{"collectionFormat='multi' is only valid for in=query or in=formData"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var param swagger.Parameter

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &param)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := param.Validate(t.Context())
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

func TestParameter_Validate_FileTypeConsumes_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		consumes []string
	}{
		{
			name: "file_parameter_with_multipart_form_data",
			yml: `name: file
in: formData
type: file`,
			consumes: []string{"multipart/form-data"},
		},
		{
			name: "file_parameter_with_urlencoded",
			yml: `name: file
in: formData
type: file`,
			consumes: []string{"application/x-www-form-urlencoded"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var param swagger.Parameter

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &param)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			// Create operation context with appropriate consumes
			var operation swagger.Operation
			_, err = marshaller.Unmarshal(t.Context(), bytes.NewBufferString(`consumes:
  - `+tt.consumes[0]+`
responses:
  200:
    description: OK`), &operation)
			require.NoError(t, err)

			errs := param.Validate(t.Context(), validation.WithContextObject(&operation))
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestParameter_Validate_FileTypeConsumes_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		opYml    string
		wantErrs []string
	}{
		{
			name: "file_parameter_without_consumes",
			yml: `name: file
in: formData
type: file`,
			opYml: `responses:
  200:
    description: OK`,
			wantErrs: []string{"parameter with type=file requires operation to have consumes defined"},
		},
		{
			name: "file_parameter_with_invalid_consumes",
			yml: `name: file
in: formData
type: file`,
			opYml: `consumes:
  - application/json
responses:
  200:
    description: OK`,
			wantErrs: []string{"parameter with type=file requires operation consumes to be 'multipart/form-data' or 'application/x-www-form-urlencoded'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var param swagger.Parameter

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &param)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			// Create operation context
			var operation swagger.Operation
			_, err = marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.opYml), &operation)
			require.NoError(t, err)

			var allErrors []error
			validateErrs := param.Validate(t.Context(), validation.WithContextObject(&operation))
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

func TestSecurityRequirement_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_oauth2_with_scopes",
			yml: `oauth:
  - read
  - write`,
		},
		{
			name: "valid_apiKey_empty_scopes",
			yml:  `apiKey: []`,
		},
		{
			name: "valid_basic_empty_scopes",
			yml:  `basic: []`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var secReq swagger.SecurityRequirement

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &secReq)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			// Create Swagger context with security definitions
			doc := &swagger.Swagger{
				SecurityDefinitions: sequencedmap.New(
					sequencedmap.NewElem("oauth", &swagger.SecurityScheme{Type: swagger.SecuritySchemeTypeOAuth2}),
					sequencedmap.NewElem("apiKey", &swagger.SecurityScheme{Type: swagger.SecuritySchemeTypeAPIKey}),
					sequencedmap.NewElem("basic", &swagger.SecurityScheme{Type: swagger.SecuritySchemeTypeBasic}),
				),
			}

			errs := secReq.Validate(t.Context(), validation.WithContextObject(doc))
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestSecurityRequirement_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name:     "undefined_security_scheme",
			yml:      `undefined: []`,
			wantErrs: []string{"security requirement `undefined` does not match any security scheme"},
		},
		{
			name:     "apiKey_with_non_empty_scopes",
			yml:      `apiKey: ["some_scope"]`,
			wantErrs: []string{"security requirement `apiKey` must have empty scopes array for non-oauth2"},
		},
		{
			name:     "basic_with_non_empty_scopes",
			yml:      `basic: ["some_scope"]`,
			wantErrs: []string{"security requirement `basic` must have empty scopes array for non-oauth2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var secReq swagger.SecurityRequirement

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &secReq)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			// Create Swagger context with security definitions
			doc := &swagger.Swagger{
				SecurityDefinitions: sequencedmap.New(
					sequencedmap.NewElem("oauth", &swagger.SecurityScheme{Type: swagger.SecuritySchemeTypeOAuth2}),
					sequencedmap.NewElem("apiKey", &swagger.SecurityScheme{Type: swagger.SecuritySchemeTypeAPIKey}),
					sequencedmap.NewElem("basic", &swagger.SecurityScheme{Type: swagger.SecuritySchemeTypeBasic}),
				),
			}

			var allErrors []error
			validateErrs := secReq.Validate(t.Context(), validation.WithContextObject(doc))
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
