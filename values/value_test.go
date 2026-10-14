package values_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/values"
	"github.com/stretchr/testify/assert"
	goyaml "go.yaml.in/yaml/v3"
	legacyyaml "gopkg.in/yaml.v3"
)

func TestValues_LegacyConversion(t *testing.T) {
	t.Parallel()

	legacyNode := &legacyyaml.Node{
		Kind:  legacyyaml.ScalarNode,
		Value: "test-value",
		Tag:   "!!str",
	}

	val := values.FromLegacyNode(legacyNode)
	assert.NotNil(t, val)
	assert.Equal(t, "test-value", val.Value)
	assert.Equal(t, goyaml.ScalarNode, val.Kind)

	convertedBack := values.ToLegacyNode(val)
	assert.NotNil(t, convertedBack)
	assert.Equal(t, "test-value", convertedBack.Value)
	assert.Equal(t, legacyyaml.ScalarNode, convertedBack.Kind)
}
