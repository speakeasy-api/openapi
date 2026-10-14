package hashing_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/hashing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	goyaml "go.yaml.in/yaml/v3"
	legacyyaml "gopkg.in/yaml.v3"
)

func TestHash_YAMLNodeParity(t *testing.T) {
	t.Parallel()

	yamlData := `info:
  title: Test API
  version: "1.0"
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
`
	var modernNode goyaml.Node
	err := goyaml.Unmarshal([]byte(yamlData), &modernNode)
	require.NoError(t, err)

	var legacyNode legacyyaml.Node
	err = legacyyaml.Unmarshal([]byte(yamlData), &legacyNode)
	require.NoError(t, err)

	modernHashPtr := hashing.Hash(&modernNode)
	legacyHashPtr := hashing.Hash(&legacyNode)
	assert.Equal(t, modernHashPtr, legacyHashPtr, "legacy and modern yaml node pointers should yield identical hash")

	modernHashVal := hashing.Hash(modernNode)
	legacyHashVal := hashing.Hash(legacyNode)
	assert.Equal(t, modernHashVal, legacyHashVal, "legacy and modern yaml node values should yield identical hash")
}
