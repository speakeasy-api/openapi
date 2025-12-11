package oas3_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
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

func TestXML_GetName_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		xml      *oas3.XML
		expected string
	}{
		{
			name:     "nil xml returns empty",
			xml:      nil,
			expected: "",
		},
		{
			name:     "returns name",
			xml:      &oas3.XML{Name: pointer.From("element")},
			expected: "element",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.xml.GetName())
		})
	}
}

func TestXML_GetNamespace_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		xml      *oas3.XML
		expected string
	}{
		{
			name:     "nil xml returns empty",
			xml:      nil,
			expected: "",
		},
		{
			name:     "returns namespace",
			xml:      &oas3.XML{Namespace: pointer.From("https://example.com")},
			expected: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.xml.GetNamespace())
		})
	}
}

func TestXML_GetPrefix_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		xml      *oas3.XML
		expected string
	}{
		{
			name:     "nil xml returns empty",
			xml:      nil,
			expected: "",
		},
		{
			name:     "returns prefix",
			xml:      &oas3.XML{Prefix: pointer.From("ex")},
			expected: "ex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.xml.GetPrefix())
		})
	}
}

func TestXML_GetAttribute_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		xml      *oas3.XML
		expected bool
	}{
		{
			name:     "nil xml returns false",
			xml:      nil,
			expected: false,
		},
		{
			name:     "returns true",
			xml:      &oas3.XML{Attribute: pointer.From(true)},
			expected: true,
		},
		{
			name:     "returns false",
			xml:      &oas3.XML{Attribute: pointer.From(false)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.xml.GetAttribute())
		})
	}
}

func TestXML_GetWrapped_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		xml      *oas3.XML
		expected bool
	}{
		{
			name:     "nil xml returns false",
			xml:      nil,
			expected: false,
		},
		{
			name:     "returns true",
			xml:      &oas3.XML{Wrapped: pointer.From(true)},
			expected: true,
		},
		{
			name:     "returns false",
			xml:      &oas3.XML{Wrapped: pointer.From(false)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.xml.GetWrapped())
		})
	}
}

func TestXML_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		xml         *oas3.XML
		expectEmpty bool
	}{
		{
			name:        "nil xml returns empty extensions",
			xml:         nil,
			expectEmpty: true,
		},
		{
			name:        "nil extensions returns empty extensions",
			xml:         &oas3.XML{},
			expectEmpty: true,
		},
		{
			name:        "returns extensions",
			xml:         &oas3.XML{Extensions: extensions.New()},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.xml.GetExtensions()
			assert.NotNil(t, result)
			if tt.expectEmpty {
				assert.Equal(t, 0, result.Len())
			}
		})
	}
}

func TestXML_IsEqual_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        *oas3.XML
		b        *oas3.XML
		expected bool
	}{
		{
			name:     "both nil returns true",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "a nil b not nil returns false",
			a:        nil,
			b:        &oas3.XML{Name: pointer.From("element")},
			expected: false,
		},
		{
			name:     "a not nil b nil returns false",
			a:        &oas3.XML{Name: pointer.From("element")},
			b:        nil,
			expected: false,
		},
		{
			name:     "equal name returns true",
			a:        &oas3.XML{Name: pointer.From("element")},
			b:        &oas3.XML{Name: pointer.From("element")},
			expected: true,
		},
		{
			name:     "different name returns false",
			a:        &oas3.XML{Name: pointer.From("element1")},
			b:        &oas3.XML{Name: pointer.From("element2")},
			expected: false,
		},
		{
			name:     "equal namespace returns true",
			a:        &oas3.XML{Namespace: pointer.From("https://example.com")},
			b:        &oas3.XML{Namespace: pointer.From("https://example.com")},
			expected: true,
		},
		{
			name:     "different namespace returns false",
			a:        &oas3.XML{Namespace: pointer.From("https://example1.com")},
			b:        &oas3.XML{Namespace: pointer.From("https://example2.com")},
			expected: false,
		},
		{
			name:     "equal prefix returns true",
			a:        &oas3.XML{Prefix: pointer.From("ex")},
			b:        &oas3.XML{Prefix: pointer.From("ex")},
			expected: true,
		},
		{
			name:     "different prefix returns false",
			a:        &oas3.XML{Prefix: pointer.From("ex1")},
			b:        &oas3.XML{Prefix: pointer.From("ex2")},
			expected: false,
		},
		{
			name:     "equal attribute returns true",
			a:        &oas3.XML{Attribute: pointer.From(true)},
			b:        &oas3.XML{Attribute: pointer.From(true)},
			expected: true,
		},
		{
			name:     "different attribute returns false",
			a:        &oas3.XML{Attribute: pointer.From(true)},
			b:        &oas3.XML{Attribute: pointer.From(false)},
			expected: false,
		},
		{
			name:     "equal wrapped returns true",
			a:        &oas3.XML{Wrapped: pointer.From(true)},
			b:        &oas3.XML{Wrapped: pointer.From(true)},
			expected: true,
		},
		{
			name:     "different wrapped returns false",
			a:        &oas3.XML{Wrapped: pointer.From(true)},
			b:        &oas3.XML{Wrapped: pointer.From(false)},
			expected: false,
		},
		{
			name:     "both nil extensions returns true",
			a:        &oas3.XML{Name: pointer.From("element")},
			b:        &oas3.XML{Name: pointer.From("element")},
			expected: true,
		},
		{
			name:     "one nil extensions returns false",
			a:        &oas3.XML{Name: pointer.From("element"), Extensions: extensions.New()},
			b:        &oas3.XML{Name: pointer.From("element")},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.a.IsEqual(tt.b))
		})
	}
}
