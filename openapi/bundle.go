package openapi

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/speakeasy-api/openapi/hashing"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// BundleNamingStrategy defines how external references should be named when bundled.
type BundleNamingStrategy int

const (
	// BundleNamingCounter uses counter-based suffixes like User_1, User_2 for conflicts
	BundleNamingCounter BundleNamingStrategy = iota
	// BundleNamingFilePath uses file path-based naming like file_path_somefile_yaml~User
	BundleNamingFilePath
)

// BundleOptions represents the options available when bundling an OpenAPI document.
type BundleOptions struct {
	// ResolveOptions are the options to use when resolving references during bundling.
	ResolveOptions ResolveOptions
	// NamingStrategy determines how external references are named when brought into components.
	NamingStrategy BundleNamingStrategy
}

// Bundle transforms an OpenAPI document by bringing all external references into the components section,
// creating a self-contained document that maintains the reference structure but doesn't depend on external files.
// This operation modifies the document in place.
//
// Why use Bundle?
//
//   - **Create portable documents**: Combine multiple OpenAPI files into a single document while preserving references
//   - **Maintain reference structure**: Keep the benefits of references for tooling that supports them
//   - **Simplify distribution**: Share a single file that contains all dependencies
//   - **Optimize for reference-aware tools**: Provide complete documents to tools that work well with references
//   - **Prepare for further processing**: Create a foundation for subsequent inlining or other transformations
//   - **Handle complex API architectures**: Combine modular API definitions into unified specifications
//
// What you'll get:
//
// Before bundling:
//
//	{
//	  "openapi": "3.1.0",
//	  "paths": {
//	    "/users": {
//	      "get": {
//	        "responses": {
//	          "200": {
//	            "content": {
//	              "application/json": {
//	                "schema": {
//	                  "$ref": "external_api.yaml#/User"
//	                }
//	              }
//	            }
//	          }
//	        }
//	      }
//	    }
//	  }
//	}
//
// After bundling (with BundleNamingFilePath):
//
//	{
//	  "openapi": "3.1.0",
//	  "paths": {
//	    "/users": {
//	      "get": {
//	        "responses": {
//	          "200": {
//	            "content": {
//	              "application/json": {
//	                "schema": {
//	                  "$ref": "#/components/schemas/external_api_yaml~User"
//	                }
//	              }
//	            }
//	          }
//	        }
//	      }
//	    }
//	  },
//	  "components": {
//	    "schemas": {
//	      "external_api_yaml~User": {
//	        "type": "object",
//	        "properties": {
//	          "id": {"type": "string"},
//	          "name": {"type": "string"}
//	        }
//	      }
//	    }
//	  }
//	}
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - doc: The OpenAPI document to bundle (modified in place)
//   - opts: Configuration options for bundling
//
// Returns:
//   - error: Any error that occurred during bundling
func Bundle(ctx context.Context, doc *OpenAPI, opts BundleOptions) error {
	if doc == nil {
		return nil
	}

	// Track external references and their new component names
	externalRefs := make(map[string]string) // original ref -> new component name
	componentSchemas := sequencedmap.New[string, *oas3.JSONSchema[oas3.Referenceable]]()
	componentNames := make(map[string]bool) // track used names to avoid conflicts
	schemaHashes := make(map[string]string) // component name -> hash for conflict detection

	// Initialize existing component names and hashes to avoid conflicts
	if doc.Components != nil && doc.Components.Schemas != nil {
		for name, schema := range doc.Components.Schemas.All() {
			componentNames[name] = true
			if schema != nil {
				schemaHashes[name] = hashing.Hash(schema)
			}
		}
	}

	// First pass: collect all external references and resolve them
	for item := range Walk(ctx, doc) {
		err := item.Match(Matcher{
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				return bundleSchema(ctx, schema, opts, externalRefs, componentSchemas, componentNames, schemaHashes)
			},
			ReferencedPathItem: func(ref *ReferencedPathItem) error {
				return bundleReference(ctx, ref, opts, externalRefs, componentNames)
			},
			ReferencedParameter: func(ref *ReferencedParameter) error {
				return bundleReference(ctx, ref, opts, externalRefs, componentNames)
			},
			ReferencedExample: func(ref *ReferencedExample) error {
				return bundleReference(ctx, ref, opts, externalRefs, componentNames)
			},
			ReferencedRequestBody: func(ref *ReferencedRequestBody) error {
				return bundleReference(ctx, ref, opts, externalRefs, componentNames)
			},
			ReferencedResponse: func(ref *ReferencedResponse) error {
				return bundleReference(ctx, ref, opts, externalRefs, componentNames)
			},
			ReferencedHeader: func(ref *ReferencedHeader) error {
				return bundleReference(ctx, ref, opts, externalRefs, componentNames)
			},
			ReferencedCallback: func(ref *ReferencedCallback) error {
				return bundleReference(ctx, ref, opts, externalRefs, componentNames)
			},
			ReferencedLink: func(ref *ReferencedLink) error {
				return bundleReference(ctx, ref, opts, externalRefs, componentNames)
			},
			ReferencedSecurityScheme: func(ref *ReferencedSecurityScheme) error {
				return bundleReference(ctx, ref, opts, externalRefs, componentNames)
			},
		})
		if err != nil {
			return fmt.Errorf("failed to bundle item at %s: %w", item.Location.ToJSONPointer().String(), err)
		}
	}

	// Rewrite references within bundled schemas to handle circular references
	err := rewriteRefsInBundledSchemas(ctx, componentSchemas, externalRefs)
	if err != nil {
		return fmt.Errorf("failed to rewrite references in bundled schemas: %w", err)
	}

	// Second pass: update all references to point to new component names
	err = updateReferencesToComponents(ctx, doc, externalRefs)
	if err != nil {
		return fmt.Errorf("failed to update references: %w", err)
	}

	// Add collected schemas to components
	addSchemasToComponents(doc, componentSchemas)

	return nil
}

