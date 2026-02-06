package swagger_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/swagger"
	"github.com/stretchr/testify/require"
)

func TestOperation_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "minimal_valid_operation",
			yml: `responses:
  200:
    description: Success`,
		},
		{
			name: "complete_operation",
			yml: `summary: Get users
description: Retrieve a list of users
operationId: getUsers
tags:
  - users
consumes:
  - application/json
produces:
  - application/json
parameters:
  - name: limit
    in: query
    type: integer
responses:
  200:
    description: Success
  404:
    description: Not found`,
		},
		{
			name: "operation_with_schemes",
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

			var operation swagger.Operation

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &operation)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := operation.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
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
			name:     "missing_responses",
			yml:      `summary: Test operation`,
			wantErrs: []string{"`operation.responses` is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var operation swagger.Operation

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &operation)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := operation.Validate(t.Context())
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
