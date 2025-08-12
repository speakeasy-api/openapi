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

func TestExample_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid example with all fields",
			yml: `
summary: Example of a pet
description: A pet object example
value:
  id: 1
  name: doggie
  status: available
x-test: some-value
`,
		},
		{
			name: "valid example with value only",
			yml: `
value:
  name: test
  id: 123
`,
		},
		{
			name: "valid example with external value only",
			yml: `
externalValue: https://example.com/examples/user.json
`,
		},
		{
			name: "valid example with summary and description",
			yml: `
summary: User example
description: An example user object
value:
  username: johndoe
  email: john@example.com
`,
		},
		{
			name: "valid example with complex value",
			yml: `
summary: Complex object
value:
  user:
    id: 1
    profile:
      name: John
      settings:
        theme: dark
  metadata:
    created: "2023-01-01"
`,
		},
		{
			name: "valid example with string value",
			yml: `
summary: String example
value: "Hello World"
`,
		},
		{
			name: "valid example with number value",
			yml: `
summary: Number example
value: 42
`,
		},
		{
			name: "valid example with boolean value",
			yml: `
summary: Boolean example
value: true
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var example openapi.Example
			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &example)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := example.Validate(context.Background())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, example.Valid, "expected example to be valid")
		})
	}
}

func TestExample_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "invalid external value URL",
			yml: `
summary: Example with invalid URL
externalValue: ":invalid"
`,
			wantErrs: []string{"[3:16] example field externalValue is not a valid uri: parse \":invalid\": missing protocol scheme"},
		},
		{
			name: "invalid external value URL with spaces",
			yml: `
externalValue: ":invalid url"
`,
			wantErrs: []string{"[2:16] example field externalValue is not a valid uri: parse \":invalid url\": missing protocol scheme"},
		},
		{
			name: "both value and external value provided",
			yml: `
summary: Invalid example
value: "test"
externalValue: "https://example.com/test.json"
`,
			wantErrs: []string{"[3:8] example field value and externalValue are mutually exclusive"},
		},
		{
			name: "multiple validation errors",
			yml: `
value: "test"
externalValue: ":invalid"
`,
			wantErrs: []string{
				"[2:8] example field value and externalValue are mutually exclusive",
				"[3:16] example field externalValue is not a valid uri: parse \":invalid\": missing protocol scheme",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var example openapi.Example
			validationErrs, err := marshaller.Unmarshal(context.Background(), bytes.NewBuffer([]byte(tt.yml)), &example)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := example.Validate(context.Background())
			require.NotEmpty(t, errs, "expected validation errors")
			require.False(t, example.Valid, "expected example to be invalid")

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