// bundleSchema handles bundling of JSON schemas with external references
func bundleSchema(ctx context.Context, schema *oas3.JSONSchema[oas3.Referenceable], opts BundleOptions, externalRefs map[string]string, componentSchemas *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]], componentNames map[string]bool, schemaHashes map[string]string) error {
	if !schema.IsReference() {
		return nil
	}

	ref := string(schema.GetRef())

	// Check if this is an external reference using the utility function
	classification, err := utils.ClassifyReference(ref)
	if err != nil || classification.IsFragment {
		//nolint:nilerr
		return nil // Internal reference or invalid, skip
	}

	// Check if we've already processed this reference
	if _, exists := externalRefs[ref]; exists {
		return nil
	}

	// Resolve the external reference
	resolveOpts := oas3.ResolveOptions{
		RootDocument:   opts.ResolveOptions.RootDocument,
		TargetLocation: opts.ResolveOptions.TargetLocation,
	}

	if _, err := schema.Resolve(ctx, resolveOpts); err != nil {
		return fmt.Errorf("failed to resolve external schema reference %s: %w", ref, err)
	}

	// Get the resolved schema
	resolvedSchema := schema.GetResolvedSchema()
	if resolvedSchema == nil {
		return fmt.Errorf("failed to get resolved schema for reference %s", ref)
	}

	// Convert back to referenceable schema for storage
	resolvedRefSchema := (*oas3.JSONSchema[oas3.Referenceable])(resolvedSchema)

	// Hash the resolved schema for conflict detection
	resolvedHash := hashing.Hash(resolvedRefSchema)

	// Generate component name with smart conflict resolution
	componentName := generateComponentNameWithHashConflictResolution(ref, opts.NamingStrategy, componentNames, schemaHashes, resolvedHash)

	// Only add to componentSchemas if it's a new schema (not a duplicate)
	if _, exists := schemaHashes[componentName]; !exists {
		componentNames[componentName] = true
		schemaHashes[componentName] = resolvedHash
		componentSchemas.Set(componentName, resolvedRefSchema)

		// Recursively process any external references within this resolved schema
		err = processNestedExternalReferences(ctx, resolvedRefSchema, opts, externalRefs, componentSchemas, componentNames, schemaHashes)
		if err != nil {
			return fmt.Errorf("failed to process nested references in %s: %w", ref, err)
		}
	}

	// Store the mapping
	externalRefs[ref] = componentName

	return nil
}

