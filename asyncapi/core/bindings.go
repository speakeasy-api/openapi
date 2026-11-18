package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	values "github.com/speakeasy-api/openapi/values/core"
	"go.yaml.in/yaml/v3"
)

// Bindings is the backing model for the Server, Channel, Operation and Message Bindings Objects.
// Each protocol binding is kept as a raw value so unknown protocols and fields survive unchanged.
type Bindings struct {
	marshaller.CoreModel `model:"bindings"`
	*sequencedmap.Map[string, marshaller.Node[values.Value]]

	Extensions core.Extensions `key:"extensions"`
}

// NewBindings creates a new Bindings object with the embedded map initialized.
func NewBindings() *Bindings {
	return &Bindings{
		Map: sequencedmap.New[string, marshaller.Node[values.Value]](),
	}
}

// GetMapKeyNodeOrRoot returns the key node for the provided protocol, or the provided root node if not found.
func (b *Bindings) GetMapKeyNodeOrRoot(key string, rootNode *yaml.Node) *yaml.Node {
	if !b.IsInitialized() || b.RootNode == nil {
		return rootNode
	}

	for i := 0; i < len(b.RootNode.Content); i += 2 {
		if b.RootNode.Content[i].Value == key {
			return b.RootNode.Content[i]
		}
	}

	return rootNode
}
