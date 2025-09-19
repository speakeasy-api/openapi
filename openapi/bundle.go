package openapi

import (
	"context"
	"errors"
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

	componentStorage := &componentStorage{
		schemaStorage:    sequencedmap.New[string, *oas3.JSONSchema[oas3.Referenceable]](),
		referenceStorage: sequencedmap.New[string, *sequencedmap.Map[string, any]](),
		externalRefs:     make(map[string]string),
		componentNames:   make(map[string]bool),
		schemaHashes:     make(map[string]string),
	}

	// Initialize existing component names and hashes to avoid conflicts
	if doc.Components != nil && doc.Components.Schemas != nil {
		for name, schema := range doc.Components.Schemas.All() {
			componentStorage.componentNames[name] = true
			if schema != nil {
				componentStorage.schemaHashes[name] = hashing.Hash(schema)
			}
		}
	}

	if err := bundleObject(ctx, doc, opts.NamingStrategy, "", opts.ResolveOptions, componentStorage); err != nil {
		return err
	}

	// Rewrite references within bundled schemas to handle circular references
	err := rewriteRefsInBundledSchemas(ctx, componentStorage)
	if err != nil {
		return fmt.Errorf("failed to rewrite references in bundled schemas: %w", err)
	}

	// Second pass: update all references to point to new component names
	err = updateReferencesToComponents(ctx, doc, componentStorage)
	if err != nil {
		return fmt.Errorf("failed to update references: %w", err)
	}

	// Add collected components to document
	addComponentsToDocument(doc, componentStorage)

	return nil
}

type componentStorage struct {
	schemaStorage    *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]]
	referenceStorage *sequencedmap.Map[string, *sequencedmap.Map[string, any]]
	externalRefs     map[string]string // original ref -> new component name
	componentNames   map[string]bool   // track used names to avoid conflicts
	schemaHashes     map[string]string // component name -> hash for conflict detection
}

func bundleObject[T any](ctx context.Context, obj *T, namingStrategy BundleNamingStrategy, parentLocation string, opts ResolveOptions, componentStorage *componentStorage) error {
	for item := range Walk(ctx, obj) {
		err := item.Match(Matcher{
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				return bundleSchema(ctx, schema, namingStrategy, parentLocation, opts, componentStorage)
			},
			ReferencedPathItem: func(ref *ReferencedPathItem) error {
				return bundleGenericReference(ctx, ref, namingStrategy, parentLocation, opts, componentStorage, "pathItems")
			},
			ReferencedParameter: func(ref *ReferencedParameter) error {
				return bundleGenericReference(ctx, ref, namingStrategy, parentLocation, opts, componentStorage, "parameters")
			},
			ReferencedExample: func(ref *ReferencedExample) error {
				return bundleGenericReference(ctx, ref, namingStrategy, parentLocation, opts, componentStorage, "examples")
			},
			ReferencedRequestBody: func(ref *ReferencedRequestBody) error {
				return bundleGenericReference(ctx, ref, namingStrategy, parentLocation, opts, componentStorage, "requestBodies")
			},
			ReferencedResponse: func(ref *ReferencedResponse) error {
				return bundleGenericReference(ctx, ref, namingStrategy, parentLocation, opts, componentStorage, "responses")
			},
			ReferencedHeader: func(ref *ReferencedHeader) error {
				return bundleGenericReference(ctx, ref, namingStrategy, parentLocation, opts, componentStorage, "headers")
			},
			ReferencedCallback: func(ref *ReferencedCallback) error {
				return bundleGenericReference(ctx, ref, namingStrategy, parentLocation, opts, componentStorage, "callbacks")
			},
			ReferencedLink: func(ref *ReferencedLink) error {
				return bundleGenericReference(ctx, ref, namingStrategy, parentLocation, opts, componentStorage, "links")
			},
			ReferencedSecurityScheme: func(ref *ReferencedSecurityScheme) error {
				return bundleGenericReference(ctx, ref, namingStrategy, parentLocation, opts, componentStorage, "securitySchemes")
			},
		})
		if err != nil {
			return fmt.Errorf("failed to bundle item at %s: %w", item.Location.ToJSONPointer().String(), err)
		}
	}

	return nil
}