// processNestedExternalReferences recursively processes external references within a resolved schema
func processNestedExternalReferences(ctx context.Context, schema *oas3.JSONSchema[oas3.Referenceable], opts BundleOptions, externalRefs map[string]string, componentSchemas *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]], componentNames map[string]bool, schemaHashes map[string]string) error {
	if schema == nil {
		return nil
	}

	// Walk through the schema to find any external references
	for item := range oas3.Walk(ctx, schema) {
		err := item.Match(oas3.SchemaMatcher{
			Schema: func(nestedSchema *oas3.JSONSchema[oas3.Referenceable]) error {
				// First, process the nested schema to bundle any external references
				err := bundleSchema(ctx, nestedSchema, opts, externalRefs, componentSchemas, componentNames, schemaHashes)
				if err != nil {
					return err
				}

				// Only process nested external references during the first pass
				// Reference updating will be handled in the second pass
				if nestedSchema.IsReference() {
					// Just process the nested schema to bundle any external references
					// Don't update references here - that will be done in the second pass
					err := bundleSchema(ctx, nestedSchema, opts, externalRefs, componentSchemas, componentNames, schemaHashes)
					if err != nil {
						return err
					}
				}

				return nil
			},
		})
		if err != nil {
			return fmt.Errorf("failed to process nested schema: %w", err)
		}
	}

	return nil
}

// rewriteRefsInBundledSchemas rewrites references within bundled schemas to point to their new component locations
func rewriteRefsInBundledSchemas(ctx context.Context, componentSchemas *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]], externalRefs map[string]string) error {
	// Walk through each bundled schema and rewrite internal references
	for _, schema := range componentSchemas.All() {
		err := rewriteRefsInSchema(ctx, schema, externalRefs)
		if err != nil {
			return err
		}
	}
	return nil
}

// rewriteRefsInSchema rewrites references within a single schema
func rewriteRefsInSchema(ctx context.Context, schema *oas3.JSONSchema[oas3.Referenceable], externalRefs map[string]string) error {
	if schema == nil {
		return nil
	}

	// Walk through the schema and rewrite references
	for item := range oas3.Walk(ctx, schema) {
		err := item.Match(oas3.SchemaMatcher{
			Schema: func(s *oas3.JSONSchema[oas3.Referenceable]) error {
				schemaObj := s.GetLeft()
				if schemaObj != nil && schemaObj.Ref != nil {
					refStr := schemaObj.Ref.String()

					// Check for direct external reference match
					if newName, exists := externalRefs[refStr]; exists {
						newRef := "#/components/schemas/" + newName
						*schemaObj.Ref = references.Reference(newRef)
					} else if strings.HasPrefix(refStr, "#/") && !strings.HasPrefix(refStr, "#/components/") {
						// Handle circular references within external schemas
						// e.g., "#/User" should be mapped to "#/components/schemas/User_1"
						defName := strings.TrimPrefix(refStr, "#/")
						for externalRef, componentName := range externalRefs {
							// Check if the external reference ends with this fragment
							// e.g., "external_conflicting_user.yaml#/User" ends with "#/User"
							if strings.HasSuffix(externalRef, "#/"+defName) {
								newRef := "#/components/schemas/" + componentName
								*schemaObj.Ref = references.Reference(newRef)
								break
							}
						}
					}
				}
				return nil
			},
		})
		if err != nil {
			return fmt.Errorf("failed to rewrite reference in schema: %w", err)
		}
	}
	return nil
}

