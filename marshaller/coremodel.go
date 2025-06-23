package marshaller

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/speakeasy-api/openapi/json"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
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
		fmt.Printf("DEBUG: Taking JSON path\n")
		if err := json.YAMLToJSON(c.RootNode, cfg.Indentation, w); err != nil {
			return err
		}
	default:
		fmt.Printf("DEBUG: Unknown output format: %v\n", cfg.OutputFormat)
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
