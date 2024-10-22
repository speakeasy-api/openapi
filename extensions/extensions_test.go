package extensions_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	coreExtensions "github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type ModelWithExtensions struct {
	Test string

	Extensions *extensions.Extensions

	core CoreModelWithExtensions //nolint:unused
}

type CoreModelWithExtensions struct {
	Test       marshaller.Node[string]   `key:"test"`
	Extensions coreExtensions.Extensions `key:"extensions"`

	RootNode *yaml.Node
}

type TestModel struct {
	Name  string
	Value *yaml.Node

	core TestCoreModel //nolint:unused
}

type TestCoreModel struct {
	Name  marshaller.Node[string]     `key:"name"`
	Value marshaller.Node[*yaml.Node] `key:"value" required:"true"`

	RootNode *yaml.Node
}

func TestUnmarshalExtensionModel_Success(t *testing.T) {
	ctx := context.Background()

	data, err := io.ReadAll(bytes.NewReader([]byte(`
test: hello world
x-speakeasy-test:
  name: test
  value: 1
`)))
	require.NoError(t, err)

	var root yaml.Node
	err = yaml.Unmarshal(data, &root)
	require.NoError(t, err)

	var c CoreModelWithExtensions
	err = marshaller.Unmarshal(ctx, &root, &c)
	require.NoError(t, err)

	m := &ModelWithExtensions{}
	err = marshaller.PopulateModel(c, m)
	require.NoError(t, err)

	var testModel TestModel
	err = extensions.UnmarshalExtensionModel[TestModel, TestCoreModel](ctx, m.Extensions, "x-speakeasy-test", &testModel)
	require.NoError(t, err)

	assert.Equal(t, "test", testModel.Name)
	assert.Equal(t, testutils.CreateIntYamlNode(1, 5, 10), testModel.Value)
}
