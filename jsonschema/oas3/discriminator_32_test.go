package oas3_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscriminator_OpenAPI32_DefaultMapping_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		yml                string
		expectedDefaultMap string
	}{
		{
			name: "discriminator with propertyName and defaultMapping",
			yml: `
propertyName: petType
defaultMapping: "#/components/schemas/OtherPet"
mapping:
  cat: "#/components/schemas/Cat"
  dog: "#/components/schemas/Dog"
`,
			expectedDefaultMap: "#/components/schemas/OtherPet",
		},
		{
			name: "discriminator with defaultMapping using schema name",
			yml: `
propertyName: petType
defaultMapping: OtherPet
mapping:
  cat: Cat
  dog: Dog
`,
			expectedDefaultMap: "OtherPet",
		},
		{
			name: "discriminator with propertyName and defaultMapping but no mapping",
			yml: `
propertyName: petType
defaultMapping: DefaultPet
`,
			expectedDefaultMap: "DefaultPet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var discriminator oas3.Discriminator
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &discriminator)
			require.NoError(t, err)
			require.Empty(t, validationErrs)

			assert.Equal(t, tt.expectedDefaultMap, discriminator.GetDefaultMapping())
		})
	}
}

func TestDiscriminator_OpenAPI32_IsEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		disc1    *oas3.Discriminator
		disc2    *oas3.Discriminator
		expected bool
	}{
		{
			name: "equal discriminators with defaultMapping",
			disc1: &oas3.Discriminator{
				PropertyName:   "petType",
				DefaultMapping: pointer.From("OtherPet"),
			},
			disc2: &oas3.Discriminator{
				PropertyName:   "petType",
				DefaultMapping: pointer.From("OtherPet"),
			},
			expected: true,
		},
		{
			name: "different defaultMapping",
			disc1: &oas3.Discriminator{
				PropertyName:   "petType",
				DefaultMapping: pointer.From("OtherPet"),
			},
			disc2: &oas3.Discriminator{
				PropertyName:   "petType",
				DefaultMapping: pointer.From("DefaultPet"),
			},
			expected: false,
		},
		{
			name: "one with defaultMapping, one without",
			disc1: &oas3.Discriminator{
				PropertyName:   "petType",
				DefaultMapping: pointer.From("OtherPet"),
			},
			disc2: &oas3.Discriminator{
				PropertyName: "petType",
			},
			expected: false,
		},
		{
			name: "both with same propertyName and defaultMapping",
			disc1: &oas3.Discriminator{
				PropertyName:   "petType",
				DefaultMapping: pointer.From("OtherPet"),
			},
			disc2: &oas3.Discriminator{
				PropertyName:   "petType",
				DefaultMapping: pointer.From("OtherPet"),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.disc1.IsEqual(tt.disc2))
		})
	}
}
