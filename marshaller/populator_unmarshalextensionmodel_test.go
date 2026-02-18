package marshaller_test

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	oascore "github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func init() {
	marshaller.RegisterType(func() *CustomSecurityConfig {
		return &CustomSecurityConfig{}
	})
	marshaller.RegisterType(func() *CoreCustomSecurityConfig {
		return &CoreCustomSecurityConfig{}
	})
}

// CustomSecurityConfig represents a custom security configuration extension
type CustomSecurityConfig struct {
	marshaller.Model[CoreCustomSecurityConfig]

	UsesScopes *bool
	Schema     *oas3.JSONSchema[oas3.Referenceable]
}

// CoreCustomSecurityConfig represents the core model for custom security configuration
type CoreCustomSecurityConfig struct {
	marshaller.CoreModel `model:"coreCustomSecurityConfig"`

	UsesScopes marshaller.Node[*bool]              `key:"usesScopes"`
	Schema     marshaller.Node[oascore.JSONSchema] `key:"schema" required:"true"`
}

// ModelWithExtensions represents a model that has extensions
type ModelWithExtensions struct {
	marshaller.Model[CoreModelWithExtensions]

	Test       string
	Extensions *extensions.Extensions
}

// CoreModelWithExtensions represents the core model with extensions
type CoreModelWithExtensions struct {
	marshaller.CoreModel `model:"coreModelWithExtensions"`

	Test       marshaller.Node[string]                                `key:"test"`
	Extensions *sequencedmap.Map[string, marshaller.Node[*yaml.Node]] `key:"extensions"`
}

// TestUnmarshalExtensionModel_Success tests unmarshalling an extension model with missing optional fields
func TestUnmarshalExtensionModel_Success(t *testing.T) {
	t.Parallel()
	// Create a YAML document with an extension that has a 'schema' field but is missing 'usesScopes'
	// The 'usesScopes' field should be treated as nil/unset, not cause a panic
	yamlContent := `
test: hello world
x-speakeasy-custom-security-scheme:
  schema:
    type: object
    properties:
      customField:
        type: string
    required:
      - customField
`

	// Unmarshal the YAML into a model with extensions
	m := getTestModelWithExtensions(t.Context(), t, yamlContent)

	// Verify the extension was parsed
	require.Equal(t, 1, m.Extensions.Len(), "should have one extension")
	require.True(t, m.Extensions.Has("x-speakeasy-custom-security-scheme"), "should have the custom security scheme extension")

	// Unmarshal the specific extension model
	// This should succeed even when some fields are missing from the YAML
	var css CustomSecurityConfig
	vErrs, err := extensions.UnmarshalExtensionModel[CustomSecurityConfig, CoreCustomSecurityConfig](
		t.Context(),
		m.Extensions,
		"x-speakeasy-custom-security-scheme",
		&css,
	)

	// Should not error
	require.NoError(t, err, "should successfully unmarshal extension model")
	assert.Empty(t, vErrs, "should have no validation errors")

	// Should populate the schema field that was present in YAML
	assert.NotNil(t, css.Schema, "schema field should be populated")

	// Should leave the missing usesScopes field as nil (not panic)
	assert.Nil(t, css.UsesScopes, "usesScopes field should be nil when not present in YAML")
}

// getTestModelWithExtensions creates a model with extensions from YAML
func getTestModelWithExtensions(ctx context.Context, t *testing.T, data string) *ModelWithExtensions {
	t.Helper()

	var root yaml.Node
	err := yaml.Unmarshal([]byte(data), &root)
	require.NoError(t, err)

	var c CoreModelWithExtensions
	validationErrs, err := marshaller.UnmarshalCore(ctx, "", &root, &c)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	m := &ModelWithExtensions{}
	err = marshaller.PopulateWithContext(c, m, nil)
	require.NoError(t, err)

	return m
}
