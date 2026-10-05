package marshaller_test

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/speakeasy-api/openapi/marshaller"
	testmodels "github.com/speakeasy-api/openapi/marshaller/tests"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestUnmarshal_DocumentInput_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		reader io.Reader
		want   string
	}{
		{"empty document", strings.NewReader(""), "empty document"},
		{"reader failure", iotest.ErrReader(errors.New("read failure")), "failed to read document: read failure"},
		{"invalid YAML", strings.NewReader("["), "failed to unmarshal document"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var model testmodels.TestPrimitiveHighModel
			validationErrs, err := marshaller.Unmarshal(t.Context(), tt.reader, &model)
			require.ErrorContains(t, err, tt.want, "document input errors should be reported")
			assert.Empty(t, validationErrs, "input failures should not be reported as model validation errors")
		})
	}
}

func TestUnmarshalCore_DocumentCardinality_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		node *yaml.Node
		want string
	}{
		{"no root", &yaml.Node{Kind: yaml.DocumentNode}, "expected 1 node, got `0`"},
		{"multiple roots", &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{yml.CreateStringNode("a"), yml.CreateStringNode("b")}}, "expected 1 node, got `2`"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var output string
			_, err := marshaller.UnmarshalCore(t.Context(), "test", tt.node, &output)
			require.ErrorContains(t, err, tt.want, "a document should contain exactly one root node")
		})
	}
}

func TestUnmarshalModel_InvalidModel_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output any
		want   string
	}{
		{"scalar output", new(int), "expected a struct"},
		{"empty struct", &struct{}{}, "expected embedded CoreModel field"},
		{"wrong first field", &struct{ Value string }{}, "expected embedded CoreModel field to be of type CoreModel"},
		{"missing model tag", &struct{ marshaller.CoreModel }{}, "expected embedded CoreModel field to have a 'model' tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := marshaller.UnmarshalModel(t.Context(), parseYAML(t, "{}"), tt.output)
			require.ErrorContains(t, err, tt.want, "invalid custom models should return a useful contract error")
		})
	}
}

func TestUnmarshalCore_UnsupportedMaps_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		output any
	}{
		{"plain map", "{key: value}", &map[string]string{}},
		{"map in sequence", "[{key: value}]", &[]map[string]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := marshaller.UnmarshalCore(t.Context(), "test", parseYAML(t, tt.input), tt.output)
			require.ErrorContains(t, err, "currently unsupported out kind: `map`", "unsupported maps should return errors, including within slices")
		})
	}
}

func TestUnmarshalCore_RawNode_Success(t *testing.T) {
	t.Parallel()

	node := yml.CreateStringNode("value")
	var output yaml.Node
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "test", node, &output)
	require.NoError(t, err, "raw node targets should be supported")
	assert.Empty(t, validationErrs, "raw nodes should not require scalar conversion")
	assert.Equal(t, *node, output, "raw node metadata should be preserved")
}

func TestUnmarshalCore_UnresolvedAlias_Success(t *testing.T) {
	t.Parallel()

	output := "unchanged"
	validationErrs, err := marshaller.UnmarshalCore(t.Context(), "test", &yaml.Node{Kind: yaml.AliasNode}, &output)
	require.NoError(t, err, "aliases without a resolved node should be ignored")
	assert.Empty(t, validationErrs, "unresolved aliases should not add validation errors")
	assert.Equal(t, "unchanged", output, "an unresolved alias should not overwrite the target")
}
