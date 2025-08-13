package marshaller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/json"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v4"
)

type CoreModeler interface {
	GetRootNode() *yaml.Node
	SetRootNode(rootNode *yaml.Node)
	GetValid() bool
	GetValidYaml() bool
	SetValid(valid, validYaml bool)
	DetermineValidity(errs []error)
	SetConfig(config *yml.Config)
	GetConfig() *yml.Config
	Marshal(ctx context.Context, w io.Writer) error
}

type CoreModel struct {
	RootNode  *yaml.Node  // RootNode is the node that was unmarshaled into this model
	Valid     bool        // Valid indicates whether the model passed validation, ie all its required fields were present and ValidYaml is true
	ValidYaml bool        // ValidYaml indicates whether the model's underlying YAML representation is valid, for example a mapping node was received for a model
	Config    *yml.Config // Generally only set on the top-level model that was unmarshaled
}

var _ CoreModeler = (*CoreModel)(nil)

func (c CoreModel) GetRootNode() *yaml.Node {
	return c.RootNode
}

func (c CoreModel) GetRootNodeLine() int {
	if c.RootNode == nil {
		return -1
	}
	return c.RootNode.Line
}

func (c *CoreModel) SetRootNode(rootNode *yaml.Node) {
	c.RootNode = rootNode
}

func (c CoreModel) GetValid() bool {
	return c.Valid
}

func (c CoreModel) GetValidYaml() bool {
	return c.ValidYaml
}

func (c *CoreModel) DetermineValidity(errs []error) {
	containsYamlErr := false
	containsValidationErr := false

	for _, err := range errs {
		if errors.Is(err, &validation.TypeMismatchError{}) {
			containsYamlErr = true
		} else {
			containsValidationErr = true
		}
	}

	c.SetValid(!containsValidationErr, !containsYamlErr)
}

func (c *CoreModel) SetValid(valid, validYaml bool) {
	c.Valid = valid && validYaml
	c.ValidYaml = validYaml
}

func (c *CoreModel) SetConfig(config *yml.Config) {
	c.Config = config
}

func (c *CoreModel) GetConfig() *yml.Config {
	return c.Config
}

// GetJSONPointer returns the JSON pointer path from the topLevelRootNode to this CoreModel's RootNode.
// Returns an empty string if the node is not found or if either node is nil.
// The returned pointer follows RFC6901 format (e.g., "/path/to/node").
func (c *CoreModel) GetJSONPointer(topLevelRootNode *yaml.Node) string {
	if c.RootNode == nil || topLevelRootNode == nil {
		return ""
	}

	// If the nodes are the same, return root pointer
	if c.RootNode == topLevelRootNode {
		return "/"
	}

	// Find the path from topLevelRootNode to c.RootNode
	path := findNodePath(topLevelRootNode, c.RootNode, []string{})
	if path == nil {
		return ""
	}

	// Convert path to JSON pointer format
	return buildJSONPointer(path)
}

// Marshal will marshal the core model to the provided io.Writer.
// This method handles both YAML and JSON output based on the context configuration.
func (c *CoreModel) Marshal(ctx context.Context, w io.Writer) error {
	cfg := yml.GetConfigFromContext(ctx)

	switch cfg.OutputFormat {
	case yml.OutputFormatYAML:
		// Check if we need to reset node styles (original was JSON, now want YAML)
		if cfg.OriginalFormat == yml.OutputFormatJSON && cfg.OutputFormat == yml.OutputFormatYAML {
			resetNodeStylesForYAML(c.RootNode, cfg)
		}

		enc := yaml.NewEncoder(w)
		enc.SetIndent(cfg.Indentation)
		if err := enc.Encode(c.RootNode); err != nil {
			return err
		}
	case yml.OutputFormatJSON:
		if err := json.YAMLToJSON(c.RootNode, cfg.Indentation, w); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %s", cfg.OutputFormat)
	}

	return nil
}

// resetNodeStylesForYAML recursively resets node styles to force YAML output
// This is used when converting from JSON input to YAML output
func resetNodeStylesForYAML(node *yaml.Node, cfg *yml.Config) {
	resetNodeStylesForYAMLRecursive(node, cfg, false)
}

