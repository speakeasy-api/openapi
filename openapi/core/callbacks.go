package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"go.yaml.in/yaml/v4"
)

type Callback struct {
	marshaller.CoreModel `model:"callback"`
	*sequencedmap.Map[string, *Reference[*PathItem]]

	Extensions core.Extensions `key:"extensions"`
}

func NewCallback() *Callback {
	return &Callback{
		Map: sequencedmap.New[string, *Reference[*PathItem]](),
	}
}

func (c *Callback) GetMapKeyNodeOrRoot(key string, rootNode *yaml.Node) *yaml.Node {
	if !c.IsInitialized() {
		return rootNode
	}

	if c.RootNode == nil {
		return rootNode
	}

	for i := 0; i < len(c.RootNode.Content); i += 2 {
		if c.RootNode.Content[i].Value == key {
			return c.RootNode.Content[i]
		}
	}

	return rootNode
}

func (c *Callback) GetMapKeyNodeOrRootLine(key string, rootNode *yaml.Node) int {
	node := c.GetMapKeyNodeOrRoot(key, rootNode)
	if node == nil {
		return -1
	}
	return node.Line
}