// bundleSchema handles bundling of JSON schemas with external references
func bundleSchema(ctx context.Context, schema *oas3.JSONSchema[oas3.Referenceable], namingStrategy BundleNamingStrategy, parentLocation string, opts ResolveOptions, componentStorage *componentStorage) error {
	if !schema.IsReference() {
		return nil
	}

	ref, classification := handleReference(schema.GetRef(), parentLocation, opts.TargetLocation)
	if classification == nil {
		return nil // Invalid reference, skip
	}

	// If it's a fragment reference, check if it's pointing to a different document
	if classification.IsFragment {
		return nil // Internal reference within the root document, skip
	}

	// Check if we've already processed this reference
	if _, exists := componentStorage.externalRefs[ref]; exists {
		return nil
	}

	// Resolve the external reference
	resolveOpts := oas3.ResolveOptions{
		RootDocument:   opts.RootDocument,
		TargetDocument: opts.TargetDocument,
		TargetLocation: opts.TargetLocation,
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
	componentName, err := generateComponentNameWithHashConflictResolution(ref, namingStrategy, componentStorage.componentNames, componentStorage.schemaHashes, resolvedHash, opts.TargetLocation)
	if err != nil {
		return fmt.Errorf("failed to generate component name for %s: %w", ref, err)
	}

	// Store the mapping
	componentStorage.externalRefs[ref] = componentName

	// Only add to componentSchemas if it's a new schema (not a duplicate)
	if _, exists := componentStorage.schemaHashes[componentName]; !exists {
		componentStorage.componentNames[componentName] = true
		componentStorage.schemaHashes[componentName] = resolvedHash
		componentStorage.schemaStorage.Set(componentName, resolvedRefSchema)

		targetDocInfo := schema.GetReferenceResolutionInfo()

		if err := bundleObject(ctx, resolvedRefSchema, namingStrategy, opts.TargetLocation, references.ResolveOptions{
			RootDocument:   opts.RootDocument,
			TargetDocument: targetDocInfo.ResolvedDocument,
			TargetLocation: targetDocInfo.AbsoluteReference,
		}, componentStorage); err != nil {
			return fmt.Errorf("failed to bundle nested references in %s: %w", ref, err)
		}
	}

	return nil
}

// rewriteRefsInBundledSchemas rewrites references within bundled schemas to point to their new component locations
func rewriteRefsInBundledSchemas(ctx context.Context, componentStorage *componentStorage) error {
	// Walk through each bundled schema and rewrite internal references
	for _, schema := range componentStorage.schemaStorage.All() {
		err := rewriteRefsInSchema(ctx, schema, componentStorage)
		if err != nil {
			return err
		}
	}
	return nil
}

// rewriteRefsInSchema rewrites references within a single schema
func rewriteRefsInSchema(ctx context.Context, schema *oas3.JSONSchema[oas3.Referenceable], componentStorage *componentStorage) error {
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
					if newName, exists := componentStorage.externalRefs[refStr]; exists {
						newRef := "#/components/schemas/" + newName
						*schemaObj.Ref = references.Reference(newRef)
					} else if strings.HasPrefix(refStr, "#/") && !strings.HasPrefix(refStr, "#/components/") {
						// Handle circular references within external schemas
						// e.g., "#/User" should be mapped to "#/components/schemas/User_1"
						defName := strings.TrimPrefix(refStr, "#/")
						for externalRef, componentName := range componentStorage.externalRefs {
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

// bundleGenericReference handles bundling of generic OpenAPI component references
func bundleGenericReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ctx context.Context, ref *Reference[T, V, C], namingStrategy BundleNamingStrategy, parentLocation string, opts ResolveOptions, componentStorage *componentStorage, componentType string) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}

	refStr, classification := handleReference(ref.GetReference(), parentLocation, opts.TargetLocation)
	if classification == nil {
		return nil // Invalid reference, skip
	}

	if classification.IsFragment {
		return nil // Internal reference within the root document, skip
	}

	// Check if we've already processed this reference
	if _, exists := componentStorage.externalRefs[refStr]; exists {
		return nil
	}

	// Resolve the external reference
	resolveOpts := ResolveOptions{
		RootDocument:   opts.RootDocument,
		TargetDocument: opts.TargetDocument,
		TargetLocation: opts.TargetLocation,
	}

	_, resolveErr := ref.Resolve(ctx, resolveOpts)
	if resolveErr != nil {
		return fmt.Errorf("failed to resolve external %s reference %s: %w", componentType, refStr, resolveErr)
	}

	// Generate component name
	componentName, err := generateComponentName(refStr, namingStrategy, componentStorage.componentNames, opts.TargetLocation)
	if err != nil {
		return fmt.Errorf("failed to generate component name for %s: %w", refStr, err)
	}
	componentStorage.componentNames[componentName] = true

	// Store the mapping
	componentStorage.externalRefs[refStr] = componentName

	// Get the resolved content and create a new non-reference version
	resolvedValue := ref.GetObject()
	if resolvedValue == nil {
		return fmt.Errorf("failed to get resolved %s for reference %s", componentType, refStr)
	}

	// Create a new Reference with the resolved content (not a reference)
	bundledRef := &Reference[T, V, C]{}
	bundledRef.Object = resolvedValue

	if !componentStorage.referenceStorage.Has(componentType) {
		componentStorage.referenceStorage.Set(componentType, sequencedmap.New[string, any]())
	}

	// Store the resolved component (not the reference) if it's a new component
	if !componentStorage.referenceStorage.GetOrZero(componentType).Has(componentName) {
		componentStorage.referenceStorage.GetOrZero(componentType).Set(componentName, bundledRef)

		targetDocInfo := ref.GetReferenceResolutionInfo()

		if err := bundleObject(ctx, bundledRef, namingStrategy, opts.TargetLocation, references.ResolveOptions{
			RootDocument:   opts.RootDocument,
			TargetDocument: targetDocInfo.ResolvedDocument,
			TargetLocation: targetDocInfo.AbsoluteReference,
		}, componentStorage); err != nil {
			return fmt.Errorf("failed to bundle nested references in %s: %w", ref.GetReference(), err)
		}
	}

	return nil
}

// generateComponentName creates a new component name based on the reference and naming strategy
func generateComponentName(ref string, strategy BundleNamingStrategy, usedNames map[string]bool, targetLocation string) (string, error) {
	switch strategy {
	case BundleNamingFilePath:
		return generateFilePathBasedNameWithConflictResolution(ref, usedNames, targetLocation)
	case BundleNamingCounter:
		return generateCounterBasedName(ref, usedNames), nil
	default:
		return generateCounterBasedName(ref, usedNames), nil
	}
}

// generateComponentNameWithHashConflictResolution creates a component name with smart conflict resolution based on content hashes
func generateComponentNameWithHashConflictResolution(ref string, strategy BundleNamingStrategy, usedNames map[string]bool, schemaHashes map[string]string, resolvedHash string, targetLocation string) (string, error) {
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
			return simpleName, nil
		}
		// Different content with same name - need conflict resolution
		// Fall back to the configured naming strategy for conflict resolution
		return generateComponentName(ref, strategy, usedNames, targetLocation)
	}

	// No conflict, use simple name
	return simpleName, nil
}

