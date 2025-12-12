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
	// BundleNamingFilePath uses file path-based naming like file_path_somefile_yaml__User
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
//	                  "$ref": "#/components/schemas/external_api_yaml__User"
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
//	      "external_api_yaml__User": {
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

	// Make target location absolute at the entry point
	targetLocation := opts.ResolveOptions.TargetLocation
	if targetLocation != "" && !filepath.IsAbs(targetLocation) {
		if absTarget, err := filepath.Abs(targetLocation); err == nil {
			targetLocation = absTarget
			opts.ResolveOptions.TargetLocation = absTarget
		}
		// Error getting absolute path is not fatal - we continue with the relative path
		// This allows processing to proceed even if there are issues with path resolution
	}

	componentStorage := &componentStorage{
		schemaStorage:      sequencedmap.New[string, *oas3.JSONSchema[oas3.Referenceable]](),
		referenceStorage:   sequencedmap.New[string, *sequencedmap.Map[string, any]](),
		refs:               make(map[string]string),
		componentNames:     make(map[string]bool),
		schemaHashes:       make(map[string]string),
		schemaLocations:    make(map[string]string),
		componentLocations: make(map[string]string),
		rootLocation:       targetLocation,
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

	if err := bundleObject(ctx, doc, opts.NamingStrategy, opts.ResolveOptions, componentStorage); err != nil {
		return err
	}

	// Rewrite references within bundled schemas to handle circular references
	err := rewriteRefsInBundledSchemas(ctx, componentStorage)
	if err != nil {
		return fmt.Errorf("failed to rewrite references in bundled schemas: %w", err)
	}

	// Rewrite references within bundled components (responses, headers, etc.)
	err = rewriteRefsInBundledComponents(ctx, componentStorage)
	if err != nil {
		return fmt.Errorf("failed to rewrite references in bundled components: %w", err)
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
	schemaStorage      *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]]
	referenceStorage   *sequencedmap.Map[string, *sequencedmap.Map[string, any]]
	refs               map[string]string // absolute ref -> component name
	componentNames     map[string]bool   // track used names to avoid conflicts
	schemaHashes       map[string]string // component name -> hash for conflict detection
	schemaLocations    map[string]string // component name -> absolute source location (for rewriting refs)
	componentLocations map[string]string // componentType/componentName -> absolute source location
	rootLocation       string            // absolute path to root document for relative path calculation
}

func bundleObject[T any](ctx context.Context, obj *T, namingStrategy BundleNamingStrategy, opts ResolveOptions, componentStorage *componentStorage) error {
	for item := range Walk(ctx, obj) {
		err := item.Match(Matcher{
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				return bundleSchema(ctx, schema, namingStrategy, opts, componentStorage)
			},
			ReferencedPathItem: func(ref *ReferencedPathItem) error {
				return bundleGenericReference(ctx, ref, namingStrategy, opts, componentStorage, "pathItems")
			},
			ReferencedParameter: func(ref *ReferencedParameter) error {
				return bundleGenericReference(ctx, ref, namingStrategy, opts, componentStorage, "parameters")
			},
			ReferencedExample: func(ref *ReferencedExample) error {
				return bundleGenericReference(ctx, ref, namingStrategy, opts, componentStorage, "examples")
			},
			ReferencedRequestBody: func(ref *ReferencedRequestBody) error {
				return bundleGenericReference(ctx, ref, namingStrategy, opts, componentStorage, "requestBodies")
			},
			ReferencedResponse: func(ref *ReferencedResponse) error {
				return bundleGenericReference(ctx, ref, namingStrategy, opts, componentStorage, "responses")
			},
			ReferencedHeader: func(ref *ReferencedHeader) error {
				return bundleGenericReference(ctx, ref, namingStrategy, opts, componentStorage, "headers")
			},
			ReferencedCallback: func(ref *ReferencedCallback) error {
				return bundleGenericReference(ctx, ref, namingStrategy, opts, componentStorage, "callbacks")
			},
			ReferencedLink: func(ref *ReferencedLink) error {
				return bundleGenericReference(ctx, ref, namingStrategy, opts, componentStorage, "links")
			},
			ReferencedSecurityScheme: func(ref *ReferencedSecurityScheme) error {
				return bundleGenericReference(ctx, ref, namingStrategy, opts, componentStorage, "securitySchemes")
			},
		})
		if err != nil {
			return fmt.Errorf("failed to bundle item at %s: %w", item.Location.ToJSONPointer().String(), err)
		}
	}

	return nil
}