// bundleReference handles bundling of generic OpenAPI component references
func bundleReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ctx context.Context, ref *Reference[T, V, C], opts BundleOptions, externalRefs map[string]string, componentNames map[string]bool) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}

	refValue := ref.GetReference()
	refStr := string(refValue)

	// Check if this is an external reference using the utility function
	classification, err := utils.ClassifyReference(refStr)
	if err != nil || classification.IsFragment {
		//nolint:nilerr
		return nil // Internal reference or invalid, skip
	}

	// Check if we've already processed this reference
	if _, exists := externalRefs[refStr]; exists {
		return nil
	}

	// For OpenAPI component references, we need to resolve and bring the content into components
	// This is simpler than schema references as they don't have circular reference concerns
	resolveOpts := ResolveOptions{
		RootDocument:   opts.ResolveOptions.RootDocument,
		TargetLocation: opts.ResolveOptions.TargetLocation,
	}

	_, resolveErr := ref.Resolve(ctx, resolveOpts)
	if resolveErr != nil {
		return fmt.Errorf("failed to resolve external reference %s: %w", refStr, resolveErr)
	}

	// Generate component name
	componentName := generateComponentName(refStr, opts.NamingStrategy, componentNames)
	componentNames[componentName] = true

	// Store the mapping - for OpenAPI references, we'll handle them in the second pass
	externalRefs[refStr] = componentName

	// Note: For non-schema references, we don't store them in componentSchemas
	// as they will be handled differently based on their type

	return nil
}

// generateComponentName creates a new component name based on the reference and naming strategy
func generateComponentName(ref string, strategy BundleNamingStrategy, usedNames map[string]bool) string {
	switch strategy {
	case BundleNamingFilePath:
		return generateFilePathBasedNameWithConflictResolution(ref, usedNames)
	case BundleNamingCounter:
		return generateCounterBasedName(ref, usedNames)
	default:
		return generateCounterBasedName(ref, usedNames)
	}
}

// generateComponentNameWithHashConflictResolution creates a component name with smart conflict resolution based on content hashes
func generateComponentNameWithHashConflictResolution(ref string, strategy BundleNamingStrategy, usedNames map[string]bool, schemaHashes map[string]string, resolvedHash string) string {
	// Parse the reference to extract the simple name
	parts := strings.Split(ref, "#")
	if len(parts) == 0 {
		parts = []string{ref} // Fallback, though this should never happen
	}
	fragment := ""
	if len(parts) > 1 {
		fragment = parts[1]
	}

	var simpleName string
	if fragment == "" || fragment == "/" {
		// Top-level file reference - use filename as simple name
		filePath := parts[0]
		baseName := filepath.Base(filePath)
		ext := filepath.Ext(baseName)
		if ext != "" {
			baseName = baseName[:len(baseName)-len(ext)]
		}
		simpleName = regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(baseName, "_")
	} else {
		// Reference to specific schema within file - extract schema name
		cleanFragment := strings.TrimPrefix(fragment, "/")
		fragmentParts := strings.Split(cleanFragment, "/")
		if len(fragmentParts) == 0 {
			// This should never happen as strings.Split never returns nil or empty slice
			simpleName = "unknown"
		} else {
			simpleName = fragmentParts[len(fragmentParts)-1]
		}
	}

	// Check if a schema with this simple name already exists
	if existingHash, exists := schemaHashes[simpleName]; exists {
		if existingHash == resolvedHash {
			// Same content, reuse existing schema
			return simpleName
		}
		// Different content with same name - need conflict resolution
		// Fall back to the configured naming strategy for conflict resolution
		return generateComponentName(ref, strategy, usedNames)
	}

	// No conflict, use simple name
	return simpleName
}

