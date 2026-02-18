package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"go.yaml.in/yaml/v4"
)

type Responses struct {
	marshaller.CoreModel `model:"responses"`
	*sequencedmap.Map[string, *Reference[*Response]]

	Default    marshaller.Node[*Reference[*Response]] `key:"default"`
	Extensions core.Extensions                        `key:"extensions"`
}

func NewResponses() *Responses {
	return &Responses{
		Map: sequencedmap.New[string, *Reference[*Response]](),
	}
}

func (r *Responses) GetMapKeyNodeOrRoot(key string, rootNode *yaml.Node) *yaml.Node {
	if !r.IsInitialized() {
		return rootNode
	}

	if r.RootNode == nil {
		return rootNode
	}

	for i := 0; i < len(r.RootNode.Content); i += 2 {
		if r.RootNode.Content[i].Value == key {
			return r.RootNode.Content[i]
		}
	}

	return rootNode
}

func (r *Responses) GetMapKeyNodeOrRootLine(key string, rootNode *yaml.Node) int {
	node := r.GetMapKeyNodeOrRoot(key, rootNode)
	if node == nil {
		return -1
	}
	return node.Line
}

type Response struct {
	marshaller.CoreModel `model:"response"`

	Description marshaller.Node[string]                                         `key:"description"`
	Headers     marshaller.Node[*sequencedmap.Map[string, *Reference[*Header]]] `key:"headers"`
	Content     marshaller.Node[*sequencedmap.Map[string, *MediaType]]          `key:"content"`
	Links       marshaller.Node[*sequencedmap.Map[string, *Reference[*Link]]]   `key:"links"`
	Extensions  core.Extensions                                                 `key:"extensions"`
}
