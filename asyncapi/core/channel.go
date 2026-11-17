package core

import (
	"context"
	"slices"

	"github.com/speakeasy-api/openapi/extensions/core"
	schemacore "github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v3"
)

// Channel is the backing model for the Channel Object.
type Channel struct {
	marshaller.CoreModel `model:"channel"`

	Address      marshaller.Node[*string]                                           `key:"address"`
	Messages     marshaller.Node[*sequencedmap.Map[string, *Reference[*Message]]]   `key:"messages"`
	Title        marshaller.Node[*string]                                           `key:"title"`
	Summary      marshaller.Node[*string]                                           `key:"summary"`
	Description  marshaller.Node[*string]                                           `key:"description"`
	Servers      marshaller.Node[[]*ReferenceObject]                                `key:"servers"`
	Parameters   marshaller.Node[*sequencedmap.Map[string, *Reference[*Parameter]]] `key:"parameters"`
	Tags         marshaller.Node[[]*Tag]                                            `key:"tags"`
	ExternalDocs marshaller.Node[*schemacore.ExternalDocumentation]                 `key:"externalDocs"`
	Bindings     marshaller.Node[*Reference[*Bindings]]                             `key:"bindings"`

	Extensions core.Extensions `key:"extensions"`
}

// SyncChanges syncs the public Channel model back to this backing model.
//
// A null address and an omitted address both mean the address is unknown, so the public model represents both as
// nil. To round-trip documents unchanged, an authored `address: null` is restored when the public address is
// still unknown.
func (c *Channel) SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error) {
	var nullKey, nullValue *yaml.Node
	index := -1
	if resolved := yml.ResolveAlias(valueNode); resolved != nil && resolved.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(resolved.Content); i += 2 {
			value := yml.ResolveAlias(resolved.Content[i+1])
			if resolved.Content[i].Value == "address" && value != nil && value.Tag == "!!null" {
				nullKey, nullValue, index = resolved.Content[i], resolved.Content[i+1], i
				break
			}
		}
	}

	node, err := marshaller.SyncValue(ctx, model, c, valueNode, true)
	if err != nil {
		return nil, err
	}

	resolved := yml.ResolveAlias(node)
	if nullKey == nil || resolved == nil || resolved.Kind != yaml.MappingNode {
		return node, nil
	}
	if _, _, found := yml.GetMapElementNodes(ctx, resolved, "address"); found {
		return node, nil
	}

	index = min(index, len(resolved.Content))
	resolved.Content = slices.Insert(resolved.Content, index, nullKey, nullValue)
	c.Address = marshaller.Node[*string]{Key: "address", KeyNode: nullKey, ValueNode: nullValue, Present: true}

	return node, nil
}

// Parameter is the backing model for the Parameter Object.
type Parameter struct {
	marshaller.CoreModel `model:"parameter"`

	Enum        marshaller.Node[[]marshaller.Node[string]] `key:"enum"`
	Default     marshaller.Node[*string]                   `key:"default"`
	Description marshaller.Node[*string]                   `key:"description"`
	Examples    marshaller.Node[[]marshaller.Node[string]] `key:"examples"`
	Location    marshaller.Node[*string]                   `key:"location"`

	Extensions core.Extensions `key:"extensions"`
}
