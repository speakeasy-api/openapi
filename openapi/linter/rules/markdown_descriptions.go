package rules

import (
	"gopkg.in/yaml.v3"
)

// GetFieldValueNode gets the YAML value node for a specific field from any object with a GetCore() method.
// This is useful for precise error reporting by providing the exact YAML location of a field.
//
// Example usage:
//
//	node := GetFieldValueNode(operation, "description", doc)
//	node := GetFieldValueNode(schema, "type", doc)
//
// Returns the field's value node if found, or a fallback node (object root or document root).
func GetFieldValueNode(obj any, fieldName string, fallbackDoc any) *yaml.Node {
	// Get root node directly from the object
	type rootNodeGetter interface {
		GetRootNode() *yaml.Node
	}

	var rootNode *yaml.Node
	if rng, ok := obj.(rootNodeGetter); ok {
		rootNode = rng.GetRootNode()
	}

	// If we have a root node, search for the field
	if rootNode != nil {
		if fieldNode := findFieldValueInNode(rootNode, fieldName); fieldNode != nil {
			return fieldNode
		}
		// Field not found, return object root as fallback
		return rootNode
	}

	// Fallback to document root
	if fallbackDoc != nil {
		if rng, ok := fallbackDoc.(rootNodeGetter); ok {
			node := rng.GetRootNode()
			if node != nil {
				return node
			}
		}
	}

	return nil
}

// findFieldValueInNode searches for a field's value node in a mapping node
func findFieldValueInNode(node *yaml.Node, fieldName string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode != nil && keyNode.Value == fieldName {
			if i+1 < len(node.Content) {
				return node.Content[i+1]
			}
		}
	}

	return nil
}
