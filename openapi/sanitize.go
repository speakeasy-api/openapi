package openapi

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

// SanitizeOptions configures the sanitization behavior.
// Can be loaded from a YAML config file or constructed programmatically.
// Zero values provide aggressive cleanup (remove everything non-standard).
type SanitizeOptions struct {
	// ExtensionPatterns specifies glob patterns for selective extension removal.
	// nil: Remove ALL extensions (default)
	// []: Keep ALL extensions (empty array)
	// ["x-go-*", ...]: Remove only extensions matching these patterns
	ExtensionPatterns []string `yaml:"extensionPatterns"`

	// KeepUnusedComponents preserves unused components in the document.
	// Default (false): removes unused components.
	// Set to true to preserve all components regardless of usage.
	KeepUnusedComponents bool `yaml:"keepUnusedComponents,omitempty"`

	// KeepUnknownProperties preserves properties not defined in the OpenAPI specification.
	// Default (false): removes unknown/unrecognized properties.
	// Set to true to preserve all properties even if not in the OpenAPI spec.
	// Note: Extensions (x-*) are handled separately by ExtensionPatterns.
	KeepUnknownProperties bool `yaml:"keepUnknownProperties,omitempty"`
}

// Sanitize cleans an OpenAPI document by removing unwanted elements.
// By default (nil options or zero values), it provides aggressive cleanup:
//   - Removes all extensions (x-*)
//   - Removes unused components
//   - Removes unknown properties
//
// This function modifies the document in place.
//
// Why use Sanitize?
//
//   - **Standards compliance**: Remove vendor-specific extensions for standards-compliant specs
//   - **Clean distribution**: Prepare specifications for public sharing or publishing
//   - **Reduce document size**: Remove unnecessary extensions, components, and properties
//   - **Selective cleanup**: Use patterns to target specific extension families
//   - **Integration ready**: Combine multiple cleanup operations in one pass
//
// What gets sanitized by default:
//
//   - All x-* extensions throughout the document
//   - Unused components (schemas, responses, parameters, etc.)
//   - Unknown properties not defined in the OpenAPI specification
//
// Extension removal behavior:
//   - If opts is nil or opts.ExtensionPatterns is empty: removes ALL x-* extensions
//   - If opts.ExtensionPatterns has values: removes only matching extensions
//
// Example usage:
//
//	// Default sanitization: remove all extensions, unused components, and unknown properties
//	err := Sanitize(ctx, doc, nil)
//	if err != nil {
//		return fmt.Errorf("failed to sanitize document: %w", err)
//	}
//
//	// Remove only x-go-* extensions, keep everything else
//	opts := &SanitizeOptions{
//		ExtensionPatterns: []string{"x-go-*"},
//		KeepUnusedComponents: true,
//		KeepUnknownProperties: true,
//	}
//	err := Sanitize(ctx, doc, opts)
//	if err != nil {
//		return fmt.Errorf("failed to sanitize document: %w", err)
//	}
//
//	// Remove extensions and unknown properties, but keep components
//	opts := &SanitizeOptions{
//		KeepUnusedComponents: true,
//	}
//	err := Sanitize(ctx, doc, opts)
//
// Parameters:
//   - ctx: Context for the operation
//   - doc: The OpenAPI document to sanitize (modified in place)
//   - opts: Sanitization options (nil uses defaults: aggressive cleanup)
//
// Returns:
//   - error: Any error that occurred during sanitization
func Sanitize(ctx context.Context, doc *OpenAPI, opts *SanitizeOptions) error {
	if doc == nil {
		return nil
	}

	// Use default options if nil
	if opts == nil {
		opts = &SanitizeOptions{}
	}

	// Remove extensions based on configuration
	if err := removeExtensions(ctx, doc, opts); err != nil {
		return fmt.Errorf("failed to remove extensions: %w", err)
	}

	// Remove unknown properties if not keeping them
	if !opts.KeepUnknownProperties {
		if err := removeUnknownProperties(ctx, doc); err != nil {
			return fmt.Errorf("failed to remove unknown properties: %w", err)
		}
	}

	// Clean unused components if not keeping them
	if !opts.KeepUnusedComponents {
		if err := Clean(ctx, doc); err != nil {
			return fmt.Errorf("failed to clean unused components: %w", err)
		}
	}

	return nil
}

