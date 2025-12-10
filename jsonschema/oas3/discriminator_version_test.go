package oas3_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscriminator_VersionAwareValidation_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		yml     string
		version string
		isValid bool
	}{
		{
			name: "OAS 3.1 - propertyName required",
			yml: `
propertyName: petType
mapping:
  cat: Cat
  dog: Dog
`,
			version: "3.1.0",
			isValid: true,
		},
		{
			name: "OAS 3.2 - propertyName with defaultMapping",
			yml: `
propertyName: petType
defaultMapping: OtherPet
mapping:
  cat: Cat
  dog: Dog
`,
			version: "3.2.0",
			isValid: true,
		},
		{
			name: "OAS 3.2 - propertyName without defaultMapping (property is required in schema)",
			yml: `
propertyName: petType
mapping:
  cat: Cat
  dog: Dog
`,
			version: "3.2.0",
			isValid: true,
		},
		{
			name: "OAS 3.1 - missing propertyName should fail",
			yml: `mapping:
  cat: Cat
  dog: Dog
`,
			version: "3.1.0",
			isValid: false,
		},
		{
			name: "OAS 3.2 - missing propertyName should also fail",
			yml: `defaultMapping: OtherPet
mapping:
  cat: Cat
  dog: Dog
`,
			version: "3.2.0",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var discriminator oas3.Discriminator

			// Collect all errors from unmarshalling
			var allErrors []error
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &discriminator)
			require.NoError(t, err)
			allErrors = append(allErrors, validationErrs...)

			// Create version context
			opts := []validation.Option{
				validation.WithContextObject(&oas3.ParentDocumentVersion{
					OpenAPI: &tt.version,
				}),
			}

			// Collect validation errors
			validateErrs := discriminator.Validate(t.Context(), opts...)
			allErrors = append(allErrors, validateErrs...)

			if tt.isValid {
				assert.Empty(t, allErrors, "expected no validation errors for version %s", tt.version)
				assert.True(t, discriminator.Valid, "expected discriminator to be valid for version %s", tt.version)
			} else {
				assert.NotEmpty(t, allErrors, "expected validation errors for version %s", tt.version)
				assert.False(t, discriminator.Valid, "expected discriminator to be invalid for version %s", tt.version)
			}
		})
	}
}

func TestDiscriminator_OpenAPI32_CompleteExample_Success(t *testing.T) {
	t.Parallel()

	// Full example from the spec showing optional propertyName with defaultMapping
	yml := `
propertyName: petType
defaultMapping: OtherPet
mapping:
  cat: Cat
  dog: Dog
  lizard: Lizard
x-custom-extension: value
`

	var discriminator oas3.Discriminator
	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &discriminator)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Validate with OpenAPI 3.2
	version := "3.2.0"
	opts := []validation.Option{
		validation.WithContextObject(&oas3.ParentDocumentVersion{
			OpenAPI: &version,
		}),
	}

	errs := discriminator.Validate(t.Context(), opts...)
	require.Empty(t, errs)
	require.True(t, discriminator.Valid)

	// Verify all fields
	assert.Equal(t, "petType", discriminator.GetPropertyName())
	assert.Equal(t, "OtherPet", discriminator.GetDefaultMapping())

	mapping := discriminator.GetMapping()
	require.NotNil(t, mapping)
	assert.Equal(t, 3, mapping.Len())

	cat, ok := mapping.Get("cat")
	assert.True(t, ok)
	assert.Equal(t, "Cat", cat)

	dog, ok := mapping.Get("dog")
	assert.True(t, ok)
	assert.Equal(t, "Dog", dog)

	lizard, ok := mapping.Get("lizard")
	assert.True(t, ok)
	assert.Equal(t, "Lizard", lizard)

	extensions := discriminator.GetExtensions()
	require.NotNil(t, extensions)

	ext, ok := extensions.Get("x-custom-extension")
	require.True(t, ok)
	assert.Equal(t, "value", ext.Value)
}

func TestDiscriminator_BackwardCompatibility_Success(t *testing.T) {
	t.Parallel()

	// Verify that existing OAS 3.1 discriminators still work
	yml := `
propertyName: petType
mapping:
  cat: "#/components/schemas/Cat"
  dog: "#/components/schemas/Dog"
`

	var discriminator oas3.Discriminator
	validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(yml), &discriminator)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	// Test with both 3.1 and 3.2 versions
	for _, version := range []string{"3.1.0", "3.2.0"} {
		t.Run("version_"+version, func(t *testing.T) {
			t.Parallel()

			opts := []validation.Option{
				validation.WithContextObject(&oas3.ParentDocumentVersion{
					OpenAPI: pointer.From(version),
				}),
			}

			errs := discriminator.Validate(t.Context(), opts...)
			assert.Empty(t, errs, "valid OAS 3.1 discriminator should work in version %s", version)
			assert.True(t, discriminator.Valid)
		})
	}
}
