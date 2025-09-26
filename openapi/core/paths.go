package core

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

type Paths struct {
	marshaller.CoreModel `model:"paths"`
	sequencedmap.Map[string, *Reference[*PathItem]]

	Extensions core.Extensions `key:"extensions"`
}

func NewPaths() *Paths {
	return &Paths{
		Map: *sequencedmap.New[string, *Reference[*PathItem]](),
	}
}

type PathItem struct {
	marshaller.CoreModel `model:"pathItem"`
	sequencedmap.Map[string, *Operation]

	Summary     marshaller.Node[*string] `key:"summary"`
	Description marshaller.Node[*string] `key:"description"`

	Servers    marshaller.Node[[]*Server]                `key:"servers"`
	Parameters marshaller.Node[[]*Reference[*Parameter]] `key:"parameters"`

	AdditionalOperations marshaller.Node[*sequencedmap.Map[string, marshaller.Node[*Operation]]] `key:"additionalOperations"`

	Extensions core.Extensions `key:"extensions"`
}

func NewPathItem() *PathItem {
	return &PathItem{
		Map: *sequencedmap.New[string, *Operation](),
	}
}
func (n PathItem) GetMapKeyNodeOrRoot(key string, rootNode *yaml.Node) *yaml.Node {
	keyNode, _, found := yml.GetMapElementNodes(context.Background(), n.RootNode, key)
	if found {
		return keyNode
	}
	return rootNode
}
