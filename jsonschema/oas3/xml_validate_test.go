package oas3_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/require"
)

func TestXML_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid XML with all fields",
			yml: `
name: user
namespace: https://example.com/schema
prefix: ex
attribute: true
wrapped: false
x-test: some-value
`,
		},
		{
			name: "valid XML with name only",
			yml: `
name: user
`,
		},
		{
			name: "valid XML with namespace only",
			yml: `
namespace: https://example.com/schema
`,
		},
		{
			name: "valid XML with prefix only",
			yml: `
prefix: ex
`,
		},
		{
			name: "valid XML with boolean flags",
			yml: `
name: item
attribute: false
wrapped: true
`,
		},
		{
			name: "valid XML with absolute namespace URI",
			yml: `
name: element
namespace: http://www.w3.org/2001/XMLSchema
prefix: xs
`,
		},
		{
			name: "empty XML object",
			yml: `
name: ""
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var xml oas3.XML
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &xml)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := xml.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, xml.Valid, "expected XML to be valid")
		})
	}
}

func TestXML_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name: "invalid namespace URI - missing protocol",
			yml: `
name: user
namespace: "example.com/schema"
`,
			wantErrs: []string{"namespace must be an absolute uri"},
		},
		{
			name: "invalid namespace URI - malformed",
			yml: `
name: user
namespace: ":invalid"
`,
			wantErrs: []string{"namespace is not a valid uri: parse \":invalid\": missing protocol scheme"},
		},
		{
			name: "invalid namespace URI - relative path",
			yml: `
name: user
namespace: "/relative/path"
`,
			wantErrs: []string{"namespace must be an absolute uri"},
		},
		{
			name: "invalid namespace URI - with spaces",
			yml: `
name: user
namespace: ":invalid namespace"
`,
			wantErrs: []string{"namespace is not a valid uri: parse \":invalid namespace\": missing protocol scheme"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var xml oas3.XML
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &xml)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := xml.Validate(t.Context())
			require.NotEmpty(t, errs, "expected validation errors")
			require.False(t, xml.Valid, "expected XML to be invalid")

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
