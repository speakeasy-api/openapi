package openapi_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/require"
)

func TestTag_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid tag with all fields",
			yml: `
name: pets
description: Everything about your pets
externalDocs:
  description: Find out more
  url: https://example.com/pets
x-test: some-value
`,
		},
		{
			name: "valid tag with name only",
			yml: `
name: users
`,
		},
		{
			name: "valid tag with name and description",
			yml: `
name: orders
description: Access to Petstore orders
`,
		},
		{
			name: "valid tag with name and external docs",
			yml: `
name: store
externalDocs:
  url: https://example.com/store
`,
		},
		{
			name: "valid tag with complex external docs",
			yml: `
name: admin
description: Administrative operations
externalDocs:
  description: Admin documentation
  url: https://admin.example.com/docs
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var tag openapi.Tag
			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &tag)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := tag.Validate(context.Background())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, tag.Valid, "expected tag to be valid")
		})
	}
}

func TestTag_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "missing name",
			yml: `
description: A tag without name
`,
			wantErrs: []string{"[2:1] tag field name is missing"},
		},
		{
			name: "empty name",
			yml: `
name: ""
description: A tag with empty name
`,
			wantErrs: []string{"[2:7] tag field name is required"},
		},
		{
			name: "invalid external docs URL",
			yml: `
name: test
externalDocs:
  url: ":invalid"
`,
			wantErrs: []string{"[4:8] externalDocumentation field url is not a valid uri: parse \":invalid\": missing protocol scheme"},
		},
		{
			name: "external docs without URL",
			yml: `
name: test
externalDocs:
  description: Documentation without URL
`,
			wantErrs: []string{"[4:3] externalDocumentation field url is missing"},
		},
		{
			name: "multiple validation errors",
			yml: `
name: ""
externalDocs:
  url: ":invalid"
`,
			wantErrs: []string{
				"[2:7] tag field name is required",
				"[4:8] externalDocumentation field url is not a valid uri: parse \":invalid\": missing protocol scheme",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var tag openapi.Tag

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &tag)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := tag.Validate(context.Background())
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
