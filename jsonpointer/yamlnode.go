package jsonpointer

import (
	"fmt"
	"strconv"

	"go.yaml.in/yaml/v4"
)

func getYamlNodeTarget(node *yaml.Node, currentPart navigationPart, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	if node == nil {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("yaml node is nil at %s", currentPath))
	}

	// Resolve alias nodes
	for node.Kind == yaml.AliasNode {
		if node.Alias == nil {
			return nil, nil, ErrNotFound.Wrap(fmt.Errorf("yaml alias node has nil alias at %s", currentPath))
		}
		node = node.Alias
	}

	// Handle DocumentNode by delegating to its content
	if node.Kind == yaml.DocumentNode {
		return getYamlDocumentTarget(node, currentPart, stack, currentPath, o)
	}

	// Special case: if this is root access ("/") with empty stack and empty currentPart
	if len(stack) == 0 && currentPart.Value == "" {
		// For DocumentNode, return its content (the actual root data)
		if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
			return node.Content[0], stack, nil
		}
		return node, stack, nil
	}

	switch node.Kind {
	case yaml.DocumentNode:
		return getYamlDocumentTarget(node, currentPart, stack, currentPath, o)
	case yaml.MappingNode:
		return getYamlMappingTarget(node, currentPart, stack, currentPath, o)
	case yaml.SequenceNode:
		return getYamlSequenceTarget(node, currentPart, stack, currentPath, o)
	case yaml.ScalarNode:
		return nil, nil, ErrInvalidPath.Wrap(fmt.Errorf("cannot navigate through scalar yaml node at %s", currentPath))
	default:
		return nil, nil, ErrInvalidPath.Wrap(fmt.Errorf("unsupported yaml node kind %v at %s", node.Kind, currentPath))
	}
}

func getYamlDocumentTarget(node *yaml.Node, currentPart navigationPart, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	if len(node.Content) == 0 {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("document node has no content at %s", currentPath))
	}
	// Document nodes typically have a single root content node
	// We need to continue with the current part and stack on the document's content
	return getYamlNodeTarget(node.Content[0], currentPart, stack, currentPath, o)
}

func getYamlMappingTarget(node *yaml.Node, currentPart navigationPart, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	key := currentPart.unescapeValue()

	// YAML mapping nodes have content in pairs: [key1, value1, key2, value2, ...]
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break // Malformed mapping, skip
		}

		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		// Resolve aliases for key comparison
		resolvedKeyNode := keyNode
		for resolvedKeyNode.Kind == yaml.AliasNode && resolvedKeyNode.Alias != nil {
			resolvedKeyNode = resolvedKeyNode.Alias
		}

		if resolvedKeyNode.Kind == yaml.ScalarNode && resolvedKeyNode.Value == key {
			// If there are no more navigation parts in the stack, return the value node directly
			if len(stack) == 0 {
				return valueNode, stack, nil
			}
			return getCurrentStackTarget(valueNode, stack, currentPath, o)
		}
	}

	// If key not found, check for YAML merge keys (<<: *alias)
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}

		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		// Look for merge key "<<"
		if keyNode.Kind == yaml.ScalarNode && keyNode.Value == "<<" {
			// Resolve the alias
			aliasNode := valueNode
			for aliasNode.Kind == yaml.AliasNode && aliasNode.Alias != nil {
				aliasNode = aliasNode.Alias
			}

			if aliasNode.Kind == yaml.MappingNode {
				// Recursively search in the aliased mapping
				result, newStack, err := getYamlMappingTarget(aliasNode, currentPart, stack, currentPath, o)
				if err == nil {
					return result, newStack, nil
				}
			}
		}
	}

	return nil, nil, ErrNotFound.Wrap(fmt.Errorf("key %s not found in yaml mapping at %s", key, currentPath))
}

func getYamlSequenceTarget(node *yaml.Node, currentPart navigationPart, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	if currentPart.Type != partTypeIndex {
		return nil, nil, ErrInvalidPath.Wrap(fmt.Errorf("expected index, got %s at %s", currentPart.Type, currentPath))
	}

	index, err := strconv.Atoi(currentPart.Value)
	if err != nil {
		return nil, nil, ErrInvalidPath.Wrap(fmt.Errorf("invalid index %s at %s", currentPart.Value, currentPath))
	}

	if index < 0 || index >= len(node.Content) {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("index %d out of range for yaml sequence of length %d at %s", index, len(node.Content), currentPath))
	}

	// If there are no more navigation parts in the stack, return the element node directly
	if len(stack) == 0 {
		return node.Content[index], stack, nil
	}
	return getCurrentStackTarget(node.Content[index], stack, currentPath, o)
}