// generateFilePathBasedNameWithConflictResolution tries to use simple names first, falling back to file-path-based names for conflicts
func generateFilePathBasedNameWithConflictResolution(ref string, usedNames map[string]bool) string {
	// Parse the reference to extract file path and fragment
	parts := strings.Split(ref, "#")
	if len(parts) == 0 {
		// This should never happen as strings.Split never returns nil or empty slice
		return "unknown"
	}
	fragment := ""
	if len(parts) > 1 {
		fragment = parts[1]
	}

	var simpleName string
	if fragment == "" || fragment == "/" {
		// Top-level file reference - use filename as simple name
		filePath := parts[0]
		baseName := filepath.Base(filePath)
		ext := filepath.Ext(baseName)
		if ext != "" {
			baseName = baseName[:len(baseName)-len(ext)]
		}
		simpleName = regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(baseName, "_")
	} else {
		// Reference to specific schema within file - extract schema name
		cleanFragment := strings.TrimPrefix(fragment, "/")
		fragmentParts := strings.Split(cleanFragment, "/")
		if len(fragmentParts) == 0 {
			// This should never happen as strings.Split never returns nil or empty slice
			simpleName = "unknown"
		} else {
			simpleName = fragmentParts[len(fragmentParts)-1]
		}
	}

	// Try simple name first
	if !usedNames[simpleName] {
		return simpleName
	}

	// If there's a conflict, fall back to file-path-based naming
	return generateFilePathBasedName(ref, usedNames)
}

// generateFilePathBasedName creates names like "some_path_external_yaml~User" or "some_path_external_yaml" for top-level refs
func generateFilePathBasedName(ref string, usedNames map[string]bool) string {
	// Parse the reference to extract file path and fragment
	parts := strings.Split(ref, "#")
	if len(parts) == 0 {
		// This should never happen as strings.Split never returns nil or empty slice
		return "unknown"
	}
	filePath := parts[0]
	fragment := ""
	if len(parts) > 1 {
		fragment = parts[1]
	}

	// Convert full file path to safe component name
	// Clean the path but keep extension for uniqueness
	cleanPath := filepath.Clean(filePath)

	// Remove leading "./" if present
	cleanPath = strings.TrimPrefix(cleanPath, "./")

	// Replace extension dot with underscore to keep it but make it safe
	ext := filepath.Ext(cleanPath)
	if ext != "" {
		cleanPath = cleanPath[:len(cleanPath)-len(ext)] + "_" + ext[1:] // Remove dot, add underscore
	}

	// Replace path separators and unsafe characters with underscores
	safeFileName := regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(cleanPath, "_")

	var componentName string
	if fragment == "" || fragment == "/" {
		// Top-level file reference
		componentName = safeFileName
	} else {
		// Reference to specific schema within file
		// Clean up fragment (remove leading slash and convert path separators)
		cleanFragment := strings.TrimPrefix(fragment, "/")
		cleanFragment = strings.ReplaceAll(cleanFragment, "/", "_")
		componentName = safeFileName + "~" + cleanFragment
	}

	// Ensure uniqueness
	originalName := componentName
	counter := 1
	for usedNames[componentName] {
		componentName = fmt.Sprintf("%s_%d", originalName, counter)
		counter++
	}

	return componentName
}

// generateCounterBasedName creates names like "User_1", "User_2" for conflicts
func generateCounterBasedName(ref string, usedNames map[string]bool) string {
	// Extract the schema name from the reference
	parts := strings.Split(ref, "#")
	if len(parts) == 0 {
		// This should never happen as strings.Split never returns nil or empty slice
		return "unknown"
	}
	fragment := ""
	if len(parts) > 1 {
		fragment = parts[1]
	}

	var baseName string
	if fragment == "" || fragment == "/" {
		// Top-level file reference - use filename
		filePath := parts[0]
		baseName = filepath.Base(filePath)
		ext := filepath.Ext(baseName)
		if ext != "" {
			baseName = baseName[:len(baseName)-len(ext)]
		}
		// Replace unsafe characters
		baseName = regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(baseName, "_")
	} else {
		// Extract last part of fragment as schema name
		fragmentParts := strings.Split(strings.TrimPrefix(fragment, "/"), "/")
		if len(fragmentParts) == 0 {
			// This should never happen as strings.Split never returns nil or empty slice
			baseName = "unknown"
		} else {
			baseName = fragmentParts[len(fragmentParts)-1]
		}
	}

	// Ensure uniqueness with counter
	componentName := baseName
	counter := 1
	for usedNames[componentName] {
		componentName = fmt.Sprintf("%s_%d", baseName, counter)
		counter++
	}

	return componentName
}

