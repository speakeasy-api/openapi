package core_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindings_GetMapKeyNodeOrRoot_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{name: "returns key node when protocol exists", key: "kafka", expected: "kafka"},
		{name: "returns root when protocol is missing", key: "mqtt", expected: "root"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var bindings core.Bindings
			_, err := marshaller.UnmarshalCore(t.Context(), "", parseYAML(t, "ws: {}\nkafka:\n  topic: orders\n"), &bindings)
			require.NoError(t, err, "unmarshal should succeed")

			root := yml.CreateStringNode("root")
			assert.Equal(t, tt.expected, bindings.GetMapKeyNodeOrRoot(tt.key, root).Value, "should return the expected node")
		})
	}
}

func TestBindings_GetMapKeyNodeOrRoot_ReturnsRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bindings *core.Bindings
	}{
		{name: "uninitialized bindings", bindings: &core.Bindings{}},
		{name: "bindings without a root node", bindings: core.NewBindings()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := yml.CreateStringNode("root")
			assert.Same(t, root, tt.bindings.GetMapKeyNodeOrRoot("kafka", root), "should return the root node")
		})
	}
}

func TestNewBindings_Success(t *testing.T) {
	t.Parallel()

	bindings := core.NewBindings()
	require.NotNil(t, bindings.Map, "map should be initialized")
	assert.Equal(t, 0, bindings.Len(), "new bindings should be empty")
}
