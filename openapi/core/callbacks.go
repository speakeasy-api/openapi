package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"gopkg.in/yaml.v3"
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

	for i := 0; i < len(rootNode.Content); i += 2 {
		if rootNode.Content[i].Value == key {
			return rootNode.Content[i]
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
