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

func TestOperation_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid minimal operation",
			yml: `
responses:
  "200":
    description: Success
`,
		},
		{
			name: "valid operation with all fields",
			yml: `
operationId: getUserById
summary: Get user by ID
description: Retrieves a user by their unique identifier
tags:
  - users
  - accounts
deprecated: false
servers:
  - url: https://api.example.com/v1
    description: Production server
parameters:
  - name: userId
    in: path
    required: true
    schema:
      type: string
requestBody:
  description: User data
  required: true
  content:
    application/json:
      schema:
        type: object
responses:
  "200":
    description: User found
    content:
      application/json:
        schema:
          type: object
  "404":
    description: User not found
callbacks:
  userCreated:
    "{$request.body#/callbackUrl}":
      post:
        responses:
          "200":
            description: Callback received
externalDocs:
  description: More info
  url: https://example.com/docs
x-test: some-value
`,
		},
		{
			name: "valid operation with deprecated flag",
			yml: `
deprecated: true
responses:
  "200":
    description: Success (deprecated)
`,
		},
		{
			name: "valid operation with complex parameters",
			yml: `
parameters:
  - name: limit
    in: query
    schema:
      type: integer
      minimum: 1
      maximum: 100
  - name: offset
    in: query
    schema:
      type: integer
      minimum: 0
responses:
  "200":
    description: Success
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var operation openapi.Operation
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &operation)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := operation.Validate(t.Context(), validation.WithContextObject(openapi.NewOpenAPI()))
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, operation.Valid, "expected operation to be valid")
		})
	}
}

func TestOperation_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "invalid external docs URL",
			yml: `
responses:
  "200":
    description: Success
externalDocs:
  description: Invalid docs
  url: ":invalid"
`,
			wantErrs: []string{"[7:8] externalDocumentation.url is not a valid uri: parse \":invalid\": missing protocol scheme"},
		},
		{
			name: "invalid server URL",
			yml: `
responses:
  "200":
    description: Success
servers:
  - url: ":invalid"
    description: Invalid server
`,
			wantErrs: []string{"[6:10] server.url is not a valid uri: parse \":invalid\": missing protocol scheme"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var operation openapi.Operation
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &operation)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := operation.Validate(t.Context())
			require.NotEmpty(t, errs, "expected validation errors")
			require.False(t, operation.Valid, "expected operation to be invalid")

			// Check that all expected error messages are present
			var errMessages []string
			for _, err := range errs {
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
