package marshaller

import (
	"context"
	"fmt"
	"iter"
	"reflect"
	"slices"

	"github.com/speakeasy-api/openapi/errors"
	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

type Extension = *yaml.Node

type ExtensionCoreMap interface {
	Get(string) (Node[Extension], bool)
	Set(string, Node[Extension])
	Delete(string)
	All() iter.Seq2[string, Node[Extension]]
	Init()
}

type ExtensionMap interface {
	Set(string, Extension)
	Init()
	SetCore(any)
}

type ExtensionSourceIterator interface {
	All() iter.Seq2[string, Extension]
}

func unmarshalExtension(keyNode *yaml.Node, valueNode *yaml.Node, extensionsField reflect.Value) error {
	if !extensionsField.CanSet() {
		return errors.New("Extensions field is not settable")
	}

	if extensionsField.IsNil() {
		extensionsField.Set(reflect.New(extensionsField.Type().Elem()))
	}

	exts, ok := extensionsField.Interface().(ExtensionCoreMap)
	if !ok {
		return fmt.Errorf("expected ExtensionCoreMap, got %v", extensionsField.Type())
	}

	exts.Init()

	exts.Set(keyNode.Value, Node[Extension]{
		Key:       keyNode.Value,
		KeyNode:   keyNode,
		Value:     valueNode,
		ValueNode: valueNode,
	})

	return nil
}

func syncExtensions(ctx context.Context, source any, target reflect.Value, mapNode *yaml.Node) (*yaml.Node, error) {
	iterator, ok := source.(ExtensionSourceIterator)
	if !ok {
		return nil, fmt.Errorf("expected ExtensionSourceIterator, got %v", reflect.TypeOf(source))
	}

	if target.Kind() == reflect.Ptr && target.IsNil() {
		target.Set(reflect.New(target.Type().Elem()))
	}

	targetMap, ok := target.Interface().(ExtensionCoreMap)
	if !ok {
		return nil, fmt.Errorf("expected ExtensionCoreMap, got %v", reflect.TypeOf(target))
	}

	targetMap.Init()

	presentKeys := []string{}

	for key, extNode := range iterator.All() {
		node, ok := targetMap.Get(key)
		presentKeys = append(presentKeys, key)

		var keyNode, valueNode *yaml.Node
		if !ok {
			keyNode = yml.CreateOrUpdateKeyNode(ctx, key, nil)
			valueNode = extNode
			node = Node[Extension]{
				Key:       key,
				KeyNode:   keyNode,
				Value:     extNode,
				ValueNode: extNode,
			}
		} else {
			var err error
			keyNode, valueNode, err = node.SyncValue(ctx, key, extNode)
			if err != nil {
				return nil, err
			}
		}

		mapNode = yml.CreateOrUpdateMapNodeElement(ctx, key, keyNode, valueNode, mapNode)
		targetMap.Set(key, node)
	}

	for key := range targetMap.All() {
		if !slices.Contains(presentKeys, key) {
			mapNode = yml.DeleteMapNodeElement(ctx, key, mapNode)
			targetMap.Delete(key)
		}
	}

	return mapNode, nil
}
