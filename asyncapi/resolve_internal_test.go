package asyncapi

import (
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi/core"
	jsonschema "github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestDescribeObject_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		obj      any
		expected string
	}{
		{name: "document", obj: &AsyncAPI{}, expected: "the whole document"},
		{name: "server", obj: &Server{}, expected: "a server"},
		{name: "server variable", obj: &ServerVariable{}, expected: "a server variable"},
		{name: "channel", obj: &ReferencedChannel{}, expected: "a channel"},
		{name: "parameter", obj: &Parameter{}, expected: "a parameter"},
		{name: "operation", obj: &Operation{}, expected: "an operation"},
		{name: "operation trait", obj: &OperationTrait{}, expected: "an operation trait"},
		{name: "operation reply", obj: &OperationReply{}, expected: "an operation reply"},
		{name: "operation reply address", obj: &OperationReplyAddress{}, expected: "an operation reply address"},
		{name: "message", obj: &ReferencedMessage{}, expected: "a message"},
		{name: "message trait", obj: &MessageTrait{}, expected: "a message trait"},
		{name: "correlation ID", obj: &CorrelationID{}, expected: "a correlation ID"},
		{name: "security scheme", obj: &SecurityScheme{}, expected: "a security scheme"},
		{name: "bindings", obj: &Bindings{}, expected: "bindings"},
		{name: "tag", obj: &Tag{}, expected: "a tag"},
		{name: "external documentation", obj: &ExternalDocumentation{}, expected: "external documentation"},
		{name: "schema", obj: &jsonschema.SchemaUnion{}, expected: "a schema"},
		{name: "components", obj: &Components{}, expected: "the components object"},
		{name: "other", obj: &Info{}, expected: "an unsupported location"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, describeObject(tt.obj), "description should match")
		})
	}
}

func TestUnmarshalReference_Error(t *testing.T) {
	t.Parallel()

	_, _, err := unmarshalReference[Channel, *Channel, *core.Channel](t.Context(), &yaml.Node{Kind: yaml.DocumentNode}, false)
	require.EqualError(t, err, "expected 1 node, got `0` at line `0`, column `0`", "an empty document should fail to unmarshal")
}

func TestUnmarshalReference_SkipValidation_Success(t *testing.T) {
	t.Parallel()

	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "channel"}

	_, validationErrs, err := unmarshalReference[Channel, *Channel, *core.Channel](t.Context(), node, false)
	require.NoError(t, err, "unmarshal should succeed")
	assert.Len(t, validationErrs, 1, "a scalar should be reported")

	ref, validationErrs, err := unmarshalReference[Channel, *Channel, *core.Channel](t.Context(), node, true)
	require.NoError(t, err, "unmarshal should succeed")
	assert.NotNil(t, ref, "reference should be returned")
	assert.Nil(t, validationErrs, "validation should be skipped")
}

func TestUnmarshalSchemaUnion_Error(t *testing.T) {
	t.Parallel()

	_, _, err := unmarshalSchemaUnion(t.Context(), &yaml.Node{Kind: yaml.DocumentNode}, false)
	require.EqualError(t, err, "expected 1 node, got `0` at line `0`, column `0`", "an empty document should fail to unmarshal")
}

func TestUnmarshalSchemaUnion_SkipValidation_Success(t *testing.T) {
	t.Parallel()

	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "schema"}

	_, validationErrs, err := unmarshalSchemaUnion(t.Context(), node, false)
	require.NoError(t, err, "unmarshal should succeed")
	assert.NotEmpty(t, validationErrs, "a scalar should be reported")

	schema, validationErrs, err := unmarshalSchemaUnion(t.Context(), node, true)
	require.NoError(t, err, "unmarshal should succeed")
	assert.NotNil(t, schema, "schema should be returned")
	assert.Nil(t, validationErrs, "validation should be skipped")
}

func TestResolveNode_Success(t *testing.T) {
	t.Parallel()

	scalar := &yaml.Node{Kind: yaml.ScalarNode, Value: "value"}

	tests := []struct {
		name     string
		node     *yaml.Node
		expected *yaml.Node
	}{
		{name: "nil", node: nil, expected: nil},
		{name: "empty document", node: &yaml.Node{Kind: yaml.DocumentNode}, expected: nil},
		{name: "document", node: &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{scalar}}, expected: scalar},
		{name: "alias", node: &yaml.Node{Kind: yaml.AliasNode, Alias: scalar}, expected: scalar},
		{name: "scalar", node: scalar, expected: scalar},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Same(t, tt.expected, resolveNode(tt.node), "resolved node should match")
		})
	}
}

func TestMappingScalar_Success(t *testing.T) {
	t.Parallel()

	key := &yaml.Node{Kind: yaml.ScalarNode, Value: "schemaFormat"}
	format := &yaml.Node{Kind: yaml.ScalarNode, Value: "application/vnd.apache.avro;version=1.9.0"}

	tests := []struct {
		name     string
		node     *yaml.Node
		expected string
	}{
		{name: "not a mapping", node: &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{format}}, expected: ""},
		{name: "empty document", node: &yaml.Node{Kind: yaml.DocumentNode}, expected: ""},
		{name: "missing key", node: &yaml.Node{Kind: yaml.MappingNode}, expected: ""},
		{name: "non-scalar value", node: &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{key, {Kind: yaml.MappingNode}}}, expected: ""},
		{name: "scalar value", node: &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{key, format}}, expected: "application/vnd.apache.avro;version=1.9.0"},
		{name: "aliased value", node: &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{key, {Kind: yaml.AliasNode, Alias: format}}}, expected: "application/vnd.apache.avro;version=1.9.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, mappingScalar(tt.node, "schemaFormat"), "value should match")
		})
	}
}
