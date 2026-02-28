package core

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func parseYAML(t *testing.T, yml string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)
	return node.Content[0]
}

func TestXML_Unmarshal_AllFields_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "all fields populated",
			yaml: `
name: Person
namespace: http://example.com/schema/Person
prefix: per
attribute: true
wrapped: false
x-custom: value
`,
		},
		{
			name: "only required fields",
			yaml: `
name: Item
`,
		},
		{
			name: "namespace and prefix",
			yaml: `
namespace: http://example.com/ns
prefix: ex
`,
		},
		{
			name: "boolean flags",
			yaml: `
attribute: true
wrapped: true
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target XML
			validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &target)

			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should not have validation errors")
			assert.NotNil(t, target, "XML should not be nil")
		})
	}
}

func TestXML_Unmarshal_NameField_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yaml         string
		expectedName string
	}{
		{
			name:         "simple name",
			yaml:         `name: Person`,
			expectedName: "Person",
		},
		{
			name:         "camelCase name",
			yaml:         `name: personDetails`,
			expectedName: "personDetails",
		},
		{
			name:         "PascalCase name",
			yaml:         `name: PersonDetails`,
			expectedName: "PersonDetails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target XML
			validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &target)

			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should not have validation errors")
			require.NotNil(t, target.Name.Value, "name should be set")
			assert.Equal(t, tt.expectedName, *target.Name.Value, "should parse name correctly")
		})
	}
}

func TestXML_Unmarshal_NamespaceField_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		yaml              string
		expectedNamespace string
	}{
		{
			name:              "http namespace",
			yaml:              `namespace: http://example.com/schema`,
			expectedNamespace: "http://example.com/schema",
		},
		{
			name:              "https namespace",
			yaml:              `namespace: https://example.com/api/v1`,
			expectedNamespace: "https://example.com/api/v1",
		},
		{
			name:              "urn namespace",
			yaml:              `namespace: urn:example:schema`,
			expectedNamespace: "urn:example:schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target XML
			validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &target)

			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should not have validation errors")
			require.NotNil(t, target.Namespace.Value, "namespace should be set")
			assert.Equal(t, tt.expectedNamespace, *target.Namespace.Value, "should parse namespace correctly")
		})
	}
}

func TestXML_Unmarshal_PrefixField_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yaml           string
		expectedPrefix string
	}{
		{
			name:           "short prefix",
			yaml:           `prefix: ex`,
			expectedPrefix: "ex",
		},
		{
			name:           "longer prefix",
			yaml:           `prefix: example`,
			expectedPrefix: "example",
		},
		{
			name:           "single char prefix",
			yaml:           `prefix: x`,
			expectedPrefix: "x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target XML
			validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &target)

			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should not have validation errors")
			require.NotNil(t, target.Prefix.Value, "prefix should be set")
			assert.Equal(t, tt.expectedPrefix, *target.Prefix.Value, "should parse prefix correctly")
		})
	}
}

func TestXML_Unmarshal_AttributeField_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		yaml              string
		expectedAttribute bool
	}{
		{
			name:              "attribute true",
			yaml:              `attribute: true`,
			expectedAttribute: true,
		},
		{
			name:              "attribute false",
			yaml:              `attribute: false`,
			expectedAttribute: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target XML
			validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &target)

			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should not have validation errors")
			require.NotNil(t, target.Attribute.Value, "attribute should be set")
			assert.Equal(t, tt.expectedAttribute, *target.Attribute.Value, "should parse attribute correctly")
		})
	}
}

func TestXML_Unmarshal_WrappedField_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		yaml            string
		expectedWrapped bool
	}{
		{
			name:            "wrapped true",
			yaml:            `wrapped: true`,
			expectedWrapped: true,
		},
		{
			name:            "wrapped false",
			yaml:            `wrapped: false`,
			expectedWrapped: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target XML
			validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, tt.yaml), &target)

			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "should not have validation errors")
			require.NotNil(t, target.Wrapped.Value, "wrapped should be set")
			assert.Equal(t, tt.expectedWrapped, *target.Wrapped.Value, "should parse wrapped correctly")
		})
	}
}

func TestXML_Unmarshal_Extensions_Success(t *testing.T) {
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
x-custom: value
`,
			extensionKey:  "x-custom",
			expectedValue: "value",
		},
		{
			name: "multiple extensions",
			yaml: `
x-first: value1
x-second: value2
`,
			extensionKey:  "x-first",
			expectedValue: "value1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			var target XML
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

func TestXML_Unmarshal_EmptyObject_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	yaml := `{}`

	var target XML
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", parseYAML(t, yaml), &target)

	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "should not have validation errors")
	assert.Nil(t, target.Name.Value, "name should be nil")
	assert.Nil(t, target.Namespace.Value, "namespace should be nil")
	assert.Nil(t, target.Prefix.Value, "prefix should be nil")
	assert.Nil(t, target.Attribute.Value, "attribute should be nil")
	assert.Nil(t, target.Wrapped.Value, "wrapped should be nil")
}
