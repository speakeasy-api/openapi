package rules

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func parseNode(t *testing.T, doc string) *yaml.Node {
	t.Helper()

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(doc), &node), "yaml should parse")
	return node.Content[0]
}

func TestVisitScalars_FollowsAliases_Success(t *testing.T) {
	t.Parallel()

	node := parseNode(t, `a: &shared
  value: one
b: *shared
`)
	values := []string{}
	visitScalars(node, func(scalar *yaml.Node) { values = append(values, scalar.Value) })
	assert.Equal(t, []string{"a", "value", "one", "b"}, values, "aliased nodes should be visited once")
}

func TestMapValue_Success(t *testing.T) {
	t.Parallel()

	node := parseNode(t, `a: &shared
  value: one
b: *shared
`)
	assert.Equal(t, "one", mapValue(mapValue(node, "b"), "value").Value, "aliases should be followed")
	assert.Nil(t, mapValue(nil, "a"), "nil nodes have no values")
	assert.Nil(t, mapValue(mapValue(mapValue(node, "a"), "value"), "x"), "scalars have no values")
	assert.Nil(t, mapValue(node, "missing"), "missing keys have no values")
}

func TestMapKey_Success(t *testing.T) {
	t.Parallel()

	node := parseNode(t, "a: 1\n")
	assert.Equal(t, "a", mapKey(node, "a").Value, "key should be found")
	assert.Nil(t, mapKey(node, "missing"), "missing keys should not be found")
	assert.Nil(t, mapKey(nil, "a"), "nil nodes have no keys")
}

func TestResolveActions_Nil_Success(t *testing.T) {
	t.Parallel()

	a := &arazzo.Arazzo{}
	assert.Nil(t, resolveSuccessAction(a, nil), "nil success actions resolve to nil")
	assert.Nil(t, resolveFailureAction(a, nil), "nil failure actions resolve to nil")
}

func TestMarkComponentUse_UnknownSection_Success(t *testing.T) {
	t.Parallel()

	used := map[string]map[string]bool{}
	markComponentUse(used, "unknown", "name", "", "")
	assert.Empty(t, used, "unknown sections should be ignored")
}

func TestUsedInputs_NoRootNode_Success(t *testing.T) {
	t.Parallel()

	used, all := usedInputs(&arazzo.Workflow{})
	assert.Empty(t, used, "workflows without nodes use no inputs")
	assert.False(t, all, "workflows without nodes don't use all inputs")
}
