package sequencedmap_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestSequencedMap_Standard_Yaml_Unmarshal(t *testing.T) {
	t.Parallel()

	type TestStruct struct {
		Map *sequencedmap.Map[string, string] `yaml:"map"`
	}

	yamlStr := `map:
  key 1: value 1
  key 2: value 2
  key 3: value 3
`

	var ts TestStruct
	err := yaml.Unmarshal([]byte(yamlStr), &ts)
	require.NoError(t, err)

	assert.Equal(t, TestStruct{
		Map: sequencedmap.New(sequencedmap.NewElem("key 1", "value 1"), sequencedmap.NewElem("key 2", "value 2"), sequencedmap.NewElem("key 3", "value 3")),
	}, ts)
}

func TestSequencedMap_Yaml_Marshal_Unmarshal_RoundTrip(t *testing.T) {
	t.Parallel()

	type TestStruct struct {
		Map *sequencedmap.Map[string, string] `yaml:"map"`
	}

	// Create original data
	original := TestStruct{
		Map: sequencedmap.New(
			sequencedmap.NewElem("first", "value1"),
			sequencedmap.NewElem("second", "value2"),
			sequencedmap.NewElem("third", "value3"),
		),
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(original)
	require.NoError(t, err)

	// Unmarshal back
	var result TestStruct
	err = yaml.Unmarshal(yamlData, &result)
	require.NoError(t, err)

	// Verify the round trip preserved order and values
	assert.Equal(t, original, result)

	// Verify order is preserved
	keys := make([]string, 0)
	for key := range result.Map.All() {
		keys = append(keys, key)
	}
	assert.Equal(t, []string{"first", "second", "third"}, keys)
}

func TestSequencedMap_Yaml_Unmarshal_EmptyMap(t *testing.T) {
	t.Parallel()

	type TestStruct struct {
		Map *sequencedmap.Map[string, string] `yaml:"map"`
	}

	yamlStr := `map: {}`

	var ts TestStruct
	err := yaml.Unmarshal([]byte(yamlStr), &ts)
	require.NoError(t, err)

	assert.Equal(t, TestStruct{
		Map: sequencedmap.New[string, string](),
	}, ts)
	assert.Equal(t, 0, ts.Map.Len())
}

func TestSequencedMap_Yaml_Unmarshal_NilMap(t *testing.T) {
	t.Parallel()

	type TestStruct struct {
		Map *sequencedmap.Map[string, string] `yaml:"map"`
	}

	yamlStr := `map: null`

	var ts TestStruct
	err := yaml.Unmarshal([]byte(yamlStr), &ts)
	require.NoError(t, err)

	assert.Nil(t, ts.Map)
}

func TestSequencedMap_Yaml_Unmarshal_IntegerKeys(t *testing.T) {
	t.Parallel()

	type TestStruct struct {
		Map *sequencedmap.Map[int, string] `yaml:"map"`
	}

	yamlStr := `map:
  1: first
  2: second
  3: third
`

	var ts TestStruct
	err := yaml.Unmarshal([]byte(yamlStr), &ts)
	require.NoError(t, err)

	expected := TestStruct{
		Map: sequencedmap.New(
			sequencedmap.NewElem(1, "first"),
			sequencedmap.NewElem(2, "second"),
			sequencedmap.NewElem(3, "third"),
		),
	}
	assert.Equal(t, expected, ts)
}

func TestSequencedMap_Yaml_Unmarshal_Error_InvalidYaml(t *testing.T) {
	t.Parallel()

	type TestStruct struct {
		Map *sequencedmap.Map[string, string] `yaml:"map"`
	}

	yamlStr := `map: "not a map"`

	var ts TestStruct
	err := yaml.Unmarshal([]byte(yamlStr), &ts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot unmarshal")
}
