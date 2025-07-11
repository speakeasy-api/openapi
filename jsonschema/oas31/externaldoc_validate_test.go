package oas31_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas31"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/require"
)

func TestExternalDoc_Validate_Success(t *testing.T) {
	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid external doc with all fields",
			yml: `
description: Additional documentation
url: https://example.com/docs
x-test: some-value
`,
		},
		{
			name: "valid external doc with URL only",
			yml: `
url: https://example.com/docs
`,
		},
		{
			name: "valid external doc with description and URL",
			yml: `
description: API documentation
url: https://api.example.com/docs
`,
		},
		{
			name: "valid external doc with HTTP URL",
			yml: `
description: Documentation
url: http://example.com/docs
`,
		},
		{
			name: "valid external doc with complex URL",
			yml: `
description: API Reference
url: https://api.example.com/v1/docs?section=reference
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var extDoc oas31.ExternalDocumentation
			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &extDoc)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := extDoc.Validate(context.Background())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, extDoc.Valid, "expected external doc to be valid")
		})
	}
}

func TestExternalDoc_Validate_Error(t *testing.T) {
	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "missing URL",
			yml: `
description: Some documentation
`,
			wantErrs: []string{"[2:1] field url is missing"},
		},
		{
			name: "empty URL",
			yml: `
description: Some documentation
url: ""
`,
			wantErrs: []string{"[3:6] url is required"},
		},
		{
			name: "invalid URL format",
			yml: `
description: Some documentation
url: ":invalid"
`,
			wantErrs: []string{"url is not a valid uri"},
		},
		{
			name: "invalid URL with spaces",
			yml: `
description: Some documentation
url: ":invalid url"
`,
			wantErrs: []string{"url is not a valid uri"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var extDoc oas31.ExternalDocumentation

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &extDoc)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := extDoc.Validate(context.Background())
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
