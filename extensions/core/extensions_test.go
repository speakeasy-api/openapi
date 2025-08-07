package core_test

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type TestCoreModel struct {
	marshaller.CoreModel
	Name  marshaller.Node[string]     `key:"name"`
	Value marshaller.Node[*yaml.Node] `key:"value" required:"true"`
}

func (t *TestCoreModel) Unmarshal(ctx context.Context, node *yaml.Node) ([]error, error) {
	return marshaller.UnmarshalModel(ctx, node, t)
}

func TestUnmarshalExtensionModel_Success(t *testing.T) {
	t.Parallel()
	e := sequencedmap.New(
		sequencedmap.NewElem("x-speakeasy-test", marshaller.Node[*yaml.Node]{
			Value: testutils.CreateMapYamlNode([]*yaml.Node{
				testutils.CreateStringYamlNode("name", 0, 0),
				testutils.CreateStringYamlNode("test", 0, 0),
				testutils.CreateStringYamlNode("value", 0, 0),
				testutils.CreateIntYamlNode(1, 0, 0),
			}, 0, 0),
		}),
	)

	tcm, validationErrs, err := core.UnmarshalExtensionModel[TestCoreModel](context.Background(), e, "x-speakeasy-test")
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	assert.Equal(t, "test", tcm.Name.Value)
	assert.Equal(t, testutils.CreateIntYamlNode(1, 0, 0), tcm.Value.Value)
}
