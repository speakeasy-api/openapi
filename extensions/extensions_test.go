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
	marshaller.Model[CoreModelWithExtensions]

	Test       string
	Extensions *extensions.Extensions
}

type CoreModelWithExtensions struct {
	marshaller.CoreModel
	Test       marshaller.Node[string]   `key:"test"`
	Extensions coreExtensions.Extensions `key:"extensions"`
}

type TestModel struct {
	marshaller.Model[TestCoreModel]

	Name  string
	Value yaml.Node
}

type TestCoreModel struct {
	marshaller.CoreModel

	Name  marshaller.Node[string]     `key:"name"`
	Value marshaller.Node[*yaml.Node] `key:"value" required:"true"`
}

func TestUnmarshalExtensionModel_Success(t *testing.T) {
	ctx := context.Background()

	m := getTestModelWithExtensions(ctx, t, `
test: hello world
x-speakeasy-test:
  name: test
  value: 1`)

	var testModel TestModel
	validationErrs, err := extensions.UnmarshalExtensionModel[TestModel, TestCoreModel](ctx, m.Extensions, "x-speakeasy-test", &testModel)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	assert.Equal(t, "test", testModel.Name)
	assert.Equal(t, *testutils.CreateIntYamlNode(1, 5, 10), testModel.Value)
}

func TestGetExtensionValue_Success(t *testing.T) {
	ctx := context.Background()

	m := getTestModelWithExtensions(ctx, t, `
test: hello world
x-int: 1
x-string: hi
x-bool: true
x-simple-map:
  key1: value1
  key2: value2
x-simple-model:
  name: test
  value: 1`)

	intVal, err := extensions.GetExtensionValue[int](m.Extensions, "x-int")
	require.NoError(t, err)
	require.NotNil(t, intVal)
	assert.Equal(t, 1, *intVal)

	stringVal, err := extensions.GetExtensionValue[string](m.Extensions, "x-string")
	require.NoError(t, err)
	require.NotNil(t, stringVal)
	assert.Equal(t, "hi", *stringVal)

	boolVal, err := extensions.GetExtensionValue[bool](m.Extensions, "x-bool")
	require.NoError(t, err)
	require.NotNil(t, boolVal)
	assert.Equal(t, true, *boolVal)

	simpleMapVal, err := extensions.GetExtensionValue[map[string]string](m.Extensions, "x-simple-map")
	require.NoError(t, err)
	require.NotNil(t, simpleMapVal)
	assert.Equal(t, map[string]string{"key1": "value1", "key2": "value2"}, *simpleMapVal)

	simpleModelVal, err := extensions.GetExtensionValue[TestModel](m.Extensions, "x-simple-model")
	require.NoError(t, err)
	require.NotNil(t, simpleModelVal)
	assert.Equal(t, "test", simpleModelVal.Name)
	assert.Equal(t, *testutils.CreateIntYamlNode(1, 11, 10), simpleModelVal.Value)
}

func getTestModelWithExtensions(ctx context.Context, t *testing.T, data string) *ModelWithExtensions {
	t.Helper()

	d, err := io.ReadAll(bytes.NewReader([]byte(data)))
	require.NoError(t, err)

	var root yaml.Node
	err = yaml.Unmarshal(d, &root)
	require.NoError(t, err)

	var c CoreModelWithExtensions
	validationErrs, err := marshaller.UnmarshalCore(ctx, &root, &c)
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	m := &ModelWithExtensions{}
	err = marshaller.Populate(c, m)
	require.NoError(t, err)

	return m
}
