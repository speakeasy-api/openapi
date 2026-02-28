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
	"go.yaml.in/yaml/v4"
)

// ExtensionFilter specifies patterns for filtering extensions using allowed and denied lists.
type ExtensionFilter struct {
	// Keep specifies glob patterns for extensions to keep (allow list).
	// If provided and not empty, ONLY extensions matching these patterns are kept.
	// When both Keep and Remove are provided, extensions matching Keep are kept even if they also match Remove.
	// Use ["*"] to keep all extensions.
	// nil or []: Defaults to deny list mode using Remove patterns, or removes all if Remove is also empty.
	Keep []string `yaml:"keep,omitempty"`

	// Remove specifies glob patterns for extensions to remove (deny list).
	// Extensions matching these patterns are removed.
	// When both Keep and Remove are provided, Keep takes precedence (allow overrides deny).
	// nil or []: If Keep is also empty, removes all extensions.
	Remove []string `yaml:"remove,omitempty"`
}

// SanitizeOptions configures the sanitization behavior.
// Can be loaded from a YAML config file or constructed programmatically.
// Zero values provide aggressive cleanup (remove everything non-standard).
type SanitizeOptions struct {
	// ExtensionPatterns specifies patterns for selective extension filtering.
	// Supports whitelist (Keep), blacklist (Remove), or both.
	//
	// Default behavior:
	//   nil: Remove ALL extensions (default, aggressive cleanup)
	//   &ExtensionFilter{}: Remove ALL extensions (empty whitelist and blacklist)
	//
	// Whitelist mode (Keep provided, Remove empty):
	//   When Keep is provided and not empty, ONLY extensions matching Keep patterns are kept.
	//
	//   &ExtensionFilter{Keep: ["x-speakeasy-*"]}: Keep only x-speakeasy-*, remove all others
	//   &ExtensionFilter{Keep: ["*"]}: Keep ALL extensions (wildcard matches everything)
	//
	// Blacklist mode (Remove provided, Keep empty):
	//   When Keep is empty/nil and Remove is provided, only extensions matching Remove are removed.
	//   All other extensions are kept.
	//
	//   &ExtensionFilter{Remove: ["x-go-*"]}: Remove only x-go-*, keep all others
	//   &ExtensionFilter{Remove: ["x-go-*", "x-internal-*"]}: Remove x-go-* and x-internal-*, keep others
	//
	// Combined mode (both Keep and Remove provided):
	//   When both are provided, Keep takes precedence (whitelist overrides blacklist).
	//   Extensions matching Keep are kept even if they also match Remove.
	//
	//   &ExtensionFilter{Keep: ["x-speakeasy-schema*"], Remove: ["x-speakeasy-*"]}:
	//     Remove all x-speakeasy-{something} extensions except x-speakeasy-schema-{something} (whitelist overrides wildcard blacklist)
	//
	// Examples:
	//   // Remove all extensions (default)
	//   opts := &SanitizeOptions{ExtensionPatterns: nil}
	//
	//   // Remove all extensions (explicit)
	//   opts := &SanitizeOptions{ExtensionPatterns: &ExtensionFilter{}}
	//
	//   // Blacklist: Remove only x-go-* extensions
	//   opts := &SanitizeOptions{ExtensionPatterns: &ExtensionFilter{Remove: []string{"x-go-*"}}}
	//
	//   // Whitelist: Keep only x-speakeasy-* extensions
	//   opts := &SanitizeOptions{ExtensionPatterns: &ExtensionFilter{Keep: []string{"x-speakeasy-*"}}}
	//
	//   // Whitelist: Keep all extensions
	//   opts := &SanitizeOptions{ExtensionPatterns: &ExtensionFilter{Keep: []string{"*"}}}
	//
	//   // Combined: Remove all except x-speakeasy-* (whitelist narrows broad blacklist)
	//   opts := &SanitizeOptions{
	//       ExtensionPatterns: &ExtensionFilter{
	//           Keep:   []string{"x-speakeasy-schema-*"},
	//           Remove: []string{"x-speakeasy-*"}, // Remove all x-speakeasy-* extensions except those matching x-speakeasy-schema-*
	//       },
	//   }
	ExtensionPatterns *ExtensionFilter `yaml:"extensionPatterns,omitempty"`

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

// SanitizeResult contains the results of a sanitization operation.
type SanitizeResult struct {
	// Warnings contains non-fatal issues encountered during sanitization.
	// These typically include invalid glob patterns that were skipped.
	Warnings []string
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
//   - If opts is nil or opts.ExtensionPatterns is nil: removes ALL x-* extensions (default)
//   - If opts.ExtensionPatterns is &ExtensionFilter{}: removes ALL extensions (empty filter)
//   - Use Keep patterns for whitelist mode, Remove patterns for blacklist mode
//
// Example usage:
//
//	// Default sanitization: remove all extensions, unused components, and unknown properties
//	result, err := Sanitize(ctx, doc, nil)
//	if err != nil {
//		return fmt.Errorf("failed to sanitize document: %w", err)
//	}
//	for _, warning := range result.Warnings {
//		fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
//	}
//
//	// Blacklist: Remove only x-go-* extensions, keep everything else
//	opts := &SanitizeOptions{
//		ExtensionPatterns: &ExtensionFilter{Remove: []string{"x-go-*"}},
//		KeepUnusedComponents: true,
//		KeepUnknownProperties: true,
//	}
//	result, err := Sanitize(ctx, doc, opts)
//	if err != nil {
//		return fmt.Errorf("failed to sanitize document: %w", err)
//	}
//
//	// Whitelist: Keep only x-speakeasy-* extensions, remove all others
//	opts := &SanitizeOptions{
//		ExtensionPatterns: &ExtensionFilter{Keep: []string{"x-speakeasy-*"}},
//	}
//	result, err := Sanitize(ctx, doc, opts)
//
// Parameters:
//   - ctx: Context for the operation
//   - doc: The OpenAPI document to sanitize (modified in place)
//   - opts: Sanitization options (nil uses defaults: aggressive cleanup)
//
// Returns:
//   - *SanitizeResult: Result containing any warnings from the operation
//   - error: Any error that occurred during sanitization
func Sanitize(ctx context.Context, doc *OpenAPI, opts *SanitizeOptions) (*SanitizeResult, error) {
	result := &SanitizeResult{}

	if doc == nil {
		return result, nil
	}

	// Use default options if nil
	if opts == nil {
		opts = &SanitizeOptions{}
	}

	// Remove extensions based on configuration
	warnings, err := removeExtensions(ctx, doc, opts)
	if err != nil {
		return result, fmt.Errorf("failed to remove extensions: %w", err)
	}
	result.Warnings = append(result.Warnings, warnings...)

	// Remove unknown properties if not keeping them
	if !opts.KeepUnknownProperties {
		if err := removeUnknownProperties(ctx, doc); err != nil {
			return result, fmt.Errorf("failed to remove unknown properties: %w", err)
		}
	}

	// Clean unused components if not keeping them
	if !opts.KeepUnusedComponents {
		if err := Clean(ctx, doc); err != nil {
			return result, fmt.Errorf("failed to clean unused components: %w", err)
		}
	}

	return result, nil
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
	f, err := os.Open(path) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	return LoadSanitizeConfig(f)
}

// removeExtensions walks through the document and removes extensions based on options.
// determineRemovalAction determines whether an extension should be removed based on filtering rules.
func determineRemovalAction(
	key string,
	removeAll bool,
	hasWhitelist bool,
	keepPatterns []string,
	removePatterns []string,
	keepPatternUsage map[string]*matchInfo,
	removePatternUsage map[string]*matchInfo,
) bool {
	switch {
	case removeAll:
		// Default: remove all extensions
		return true

	case hasWhitelist && len(removePatterns) == 0:
		// Pure whitelist mode: remove unless it matches Keep patterns
		shouldRemove := true // default to remove

		// Check if extension matches any Keep pattern
		for _, pattern := range keepPatterns {
			info := keepPatternUsage[pattern]
			if info == nil {
				continue
			}
			matched, err := filepath.Match(pattern, key)
			if err != nil {
				info.invalid = true
				continue
			}
			if matched {
				info.matched = true
				shouldRemove = false // keep it
				break
			}
		}
		return shouldRemove

	case hasWhitelist && len(removePatterns) > 0:
		// Combined mode: Apply Remove patterns, but Keep overrides
		// First check if it matches Remove patterns
		matchesRemove := false
		for _, pattern := range removePatterns {
			info := removePatternUsage[pattern]
			if info == nil {
				continue
			}
			matched, err := filepath.Match(pattern, key)
			if err != nil {
				info.invalid = true
				continue
			}
			if matched {
				matchesRemove = true
				break
			}
		}

		if matchesRemove {
			// It matches Remove, but check if Keep overrides
			shouldRemove := true // default to remove
			for _, pattern := range keepPatterns {
				info := keepPatternUsage[pattern]
				if info == nil {
					continue
				}
				matched, err := filepath.Match(pattern, key)
				if err != nil {
					info.invalid = true
					continue
				}
				if matched {
					info.matched = true
					shouldRemove = false // keep it (whitelist overrides blacklist)
					break
				}
			}

			// Track that Remove pattern matched (even if Keep overrode it)
			if matchesRemove {
				for _, pattern := range removePatterns {
					matched, err := filepath.Match(pattern, key)
					if err == nil && matched {
						if info := removePatternUsage[pattern]; info != nil {
							info.matched = true
						}
					}
				}
			}
			return shouldRemove
		}
		// If it doesn't match Remove patterns, keep it (not affected)
		return false

	case len(removePatterns) > 0:
		// Pure blacklist mode: keep unless it matches Remove patterns
		shouldRemove := false // default to keep

		// Check if extension matches any Remove pattern
		for _, pattern := range removePatterns {
			info := removePatternUsage[pattern]
			if info == nil {
				continue
			}
			matched, err := filepath.Match(pattern, key)
			if err != nil {
				info.invalid = true
				continue
			}
			if matched {
				info.matched = true
				shouldRemove = true // remove it (matches blacklist)
				break
			}
		}
		return shouldRemove

	default:
		return false
	}
}

// matchInfo tracks pattern usage and validity for warning generation.
type matchInfo struct {
	invalid bool // true if pattern has invalid syntax
	matched bool // true if pattern matched at least one extension
}

// Returns a slice of warnings for invalid patterns or patterns that matched nothing.
func removeExtensions(ctx context.Context, doc *OpenAPI, opts *SanitizeOptions) ([]string, error) {
	// Determine removal strategy based on ExtensionPatterns:
	// - nil: remove ALL extensions (default)
	// - &ExtensionFilter{}: remove ALL extensions (empty whitelist and blacklist)
	// - &ExtensionFilter{Keep: [...}}: whitelist mode - keep only matching extensions
	// - &ExtensionFilter{Remove: [...}}: blacklist mode - remove only matching extensions
	// - &ExtensionFilter{Keep: [...], Remove: [...}}: whitelist overrides blacklist

	var keepPatterns, removePatterns []string
	hasWhitelist := false
	removeAll := true // Default: remove all extensions

	if opts != nil && opts.ExtensionPatterns != nil {
		filter := opts.ExtensionPatterns

		// Check for whitelist (Keep patterns)
		if len(filter.Keep) > 0 {
			keepPatterns = filter.Keep
			hasWhitelist = true
			removeAll = false
		}

		// Check for blacklist (Remove patterns)
		if len(filter.Remove) > 0 {
			removePatterns = filter.Remove
			if !hasWhitelist {
				// Blacklist mode only (no whitelist)
				removeAll = false
			}
		}

		// If both Keep and Remove are empty, remove all (empty filter)
		if len(filter.Keep) == 0 && len(filter.Remove) == 0 {
			removeAll = true
		}
	}

	// Track pattern usage for warnings
	keepPatternUsage := make(map[string]*matchInfo)
	removePatternUsage := make(map[string]*matchInfo)

	// Initialize tracking for all patterns
	for _, pattern := range keepPatterns {
		keepPatternUsage[pattern] = &matchInfo{}
	}
	for _, pattern := range removePatterns {
		removePatternUsage[pattern] = &matchInfo{}
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
					shouldRemove := determineRemovalAction(
						key,
						removeAll,
						hasWhitelist,
						keepPatterns,
						removePatterns,
						keepPatternUsage,
						removePatternUsage,
					)

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
			return nil, fmt.Errorf("failed to process extensions: %w", err)
		}
	}

	// Generate warnings for invalid patterns and patterns that never matched
	var warnings []string

	// Check Keep patterns
	for _, pattern := range keepPatterns {
		info := keepPatternUsage[pattern]
		if info == nil {
			continue
		}
		if info.invalid {
			warnings = append(warnings, fmt.Sprintf("invalid keep pattern '%s' was skipped", pattern))
		} else if !info.matched {
			warnings = append(warnings, fmt.Sprintf("keep pattern '%s' did not match any extensions in the document", pattern))
		}
	}

	// Check Remove patterns
	for _, pattern := range removePatterns {
		info := removePatternUsage[pattern]
		if info == nil {
			continue
		}
		if info.invalid {
			warnings = append(warnings, fmt.Sprintf("invalid remove pattern '%s' was skipped", pattern))
		} else if !info.matched {
			warnings = append(warnings, fmt.Sprintf("remove pattern '%s' did not match any extensions in the document", pattern))
		}
	}

	return warnings, nil
}

// removeUnknownProperties removes properties that are not defined in the OpenAPI specification.
// It uses the UnknownProperties list tracked during unmarshalling to identify and remove
// unknown keys from the YAML nodes.
func removeUnknownProperties(ctx context.Context, doc *OpenAPI) error {
	// Walk through the document and clean unknown properties from all models
	// We need specific matchers for wrapped types (Referenced*, JSONSchema)
	for item := range Walk(ctx, doc) {
		err := item.Match(Matcher{
			Any:    cleanUnknownPropertiesFromModel,
			Schema: cleanUnknownPropertiesFromJSONSchema,
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

	var directCore any
	if coreModel, ok := model.(coreGetter); ok {
		directCore = coreModel.GetCoreAny()
		if directCore != nil {
			if coreModeler, ok := directCore.(marshaller.CoreModeler); ok {
				if len(coreModeler.GetUnknownProperties()) > 0 {
					return directCore
				}
			} else {
				return directCore
			}
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
			if innerCore := getCoreModelFromAny(inner); innerCore != nil {
				return innerCore
			}
		}
	}

	return directCore
}

// getRootNodeFromAny attempts to extract the root yaml.Node from various OpenAPI types.
// This is used for node-to-operation mapping during indexing.
func getRootNodeFromAny(model any) *yaml.Node {
	if model == nil {
		return nil
	}

	// Try direct GetRootNode()
	type rootNodeGetter interface {
		GetRootNode() *yaml.Node
	}

	if getter, ok := model.(rootNodeGetter); ok {
		return getter.GetRootNode()
	}

	// Try navigable node (for EitherValue wrappers)
	type navigableNoder interface {
		GetNavigableNode() (any, error)
	}

	if navigable, ok := model.(navigableNoder); ok {
		inner, err := navigable.GetNavigableNode()
		if err == nil && inner != nil {
			// Recursively try to get root node from the inner value
			return getRootNodeFromAny(inner)
		}
	}

	// Try to get core model and extract root node from there
	if core := getCoreModelFromAny(model); core != nil {
		if getter, ok := core.(rootNodeGetter); ok {
			return getter.GetRootNode()
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
	removeSet := make(map[string]struct{}, len(keysToRemove))
	for _, key := range keysToRemove {
		removeSet[key] = struct{}{}
	}

	// Filter content to exclude keys in the remove set
	newContent := make([]*yaml.Node, 0, len(node.Content))
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}

		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Kind == yaml.ScalarNode {
			if _, shouldRemove := removeSet[keyNode.Value]; shouldRemove {
				// Skip this key-value pair (it's unknown)
				continue
			}
		}

		// Keep this key-value pair
		newContent = append(newContent, keyNode, valueNode)
	}

	// Update the node's content
	node.Content = newContent
}