// updateReferencesToComponents updates all references in the document to point to new component names
func updateReferencesToComponents(ctx context.Context, doc *OpenAPI, externalRefs map[string]string) error {
	for item := range Walk(ctx, doc) {
		err := item.Match(Matcher{
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				if schema.IsReference() {
					ref := string(schema.GetRef())
					if newName, exists := externalRefs[ref]; exists {
						// Update the reference to point to the new component
						newRef := "#/components/schemas/" + newName
						*schema.GetLeft().Ref = references.Reference(newRef)
					} else if strings.HasPrefix(ref, "#/") && !strings.HasPrefix(ref, "#/components/") {
						// Handle circular references within external schemas
						// Look for a matching external reference that ends with this fragment
						for externalRef, componentName := range externalRefs {
							// Check if the external reference ends with this fragment
							// e.g., "external_conflicting_user.yaml#/User" ends with "#/User"
							if strings.HasSuffix(externalRef, ref) {
								// Update the circular reference to point to the bundled component
								newRef := "#/components/schemas/" + componentName
								*schema.GetLeft().Ref = references.Reference(newRef)
								break
							}
						}
					}
				}
				return nil
			},
			ReferencedPathItem: func(ref *ReferencedPathItem) error {
				return updateReference(ref, externalRefs, "pathItems")
			},
			ReferencedParameter: func(ref *ReferencedParameter) error {
				return updateReference(ref, externalRefs, "parameters")
			},
			ReferencedExample: func(ref *ReferencedExample) error {
				return updateReference(ref, externalRefs, "examples")
			},
			ReferencedRequestBody: func(ref *ReferencedRequestBody) error {
				return updateReference(ref, externalRefs, "requestBodies")
			},
			ReferencedResponse: func(ref *ReferencedResponse) error {
				return updateReference(ref, externalRefs, "responses")
			},
			ReferencedHeader: func(ref *ReferencedHeader) error {
				return updateReference(ref, externalRefs, "headers")
			},
			ReferencedCallback: func(ref *ReferencedCallback) error {
				return updateReference(ref, externalRefs, "callbacks")
			},
			ReferencedLink: func(ref *ReferencedLink) error {
				return updateReference(ref, externalRefs, "links")
			},
			ReferencedSecurityScheme: func(ref *ReferencedSecurityScheme) error {
				return updateReference(ref, externalRefs, "securitySchemes")
			},
		})
		if err != nil {
			return fmt.Errorf("failed to update reference at %s: %w", item.Location.ToJSONPointer().String(), err)
		}
	}
	return nil
}

// updateReference updates a generic reference to point to the new component name
func updateReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ref *Reference[T, V, C], externalRefs map[string]string, componentSection string) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}

	refStr := string(ref.GetReference())
	if newName, exists := externalRefs[refStr]; exists {
		// Update the reference to point to the new component
		newRef := "#/components/" + componentSection + "/" + newName
		*ref.Reference = references.Reference(newRef)
	}
	return nil
}

// addSchemasToComponents adds the collected schemas to the document's components section
func addSchemasToComponents(doc *OpenAPI, componentSchemas *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]]) {
	if componentSchemas.Len() == 0 {
		return
	}

	// Ensure components section exists
	if doc.Components == nil {
		doc.Components = &Components{}
	}
	if doc.Components.Schemas == nil {
		doc.Components.Schemas = sequencedmap.New[string, *oas3.JSONSchema[oas3.Referenceable]]()
	}

	// Add all collected schemas in insertion order
	for name, schema := range componentSchemas.All() {
		doc.Components.Schemas.Set(name, schema)
	}
}
