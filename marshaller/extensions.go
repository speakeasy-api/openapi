package marshaller

import (
	"context"
	"fmt"
	"iter"
	"reflect"
	"slices"
	"unsafe"

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

func UnmarshalExtension(keyNode *yaml.Node, valueNode *yaml.Node, extensionsField reflect.Value) error {
	resolvedKeyNode := yml.ResolveAlias(keyNode)
	resolvedValueNode := yml.ResolveAlias(valueNode)

	if !extensionsField.CanSet() {
		return fmt.Errorf("Extensions field is not settable (field type: %v) at line %d, column %d",
			extensionsField.Type(), resolvedKeyNode.Line, resolvedKeyNode.Column)
	}

	if extensionsField.IsNil() {
		extensionsField.Set(CreateInstance(extensionsField.Type().Elem()))
	}

	exts, ok := extensionsField.Interface().(ExtensionCoreMap)
	if !ok {
		return fmt.Errorf("expected ExtensionCoreMap, got %v (field type: %v) at line %d, column %d",
			extensionsField.Type(), extensionsField.Type(), resolvedKeyNode.Line, resolvedKeyNode.Column)
	}

	exts.Init()

	exts.Set(resolvedKeyNode.Value, Node[Extension]{
		Key:       resolvedKeyNode.Value,
		KeyNode:   keyNode,
		Value:     resolvedValueNode,
		ValueNode: valueNode,
	})

	return nil
}

func syncExtensions(ctx context.Context, source any, target reflect.Value, mapNode *yaml.Node) (*yaml.Node, error) {
	// Handle nil source (when Extensions field is nil)
	if source == nil {
		return mapNode, nil
	}

	// Handle case where source is a pointer to nil
	sourceVal := reflect.ValueOf(source)
	if sourceVal.Kind() == reflect.Ptr && sourceVal.IsNil() {
		return mapNode, nil
	}

	iterator, ok := source.(ExtensionSourceIterator)
	if !ok {
		return nil, fmt.Errorf("expected ExtensionSourceIterator, got %v", reflect.TypeOf(source))
	}

	if target.Kind() == reflect.Ptr && target.IsNil() {
		target.Set(CreateInstance(target.Type().Elem()))
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
			if node.Value != extNode {
				node.Value = extNode
				if node.ValueNode.Kind == yaml.AliasNode {
					node.ValueNode.Alias = node.Value
				} else {
					node.ValueNode = node.Value
				}
			}

			var err error
			keyNode, valueNode, err = node.SyncValue(ctx, key, node.ValueNode)
			if err != nil {
				return nil, err
			}
			node.KeyNode = keyNode
			node.Value = yml.ResolveAlias(valueNode)
			node.ValueNode = valueNode
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

	sUnderlying := getUnderlyingValue(reflect.ValueOf(source))

	// Update the core of the source with the updated value
	cf, ok := sUnderlying.Type().FieldByName("core")
	if ok {
		sf := sUnderlying.FieldByIndex(cf.Index)
		reflect.NewAt(sf.Type(), unsafe.Pointer(sf.UnsafeAddr())).Elem().Set(target)
	}

	return mapNode, nil
}
