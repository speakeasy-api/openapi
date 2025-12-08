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
	"gopkg.in/yaml.v3"
)

type CoreModeler interface {
	GetRootNode() *yaml.Node
	SetRootNode(rootNode *yaml.Node)
	SetDocumentNode(documentNode *yaml.Node)
	GetValid() bool
	GetValidYaml() bool
	SetValid(valid, validYaml bool)
	DetermineValidity(errs []error)
	SetConfig(config *yml.Config)
	GetConfig() *yml.Config
	Marshal(ctx context.Context, w io.Writer) error
	SetUnknownProperties(props []string)
	GetUnknownProperties() []string
}

type CoreModel struct {
	RootNode          *yaml.Node  // RootNode is the node that was unmarshaled into this model
	DocumentNode      *yaml.Node  // DocumentNode is the top-level document node (only set for top-level models) - contains header comments
	Valid             bool        // Valid indicates whether the model passed validation, ie all its required fields were present and ValidYaml is true
	ValidYaml         bool        // ValidYaml indicates whether the model's underlying YAML representation is valid, for example a mapping node was received for a model
	Config            *yml.Config // Generally only set on the top-level model that was unmarshaled
	UnknownProperties []string    // UnknownProperties lists property keys that were present in the YAML but not defined in the model (excludes extensions which start with "x-")
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

	// If we have a DocumentNode, update its content to point to the new RootNode
	if c.DocumentNode != nil && c.DocumentNode.Kind == yaml.DocumentNode {
		c.DocumentNode.Content = []*yaml.Node{rootNode}
	}
}

func (c *CoreModel) SetDocumentNode(documentNode *yaml.Node) {
	c.DocumentNode = documentNode
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

func (c *CoreModel) SetUnknownProperties(props []string) {
	c.UnknownProperties = props
}

func (c *CoreModel) GetUnknownProperties() []string {
	if c.UnknownProperties == nil {
		return []string{}
	}

	return c.UnknownProperties
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

// GetJSONPath returns the JSONPath expression from the topLevelRootNode to this CoreModel's RootNode.
// Returns an empty string if the node is not found or if either node is nil.
// The returned path follows JSONPath format (e.g., "$.paths['/some/path'].get").
func (c *CoreModel) GetJSONPath(topLevelRootNode *yaml.Node) string {
	if c == nil || c.RootNode == nil || topLevelRootNode == nil {
		return ""
	}

	// If the nodes are the same, return root path
	if c.RootNode == topLevelRootNode {
		return "$"
	}

	// Find the path from topLevelRootNode to c.RootNode
	path := findNodePath(topLevelRootNode, c.RootNode, []string{})
	if path == nil {
		return ""
	}

	// Convert path to JSONPath format
	return buildJSONPath(path)
}

// Marshal will marshal the core model to the provided io.Writer.
// This method handles both YAML and JSON output based on the context configuration.
func (c *CoreModel) Marshal(ctx context.Context, w io.Writer) error {
	cfg := yml.GetConfigFromContext(ctx)

	// Use DocumentNode if available for YAML output (to preserve comments)
	// For JSON output, use RootNode since JSON doesn't support comments
	nodeToMarshal := c.RootNode
	if c.DocumentNode != nil && cfg.OutputFormat == yml.OutputFormatYAML {
		nodeToMarshal = c.DocumentNode
	}

	switch cfg.OutputFormat {
	case yml.OutputFormatYAML:
		// Check if we need to reset node styles (original was JSON, now want YAML)
		if cfg.OriginalFormat == yml.OutputFormatJSON && cfg.OutputFormat == yml.OutputFormatYAML {
			resetNodeStylesForYAML(nodeToMarshal, cfg)
		}

		enc := yaml.NewEncoder(w)
		enc.SetIndent(cfg.Indentation)
		if err := enc.Encode(nodeToMarshal); err != nil {
			return err
		}
	case yml.OutputFormatJSON:
		if err := json.YAMLToJSONWithConfig(nodeToMarshal, cfg.IndentationStyle.ToIndent(), cfg.Indentation, cfg.TrailingNewline, w); err != nil {
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

// buildJSONPath converts a path slice to a JSONPath expression
func buildJSONPath(path []string) string {
	if len(path) == 0 {
		return "$"
	}

	var sb strings.Builder
	sb.WriteByte('$')

	for _, part := range path {
		// Check if the part is a numeric index (for arrays)
		if _, err := strconv.Atoi(part); err == nil {
			// Array index - use bracket notation
			sb.WriteByte('[')
			sb.WriteString(part)
			sb.WriteByte(']')
		} else {
			// Object property - check if it needs bracket notation
			if needsBracketNotation(part) {
				sb.WriteByte('[')
				sb.WriteByte('\'')
				sb.WriteString(escapeJSONPathProperty(part))
				sb.WriteByte('\'')
				sb.WriteByte(']')
			} else {
				// Use dot notation for simple properties
				sb.WriteByte('.')
				sb.WriteString(part)
			}
		}
	}

	return sb.String()
}

// needsBracketNotation determines if a property name needs bracket notation
func needsBracketNotation(s string) bool {
	// Use bracket notation for properties that:
	// - Start with a slash (like "/users/{id}")
	// - Contain special characters like {, }, /, spaces, etc.
	// - Are not simple identifiers
	for _, r := range s {
		if r == '/' || r == '{' || r == '}' || r == ' ' || r == '-' || r == '.' {
			return true
		}
	}
	return false
}

// escapeJSONPathProperty escapes a property name for use in JSONPath bracket notation.
func escapeJSONPathProperty(s string) string {
	// Escape single quotes by doubling them
	return strings.ReplaceAll(s, "'", "''")
}
