package openapi_test

import (
	"bytes"
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
		{
			name: "valid tag with new 3.2 fields",
			yml: `
name: products
summary: Products
description: All product-related operations
parent: catalog
kind: nav
`,
		},
		{
			name: "valid tag with registered kind values",
			yml: `
name: user-badge
summary: User Badge
kind: badge
`,
		},
		{
			name: "valid tag with custom kind value",
			yml: `
name: custom-tag
summary: Custom Tag
kind: custom-lifecycle
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var tag openapi.Tag
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &tag)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			errs := tag.Validate(t.Context())
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
			wantErrs: []string{"[2:1] tag.name is missing"},
		},
		{
			name: "empty name",
			yml: `
name: ""
description: A tag with empty name
`,
			wantErrs: []string{"[2:7] tag.name is required"},
		},
		{
			name: "invalid external docs URL",
			yml: `
name: test
externalDocs:
  url: ":invalid"
`,
			wantErrs: []string{"[4:8] externalDocumentation.url is not a valid uri: parse \":invalid\": missing protocol scheme"},
		},
		{
			name: "external docs without URL",
			yml: `
name: test
externalDocs:
  description: Documentation without URL
`,
			wantErrs: []string{"[4:3] externalDocumentation.url is missing"},
		},
		{
			name: "multiple validation errors",
			yml: `
name: ""
externalDocs:
  url: ":invalid"
`,
			wantErrs: []string{
				"[2:7] tag.name is required",
				"[4:8] externalDocumentation.url is not a valid uri: parse \":invalid\": missing protocol scheme",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var tag openapi.Tag

			// Collect all errors from both unmarshalling and validation
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &tag)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			validateErrs := tag.Validate(t.Context())
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

func TestTag_ValidateWithTags_ParentRelationships_Success(t *testing.T) {
	t.Parallel()

	// Create a hierarchy of tags: catalog -> products -> books
	catalogTag := &openapi.Tag{Name: "catalog"}
	productsTag := &openapi.Tag{Name: "products", Parent: &[]string{"catalog"}[0]}
	booksTag := &openapi.Tag{Name: "books", Parent: &[]string{"products"}[0]}
	standaloneTag := &openapi.Tag{Name: "standalone"}

	allTags := []*openapi.Tag{catalogTag, productsTag, booksTag, standaloneTag}

	for _, tag := range allTags {
		errs := tag.ValidateWithTags(t.Context(), allTags)
		require.Empty(t, errs, "expected no validation errors for tag %s", tag.Name)
	}
}

func TestTag_ValidateWithTags_ParentNotFound_Error(t *testing.T) {
	t.Parallel()

	// Create a tag with a non-existent parent
	tag := &openapi.Tag{Name: "orphan", Parent: &[]string{"nonexistent"}[0]}
	allTags := []*openapi.Tag{tag}

	errs := tag.ValidateWithTags(t.Context(), allTags)
	require.NotEmpty(t, errs, "expected validation errors")

	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "parent tag 'nonexistent' does not exist") {
			found = true
			break
		}
	}
	require.True(t, found, "expected parent not found error")
}

func TestTag_ValidateWithTags_CircularReference_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tags []*openapi.Tag
		desc string
	}{
		{
			name: "direct circular reference",
			tags: []*openapi.Tag{
				{Name: "tag1", Parent: &[]string{"tag1"}[0]}, // Self-reference
			},
			desc: "tag references itself",
		},
		{
			name: "two-tag circular reference",
			tags: []*openapi.Tag{
				{Name: "tag1", Parent: &[]string{"tag2"}[0]},
				{Name: "tag2", Parent: &[]string{"tag1"}[0]},
			},
			desc: "tag1 -> tag2 -> tag1",
		},
		{
			name: "three-tag circular reference",
			tags: []*openapi.Tag{
				{Name: "tag1", Parent: &[]string{"tag2"}[0]},
				{Name: "tag2", Parent: &[]string{"tag3"}[0]},
				{Name: "tag3", Parent: &[]string{"tag1"}[0]},
			},
			desc: "tag1 -> tag2 -> tag3 -> tag1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Check each tag that has a parent
			for _, tag := range tt.tags {
				if tag.Parent != nil {
					errs := tag.ValidateWithTags(t.Context(), tt.tags)
					require.NotEmpty(t, errs, "expected validation errors for %s", tt.desc)

					found := false
					for _, err := range errs {
						if strings.Contains(err.Error(), "circular parent reference") {
							found = true
							break
						}
					}
					require.True(t, found, "expected circular reference error for %s", tt.desc)
				}
			}
		})
	}
}

func TestTag_ValidateWithTags_ComplexHierarchy_Success(t *testing.T) {
	t.Parallel()

	// Create a complex but valid hierarchy
	// catalog
	// ├── products
	// │   ├── books
	// │   └── cds
	// └── services
	//     └── delivery

	catalogTag := &openapi.Tag{Name: "catalog", Kind: &[]string{"nav"}[0]}
	productsTag := &openapi.Tag{Name: "products", Parent: &[]string{"catalog"}[0], Kind: &[]string{"nav"}[0]}
	booksTag := &openapi.Tag{Name: "books", Parent: &[]string{"products"}[0], Kind: &[]string{"nav"}[0]}
	cdsTag := &openapi.Tag{Name: "cds", Parent: &[]string{"products"}[0], Kind: &[]string{"nav"}[0]}
	servicesTag := &openapi.Tag{Name: "services", Parent: &[]string{"catalog"}[0], Kind: &[]string{"nav"}[0]}
	deliveryTag := &openapi.Tag{Name: "delivery", Parent: &[]string{"services"}[0], Kind: &[]string{"badge"}[0]}

	allTags := []*openapi.Tag{catalogTag, productsTag, booksTag, cdsTag, servicesTag, deliveryTag}

	for _, tag := range allTags {
		errs := tag.ValidateWithTags(t.Context(), allTags)
		require.Empty(t, errs, "expected no validation errors for tag %s", tag.Name)
	}
}

func TestTagKind_Registry_Success(t *testing.T) {
	t.Parallel()

	// Test registered kinds
	registeredKinds := openapi.GetRegisteredTagKinds()
	require.Len(t, registeredKinds, 3)
	require.Contains(t, registeredKinds, openapi.TagKindNav)
	require.Contains(t, registeredKinds, openapi.TagKindBadge)
	require.Contains(t, registeredKinds, openapi.TagKindAudience)

	// Test kind validation
	require.True(t, openapi.TagKindNav.IsRegistered())
	require.True(t, openapi.TagKindBadge.IsRegistered())
	require.True(t, openapi.TagKindAudience.IsRegistered())
	require.False(t, openapi.TagKind("custom").IsRegistered())

	// Test descriptions
	require.NotEmpty(t, openapi.GetTagKindDescription(openapi.TagKindNav))
	require.NotEmpty(t, openapi.GetTagKindDescription(openapi.TagKindBadge))
	require.NotEmpty(t, openapi.GetTagKindDescription(openapi.TagKindAudience))
	require.Contains(t, openapi.GetTagKindDescription("custom"), "not in the official registry")
}
