package openapi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/require"
)

func TestCallback_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_empty_callback",
			yml:  `{}`,
		},
		{
			name: "valid_single_expression",
			yml: `
'{$request.body#/webhookUrl}':
  post:
    summary: Webhook notification
    requestBody:
      content:
        application/json:
          schema:
            type: object
    responses:
      '200':
        description: Webhook received
`,
		},
		{
			name: "valid_multiple_expressions",
			yml: `
'{$request.body#/webhookUrl}':
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
'{$request.body#/callbackUrl}':
  put:
    summary: Callback notification
    responses:
      '200':
        description: Callback received
`,
		},
		{
			name: "valid_with_extensions",
			yml: `
'{$request.body#/webhookUrl}':
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
x-custom: value
x-timeout: 30
`,
		},
		{
			name: "valid_complex_expression",
			yml: `
'{$request.body#/webhookUrl}?event={$request.body#/eventType}':
  post:
    summary: Event webhook
    responses:
      '200':
        description: Event received
      '400':
        description: Bad request
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var callback openapi.Callback

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &callback)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := callback.Validate(t.Context(), validation.WithContextObject(openapi.NewOpenAPI()))
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestCallback_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "invalid_expression_not_starting_with_dollar",
			yml: `
'request.body#/webhookUrl':
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
`,
			wantErrs: []string{"expression is not valid, must begin with $"},
		},
		{
			name: "invalid_expression_unknown_type",
			yml: `
'{$unknown.body#/webhookUrl}':
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
`,
			wantErrs: []string{"expression is not valid, must begin with one of [url, method, statusCode, request, response, inputs, outputs, steps, workflows, sourceDescriptions, components]"},
		},
		{
			name: "invalid_expression_url_with_extra_parts",
			yml: `
'{$url.extra}':
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
`,
			wantErrs: []string{"expression is not valid, extra characters after $url"},
		},
		{
			name: "invalid_expression_request_without_reference",
			yml: `
'{$request}':
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
`,
			wantErrs: []string{"expression is not valid, expected one of [header, query, path, body] after $request"},
		},
		{
			name: "invalid_expression_request_unknown_reference",
			yml: `
'{$request.unknown}':
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
`,
			wantErrs: []string{"expression is not valid, expected one of [header, query, path, body] after $request"},
		},
		{
			name: "invalid_expression_request_header_missing_token",
			yml: `
'{$request.header}':
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
`,
			wantErrs: []string{"expression is not valid, expected token after $request.header"},
		},
		{
			name: "invalid_expression_request_header_invalid_token",
			yml: `
"{$request.header.some@header}":
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
`,
			wantErrs: []string{"header reference must be a valid token"},
		},
		{
			name: "invalid_expression_request_query_missing_name",
			yml: `
'{$request.query}':
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
`,
			wantErrs: []string{"expression is not valid, expected name after $request.query"},
		},
		{
			name: "invalid_expression_request_path_missing_name",
			yml: `
'{$request.path}':
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
`,
			wantErrs: []string{"expression is not valid, expected name after $request.path"},
		},
		{
			name: "invalid_expression_request_body_with_extra_parts",
			yml: `
'{$request.body.extra}':
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
`,
			wantErrs: []string{"expression is not valid, only json pointers are allowed after $request.body"},
		},
		{
			name: "invalid_expression_invalid_json_pointer",
			yml: `
"{$request.body#some/path}":
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
`,
			wantErrs: []string{"jsonpointer must start with /"},
		},
		{
			name: "invalid_nested_pathitem_invalid_server",
			yml: `
'{$request.body#/webhookUrl}':
  servers:
    - description: Invalid server without URL
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
`,
			wantErrs: []string{"field url is missing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var callback openapi.Callback

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &callback)
			require.NoError(t, err)

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			allErrors = append(allErrors, validationErrs...)

			validateErrs := callback.Validate(t.Context(), validation.WithContextObject(openapi.NewOpenAPI()))
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

func TestCallback_Validate_ComplexExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_request_header_expression",
			yml: `
'{$request.header.Authorization}':
  post:
    summary: Webhook with auth header
    responses:
      '200':
        description: Webhook received
`,
		},
		{
			name: "valid_request_query_expression",
			yml: `
'{$request.query.callback_url}':
  post:
    summary: Webhook with query param
    responses:
      '200':
        description: Webhook received
`,
		},
		{
			name: "valid_request_path_expression",
			yml: `
'{$request.path.userId}':
  post:
    summary: Webhook with path param
    responses:
      '200':
        description: Webhook received
`,
		},
		{
			name: "valid_response_body_expression",
			yml: `
'{$response.body#/callbackUrl}':
  post:
    summary: Webhook from response body
    responses:
      '200':
        description: Webhook received
`,
		},
		{
			name: "valid_response_header_expression",
			yml: `
'{$response.header.Location}':
  post:
    summary: Webhook from response header
    responses:
      '200':
        description: Webhook received
`,
		},
		{
			name: "valid_url_method_statuscode_expressions",
			yml: `
'{$url}':
  post:
    summary: Webhook to request URL
    responses:
      '200':
        description: Webhook received
'{$method}':
  post:
    summary: Webhook with request method
    responses:
      '200':
        description: Webhook received
'{$statusCode}':
  post:
    summary: Webhook with status code
    responses:
      '200':
        description: Webhook received
`,
		},
		{
			name: "valid_complex_json_pointer",
			yml: `
'{$request.body#/webhook/config/url}':
  post:
    summary: Webhook with nested JSON pointer
    responses:
      '200':
        description: Webhook received
`,
		},
		{
			name: "valid_expression_with_query_params",
			yml: `
'{$request.body#/webhookUrl}?event={$request.body#/eventType}&source=api':
  post:
    summary: Webhook with query parameters
    responses:
      '200':
        description: Webhook received
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var callback openapi.Callback

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &callback)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := callback.Validate(t.Context(), validation.WithContextObject(openapi.NewOpenAPI()))
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestCallback_Validate_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_empty_callback_with_extensions_only",
			yml: `
x-custom: value
x-timeout: 30
`,
		},
		{
			name: "valid_callback_with_mixed_expressions_and_extensions",
			yml: `
'{$request.body#/webhookUrl}':
  post:
    summary: Webhook notification
    responses:
      '200':
        description: Webhook received
'{$response.header.Location}':
  put:
    summary: Location callback
    responses:
      '200':
        description: Callback received
x-custom: value
x-rate-limit: 100
`,
		},
		{
			name: "valid_callback_with_all_http_methods",
			yml: `
'{$request.body#/webhookUrl}':
  get:
    summary: GET webhook
    responses:
      '200':
        description: Success
  post:
    summary: POST webhook
    responses:
      '201':
        description: Created
  put:
    summary: PUT webhook
    responses:
      '200':
        description: Updated
  patch:
    summary: PATCH webhook
    responses:
      '200':
        description: Patched
  delete:
    summary: DELETE webhook
    responses:
      '204':
        description: Deleted
  head:
    summary: HEAD webhook
    responses:
      '200':
        description: Headers
  options:
    summary: OPTIONS webhook
    responses:
      '200':
        description: Options
  trace:
    summary: TRACE webhook
    responses:
      '200':
        description: Trace
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var callback openapi.Callback

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &callback)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := callback.Validate(t.Context(), validation.WithContextObject(openapi.NewOpenAPI()))
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}
