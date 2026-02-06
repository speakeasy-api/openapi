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

func TestExternalDoc_Validate_Success(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			var extDoc oas3.ExternalDocumentation
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &extDoc)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := extDoc.Validate(t.Context())
			require.Empty(t, errs, "expected no validation errors")
			require.True(t, extDoc.Valid, "expected external doc to be valid")
		})
	}
}

func TestExternalDoc_Validate_Error(t *testing.T) {
	t.Parallel()

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
			wantErrs: []string{"[2:1] error validation-required-field `externalDocumentation.url` is required"},
		},
		{
			name: "empty URL",
			yml: `
description: Some documentation
url: ""
`,
			wantErrs: []string{"[3:6] error validation-required-field `externalDocumentation.url` is required"},
		},
		{
			name: "invalid URL format",
			yml: `
description: Some documentation
url: ":invalid"
`,
			wantErrs: []string{" externalDocumentation.url is not a valid uri"},
		},
		{
			name: "invalid URL with spaces",
			yml: `
description: Some documentation
url: ":invalid url"
`,
			wantErrs: []string{" externalDocumentation.url is not a valid uri"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var extDoc oas3.ExternalDocumentation

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &extDoc)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := extDoc.Validate(t.Context())
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

func TestExternalDocumentation_GetDescription_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		extDoc   *oas3.ExternalDocumentation
		expected string
	}{
		{
			name:     "nil extDoc returns empty",
			extDoc:   nil,
			expected: "",
		},
		{
			name:     "nil description returns empty",
			extDoc:   &oas3.ExternalDocumentation{},
			expected: "",
		},
		{
			name:     "returns description",
			extDoc:   &oas3.ExternalDocumentation{Description: pointer.From("Test docs")},
			expected: "Test docs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.extDoc.GetDescription())
		})
	}
}

func TestExternalDocumentation_GetURL_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		extDoc   *oas3.ExternalDocumentation
		expected string
	}{
		{
			name:     "nil extDoc returns empty",
			extDoc:   nil,
			expected: "",
		},
		{
			name:     "returns URL",
			extDoc:   &oas3.ExternalDocumentation{URL: "https://example.com"},
			expected: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.extDoc.GetURL())
		})
	}
}

func TestExternalDocumentation_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		extDoc      *oas3.ExternalDocumentation
		expectEmpty bool
	}{
		{
			name:        "nil extDoc returns empty extensions",
			extDoc:      nil,
			expectEmpty: true,
		},
		{
			name:        "nil extensions returns empty extensions",
			extDoc:      &oas3.ExternalDocumentation{},
			expectEmpty: true,
		},
		{
			name:        "returns extensions",
			extDoc:      &oas3.ExternalDocumentation{Extensions: extensions.New()},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.extDoc.GetExtensions()
			assert.NotNil(t, result)
			if tt.expectEmpty {
				assert.Equal(t, 0, result.Len())
			}
		})
	}
}

func TestExternalDocumentation_IsEqual_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        *oas3.ExternalDocumentation
		b        *oas3.ExternalDocumentation
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
			b:        &oas3.ExternalDocumentation{URL: "https://example.com"},
			expected: false,
		},
		{
			name:     "a not nil b nil returns false",
			a:        &oas3.ExternalDocumentation{URL: "https://example.com"},
			b:        nil,
			expected: false,
		},
		{
			name:     "equal URL returns true",
			a:        &oas3.ExternalDocumentation{URL: "https://example.com"},
			b:        &oas3.ExternalDocumentation{URL: "https://example.com"},
			expected: true,
		},
		{
			name:     "different URL returns false",
			a:        &oas3.ExternalDocumentation{URL: "https://example.com"},
			b:        &oas3.ExternalDocumentation{URL: "https://other.com"},
			expected: false,
		},
		{
			name:     "equal description returns true",
			a:        &oas3.ExternalDocumentation{URL: "https://example.com", Description: pointer.From("desc")},
			b:        &oas3.ExternalDocumentation{URL: "https://example.com", Description: pointer.From("desc")},
			expected: true,
		},
		{
			name:     "different description returns false",
			a:        &oas3.ExternalDocumentation{URL: "https://example.com", Description: pointer.From("desc1")},
			b:        &oas3.ExternalDocumentation{URL: "https://example.com", Description: pointer.From("desc2")},
			expected: false,
		},
		{
			name:     "one nil description returns false",
			a:        &oas3.ExternalDocumentation{URL: "https://example.com", Description: pointer.From("desc")},
			b:        &oas3.ExternalDocumentation{URL: "https://example.com"},
			expected: false,
		},
		{
			name:     "both nil extensions returns true",
			a:        &oas3.ExternalDocumentation{URL: "https://example.com"},
			b:        &oas3.ExternalDocumentation{URL: "https://example.com"},
			expected: true,
		},
		{
			name:     "a nil extensions b not nil returns false",
			a:        &oas3.ExternalDocumentation{URL: "https://example.com"},
			b:        &oas3.ExternalDocumentation{URL: "https://example.com", Extensions: extensions.New()},
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
