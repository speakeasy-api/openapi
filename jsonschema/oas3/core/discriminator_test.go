package core

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscriminator_Unmarshal_AllFields_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "all fields populated",
			yaml: `
propertyName: petType
mapping:
  dog: "#/components/schemas/Dog"
  cat: "#/components/schemas/Cat"
defaultMapping: "#/components/schemas/Pet"
x-custom: value
`,
		},
		{
			name: "only required propertyName field",
			yaml: `
propertyName: type
`,
		},
		{
			name: "propertyName with mapping",
			yaml: `
propertyName: objectType
mapping:
  typeA: "#/components/schemas/TypeA"
  typeB: "#/components/schemas/TypeB"
`,
		},
		{
			name: "propertyName with defaultMapping",
			yaml: `
propertyName: kind
defaultMapping: "#/components/schemas/DefaultType"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target Discriminator
			validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &target)

			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should not have validation errors")
			assert.NotNil(t, target, "Discriminator should not be nil")
		})
	}
}

func TestDiscriminator_Unmarshal_PropertyNameField_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		yaml                 string
		expectedPropertyName string
	}{
		{
			name:                 "simple property name",
			yaml:                 `propertyName: type`,
			expectedPropertyName: "type",
		},
		{
			name:                 "camelCase property name",
			yaml:                 `propertyName: petType`,
			expectedPropertyName: "petType",
		},
		{
			name:                 "snake_case property name",
			yaml:                 `propertyName: pet_type`,
			expectedPropertyName: "pet_type",
		},
		{
			name:                 "kebab-case property name",
			yaml:                 `propertyName: pet-type`,
			expectedPropertyName: "pet-type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target Discriminator
			validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &target)

			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should not have validation errors")
			assert.Equal(t, tt.expectedPropertyName, target.PropertyName.Value, "should parse propertyName correctly")
		})
	}
}

func TestDiscriminator_Unmarshal_MappingField_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yaml         string
		key          string
		expectedRef  string
		expectedSize int
	}{
		{
			name: "single mapping entry",
			yaml: `
propertyName: type
mapping:
  dog: "#/components/schemas/Dog"
`,
			key:          "dog",
			expectedRef:  "#/components/schemas/Dog",
			expectedSize: 1,
		},
		{
			name: "multiple mapping entries",
			yaml: `
propertyName: type
mapping:
  dog: "#/components/schemas/Dog"
  cat: "#/components/schemas/Cat"
  bird: "#/components/schemas/Bird"
`,
			key:          "cat",
			expectedRef:  "#/components/schemas/Cat",
			expectedSize: 3,
		},
		{
			name: "mapping with external refs",
			yaml: `
propertyName: type
mapping:
  local: "#/components/schemas/Local"
  external: "https://example.com/schemas/External"
`,
			key:          "external",
			expectedRef:  "https://example.com/schemas/External",
			expectedSize: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target Discriminator
			validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &target)

			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should not have validation errors")
			require.NotNil(t, target.Mapping.Value, "mapping should be set")
			assert.Equal(t, tt.expectedSize, target.Mapping.Value.Len(), "should have correct number of mappings")

			value, found := target.Mapping.Value.Get(tt.key)
			require.True(t, found, "should find mapping key")
			assert.Equal(t, tt.expectedRef, value.Value, "should parse mapping value correctly")
		})
	}
}

func TestDiscriminator_Unmarshal_DefaultMappingField_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		yaml                   string
		expectedDefaultMapping string
	}{
		{
			name: "defaultMapping with component ref",
			yaml: `
propertyName: type
defaultMapping: "#/components/schemas/Default"
`,
			expectedDefaultMapping: "#/components/schemas/Default",
		},
		{
			name: "defaultMapping with external ref",
			yaml: `
propertyName: type
defaultMapping: "https://example.com/schemas/Default"
`,
			expectedDefaultMapping: "https://example.com/schemas/Default",
		},
		{
			name: "defaultMapping with path ref",
			yaml: `
propertyName: type
defaultMapping: "#/definitions/Default"
`,
			expectedDefaultMapping: "#/definitions/Default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target Discriminator
			validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &target)

			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should not have validation errors")
			require.NotNil(t, target.DefaultMapping.Value, "defaultMapping should be set")
			assert.Equal(t, tt.expectedDefaultMapping, *target.DefaultMapping.Value, "should parse defaultMapping correctly")
		})
	}
}

func TestDiscriminator_Unmarshal_Extensions_Success(t *testing.T) {
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
propertyName: type
x-custom: value
`,
			extensionKey:  "x-custom",
			expectedValue: "value",
		},
		{
			name: "multiple extensions",
			yaml: `
propertyName: type
x-first: value1
x-second: value2
`,
			extensionKey:  "x-first",
			expectedValue: "value1",
		},
		{
			name: "extension with all fields",
			yaml: `
propertyName: type
mapping:
  dog: "#/components/schemas/Dog"
defaultMapping: "#/components/schemas/Pet"
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

			var target Discriminator
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

func TestDiscriminator_Unmarshal_MinimalObject_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	yaml := `propertyName: type`

	var target Discriminator
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, yaml), &target)

	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "should not have validation errors")
	assert.Equal(t, "type", target.PropertyName.Value, "should parse propertyName")
	assert.Nil(t, target.Mapping.Value, "mapping should be nil")
	assert.Nil(t, target.DefaultMapping.Value, "defaultMapping should be nil")
}

func TestDiscriminator_Unmarshal_EmptyMapping_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	yaml := `
propertyName: type
mapping: {}
`

	var target Discriminator
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, yaml), &target)

	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "should not have validation errors")
	assert.Equal(t, "type", target.PropertyName.Value, "should parse propertyName")
	require.NotNil(t, target.Mapping.Value, "mapping should not be nil")
	assert.Equal(t, 0, target.Mapping.Value.Len(), "mapping should be empty")
}
