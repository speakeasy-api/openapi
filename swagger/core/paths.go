package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"go.yaml.in/yaml/v4"
)

// Paths holds the relative paths to the individual endpoints.
type Paths struct {
	marshaller.CoreModel `model:"paths"`
	*sequencedmap.Map[string, marshaller.Node[*PathItem]]

	Extensions core.Extensions `key:"extensions"`
}

func NewPaths() *Paths {
	return &Paths{
		Map: sequencedmap.New[string, marshaller.Node[*PathItem]](),
	}
}

func (p *Paths) GetMapKeyNodeOrRoot(key string, rootNode *yaml.Node) *yaml.Node {
	if !p.IsInitialized() {
		return rootNode
	}

	if p.RootNode == nil {
		return rootNode
	}

	for i := 0; i < len(p.RootNode.Content); i += 2 {
		if p.RootNode.Content[i].Value == key {
			return p.RootNode.Content[i]
		}
	}

	return rootNode
}

func (p *Paths) GetMapKeyNodeOrRootLine(key string, rootNode *yaml.Node) int {
	node := p.GetMapKeyNodeOrRoot(key, rootNode)
	if node == nil {
		return -1
	}
	return node.Line
}

// PathItem describes the operations available on a single path.
type PathItem struct {
	marshaller.CoreModel `model:"pathItem"`
	*sequencedmap.Map[string, marshaller.Node[*Operation]]

	Ref        marshaller.Node[*string]                  `key:"$ref"`
	Parameters marshaller.Node[[]*Reference[*Parameter]] `key:"parameters"`
	Extensions core.Extensions                           `key:"extensions"`
}

func NewPathItem() *PathItem {
	return &PathItem{
		Map: sequencedmap.New[string, marshaller.Node[*Operation]](),
	}
}

func (p *PathItem) GetMapKeyNodeOrRoot(key string, rootNode *yaml.Node) *yaml.Node {
	if !p.IsInitialized() {
		return rootNode
	}

	if p.RootNode == nil {
		return rootNode
	}

	for i := 0; i < len(p.RootNode.Content); i += 2 {
		if p.RootNode.Content[i].Value == key {
			return p.RootNode.Content[i]
		}
	}

	return rootNode
}

func (p *PathItem) GetMapKeyNodeOrRootLine(key string, rootNode *yaml.Node) int {
	node := p.GetMapKeyNodeOrRoot(key, rootNode)
	if node == nil {
		return -1
	}
	return node.Line
}