// generateFilePathBasedNameWithConflictResolution tries to use simple names first, falling back to file-path-based names for conflicts
func generateFilePathBasedNameWithConflictResolution(ref string, usedNames map[string]bool, targetLocation string) (string, error) {
	// Parse the reference to extract file path and fragment
	parts := strings.Split(ref, "#")
	if len(parts) == 0 {
		// This should never happen as strings.Split never returns nil or empty slice
		return "unknown", nil
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
		return simpleName, nil
	}

	// If there's a conflict, fall back to file-path-based naming
	return generateFilePathBasedName(ref, usedNames, targetLocation)
}

// generateFilePathBasedName creates names like "some_path_external_yaml~User" or "some_path_external_yaml" for top-level refs
func generateFilePathBasedName(ref string, usedNames map[string]bool, targetLocation string) (string, error) {
	// Parse the reference to extract file path and fragment
	parts := strings.Split(ref, "#")
	if len(parts) == 0 {
		// This should never happen as strings.Split never returns nil or empty slice
		return "unknown", nil
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

	// Handle parent directory references more elegantly
	// Instead of converting "../" to "___", we'll normalize the path
	normalizedPath, err := normalizePathForComponentName(cleanPath, targetLocation)
	if err != nil {
		return "", fmt.Errorf("failed to normalize path %s: %w", cleanPath, err)
	}
	cleanPath = normalizedPath

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

	return componentName, nil
}

// normalizePathForComponentName normalizes a file path to create a more readable component name
// by resolving relative paths to their actual directory names using absolute path resolution
func normalizePathForComponentName(path, targetLocation string) (string, error) {
	if targetLocation == "" {
		return "", errors.New("target location cannot be empty for path normalization")
	}

	// Get the directory of the target location
	targetDir := filepath.Dir(targetLocation)

	// Resolve the relative path against the target directory to get absolute path
	resolvedAbsPath, err := filepath.Abs(filepath.Join(targetDir, path))
	if err != nil {
		return "", fmt.Errorf("failed to resolve relative path: %w", err)
	}

	// Split the original relative path to find where the real path starts (after all the ../)
	// Handle both Unix and Windows path separators
	normalizedPath := strings.ReplaceAll(path, "\\", "/")
	pathParts := strings.Split(normalizedPath, "/")

	// Count parent directory navigations and find the start of the real path
	parentCount := 0
	realPathStart := len(pathParts) // Default to end if no real path found
	foundRealPath := false

	for i, part := range pathParts {
		if foundRealPath {
			break
		}

		switch part {
		case "..":
			parentCount++
		case ".":
			// Skip current directory references
			continue
		case "":
			// Skip empty parts
			continue
		default:
			// Found the start of the real path
			realPathStart = i
			foundRealPath = true
		}
	}

	// Get the real path parts (everything after the ../ navigation)
	var realPathParts []string
	if realPathStart < len(pathParts) {
		realPathParts = pathParts[realPathStart:]
	}

	// Use the absolute path to get the meaningful directory structure
	// Split the absolute path and take the last meaningful parts
	absParts := strings.Split(strings.ReplaceAll(resolvedAbsPath, "\\", "/"), "/")

	// We want to include the directory we land on after navigation plus the real path
	// For "../../../other/api.yaml" from "openapi/a/b/c/spec.yaml", we want "openapi/other/api.yaml"
	// So we need: landing directory (openapi) + real path parts (other/api.yaml)

	var resultParts []string

	if parentCount > 0 {
		// Find the landing directory after going up parentCount levels
		// We need at least parentCount + len(realPathParts) parts in the absolute path
		requiredParts := 1 + len(realPathParts) // 1 for landing directory + real path parts

		if len(absParts) < requiredParts {
			return "", errors.New("not enough path components in resolved absolute path")
		}

		// Take the landing directory (the directory we end up in after going up)
		landingDirIndex := len(absParts) - len(realPathParts) - 1
		if landingDirIndex >= 0 && landingDirIndex < len(absParts) {
			landingDir := absParts[landingDirIndex]
			resultParts = append(resultParts, landingDir)
		}
	}

	// Add the real path parts
	resultParts = append(resultParts, realPathParts...)

	// Join and clean up the result
	result := strings.Join(resultParts, "/")

	// Remove leading "./" if present
	result = strings.TrimPrefix(result, "./")

	return result, nil
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
func updateReferencesToComponents(ctx context.Context, doc *OpenAPI, componentStorage *componentStorage) error {
	for item := range Walk(ctx, doc) {
		err := item.Match(Matcher{
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				if schema.IsReference() {
					ref := string(schema.GetRef())
					if newName, exists := componentStorage.externalRefs[ref]; exists {
						// Update the reference to point to the new component
						newRef := "#/components/schemas/" + newName
						*schema.GetLeft().Ref = references.Reference(newRef)
					} else if strings.HasPrefix(ref, "#/") && !strings.HasPrefix(ref, "#/components/") {
						// Handle circular references within external schemas
						// Look for a matching external reference that ends with this fragment
						for externalRef, componentName := range componentStorage.externalRefs {
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
				return updateReference(ref, componentStorage, "pathItems")
			},
			ReferencedParameter: func(ref *ReferencedParameter) error {
				return updateReference(ref, componentStorage, "parameters")
			},
			ReferencedExample: func(ref *ReferencedExample) error {
				return updateReference(ref, componentStorage, "examples")
			},
			ReferencedRequestBody: func(ref *ReferencedRequestBody) error {
				return updateReference(ref, componentStorage, "requestBodies")
			},
			ReferencedResponse: func(ref *ReferencedResponse) error {
				return updateReference(ref, componentStorage, "responses")
			},
			ReferencedHeader: func(ref *ReferencedHeader) error {
				return updateReference(ref, componentStorage, "headers")
			},
			ReferencedCallback: func(ref *ReferencedCallback) error {
				return updateReference(ref, componentStorage, "callbacks")
			},
			ReferencedLink: func(ref *ReferencedLink) error {
				return updateReference(ref, componentStorage, "links")
			},
			ReferencedSecurityScheme: func(ref *ReferencedSecurityScheme) error {
				return updateReference(ref, componentStorage, "securitySchemes")
			},
		})
		if err != nil {
			return fmt.Errorf("failed to update reference at %s: %w", item.Location.ToJSONPointer().String(), err)
		}
	}
	return nil
}

// updateReference updates a generic reference to point to the new component name
func updateReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ref *Reference[T, V, C], componentStorage *componentStorage, componentSection string) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}

	refStr := string(ref.GetReference())
	if newName, exists := componentStorage.externalRefs[refStr]; exists {
		// Update the reference to point to the new component
		newRef := "#/components/" + componentSection + "/" + newName
		*ref.Reference = references.Reference(newRef)
	}
	return nil
}

// addComponentsToDocument adds all collected components to the document's components section
func addComponentsToDocument(doc *OpenAPI, componentStorage *componentStorage) {
	// Ensure components section exists
	if doc.Components == nil {
		doc.Components = &Components{}
	}

	// Add schemas
	if componentStorage.schemaStorage.Len() > 0 {
		if doc.Components.Schemas == nil {
			doc.Components.Schemas = sequencedmap.New[string, *oas3.JSONSchema[oas3.Referenceable]]()
		}
		for name, schema := range componentStorage.schemaStorage.All() {
			doc.Components.Schemas.Set(name, schema)
		}
	}

	// Add responses
	if componentStorage.referenceStorage.GetOrZero("responses").Len() > 0 {
		if doc.Components.Responses == nil {
			doc.Components.Responses = sequencedmap.New[string, *ReferencedResponse]()
		}
		for name, response := range componentStorage.referenceStorage.GetOrZero("responses").All() {
			doc.Components.Responses.Set(name, response.(*ReferencedResponse))
		}
	}

	// Add parameters
	if componentStorage.referenceStorage.GetOrZero("parameters").Len() > 0 {
		if doc.Components.Parameters == nil {
			doc.Components.Parameters = sequencedmap.New[string, *ReferencedParameter]()
		}
		for name, parameter := range componentStorage.referenceStorage.GetOrZero("parameters").All() {
			doc.Components.Parameters.Set(name, parameter.(*ReferencedParameter))
		}
	}

	// Add examples
	if componentStorage.referenceStorage.GetOrZero("examples").Len() > 0 {
		if doc.Components.Examples == nil {
			doc.Components.Examples = sequencedmap.New[string, *ReferencedExample]()
		}
		for name, example := range componentStorage.referenceStorage.GetOrZero("examples").All() {
			doc.Components.Examples.Set(name, example.(*ReferencedExample))
		}
	}

	// Add request bodies
	if componentStorage.referenceStorage.GetOrZero("requestBodies").Len() > 0 {
		if doc.Components.RequestBodies == nil {
			doc.Components.RequestBodies = sequencedmap.New[string, *ReferencedRequestBody]()
		}
		for name, requestBody := range componentStorage.referenceStorage.GetOrZero("requestBodies").All() {
			doc.Components.RequestBodies.Set(name, requestBody.(*ReferencedRequestBody))
		}
	}

	// Add headers
	if componentStorage.referenceStorage.GetOrZero("headers").Len() > 0 {
		if doc.Components.Headers == nil {
			doc.Components.Headers = sequencedmap.New[string, *ReferencedHeader]()
		}
		for name, header := range componentStorage.referenceStorage.GetOrZero("headers").All() {
			doc.Components.Headers.Set(name, header.(*ReferencedHeader))
		}
	}

	// Add callbacks
	if componentStorage.referenceStorage.GetOrZero("callbacks").Len() > 0 {
		if doc.Components.Callbacks == nil {
			doc.Components.Callbacks = sequencedmap.New[string, *ReferencedCallback]()
		}
		for name, callback := range componentStorage.referenceStorage.GetOrZero("callbacks").All() {
			doc.Components.Callbacks.Set(name, callback.(*ReferencedCallback))
		}
	}

	// Add links
	if componentStorage.referenceStorage.GetOrZero("links").Len() > 0 {
		if doc.Components.Links == nil {
			doc.Components.Links = sequencedmap.New[string, *ReferencedLink]()
		}
		for name, link := range componentStorage.referenceStorage.GetOrZero("links").All() {
			doc.Components.Links.Set(name, link.(*ReferencedLink))
		}
	}

	// Add security schemes
	if componentStorage.referenceStorage.GetOrZero("securitySchemes").Len() > 0 {
		if doc.Components.SecuritySchemes == nil {
			doc.Components.SecuritySchemes = sequencedmap.New[string, *ReferencedSecurityScheme]()
		}
		for name, securityScheme := range componentStorage.referenceStorage.GetOrZero("securitySchemes").All() {
			doc.Components.SecuritySchemes.Set(name, securityScheme.(*ReferencedSecurityScheme))
		}
	}

	// Add path items
	if componentStorage.referenceStorage.GetOrZero("pathItems").Len() > 0 {
		if doc.Components.PathItems == nil {
			doc.Components.PathItems = sequencedmap.New[string, *ReferencedPathItem]()
		}
		for name, pathItem := range componentStorage.referenceStorage.GetOrZero("pathItems").All() {
			doc.Components.PathItems.Set(name, pathItem.(*ReferencedPathItem))
		}
	}
}

func handleReference(ref references.Reference, parentLocation, targetLocation string) (string, *utils.ReferenceClassification) {
	r := ref.String()

	// Check if this is an external reference using the utility function
	classification, err := utils.ClassifyReference(r)
	if err != nil {
		return "", nil // Invalid reference, skip
	}

	// For URLs, don't do any path manipulation - return as-is
	if classification.Type == utils.ReferenceTypeURL {
		return r, classification
	}

	if parentLocation != "" {
		relPath, err := filepath.Rel(filepath.Dir(parentLocation), targetLocation)
		if err == nil {
			if classification.IsFragment {
				r = relPath + r
			} else {
				if ref.GetURI() != "" {
					r = filepath.Join(filepath.Dir(relPath), r)
				} else {
					r = filepath.Join(relPath, r)
				}
			}
		}

		// convert paths back to original separators
		// detect original separators from the original reference
		pathStyle := detectPathStyle(ref.String())
		switch pathStyle {
		case "windows":
			r = strings.ReplaceAll(r, "/", "\\")
		default:
			r = strings.ReplaceAll(r, "\\", "/")
		}

		cl, err := utils.ClassifyReference(r)
		if err == nil {
			classification = cl
		}
	}

	return r, classification
}

var winAbs = regexp.MustCompile(`^[a-zA-Z]:\\`)

func detectPathStyle(p string) string {
	switch {
	case winAbs.MatchString(p):
		return "windows"
	case strings.Contains(p, "\\"):
		return "windows"
	case strings.Contains(p, "/"):
		return "unix"
	default:
		return "unknown"
	}
}
