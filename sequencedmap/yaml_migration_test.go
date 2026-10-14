package sequencedmap_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	goyaml "go.yaml.in/yaml/v3"
	legacyyaml "gopkg.in/yaml.v3"
)

func TestSequencedMap_NativeYAMLv3RoundTrip(t *testing.T) {
	t.Parallel()

	yamlData := `first: 1
second: 2
third: 3
`
	var m sequencedmap.Map[string, int]
	err := goyaml.Unmarshal([]byte(yamlData), &m)
	require.NoError(t, err)

	assert.Equal(t, 3, m.Len())
	keys := make([]string, 0, m.Len())
	for k := range m.All() {
		keys = append(keys, k)
	}
	assert.Equal(t, []string{"first", "second", "third"}, keys)

	marshaled, err := goyaml.Marshal(&m)
	require.NoError(t, err)
	assert.Equal(t, yamlData, string(marshaled))
}

func TestSequencedMap_LegacyYAMLAdapterRoundTrip(t *testing.T) {
	t.Parallel()

	yamlData := `first: 1
second: 2
third: 3
`
	var m sequencedmap.Map[string, int]
	adapter := m.AsLegacyYAML()

	err := legacyyaml.Unmarshal([]byte(yamlData), adapter)
	require.NoError(t, err)

	assert.Equal(t, 3, m.Len())
	keys := make([]string, 0, m.Len())
	for k := range m.All() {
		keys = append(keys, k)
	}
	assert.Equal(t, []string{"first", "second", "third"}, keys)

	marshaled, err := legacyyaml.Marshal(adapter)
	require.NoError(t, err)
	assert.Equal(t, yamlData, string(marshaled))
}

func TestSequencedMap_LegacyDirectBehaviorDocumented(t *testing.T) {
	t.Parallel()

	yamlData := `first: 1
second: 2
`
	// Direct unmarshal with legacy yaml does not invoke go.yaml.in/yaml/v3 UnmarshalYAML
	var directMap sequencedmap.Map[string, int]
	err := legacyyaml.Unmarshal([]byte(yamlData), &directMap)
	require.NoError(t, err)
	assert.Equal(t, 0, directMap.Len(), "unmarshaling without adapter leaves map empty under legacy yaml")

	// Wrapping with AsLegacyYAML restores order-preserving decoding
	var adaptedMap sequencedmap.Map[string, int]
	err = legacyyaml.Unmarshal([]byte(yamlData), adaptedMap.AsLegacyYAML())
	require.NoError(t, err)
	assert.Equal(t, 2, adaptedMap.Len())
	v, ok := adaptedMap.Get("first")
	assert.True(t, ok)
	assert.Equal(t, 1, v)
}

type legacyCustomValue struct {
	customField  string
	receivedKind legacyyaml.Kind
}

func (v *legacyCustomValue) UnmarshalYAML(value *legacyyaml.Node) error {
	v.receivedKind = value.Kind
	v.customField = "custom:" + value.Value
	return nil
}

func (v legacyCustomValue) MarshalYAML() (interface{}, error) {
	return &legacyyaml.Node{
		Kind:  legacyyaml.ScalarNode,
		Value: v.customField,
	}, nil
}

func TestSequencedMap_LegacyValueUnmarshaler(t *testing.T) {
	t.Parallel()

	yamlData := `first: alpha
second: beta
`
	var m sequencedmap.Map[string, legacyCustomValue]
	err := goyaml.Unmarshal([]byte(yamlData), &m)
	require.NoError(t, err)
	assert.Equal(t, 2, m.Len())

	v1, ok := m.Get("first")
	require.True(t, ok)
	assert.Equal(t, "custom:alpha", v1.customField)
	assert.Equal(t, legacyyaml.ScalarNode, v1.receivedKind)

	v2, ok := m.Get("second")
	require.True(t, ok)
	assert.Equal(t, "custom:beta", v2.customField)
	assert.Equal(t, legacyyaml.ScalarNode, v2.receivedKind)

	marshaled, err := goyaml.Marshal(&m)
	require.NoError(t, err)
	assert.Equal(t, "first: custom:alpha\nsecond: custom:beta\n", string(marshaled))
}

func TestSequencedMap_LegacyPointerValueUnmarshalerWithAdapter(t *testing.T) {
	t.Parallel()

	yamlData := `first: alpha
second: beta
`
	var m sequencedmap.Map[string, *legacyCustomValue]
	err := legacyyaml.Unmarshal([]byte(yamlData), m.AsLegacyYAML())
	require.NoError(t, err)
	assert.Equal(t, 2, m.Len())

	v1, ok := m.Get("first")
	require.True(t, ok)
	require.NotNil(t, v1)
	assert.Equal(t, "custom:alpha", v1.customField)
	assert.Equal(t, legacyyaml.ScalarNode, v1.receivedKind)

	marshaled, err := legacyyaml.Marshal(m.AsLegacyYAML())
	require.NoError(t, err)
	assert.Equal(t, "first: custom:alpha\nsecond: custom:beta\n", string(marshaled))
}
