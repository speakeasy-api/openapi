package swagger_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/swagger"
	"github.com/stretchr/testify/require"
)

func TestTag_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_tag_with_name_only",
			yml: `name: users
description: User operations`,
		},
		{
			name: "valid_tag_with_external_docs",
			yml: `name: pets
description: Pet operations
externalDocs:
  description: Find more info here
  url: https://example.com/docs`,
		},
		{
			name: "valid_tag_minimal",
			yml:  `name: minimal`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var tag swagger.Tag

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &tag)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := tag.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
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
			name:     "missing_name",
			yml:      `description: Some description`,
			wantErrs: []string{"tag.name is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var tag swagger.Tag

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &tag)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := tag.Validate(t.Context())
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

func TestExternalDocumentation_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "valid_external_docs_with_description",
			yml: `description: Find more info here
url: https://example.com/docs`,
		},
		{
			name: "valid_external_docs_minimal",
			yml:  `url: https://example.com`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var externalDocs swagger.ExternalDocumentation

			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &externalDocs)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := externalDocs.Validate(t.Context())
			require.Empty(t, errs, "Expected no validation errors")
		})
	}
}

func TestExternalDocumentation_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		wantErrs []string
	}{
		{
			name:     "missing_url",
			yml:      `description: Some description`,
			wantErrs: []string{"externalDocumentation.url is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var externalDocs swagger.ExternalDocumentation

			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &externalDocs)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := externalDocs.Validate(t.Context())
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

func TestTag_Getters_Success(t *testing.T) {
	t.Parallel()

	yml := `name: users
description: User operations
externalDocs:
  url: https://example.com/docs
x-custom: value
`
	var tag swagger.Tag

	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &tag)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	require.Equal(t, "users", tag.GetName(), "GetName should return correct value")
	require.Equal(t, "User operations", tag.GetDescription(), "GetDescription should return correct value")
	require.NotNil(t, tag.GetExternalDocs(), "GetExternalDocs should return non-nil")
	require.NotNil(t, tag.GetExtensions(), "GetExtensions should return non-nil")
}

func TestTag_Getters_Nil(t *testing.T) {
	t.Parallel()

	var tag *swagger.Tag

	require.Empty(t, tag.GetName(), "GetName should return empty string for nil")
	require.Empty(t, tag.GetDescription(), "GetDescription should return empty string for nil")
	require.Nil(t, tag.GetExternalDocs(), "GetExternalDocs should return nil for nil tag")
	require.NotNil(t, tag.GetExtensions(), "GetExtensions should return empty extensions for nil tag")
}
