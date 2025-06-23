// Package json provides utilities for working with JSON.
package json

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

// YAMLToJSON will convert the provided YAML node to JSON in a stable way not reordering keys.
func YAMLToJSON(node *yaml.Node, indentation int, buffer io.Writer) error {
	v, err := handleYAMLNode(node)
	if err != nil {
		return err
	}

	e := json.NewEncoder(buffer)
	e.SetIndent("", strings.Repeat(" ", indentation))

	return e.Encode(v)
}

func handleYAMLNode(node *yaml.Node) (any, error) {
	switch node.Kind {
	case yaml.DocumentNode:
		return handleYAMLNode(node.Content[0])
	case yaml.SequenceNode:
		return handleSequenceNode(node)
	case yaml.MappingNode:
		return handleMappingNode(node)
	case yaml.ScalarNode:
		return handleScalarNode(node)
	case yaml.AliasNode:
		return handleYAMLNode(node.Alias)
	default:
		return nil, fmt.Errorf("unknown node kind: %s", yml.NodeKindToString(node.Kind))
	}
}

func handleMappingNode(node *yaml.Node) (any, error) {
	v := sequencedmap.New[string, any]()
	for i, n := range node.Content {
		if i%2 == 0 {
			continue
		}
		keyNode := node.Content[i-1]
		kv, err := handleYAMLNode(keyNode)
		if err != nil {
			return nil, err
		}

		if reflect.TypeOf(kv).Kind() != reflect.String {
			keyData, err := json.Marshal(kv)
			if err != nil {
				return nil, err
			}
			kv = string(keyData)
		}

		vv, err := handleYAMLNode(n)
		if err != nil {
			return nil, err
		}

		v.Set(fmt.Sprintf("%v", kv), vv)
	}

	return v, nil
}

func handleSequenceNode(node *yaml.Node) (any, error) {
	var s []yaml.Node

	if err := node.Decode(&s); err != nil {
		return nil, err
	}

	v := make([]any, len(s))
	for i, n := range s {
		vv, err := handleYAMLNode(&n)
		if err != nil {
			return nil, err
		}

		v[i] = vv
	}

	return v, nil
}

func handleScalarNode(node *yaml.Node) (any, error) {
	var v any

	if err := node.Decode(&v); err != nil {
		return nil, err
	}

	return v, nil
}
