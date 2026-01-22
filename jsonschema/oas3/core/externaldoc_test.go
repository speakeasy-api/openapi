package core

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalDocumentation_Unmarshal_AllFields_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "all fields populated",
			yaml: `
url: https://example.com/docs
description: Additional documentation
x-custom: value
`,
		},
		{
			name: "only required url field",
			yaml: `
url: https://example.com
`,
		},
		{
			name: "url with description",
			yaml: `
url: https://api.example.com/reference
description: API Reference Documentation
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target ExternalDocumentation
			validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &target)

			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should not have validation errors")
			assert.NotNil(t, target, "ExternalDocumentation should not be nil")
		})
	}
}

func TestExternalDocumentation_Unmarshal_URLField_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		yaml        string
		expectedURL string
	}{
		{
			name:        "https url",
			yaml:        `url: https://example.com/docs`,
			expectedURL: "https://example.com/docs",
		},
		{
			name:        "http url",
			yaml:        `url: http://example.com/docs`,
			expectedURL: "http://example.com/docs",
		},
		{
			name:        "url with path",
			yaml:        `url: https://api.example.com/v1/reference`,
			expectedURL: "https://api.example.com/v1/reference",
		},
		{
			name:        "url with query params",
			yaml:        `url: https://example.com/docs?version=2.0`,
			expectedURL: "https://example.com/docs?version=2.0",
		},
		{
			name:        "url with fragment",
			yaml:        `url: https://example.com/docs#section`,
			expectedURL: "https://example.com/docs#section",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target ExternalDocumentation
			validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &target)

			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should not have validation errors")
			assert.Equal(t, tt.expectedURL, target.URL.Value, "should parse url correctly")
		})
	}
}

func TestExternalDocumentation_Unmarshal_DescriptionField_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		yaml                string
		expectedDescription string
	}{
		{
			name: "simple description",
			yaml: `
url: https://example.com
description: Documentation
`,
			expectedDescription: "Documentation",
		},
		{
			name: "multi-word description",
			yaml: `
url: https://example.com
description: Complete API documentation and reference guide
`,
			expectedDescription: "Complete API documentation and reference guide",
		},
		{
			name: "description with special chars",
			yaml: `
url: https://example.com
description: "Documentation: API & SDK Guide"
`,
			expectedDescription: "Documentation: API & SDK Guide",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target ExternalDocumentation
			validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &target)

			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should not have validation errors")
			require.NotNil(t, target.Description.Value, "description should be set")
			assert.Equal(t, tt.expectedDescription, *target.Description.Value, "should parse description correctly")
		})
	}
}

func TestExternalDocumentation_Unmarshal_Extensions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		yaml          string
		extensionKey  string
		expectedValue string
	}{
		{
			name: "single extension",
			yaml: `
url: https://example.com
x-custom: value
`,
			extensionKey:  "x-custom",
			expectedValue: "value",
		},
		{
			name: "multiple extensions",
			yaml: `
url: https://example.com
x-first: value1
x-second: value2
`,
			extensionKey:  "x-first",
			expectedValue: "value1",
		},
		{
			name: "extension with url and description",
			yaml: `
url: https://example.com/docs
description: API docs
x-vendor: custom-value
`,
			extensionKey:  "x-vendor",
			expectedValue: "custom-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target ExternalDocumentation
			validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &target)

			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should not have validation errors")
			require.NotNil(t, target.Extensions, "extensions should be set")

			ext, found := target.Extensions.Get(tt.extensionKey)
			require.True(t, found, "should find extension")
			assert.Equal(t, tt.expectedValue, ext.Value.Value, "should parse extension value correctly")
		})
	}
}

func TestExternalDocumentation_Unmarshal_MinimalObject_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	yaml := `url: https://example.com`

	var target ExternalDocumentation
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, yaml), &target)

	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "should not have validation errors")
	assert.Equal(t, "https://example.com", target.URL.Value, "should parse url")
	assert.Nil(t, target.Description.Value, "description should be nil")
}