// resetNodeStylesForYAMLRecursive handles the recursive traversal with key/value context
func resetNodeStylesForYAMLRecursive(node *yaml.Node, cfg *yml.Config, isKey bool) {
	if node == nil {
		return
	}

	// Use the config's default styles for proper YAML formatting
	if node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
		if isKey {
			// Use KeyStringStyle for map keys
			node.Style = cfg.KeyStringStyle
		} else {
			// Use ValueStringStyle for values
			node.Style = cfg.ValueStringStyle
		}
	} else {
		// Reset style to default (YAML style) for non-string nodes
		node.Style = 0
	}

	// Handle different node types with proper key/value context
	switch node.Kind {
	case yaml.MappingNode:
		// In mapping nodes, alternate between keys (even indices) and values (odd indices)
		for i, child := range node.Content {
			isChildKey := (i%2 == 0) // Even indices are keys, odd indices are values
			resetNodeStylesForYAMLRecursive(child, cfg, isChildKey)
		}
	case yaml.SequenceNode, yaml.DocumentNode:
		// In sequences and documents, all children are values
		for _, child := range node.Content {
			resetNodeStylesForYAMLRecursive(child, cfg, false)
		}
	}

	// Handle alias nodes
	if node.Alias != nil {
		resetNodeStylesForYAMLRecursive(node.Alias, cfg, isKey)
	}
}

// findNodePath recursively searches for targetNode within rootNode and returns the path as a slice of strings.
// Returns nil if the target node is not found.
func findNodePath(rootNode, targetNode *yaml.Node, currentPath []string) []string {
	if rootNode == nil || targetNode == nil {
		return nil
	}

	// Resolve aliases
	resolvedRoot := resolveAlias(rootNode)
	resolvedTarget := resolveAlias(targetNode)

	// Check if we found the target node
	if resolvedRoot == resolvedTarget {
		return currentPath
	}

	// Handle DocumentNode by searching its content
	if resolvedRoot.Kind == yaml.DocumentNode {
		if len(resolvedRoot.Content) > 0 {
			return findNodePath(resolvedRoot.Content[0], targetNode, currentPath)
		}
		return nil
	}

	// Search through different node types
	switch resolvedRoot.Kind {
	case yaml.MappingNode:
		return findNodePathInMapping(resolvedRoot, targetNode, currentPath)
	case yaml.SequenceNode:
		return findNodePathInSequence(resolvedRoot, targetNode, currentPath)
	}

	return nil
}

// findNodePathInMapping searches for targetNode within a mapping node
func findNodePathInMapping(mappingNode, targetNode *yaml.Node, currentPath []string) []string {
	// YAML mapping nodes have content in pairs: [key1, value1, key2, value2, ...]
	for i := 0; i < len(mappingNode.Content); i += 2 {
		if i+1 >= len(mappingNode.Content) {
			break // Malformed mapping, skip
		}

		keyNode := mappingNode.Content[i]
		valueNode := mappingNode.Content[i+1]

		// Get the key string for the path
		keyStr := getNodeKeyString(keyNode)
		if keyStr == "" {
			continue // Skip if we can't get a valid key
		}

		// Create new path with this key
		newPath := currentPath
		newPath = append(newPath, keyStr)

		// Check if the key node itself is our target
		if keyNode == targetNode {
			return newPath // Return path pointing to the key (which resolves to value)
		}

		// Check if the value node is our target
		if result := findNodePath(valueNode, targetNode, newPath); result != nil {
			return result
		}
	}

	return nil
}

// findNodePathInSequence searches for targetNode within a sequence node
func findNodePathInSequence(sequenceNode, targetNode *yaml.Node, currentPath []string) []string {
	for i, childNode := range sequenceNode.Content {
		// Create new path with this index
		newPath := currentPath
		newPath = append(newPath, strconv.Itoa(i))

		// Check if this child node is our target or contains our target
		if result := findNodePath(childNode, targetNode, newPath); result != nil {
			return result
		}
	}

	return nil
}

// resolveAlias resolves alias nodes to their actual content
func resolveAlias(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	// Follow alias chain
	for node.Kind == yaml.AliasNode && node.Alias != nil {
		node = node.Alias
	}

	return node
}

// getNodeKeyString extracts a string representation from a key node
func getNodeKeyString(keyNode *yaml.Node) string {
	if keyNode == nil {
		return ""
	}

	// Resolve aliases
	resolved := resolveAlias(keyNode)
	if resolved == nil || resolved.Kind != yaml.ScalarNode {
		return ""
	}

	return resolved.Value
}

// buildJSONPointer converts a path slice to a JSON pointer string following RFC6901
func buildJSONPointer(path []string) string {
	if len(path) == 0 {
		return "/"
	}

	var sb strings.Builder
	for _, part := range path {
		sb.WriteByte('/')
		sb.WriteString(escapeJSONPointerToken(part))
	}

	return sb.String()
}

// escapeJSONPointerToken escapes a string for use as a reference token in a JSON pointer according to RFC6901.
// It replaces "~" with "~0" and "/" with "~1" as required by the specification.
func escapeJSONPointerToken(s string) string {
	// Replace ~ with ~0 first, then / with ~1 (order matters!)
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}