// bundleSchema handles bundling of JSON schemas with external references
func bundleSchema(ctx context.Context, schema *oas3.JSONSchema[oas3.Referenceable], namingStrategy BundleNamingStrategy, opts ResolveOptions, componentStorage *componentStorage) error {
	if !schema.IsReference() {
		return nil
	}

	ref, classification := handleReference(schema.GetRef(), opts.TargetLocation)
	if classification == nil {
		return nil // Invalid reference, skip
	}

	// Check if this is an internal reference to the root document
	if isInternalReference(ref, componentStorage.rootLocation) {
		return nil
	}

	// Check if we've already processed this reference
	if _, exists := componentStorage.refs[ref]; exists {
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
	componentName, err := generateComponentNameWithHashConflictResolution(ref, namingStrategy, componentStorage.componentNames, componentStorage.schemaHashes, resolvedHash, componentStorage.rootLocation)
	if err != nil {
		return fmt.Errorf("failed to generate component name for %s: %w", ref, err)
	}

	// Store the mapping
	componentStorage.refs[ref] = componentName

	// Only add to componentSchemas if it's a new schema (not a duplicate)
	if _, exists := componentStorage.schemaHashes[componentName]; !exists {
		componentStorage.componentNames[componentName] = true
		componentStorage.schemaHashes[componentName] = resolvedHash
		componentStorage.schemaStorage.Set(componentName, resolvedRefSchema)

		// Store the source location for this schema for later reference rewriting
		componentStorage.schemaLocations[componentName] = ref

		targetDocInfo := schema.GetReferenceResolutionInfo()

		if err := bundleObject(ctx, resolvedRefSchema, namingStrategy, references.ResolveOptions{
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
	for componentName, schema := range componentStorage.schemaStorage.All() {
		// Get the source location for this schema
		sourceLocation := componentStorage.schemaLocations[componentName]

		err := rewriteRefsInSchema(ctx, schema, componentStorage, sourceLocation)
		if err != nil {
			return err
		}
	}
	return nil
}

// prepareSourceURI extracts and normalizes a source URI for use with handleReference.
// It extracts just the URI part (removing any fragment) and converts it to native path
// format for filepath operations.
func prepareSourceURI(sourceLocation string) string {
	// Extract just the URI part from sourceLocation (remove fragment if present)
	// sourceLocation might be like "/path/to/file.yaml#/components/schemas/SchemaName"
	// but we need just "/path/to/file.yaml" for resolving relative references
	sourceURI := references.Reference(sourceLocation).GetURI()
	if sourceURI == "" {
		sourceURI = sourceLocation // Fallback if no URI part
	}

	// On Windows, convert forward slashes to backslashes for filepath operations
	// componentLocations stores paths with forward slashes (from handleReference normalization)
	// but filepath.Join needs native separators to work correctly
	if filepath.Separator == '\\' && filepath.IsAbs(sourceURI) {
		sourceURI = filepath.FromSlash(sourceURI)
	}

	return sourceURI
}

// rewriteRefsInSchema rewrites references within a single schema
func rewriteRefsInSchema(ctx context.Context, schema *oas3.JSONSchema[oas3.Referenceable], componentStorage *componentStorage, sourceLocation string) error {
	if schema == nil {
		return nil
	}

	sourceURI := prepareSourceURI(sourceLocation)

	// Walk through the schema and rewrite references
	for item := range oas3.Walk(ctx, schema) {
		err := item.Match(oas3.SchemaMatcher{
			Schema: func(s *oas3.JSONSchema[oas3.Referenceable]) error {
				schemaObj := s.GetLeft()
				if schemaObj != nil && schemaObj.Ref != nil {
					refStr := schemaObj.Ref.String()

					// Convert the reference to absolute for lookup using the source URI (without fragment)
					absRef, _ := handleReference(*schemaObj.Ref, sourceURI)

					// Check for direct reference match or circular reference
					if newName, exists := componentStorage.refs[absRef]; exists {
						newRef := "#/components/schemas/" + newName
						*schemaObj.Ref = references.Reference(newRef)
					} else if newName, found := findCircularReferenceMatch(refStr, componentStorage.refs); found {
						newRef := "#/components/schemas/" + newName
						*schemaObj.Ref = references.Reference(newRef)
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

// rewriteRefsInBundledComponents rewrites references within bundled components (responses, headers, etc.)
// to point to their new component locations. This handles cases where a bundled response contains
// header references that also need to be updated.
func rewriteRefsInBundledComponents(ctx context.Context, componentStorage *componentStorage) error {
	// Walk through each component type in referenceStorage
	for componentType, components := range componentStorage.referenceStorage.All() {
		for componentName, component := range components.All() {
			// Get the source location for this component
			sourceLocation := componentStorage.componentLocations[componentType+"/"+componentName]

			// Walk through the component and update references
			err := walkAndUpdateRefsInComponent(ctx, component, componentStorage, sourceLocation)
			if err != nil {
				return fmt.Errorf("failed to rewrite refs in %s component: %w", componentType, err)
			}
		}
	}
	return nil
}

// walkAndUpdateRefsInComponent walks through a component and updates its internal references
func walkAndUpdateRefsInComponent(ctx context.Context, component any, componentStorage *componentStorage, sourceLocation string) error {
	// We need to handle each component type that can contain references
	// The Walk function will traverse the component and find all references

	// Use type-specific walking based on the component type
	switch c := component.(type) {
	case *ReferencedResponse:
		return walkAndUpdateRefsInResponse(ctx, c, componentStorage, sourceLocation)
	case *ReferencedParameter:
		return walkAndUpdateRefsInParameter(ctx, c, componentStorage, sourceLocation)
	case *ReferencedRequestBody:
		return walkAndUpdateRefsInRequestBody(ctx, c, componentStorage, sourceLocation)
	case *ReferencedCallback:
		return walkAndUpdateRefsInCallback(ctx, c, componentStorage, sourceLocation)
	case *ReferencedPathItem:
		return walkAndUpdateRefsInPathItem(ctx, c, componentStorage, sourceLocation)
	case *ReferencedLink:
		return walkAndUpdateRefsInLink(ctx, c, componentStorage, sourceLocation)
	case *ReferencedExample:
		return walkAndUpdateRefsInExample(ctx, c, componentStorage, sourceLocation)
	case *ReferencedSecurityScheme:
		return walkAndUpdateRefsInSecurityScheme(ctx, c, componentStorage, sourceLocation)
	case *ReferencedHeader:
		return walkAndUpdateRefsInHeader(ctx, c, componentStorage, sourceLocation)
	}
	return nil
}

// walkAndUpdateRefsInResponse walks through a response and updates its internal references
func walkAndUpdateRefsInResponse(_ context.Context, response *ReferencedResponse, componentStorage *componentStorage, sourceLocation string) error {
	if response == nil || response.Object == nil {
		return nil
	}

	// Walk through the response's headers and update references
	if response.Object.Headers != nil {
		for _, header := range response.Object.Headers.All() {
			if header != nil && header.IsReference() {
				updateComponentRefWithSource(header.Reference, componentStorage, "headers", sourceLocation)
			}
		}
	}

	// Walk through the response's content schemas
	if response.Object.Content != nil {
		for _, mediaType := range response.Object.Content.All() {
			if mediaType != nil {
				if mediaType.Schema != nil && mediaType.Schema.IsReference() {
					updateSchemaRefWithSource(mediaType.Schema, componentStorage, sourceLocation)
				}
				if mediaType.ItemSchema != nil && mediaType.ItemSchema.IsReference() {
					updateSchemaRefWithSource(mediaType.ItemSchema, componentStorage, sourceLocation)
				}
			}
		}
	}

	return nil
}

// walkAndUpdateRefsInParameter walks through a parameter and updates its internal references
func walkAndUpdateRefsInParameter(_ context.Context, param *ReferencedParameter, componentStorage *componentStorage, sourceLocation string) error {
	if param == nil || param.Object == nil {
		return nil
	}

	// Walk through the parameter's schema
	if param.Object.Schema != nil && param.Object.Schema.IsReference() {
		updateSchemaRefWithSource(param.Object.Schema, componentStorage, sourceLocation)
	}

	// Walk through parameter examples
	if param.Object.Examples != nil {
		for _, example := range param.Object.Examples.All() {
			if example != nil && example.IsReference() {
				updateComponentRefWithSource(example.Reference, componentStorage, "examples", sourceLocation)
			}
		}
	}

	return nil
}

// walkAndUpdateRefsInRequestBody walks through a request body and updates its internal references
func walkAndUpdateRefsInRequestBody(_ context.Context, body *ReferencedRequestBody, componentStorage *componentStorage, sourceLocation string) error {
	if body == nil || body.Object == nil {
		return nil
	}

	// Walk through the request body's content schemas
	if body.Object.Content != nil {
		for _, mediaType := range body.Object.Content.All() {
			if mediaType != nil {
				if mediaType.Schema != nil && mediaType.Schema.IsReference() {
					updateSchemaRefWithSource(mediaType.Schema, componentStorage, sourceLocation)
				}
				if mediaType.ItemSchema != nil && mediaType.ItemSchema.IsReference() {
					updateSchemaRefWithSource(mediaType.ItemSchema, componentStorage, sourceLocation)
				}
			}
		}
	}

	return nil
}

// walkAndUpdateRefsInCallback walks through a callback and updates its internal references
func walkAndUpdateRefsInCallback(_ context.Context, callback *ReferencedCallback, componentStorage *componentStorage, sourceLocation string) error {
	if callback == nil || callback.Object == nil {
		return nil
	}

	// Callbacks contain path items with operations
	for _, pathItem := range callback.Object.All() {
		if pathItem != nil && pathItem.IsReference() {
			updateComponentRefWithSource(pathItem.Reference, componentStorage, "pathItems", sourceLocation)
		}
	}

	return nil
}

// walkAndUpdateRefsInPathItem walks through a path item and updates its internal references
func walkAndUpdateRefsInPathItem(_ context.Context, pathItem *ReferencedPathItem, componentStorage *componentStorage, sourceLocation string) error {
	if pathItem == nil || pathItem.Object == nil {
		return nil
	}

	// Path items can have parameters
	if pathItem.Object.Parameters != nil {
		for _, param := range pathItem.Object.Parameters {
			if param != nil && param.IsReference() {
				updateComponentRefWithSource(param.Reference, componentStorage, "parameters", sourceLocation)
			}
		}
	}

	return nil
}

// walkAndUpdateRefsInLink walks through a link and updates its internal references
func walkAndUpdateRefsInLink(_ context.Context, _ *ReferencedLink, _ *componentStorage, _ string) error {
	// Links don't typically contain component references that need updating
	return nil
}

// walkAndUpdateRefsInExample walks through an example and updates its internal references
func walkAndUpdateRefsInExample(_ context.Context, _ *ReferencedExample, _ *componentStorage, _ string) error {
	// Examples don't typically contain component references that need updating
	return nil
}

// walkAndUpdateRefsInSecurityScheme walks through a security scheme and updates its internal references
func walkAndUpdateRefsInSecurityScheme(_ context.Context, _ *ReferencedSecurityScheme, _ *componentStorage, _ string) error {
	// Security schemes don't typically contain component references that need updating
	return nil
}

// walkAndUpdateRefsInHeader walks through a header and updates its internal references
func walkAndUpdateRefsInHeader(_ context.Context, header *ReferencedHeader, componentStorage *componentStorage, sourceLocation string) error {
	if header == nil || header.Object == nil {
		return nil
	}

	// Walk through the header's schema
	if header.Object.Schema != nil && header.Object.Schema.IsReference() {
		updateSchemaRefWithSource(header.Object.Schema, componentStorage, sourceLocation)
	}

	// Walk through header examples
	if header.Object.Examples != nil {
		for _, example := range header.Object.Examples.All() {
			if example != nil && example.IsReference() {
				updateComponentRefWithSource(example.Reference, componentStorage, "examples", sourceLocation)
			}
		}
	}

	return nil
}

// updateSchemaRefWithSource updates a schema reference using a specific source location for resolution
func updateSchemaRefWithSource(schema *oas3.JSONSchema[oas3.Referenceable], componentStorage *componentStorage, sourceLocation string) {
	if schema == nil || !schema.IsReference() {
		return
	}

	sourceURI := prepareSourceURI(sourceLocation)
	ref := schema.GetRef()
	absRef, _ := handleReference(ref, sourceURI)

	if newName, exists := componentStorage.refs[absRef]; exists {
		newRef := "#/components/schemas/" + newName
		*schema.GetLeft().Ref = references.Reference(newRef)
	}
}

// updateComponentRefWithSource updates a component reference using a specific source location for resolution
func updateComponentRefWithSource(ref *references.Reference, componentStorage *componentStorage, componentSection string, sourceLocation string) {
	if ref == nil {
		return
	}

	sourceURI := prepareSourceURI(sourceLocation)
	absRef, _ := handleReference(*ref, sourceURI)

	if newName, exists := componentStorage.refs[absRef]; exists {
		newRef := "#/components/" + componentSection + "/" + newName
		*ref = references.Reference(newRef)
	}
}

// bundleGenericReference handles bundling of generic OpenAPI component references
func bundleGenericReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ctx context.Context, ref *Reference[T, V, C], namingStrategy BundleNamingStrategy, opts ResolveOptions, componentStorage *componentStorage, componentType string) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}

	refStr, classification := handleReference(ref.GetReference(), opts.TargetLocation)
	if classification == nil {
		return nil // Invalid reference, skip
	}

	// Check if this is an internal reference to the root document
	if isInternalReference(refStr, componentStorage.rootLocation) {
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

	// Get the final absolute reference by following the resolution chain
	// This handles chained references (e.g., common.yaml -> headers.yaml)
	finalAbsRef := getFinalAbsoluteRef(ref, refStr)

	// Normalize finalAbsRef to forward slashes for consistent map keys across platforms
	// On Windows, resolution chains may introduce backslashes, but we need forward slashes
	// to match the keys created by handleReference which always normalizes to forward slashes
	finalAbsRef = filepath.ToSlash(finalAbsRef)

	// Check if we've already processed this reference (using final absolute ref for deduplication)
	if existingName, exists := componentStorage.refs[finalAbsRef]; exists {
		// Also map the intermediate reference to the same component name
		if refStr != finalAbsRef {
			componentStorage.refs[refStr] = existingName
		}
		return nil
	}

	// Generate component name using the final absolute reference
	componentName, err := generateComponentName(finalAbsRef, namingStrategy, componentStorage.componentNames, componentStorage.rootLocation)
	if err != nil {
		return fmt.Errorf("failed to generate component name for %s: %w", finalAbsRef, err)
	}
	componentStorage.componentNames[componentName] = true

	// Store the mapping (both original and final refs point to the same component)
	componentStorage.refs[finalAbsRef] = componentName
	if refStr != finalAbsRef {
		componentStorage.refs[refStr] = componentName
	}

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

		// Get the final resolution info for nested bundling
		targetDocInfo := getFinalResolutionInfo(ref)
		if targetDocInfo == nil {
			// Fall back to the immediate resolution info if final resolution info is unavailable
			targetDocInfo = ref.GetReferenceResolutionInfo()
		}
		if targetDocInfo == nil {
			return fmt.Errorf("failed to get resolution info for %s reference %s", componentType, refStr)
		}
		componentStorage.componentLocations[componentType+"/"+componentName] = targetDocInfo.AbsoluteReference

		if err := bundleObject(ctx, bundledRef, namingStrategy, references.ResolveOptions{
			RootDocument:   opts.RootDocument,
			TargetDocument: targetDocInfo.ResolvedDocument,
			TargetLocation: targetDocInfo.AbsoluteReference,
		}, componentStorage); err != nil {
			return fmt.Errorf("failed to bundle nested references in %s: %w", ref.GetReference(), err)
		}
	}

	return nil
}

// getFinalAbsoluteRef follows the reference resolution chain to get the final absolute reference.
// This is needed for proper deduplication when we have chained references like:
// testapi.yaml -> common.yaml#/components/headers/X -> headers.yaml#/components/headers/X
// Both should resolve to the same final component.
func getFinalAbsoluteRef[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ref *Reference[T, V, C], initialAbsRef string) string {
	if ref == nil {
		return initialAbsRef
	}

	resInfo := ref.GetReferenceResolutionInfo()
	if resInfo == nil {
		return initialAbsRef
	}

	// Check if the resolved object is itself a reference (chained reference)
	if resInfo.Object != nil && resInfo.Object.IsReference() {
		// Follow the chain to get the final resolution info
		nextRefInfo := resInfo.Object.GetReferenceResolutionInfo()
		if nextRefInfo != nil {
			// Build the absolute reference from the final resolution
			finalRef := nextRefInfo.AbsoluteReference
			if nextRefInfo.Object != nil && nextRefInfo.Object.Reference != nil {
				// Add the fragment from the chained reference
				fragment := string(nextRefInfo.Object.Reference.GetJSONPointer())
				if fragment != "" {
					finalRef = finalRef + "#" + fragment
				}
			} else {
				// Use the original reference's fragment with the final file location
				origRef := resInfo.Object.GetReference()
				fragment := string(origRef.GetJSONPointer())
				if fragment != "" {
					finalRef = finalRef + "#" + fragment
				}
			}
			// Recursively follow more chains if needed
			return getFinalAbsoluteRef(resInfo.Object, finalRef)
		}
	}

	return initialAbsRef
}

// getFinalResolutionInfo follows the reference resolution chain to get the final resolution info.
// This returns the resolution info for the last step in a chained reference.
func getFinalResolutionInfo[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ref *Reference[T, V, C]) *references.ResolveResult[Reference[T, V, C]] {
	if ref == nil {
		return nil
	}

	resInfo := ref.GetReferenceResolutionInfo()
	if resInfo == nil {
		return nil
	}

	// Check if the resolved object is itself a reference (chained reference)
	if resInfo.Object != nil && resInfo.Object.IsReference() {
		// Follow the chain to get the final resolution info
		nextRefInfo := resInfo.Object.GetReferenceResolutionInfo()
		if nextRefInfo != nil {
			// Recursively follow more chains
			return getFinalResolutionInfo(resInfo.Object)
		}
	}

	return resInfo
}

// generateComponentName creates a new component name based on the reference and naming strategy
func generateComponentName(ref string, strategy BundleNamingStrategy, usedNames map[string]bool, targetLocation string) (string, error) {
	// Convert absolute path back to relative for component naming
	relativeRef := makeReferenceRelativeForNaming(ref, targetLocation)

	switch strategy {
	case BundleNamingFilePath:
		return generateFilePathBasedNameWithConflictResolution(relativeRef, usedNames, targetLocation)
	case BundleNamingCounter:
		return generateCounterBasedName(relativeRef, usedNames), nil
	default:
		return generateCounterBasedName(relativeRef, usedNames), nil
	}
}

// generateComponentNameWithHashConflictResolution creates a component name with smart conflict resolution based on content hashes
func generateComponentNameWithHashConflictResolution(ref string, strategy BundleNamingStrategy, usedNames map[string]bool, schemaHashes map[string]string, resolvedHash string, targetLocation string) (string, error) {
	// Convert absolute path back to relative for component naming
	relativeRef := makeReferenceRelativeForNaming(ref, targetLocation)

	// Extract simple name from reference
	simpleName := extractSimpleNameFromReference(relativeRef)

	// Check if a schema with this simple name already exists
	if existingHash, exists := schemaHashes[simpleName]; exists {
		if existingHash == resolvedHash {
			// Same content, reuse existing schema
			return simpleName, nil
		}
		// Different content with same name - need conflict resolution
		// Fall back to the configured naming strategy for conflict resolution (use already-relative ref)
		switch strategy {
		case BundleNamingFilePath:
			return generateFilePathBasedNameWithConflictResolution(relativeRef, usedNames, targetLocation)
		case BundleNamingCounter:
			return generateCounterBasedName(relativeRef, usedNames), nil
		default:
			return generateCounterBasedName(relativeRef, usedNames), nil
		}
	}

	// No conflict, use simple name
	return simpleName, nil
}

// generateFilePathBasedNameWithConflictResolution tries to use simple names first, falling back to file-path-based names for conflicts
func generateFilePathBasedNameWithConflictResolution(ref string, usedNames map[string]bool, targetLocation string) (string, error) {
	// Extract simple name from reference
	simpleName := extractSimpleNameFromReference(ref)

	// Try simple name first
	if !usedNames[simpleName] {
		return simpleName, nil
	}

	// If there's a conflict, fall back to file-path-based naming
	return generateFilePathBasedName(ref, usedNames, targetLocation)
}

// generateFilePathBasedName creates names like "some_path_external_yaml__User" or "some_path_external_yaml" for top-level refs
func generateFilePathBasedName(ref string, usedNames map[string]bool, targetLocation string) (string, error) {
	// Parse the reference to extract file path and fragment using references package
	reference := references.Reference(ref)
	filePath := reference.GetURI()
	fragment := string(reference.GetJSONPointer())

	// Convert full file path to safe component name
	// Clean the path but keep extension for uniqueness
	cleanPath := filepath.Clean(filePath)

	// Remove leading "./" if present
	cleanPath = strings.TrimPrefix(cleanPath, "./")

	// Normalize paths that are absolute OR contain parent directory references (..)
	if targetLocation != "" && (filepath.IsAbs(cleanPath) || strings.Contains(cleanPath, "..")) {
		// Normalize to get actual directory names instead of ../
		normalizedPath, err := normalizePathForComponentName(cleanPath, targetLocation)
		if err != nil {
			return "", fmt.Errorf("failed to normalize path %s: %w", cleanPath, err)
		}
		cleanPath = normalizedPath
	}

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
		componentName = safeFileName + "__" + cleanFragment
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
			return "", fmt.Errorf("not enough path components in resolved absolute path: got %d, need at least %d", len(absParts), requiredParts)
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
	// Extract simple name from reference
	baseName := extractSimpleNameFromReference(ref)

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

					// Convert the reference to absolute for lookup using root location
					absRef, _ := handleReference(schema.GetRef(), componentStorage.rootLocation)

					if newName, exists := componentStorage.refs[absRef]; exists {
						// Update the reference to point to the new component
						newRef := "#/components/schemas/" + newName
						*schema.GetLeft().Ref = references.Reference(newRef)
					} else if newName, found := findCircularReferenceMatch(ref, componentStorage.refs); found {
						newRef := "#/components/schemas/" + newName
						*schema.GetLeft().Ref = references.Reference(newRef)
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

	// Convert the reference to absolute for lookup using root location
	absRef, _ := handleReference(ref.GetReference(), componentStorage.rootLocation)

	if newName, exists := componentStorage.refs[absRef]; exists {
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

// handleReference processes a reference for bundling by converting it to an absolute path.
// This function is specifically designed for the Bundle operation, which needs to:
//   - Track all external references using absolute paths as unique identifiers
//   - Normalize paths to forward slashes for cross-platform consistency in the refs map
//   - Handle both file-based and URL-based references uniformly
//
// This differs from handleLocalizeReference, which preserves relative paths and original
// path separators to maintain the document structure during file copying operations.
//
// Parameters:
//   - ref: The reference to process
//   - targetLocation: The absolute path of the document containing this reference
//
// Returns:
//   - string: The absolute reference path (normalized to forward slashes for file paths)
//   - *utils.ReferenceClassification: Classification of the reference type, or nil if invalid
func handleReference(ref references.Reference, targetLocation string) (string, *utils.ReferenceClassification) {
	r := ref.String()

	// Check if this is an external reference using the utility function
	classification, err := utils.ClassifyReference(r)
	if err != nil {
		return "", nil // Invalid reference, skip
	}

	// For URLs, they're already absolute - return as-is
	if classification.Type == utils.ReferenceTypeURL {
		return r, classification
	}

	// If we have a target location, make the reference absolute
	if targetLocation != "" {
		// Classify the target location to determine how to join
		baseClassification, err := utils.ClassifyReference(targetLocation)
		if err != nil {
			// Invalid base location - cannot proceed with reference classification
			return "", nil
		}

		var absolutePath string

		// For fragment-only references, prepend the target location
		if classification.IsFragment {
			absolutePath = targetLocation + r
		} else {
			// For file path references, join with the target location
			if baseClassification.IsURL {
				// Base is URL, join using URL resolution
				joined, err := baseClassification.JoinWith(r)
				if err == nil {
					absolutePath = joined
				} else {
					// Error joining URLs is not fatal - fall back to using the original reference
					// This can happen with malformed URLs or incompatible URL structures
					absolutePath = r
				}
			} else {
				// Base is file path, resolve relative path
				// Base location is assumed to be absolute at this point
				// Split reference into path and fragment using references package
				refParsed := references.Reference(r)
				filePath := refParsed.GetURI()
				fragment := ""
				if refParsed.HasJSONPointer() {
					fragment = "#" + string(refParsed.GetJSONPointer())
				}

				// Join with target directory
				baseDir := filepath.Dir(targetLocation)
				joinedPath := filepath.Join(baseDir, filePath)

				// Clean the path and immediately normalize to forward slashes
				// This prevents issues with Windows path handling
				absPath := filepath.Clean(joinedPath)
				absPath = filepath.ToSlash(absPath)

				absolutePath = absPath + fragment
			}
		}

		r = absolutePath

		// Normalize to forward slashes for cross-platform path consistency
		// This ensures that C:\path and C:/path are treated the same
		// Apply normalization to the full absolute path (including fragment)
		refParsed := references.Reference(r)
		uri := refParsed.GetURI()

		// Always normalize absolute paths to forward slashes
		if uri != "" {
			// Check if it's an absolute path (works for both C:\path and C:/path)
			isAbs := filepath.IsAbs(uri) || (len(uri) >= 3 && uri[1] == ':' && (uri[2] == '/' || uri[2] == '\\'))

			if isAbs {
				normalizedURI := filepath.ToSlash(uri)
				if refParsed.HasJSONPointer() {
					r = normalizedURI + "#" + string(refParsed.GetJSONPointer())
				} else {
					r = normalizedURI
				}
			}
		}

		// Re-classify after making absolute and normalizing
		cl, err := utils.ClassifyReference(r)
		if err == nil {
			classification = cl
		}
		// Error re-classifying is not fatal - we keep using the previous classification
		// This maintains backward compatibility and allows processing to continue
	}

	return r, classification
}

// makeReferenceRelativeForNaming converts an absolute reference path back to a relative path
// suitable for component naming, relative to the root document location (assumed to be absolute)
func makeReferenceRelativeForNaming(ref string, rootLocation string) string {
	if rootLocation == "" {
		return ref
	}

	// Parse reference using the references package
	reference := references.Reference(ref)
	uri := reference.GetURI()

	// If there's no URI (just a fragment), return as-is
	if uri == "" {
		return ref
	}

	// On Windows, paths with forward slashes can be misclassified as URLs
	// Normalize to native separators before classification to avoid this
	normalizedURI := uri
	if filepath.Separator == '\\' && len(uri) >= 3 && uri[1] == ':' && uri[2] == '/' {
		// Windows path with forward slashes like C:/path - convert to backslashes
		normalizedURI = filepath.FromSlash(uri)
	}

	// Check if this is a URL - if so, return as-is
	// Error classifying reference is not fatal - we return the original ref unchanged
	classification, err := utils.ClassifyReference(normalizedURI)
	if err != nil || classification.IsURL {
		return ref
	}

	// If the URI is absolute, make it relative to the root document's directory
	// rootLocation is assumed to be absolute at this point
	if filepath.IsAbs(normalizedURI) {
		// Normalize rootLocation as well for consistent comparison
		normalizedRoot := rootLocation
		if filepath.Separator == '\\' {
			normalizedRoot = filepath.FromSlash(rootLocation)
		}
		rootDir := filepath.Dir(normalizedRoot)
		relPath, err := filepath.Rel(rootDir, normalizedURI)
		if err == nil {
			// Reconstruct the reference with relative path
			if reference.HasJSONPointer() {
				return relPath + "#" + string(reference.GetJSONPointer())
			}
			return relPath
		}
		// Error making path relative is not fatal - we fall through to return the original ref
		// This can happen when paths are on different drives (Windows) or incompatible
	}

	// Return as-is if we couldn't make it relative
	return ref
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

// Helper functions for DRY principle

// isInternalReference checks if a reference points to the root document
func isInternalReference(ref string, rootLocation string) bool {
	refURI := references.Reference(ref).GetURI()
	if refURI == "" {
		return true // Fragment-only reference
	}

	cleanRefURI := filepath.Clean(refURI)
	cleanRootURI := filepath.Clean(rootLocation)
	return cleanRefURI == cleanRootURI
}

// extractSimpleNameFromReference extracts a simple component name from a reference
func extractSimpleNameFromReference(ref string) string {
	reference := references.Reference(ref)
	filePath := reference.GetURI()
	fragment := string(reference.GetJSONPointer())

	if fragment == "" || fragment == "/" {
		// Top-level file reference - use filename as simple name
		baseName := filepath.Base(filePath)
		ext := filepath.Ext(baseName)
		if ext != "" {
			baseName = baseName[:len(baseName)-len(ext)]
		}
		return regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(baseName, "_")
	}

	// Reference to specific schema within file - extract schema name
	cleanFragment := strings.TrimPrefix(fragment, "/")
	fragmentParts := strings.Split(cleanFragment, "/")
	if len(fragmentParts) == 0 {
		return "unknown"
	}
	return fragmentParts[len(fragmentParts)-1]
}

// findCircularReferenceMatch finds a component name for a circular reference
func findCircularReferenceMatch(refStr string, refs map[string]string) (string, bool) {
	// Only match fragment-only references that aren't already component references
	if !strings.HasPrefix(refStr, "#/") || strings.HasPrefix(refStr, "#/components/") {
		return "", false
	}

	// Look for a matching reference that ends with this fragment
	// e.g., "/absolute/path/external_conflicting_user.yaml#/User" ends with "#/User"
	for externalRef, componentName := range refs {
		if strings.HasSuffix(externalRef, refStr) {
			return componentName, true
		}
	}

	return "", false
}
