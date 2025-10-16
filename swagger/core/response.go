package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	oascore "github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	values "github.com/speakeasy-api/openapi/values/core"
	"gopkg.in/yaml.v3"
)

// Responses is a container for the expected responses of an operation.
type Responses struct {
	marshaller.CoreModel `model:"responses"`
	*sequencedmap.Map[string, marshaller.Node[*Reference[*Response]]]

	Default    marshaller.Node[*Reference[*Response]] `key:"default"`
	Extensions core.Extensions                        `key:"extensions"`
}

func NewResponses() *Responses {
	return &Responses{
		Map: sequencedmap.New[string, marshaller.Node[*Reference[*Response]]](),
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

// Response describes a single response from an API operation.
type Response struct {
	marshaller.CoreModel `model:"response"`

	Description marshaller.Node[string]             `key:"description"`
	Schema      marshaller.Node[oascore.JSONSchema] `key:"schema"`
	Headers     marshaller.Node[*Headers]           `key:"headers"`
	Examples    marshaller.Node[*Examples]          `key:"examples"`
	Extensions  core.Extensions                     `key:"extensions"`
}

// Examples is a map of MIME types to example values.
type Examples = sequencedmap.Map[string, marshaller.Node[values.Value]]

// Headers is a map of header names to header definitions.
type Headers = sequencedmap.Map[string, marshaller.Node[*Header]]

// Header describes a single header in a response.
type Header struct {
	marshaller.CoreModel `model:"header"`

	Description      marshaller.Node[*string]                         `key:"description"`
	Type             marshaller.Node[string]                          `key:"type"`
	Format           marshaller.Node[*string]                         `key:"format"`
	Items            marshaller.Node[*Items]                          `key:"items"`
	CollectionFormat marshaller.Node[*string]                         `key:"collectionFormat"`
	Default          marshaller.Node[values.Value]                    `key:"default"`
	Maximum          marshaller.Node[*float64]                        `key:"maximum"`
	ExclusiveMaximum marshaller.Node[*bool]                           `key:"exclusiveMaximum"`
	Minimum          marshaller.Node[*float64]                        `key:"minimum"`
	ExclusiveMinimum marshaller.Node[*bool]                           `key:"exclusiveMinimum"`
	MaxLength        marshaller.Node[*int64]                          `key:"maxLength"`
	MinLength        marshaller.Node[*int64]                          `key:"minLength"`
	Pattern          marshaller.Node[*string]                         `key:"pattern"`
	MaxItems         marshaller.Node[*int64]                          `key:"maxItems"`
	MinItems         marshaller.Node[*int64]                          `key:"minItems"`
	UniqueItems      marshaller.Node[*bool]                           `key:"uniqueItems"`
	Enum             marshaller.Node[[]marshaller.Node[values.Value]] `key:"enum"`
	MultipleOf       marshaller.Node[*float64]                        `key:"multipleOf"`
	Extensions       core.Extensions                                  `key:"extensions"`
}