// LoadSanitizeConfig loads sanitize configuration from a YAML reader.
func LoadSanitizeConfig(r io.Reader) (*SanitizeOptions, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var opts SanitizeOptions
	if err := yaml.Unmarshal(data, &opts); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &opts, nil
}

// LoadSanitizeConfigFromFile loads sanitize configuration from a YAML file.
func LoadSanitizeConfigFromFile(path string) (*SanitizeOptions, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	return LoadSanitizeConfig(f)
}

// removeExtensions walks through the document and removes extensions based on options.
func removeExtensions(ctx context.Context, doc *OpenAPI, opts *SanitizeOptions) error {
	// Determine removal strategy:
	// - nil ExtensionPatterns: remove ALL extensions (default)
	// - empty array []: keep ALL extensions (explicit no-op)
	// - non-empty array: remove only matching patterns

	if opts != nil && opts.ExtensionPatterns != nil && len(opts.ExtensionPatterns) == 0 {
		// Empty array explicitly set = keep all extensions
		return nil
	}

	var patterns []string
	removeAll := true

	if opts != nil && opts.ExtensionPatterns != nil && len(opts.ExtensionPatterns) > 0 {
		// Use patterns for selective removal
		patterns = opts.ExtensionPatterns
		removeAll = false
	}

	// Walk through the document and process all Extensions
	for item := range Walk(ctx, doc) {
		err := item.Match(Matcher{
			Extensions: func(ext *extensions.Extensions) error {
				if ext == nil || ext.Len() == 0 {
					return nil
				}

				// Collect keys to remove
				keysToRemove := []string{}
				for key := range ext.All() {
					shouldRemove := false

					if removeAll {
						// Remove all extensions
						shouldRemove = true
					} else {
						// Check if extension matches any pattern
						shouldRemove = matchesAnyPattern(key, patterns)
					}

					if shouldRemove {
						keysToRemove = append(keysToRemove, key)
					}
				}

				// Remove the identified keys
				for _, key := range keysToRemove {
					ext.Delete(key)
				}

				return nil
			},
		})
		if err != nil {
			return fmt.Errorf("failed to process extensions: %w", err)
		}
	}

	return nil
}

// removeUnknownProperties removes properties that are not defined in the OpenAPI specification.
// It uses the UnknownProperties list tracked during unmarshalling to identify and remove
// unknown keys from the YAML nodes.
func removeUnknownProperties(ctx context.Context, doc *OpenAPI) error {
	// Walk through the document and clean unknown properties from all models
	// We need specific matchers for wrapped types (Referenced*, JSONSchema)
	for item := range Walk(ctx, doc) {
		err := item.Match(Matcher{
			Any: func(model any) error {
				return cleanUnknownPropertiesFromModel(model)
			},
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				return cleanUnknownPropertiesFromJSONSchema(schema)
			},
			// Handle all Referenced types by extracting their Object
			ReferencedResponse: func(ref *ReferencedResponse) error {
				if ref != nil && !ref.IsReference() && ref.Object != nil {
					return cleanUnknownPropertiesFromModel(ref.Object)
				}
				return nil
			},
			ReferencedParameter: func(ref *ReferencedParameter) error {
				if ref != nil && !ref.IsReference() && ref.Object != nil {
					return cleanUnknownPropertiesFromModel(ref.Object)
				}
				return nil
			},
			ReferencedRequestBody: func(ref *ReferencedRequestBody) error {
				if ref != nil && !ref.IsReference() && ref.Object != nil {
					return cleanUnknownPropertiesFromModel(ref.Object)
				}
				return nil
			},
			ReferencedHeader: func(ref *ReferencedHeader) error {
				if ref != nil && !ref.IsReference() && ref.Object != nil {
					return cleanUnknownPropertiesFromModel(ref.Object)
				}
				return nil
			},
			ReferencedExample: func(ref *ReferencedExample) error {
				if ref != nil && !ref.IsReference() && ref.Object != nil {
					return cleanUnknownPropertiesFromModel(ref.Object)
				}
				return nil
			},
			ReferencedLink: func(ref *ReferencedLink) error {
				if ref != nil && !ref.IsReference() && ref.Object != nil {
					return cleanUnknownPropertiesFromModel(ref.Object)
				}
				return nil
			},
			ReferencedCallback: func(ref *ReferencedCallback) error {
				if ref != nil && !ref.IsReference() && ref.Object != nil {
					return cleanUnknownPropertiesFromModel(ref.Object)
				}
				return nil
			},
			ReferencedPathItem: func(ref *ReferencedPathItem) error {
				if ref != nil && !ref.IsReference() && ref.Object != nil {
					return cleanUnknownPropertiesFromModel(ref.Object)
				}
				return nil
			},
			ReferencedSecurityScheme: func(ref *ReferencedSecurityScheme) error {
				if ref != nil && !ref.IsReference() && ref.Object != nil {
					return cleanUnknownPropertiesFromModel(ref.Object)
				}
				return nil
			},
		})
		if err != nil {
			return fmt.Errorf("failed to clean unknown properties: %w", err)
		}
	}

	return nil
}

// cleanUnknownPropertiesFromJSONSchema handles JSONSchema wrappers
func cleanUnknownPropertiesFromJSONSchema(js *oas3.JSONSchema[oas3.Referenceable]) error {
	if js == nil || !js.IsSchema() {
		return nil // Skip boolean schemas
	}

	schema := js.GetSchema()
	if schema == nil {
		return nil
	}

	// Clean unknown properties from the schema
	return cleanUnknownPropertiesFromModel(schema)
}

// cleanUnknownPropertiesFromModel removes unknown properties from a model's YAML node
// using the UnknownProperties list tracked during unmarshalling.
func cleanUnknownPropertiesFromModel(model any) error {
	// Try to get the core model
	core := getCoreModelFromAny(model)
	if core == nil {
		return nil // No core model found
	}

	// Check if core implements CoreModeler (has UnknownProperties)
	coreModeler, ok := core.(marshaller.CoreModeler)
	if !ok {
		return nil // Core doesn't implement CoreModeler
	}

	unknownProps := coreModeler.GetUnknownProperties()
	if len(unknownProps) == 0 {
		return nil // No unknown properties to remove
	}

	rootNode := coreModeler.GetRootNode()
	if rootNode == nil {
		return nil // No root node
	}

	// Remove unknown properties from the root node
	removePropertiesFromNode(rootNode, unknownProps)

	return nil
}

// getCoreModelFromAny attempts to extract a core model from various wrapper types
func getCoreModelFromAny(model any) any {
	// Try direct core getter
	type coreGetter interface {
		GetCoreAny() any
	}

	if coreModel, ok := model.(coreGetter); ok {
		core := coreModel.GetCoreAny()
		if core != nil {
			return core
		}
	}

	// Try navigable node (for EitherValue wrappers)
	type navigableNoder interface {
		GetNavigableNode() (any, error)
	}

	if navigable, ok := model.(navigableNoder); ok {
		inner, err := navigable.GetNavigableNode()
		if err == nil && inner != nil {
			// Recursively try to get core from the inner value
			return getCoreModelFromAny(inner)
		}
	}

	return nil
}

// removePropertiesFromNode removes the specified property keys from a YAML mapping node.
func removePropertiesFromNode(node *yaml.Node, keysToRemove []string) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}

	// Build a set of keys to remove for efficient lookup
	removeSet := make(map[string]bool, len(keysToRemove))
	for _, key := range keysToRemove {
		removeSet[key] = true
	}

	// Filter content to exclude keys in the remove set
	newContent := make([]*yaml.Node, 0, len(node.Content))
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}

		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Kind == yaml.ScalarNode && removeSet[keyNode.Value] {
			// Skip this key-value pair (it's unknown)
			continue
		}

		// Keep this key-value pair
		newContent = append(newContent, keyNode, valueNode)
	}

	// Update the node's content
	node.Content = newContent
}

// matchesAnyPattern checks if a string matches any of the provided glob patterns.
func matchesAnyPattern(str string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, str)
		if err != nil {
			// Invalid pattern, skip it
			continue
		}
		if matched {
			return true
		}
	}
	return false
}
