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
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// JoinConflictStrategy defines how conflicts should be resolved when joining documents.
type JoinConflictStrategy int

const (
	// JoinConflictCounter uses counter-based suffixes like User_1, User_2 for conflicts
	JoinConflictCounter JoinConflictStrategy = iota
	// JoinConflictFilePath uses file path-based naming like file_path_somefile_yaml~User
	JoinConflictFilePath
)

// JoinOptions represents the options available when joining OpenAPI documents.
type JoinOptions struct {
	// ConflictStrategy determines how conflicts are resolved when joining documents.
	ConflictStrategy JoinConflictStrategy
	// DocumentPaths maps each document to its file path for conflict resolution.
	// The key should match the document's position in the documents slice.
	DocumentPaths map[int]string
}

// JoinDocumentInfo holds information about a document being joined.
type JoinDocumentInfo struct {
	Document *OpenAPI
	FilePath string
}

// Join combines multiple OpenAPI documents into one, using conflict resolution strategies
// similar to bundling but without inlining external references. This creates a single
// document that retains references to external documents while resolving conflicts
// between local components and operations.
//
// The main document serves as the base:
//   - Its Info and OpenAPI version fields are retained
//   - For conflicting servers, security, and tags, the main document's values are kept
//
// For other fields:
//   - Operations, components, webhooks, and extensions are appended from all documents
//   - Operation conflicts create new paths with fragments containing the file name
//   - Component conflicts use the same strategy as bundling (counter or filepath naming)
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - mainDoc: The main document that serves as the base (will be modified in place)
//   - documents: Slice of JoinDocumentInfo containing additional documents and their file paths
//   - opts: Configuration options for joining
//
// Returns:
//   - error: Any error that occurred during joining
func Join(ctx context.Context, mainDoc *OpenAPI, documents []JoinDocumentInfo, opts JoinOptions) error {
	if mainDoc == nil {
		return errors.New("main document is nil")
	}

	// Track used names for conflict resolution
	usedComponentNames := make(map[string]bool)
	componentHashes := make(map[string]string)
	usedPathNames := make(map[string]bool)

	// Initialize tracking with existing components and paths from main document
	initializeUsedNames(mainDoc, usedComponentNames, componentHashes, usedPathNames)

	// Join all additional documents
	for i, docInfo := range documents {
		doc := docInfo.Document
		if doc == nil {
			continue
		}

		docPath := docInfo.FilePath
		if docPath == "" {
			// Use index as fallback if no path provided
			docPath = fmt.Sprintf("document_%d", i)
		}

		err := joinSingleDocument(ctx, mainDoc, doc, docPath, opts, usedComponentNames, componentHashes, usedPathNames)
		if err != nil {
			return fmt.Errorf("failed to join document %d (%s): %w", i, docPath, err)
		}
	}

	return nil
}

// initializeUsedNames populates the tracking maps with existing names from the lead document
func initializeUsedNames(doc *OpenAPI, usedComponentNames map[string]bool, componentHashes map[string]string, usedPathNames map[string]bool) {
	// Track existing component names and hashes
	if doc.Components != nil {
		if doc.Components.Schemas != nil {
			for name, schema := range doc.Components.Schemas.All() {
				usedComponentNames[name] = true
				if schema != nil {
					componentHashes[name] = hashing.Hash(schema)
				}
			}
		}

		// Track other component types
		trackComponentNames := func(components interface{}) {
			switch c := components.(type) {
			case *sequencedmap.Map[string, *ReferencedResponse]:
				if c != nil {
					for name := range c.All() {
						usedComponentNames[name] = true
					}
				}
			case *sequencedmap.Map[string, *ReferencedParameter]:
				if c != nil {
					for name := range c.All() {
						usedComponentNames[name] = true
					}
				}
			case *sequencedmap.Map[string, *ReferencedExample]:
				if c != nil {
					for name := range c.All() {
						usedComponentNames[name] = true
					}
				}
			case *sequencedmap.Map[string, *ReferencedRequestBody]:
				if c != nil {
					for name := range c.All() {
						usedComponentNames[name] = true
					}
				}
			case *sequencedmap.Map[string, *ReferencedHeader]:
				if c != nil {
					for name := range c.All() {
						usedComponentNames[name] = true
					}
				}
			case *sequencedmap.Map[string, *ReferencedSecurityScheme]:
				if c != nil {
					for name := range c.All() {
						usedComponentNames[name] = true
					}
				}
			case *sequencedmap.Map[string, *ReferencedLink]:
				if c != nil {
					for name := range c.All() {
						usedComponentNames[name] = true
					}
				}
			case *sequencedmap.Map[string, *ReferencedCallback]:
				if c != nil {
					for name := range c.All() {
						usedComponentNames[name] = true
					}
				}
			case *sequencedmap.Map[string, *ReferencedPathItem]:
				if c != nil {
					for name := range c.All() {
						usedComponentNames[name] = true
					}
				}
			}
		}

		trackComponentNames(doc.Components.Responses)
		trackComponentNames(doc.Components.Parameters)
		trackComponentNames(doc.Components.Examples)
		trackComponentNames(doc.Components.RequestBodies)
		trackComponentNames(doc.Components.Headers)
		trackComponentNames(doc.Components.SecuritySchemes)
		trackComponentNames(doc.Components.Links)
		trackComponentNames(doc.Components.Callbacks)
		trackComponentNames(doc.Components.PathItems)
	}

	// Track existing path names
	if doc.Paths != nil {
		for path := range doc.Paths.All() {
			usedPathNames[path] = true
		}
	}

}

// joinSingleDocument joins a single document into the result document
func joinSingleDocument(ctx context.Context, result, doc *OpenAPI, docPath string, opts JoinOptions, usedComponentNames map[string]bool, componentHashes map[string]string, usedPathNames map[string]bool) error {
	// Track component name mappings for reference updates
	componentMappings := make(map[string]string)

	// Join components with conflict resolution first (to get mappings)
	if err := joinComponents(result, doc, docPath, opts.ConflictStrategy, usedComponentNames, componentHashes, componentMappings); err != nil {
		return fmt.Errorf("failed to join components: %w", err)
	}

	// Update references in the document before joining paths and webhooks
	if err := updateReferencesInDocument(ctx, doc, componentMappings); err != nil {
		return fmt.Errorf("failed to update references in document: %w", err)
	}

	// Join paths with conflict resolution
	joinPaths(result, doc, docPath, usedPathNames)

	// Join webhooks
	joinWebhooks(result, doc)

	// Join tags with conflict resolution (append unless conflicting names)
	joinTags(result, doc)

	// Join servers and security with smart conflict resolution
	joinServersAndSecurity(result, doc)

	return nil
}

// joinPaths joins paths from source document into result, handling operation conflicts
func joinPaths(result, src *OpenAPI, srcPath string, usedPathNames map[string]bool) {
	if src.Paths == nil {
		return
	}

	// Ensure result has paths
	if result.Paths == nil {
		result.Paths = NewPaths()
	}

	for path, pathItem := range src.Paths.All() {
		if !usedPathNames[path] {
			// No conflict, add directly
			result.Paths.Set(path, pathItem)
			usedPathNames[path] = true
		} else {
			// Conflict detected, create new path with fragment
			newPath := generateConflictPath(path, srcPath)
			result.Paths.Set(newPath, pathItem)
			usedPathNames[newPath] = true
		}
	}

}

// generateConflictPath creates a new path with a fragment containing the file name
func generateConflictPath(originalPath, filePath string) string {
	// Extract filename without extension for the fragment
	baseName := filepath.Base(filePath)
	ext := filepath.Ext(baseName)
	if ext != "" {
		baseName = baseName[:len(baseName)-len(ext)]
	}

	// Clean the filename to make it URL-safe
	safeFileName := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(baseName, "_")

	// Create new path with fragment
	return fmt.Sprintf("%s#%s", originalPath, safeFileName)
}

// joinWebhooks joins webhooks from source document into result
func joinWebhooks(result, src *OpenAPI) {
	if src.Webhooks == nil {
		return
	}

	// Ensure result has webhooks
	if result.Webhooks == nil {
		result.Webhooks = sequencedmap.New[string, *ReferencedPathItem]()
	}

	for name, webhook := range src.Webhooks.All() {
		// For webhooks, we append all - no conflict resolution needed as they're named
		result.Webhooks.Set(name, webhook)
	}

}

// joinComponents joins components from source document into result with conflict resolution
func joinComponents(result, src *OpenAPI, srcPath string, strategy JoinConflictStrategy, usedComponentNames map[string]bool, componentHashes map[string]string, componentMappings map[string]string) error {
	if src.Components == nil {
		return nil
	}

	// Ensure result has components
	if result.Components == nil {
		result.Components = &Components{}
	}

	// Join schemas with hash-based conflict resolution
	joinSchemas(result.Components, src.Components, srcPath, strategy, usedComponentNames, componentHashes, componentMappings)

	// Join other component types
	if err := joinOtherComponents(result.Components, src.Components, srcPath, strategy, usedComponentNames, componentHashes, componentMappings); err != nil {
		return fmt.Errorf("failed to join other components: %w", err)
	}

	return nil
}

// joinSchemas joins schemas with smart conflict resolution based on content hashes
func joinSchemas(resultComponents, srcComponents *Components, srcPath string, strategy JoinConflictStrategy, usedComponentNames map[string]bool, componentHashes map[string]string, componentMappings map[string]string) {
	if srcComponents.Schemas == nil {
		return
	}

	// Ensure result has schemas
	if resultComponents.Schemas == nil {
		resultComponents.Schemas = sequencedmap.New[string, *oas3.JSONSchema[oas3.Referenceable]]()
	}

	for name, schema := range srcComponents.Schemas.All() {
		if schema == nil {
			continue
		}

		schemaHash := hashing.Hash(schema)

		// Check if a schema with this name already exists
		if existingHash, exists := componentHashes[name]; exists {
			if existingHash == schemaHash {
				// Same content, skip (no need to add duplicate)
				continue
			}
			// Different content with same name - need conflict resolution
			newName := generateJoinComponentName(name, srcPath, strategy, usedComponentNames)
			resultComponents.Schemas.Set(newName, schema)
			usedComponentNames[newName] = true
			componentHashes[newName] = schemaHash
			// Track the mapping for reference updates
			componentMappings[name] = newName
		} else {
			// No conflict, use original name
			resultComponents.Schemas.Set(name, schema)
			usedComponentNames[name] = true
			componentHashes[name] = schemaHash
		}
	}

}

// joinOtherComponents joins non-schema components with conflict resolution
func joinOtherComponents(resultComponents, srcComponents *Components, srcPath string, strategy JoinConflictStrategy, usedComponentNames map[string]bool, componentHashes map[string]string, componentMappings map[string]string) error {
	// Helper function to join a specific component type
	joinComponentType := func(
		getResult func() interface{},
		getSrc func() interface{},
		setResult func(interface{}),
		createNew func() interface{},
	) error {
		srcMap := getSrc()
		if srcMap == nil {
			return nil
		}

		// Ensure result has this component type
		resultMap := getResult()
		if resultMap == nil {
			resultMap = createNew()
			setResult(resultMap)
		}

		// Use reflection-like approach to handle different map types
		switch src := srcMap.(type) {
		case *sequencedmap.Map[string, *ReferencedResponse]:
			result := resultMap.(*sequencedmap.Map[string, *ReferencedResponse])
			for name, item := range src.All() {
				finalName := name
				if usedComponentNames[name] {
					finalName = generateJoinComponentName(name, srcPath, strategy, usedComponentNames)
					componentMappings[name] = finalName
				}
				result.Set(finalName, item)
				usedComponentNames[finalName] = true
			}
		case *sequencedmap.Map[string, *ReferencedParameter]:
			result := resultMap.(*sequencedmap.Map[string, *ReferencedParameter])
			for name, item := range src.All() {
				finalName := name
				if usedComponentNames[name] {
					finalName = generateJoinComponentName(name, srcPath, strategy, usedComponentNames)
					componentMappings[name] = finalName
				}
				result.Set(finalName, item)
				usedComponentNames[finalName] = true
			}
		case *sequencedmap.Map[string, *ReferencedExample]:
			result := resultMap.(*sequencedmap.Map[string, *ReferencedExample])
			for name, item := range src.All() {
				finalName := name
				if usedComponentNames[name] {
					finalName = generateJoinComponentName(name, srcPath, strategy, usedComponentNames)
					componentMappings[name] = finalName
				}
				result.Set(finalName, item)
				usedComponentNames[finalName] = true
			}
		case *sequencedmap.Map[string, *ReferencedRequestBody]:
			result := resultMap.(*sequencedmap.Map[string, *ReferencedRequestBody])
			for name, item := range src.All() {
				finalName := name
				if usedComponentNames[name] {
					finalName = generateJoinComponentName(name, srcPath, strategy, usedComponentNames)
					componentMappings[name] = finalName
				}
				result.Set(finalName, item)
				usedComponentNames[finalName] = true
			}
		case *sequencedmap.Map[string, *ReferencedHeader]:
			result := resultMap.(*sequencedmap.Map[string, *ReferencedHeader])
			for name, item := range src.All() {
				finalName := name
				if usedComponentNames[name] {
					finalName = generateJoinComponentName(name, srcPath, strategy, usedComponentNames)
					componentMappings[name] = finalName
				}
				result.Set(finalName, item)
				usedComponentNames[finalName] = true
			}
		case *sequencedmap.Map[string, *ReferencedSecurityScheme]:
			result := resultMap.(*sequencedmap.Map[string, *ReferencedSecurityScheme])
			for name, item := range src.All() {
				if item == nil {
					continue
				}

				itemHash := hashing.Hash(item)

				// Check if a security scheme with this name already exists
				if existingHash, exists := componentHashes[name]; exists {
					if existingHash == itemHash {
						// Same content, skip (no need to add duplicate)
						continue
					}
					// Different content with same name - need conflict resolution
					finalName := generateJoinComponentName(name, srcPath, strategy, usedComponentNames)
					result.Set(finalName, item)
					usedComponentNames[finalName] = true
					componentHashes[finalName] = itemHash
					componentMappings[name] = finalName
				} else {
					// No conflict, use original name
					result.Set(name, item)
					usedComponentNames[name] = true
					componentHashes[name] = itemHash
				}
			}
		case *sequencedmap.Map[string, *ReferencedLink]:
			result := resultMap.(*sequencedmap.Map[string, *ReferencedLink])
			for name, item := range src.All() {
				finalName := name
				if usedComponentNames[name] {
					finalName = generateJoinComponentName(name, srcPath, strategy, usedComponentNames)
					componentMappings[name] = finalName
				}
				result.Set(finalName, item)
				usedComponentNames[finalName] = true
			}
		case *sequencedmap.Map[string, *ReferencedCallback]:
			result := resultMap.(*sequencedmap.Map[string, *ReferencedCallback])
			for name, item := range src.All() {
				finalName := name
				if usedComponentNames[name] {
					finalName = generateJoinComponentName(name, srcPath, strategy, usedComponentNames)
					componentMappings[name] = finalName
				}
				result.Set(finalName, item)
				usedComponentNames[finalName] = true
			}
		case *sequencedmap.Map[string, *ReferencedPathItem]:
			result := resultMap.(*sequencedmap.Map[string, *ReferencedPathItem])
			for name, item := range src.All() {
				finalName := name
				if usedComponentNames[name] {
					finalName = generateJoinComponentName(name, srcPath, strategy, usedComponentNames)
					componentMappings[name] = finalName
				}
				result.Set(finalName, item)
				usedComponentNames[finalName] = true
			}
		}

		return nil
	}

	// Join responses
	if err := joinComponentType(
		func() interface{} { return resultComponents.Responses },
		func() interface{} { return srcComponents.Responses },
		func(v interface{}) { resultComponents.Responses = v.(*sequencedmap.Map[string, *ReferencedResponse]) },
		func() interface{} { return sequencedmap.New[string, *ReferencedResponse]() },
	); err != nil {
		return err
	}

	// Join parameters
	if err := joinComponentType(
		func() interface{} { return resultComponents.Parameters },
		func() interface{} { return srcComponents.Parameters },
		func(v interface{}) { resultComponents.Parameters = v.(*sequencedmap.Map[string, *ReferencedParameter]) },
		func() interface{} { return sequencedmap.New[string, *ReferencedParameter]() },
	); err != nil {
		return err
	}

	// Join examples
	if err := joinComponentType(
		func() interface{} { return resultComponents.Examples },
		func() interface{} { return srcComponents.Examples },
		func(v interface{}) { resultComponents.Examples = v.(*sequencedmap.Map[string, *ReferencedExample]) },
		func() interface{} { return sequencedmap.New[string, *ReferencedExample]() },
	); err != nil {
		return err
	}

	// Join request bodies
	if err := joinComponentType(
		func() interface{} { return resultComponents.RequestBodies },
		func() interface{} { return srcComponents.RequestBodies },
		func(v interface{}) {
			resultComponents.RequestBodies = v.(*sequencedmap.Map[string, *ReferencedRequestBody])
		},
		func() interface{} { return sequencedmap.New[string, *ReferencedRequestBody]() },
	); err != nil {
		return err
	}

	// Join headers
	if err := joinComponentType(
		func() interface{} { return resultComponents.Headers },
		func() interface{} { return srcComponents.Headers },
		func(v interface{}) { resultComponents.Headers = v.(*sequencedmap.Map[string, *ReferencedHeader]) },
		func() interface{} { return sequencedmap.New[string, *ReferencedHeader]() },
	); err != nil {
		return err
	}

	// Join security schemes
	if err := joinComponentType(
		func() interface{} { return resultComponents.SecuritySchemes },
		func() interface{} { return srcComponents.SecuritySchemes },
		func(v interface{}) {
			resultComponents.SecuritySchemes = v.(*sequencedmap.Map[string, *ReferencedSecurityScheme])
		},
		func() interface{} { return sequencedmap.New[string, *ReferencedSecurityScheme]() },
	); err != nil {
		return err
	}

	// Join links
	if err := joinComponentType(
		func() interface{} { return resultComponents.Links },
		func() interface{} { return srcComponents.Links },
		func(v interface{}) { resultComponents.Links = v.(*sequencedmap.Map[string, *ReferencedLink]) },
		func() interface{} { return sequencedmap.New[string, *ReferencedLink]() },
	); err != nil {
		return err
	}

	// Join callbacks
	if err := joinComponentType(
		func() interface{} { return resultComponents.Callbacks },
		func() interface{} { return srcComponents.Callbacks },
		func(v interface{}) { resultComponents.Callbacks = v.(*sequencedmap.Map[string, *ReferencedCallback]) },
		func() interface{} { return sequencedmap.New[string, *ReferencedCallback]() },
	); err != nil {
		return err
	}

	// Join path items
	if err := joinComponentType(
		func() interface{} { return resultComponents.PathItems },
		func() interface{} { return srcComponents.PathItems },
		func(v interface{}) { resultComponents.PathItems = v.(*sequencedmap.Map[string, *ReferencedPathItem]) },
		func() interface{} { return sequencedmap.New[string, *ReferencedPathItem]() },
	); err != nil {
		return err
	}

	return nil
}

// generateJoinComponentName creates a new component name using the same strategy as bundling
func generateJoinComponentName(originalName, filePath string, strategy JoinConflictStrategy, usedNames map[string]bool) string {
	switch strategy {
	case JoinConflictFilePath:
		return generateJoinFilePathBasedName(originalName, filePath, usedNames)
	case JoinConflictCounter:
		return generateJoinCounterBasedName(originalName, usedNames)
	default:
		return generateJoinCounterBasedName(originalName, usedNames)
	}
}

// generateJoinFilePathBasedName creates names like "some_path_external_yaml~User"
func generateJoinFilePathBasedName(originalName, filePath string, usedNames map[string]bool) string {
	// Convert file path to safe component name
	cleanPath := filepath.Clean(filePath)
	cleanPath = strings.TrimPrefix(cleanPath, "./")

	// Replace extension dot with underscore
	ext := filepath.Ext(cleanPath)
	if ext != "" {
		cleanPath = cleanPath[:len(cleanPath)-len(ext)] + "_" + ext[1:]
	}

	// Replace path separators and unsafe characters with underscores
	safeFileName := regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(cleanPath, "_")

	componentName := safeFileName + "~" + originalName

	// Ensure uniqueness
	originalComponentName := componentName
	counter := 1
	for usedNames[componentName] {
		componentName = fmt.Sprintf("%s_%d", originalComponentName, counter)
		counter++
	}

	return componentName
}

// generateJoinCounterBasedName creates names like "User_1", "User_2" for conflicts
func generateJoinCounterBasedName(originalName string, usedNames map[string]bool) string {
	componentName := originalName
	counter := 1
	for usedNames[componentName] {
		componentName = fmt.Sprintf("%s_%d", originalName, counter)
		counter++
	}

	return componentName
}

// updateReferencesInDocument updates all references in a document to use the new component names
func updateReferencesInDocument(ctx context.Context, doc *OpenAPI, componentMappings map[string]string) error {
	if len(componentMappings) == 0 {
		return nil
	}

	// Walk through the document and update references
	for item := range Walk(ctx, doc) {
		err := item.Match(Matcher{
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				if schema.IsReference() {
					ref := string(schema.GetRef())
					// Check if this is a component reference that needs updating
					if strings.HasPrefix(ref, "#/components/schemas/") {
						componentName := strings.TrimPrefix(ref, "#/components/schemas/")
						if newName, exists := componentMappings[componentName]; exists {
							newRef := "#/components/schemas/" + newName
							*schema.GetLeft().Ref = references.Reference(newRef)
						}
					}
				}
				return nil
			},
			ReferencedResponse: func(ref *ReferencedResponse) error {
				return updateComponentReference(ref, componentMappings, "responses")
			},
			ReferencedParameter: func(ref *ReferencedParameter) error {
				return updateComponentReference(ref, componentMappings, "parameters")
			},
			ReferencedExample: func(ref *ReferencedExample) error {
				return updateComponentReference(ref, componentMappings, "examples")
			},
			ReferencedRequestBody: func(ref *ReferencedRequestBody) error {
				return updateComponentReference(ref, componentMappings, "requestBodies")
			},
			ReferencedHeader: func(ref *ReferencedHeader) error {
				return updateComponentReference(ref, componentMappings, "headers")
			},
			ReferencedCallback: func(ref *ReferencedCallback) error {
				return updateComponentReference(ref, componentMappings, "callbacks")
			},
			ReferencedLink: func(ref *ReferencedLink) error {
				return updateComponentReference(ref, componentMappings, "links")
			},
			ReferencedSecurityScheme: func(ref *ReferencedSecurityScheme) error {
				return updateComponentReference(ref, componentMappings, "securitySchemes")
			},
			ReferencedPathItem: func(ref *ReferencedPathItem) error {
				return updateComponentReference(ref, componentMappings, "pathItems")
			},
		})
		if err != nil {
			return fmt.Errorf("failed to update reference at %s: %w", item.Location.ToJSONPointer().String(), err)
		}
	}

	return nil
}

// updateComponentReference updates a generic component reference to use the new name
func updateComponentReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](ref *Reference[T, V, C], componentMappings map[string]string, componentSection string) error {
	if ref == nil || !ref.IsReference() {
		return nil
	}

	refStr := string(ref.GetReference())
	expectedPrefix := "#/components/" + componentSection + "/"
	if strings.HasPrefix(refStr, expectedPrefix) {
		componentName := strings.TrimPrefix(refStr, expectedPrefix)
		if newName, exists := componentMappings[componentName]; exists {
			newRef := expectedPrefix + newName
			*ref.Reference = references.Reference(newRef)
		}
	}

	return nil
}

// joinTags joins tags from source document, appending unless there are name conflicts
func joinTags(result, src *OpenAPI) {
	if src.Tags == nil {
		return
	}

	// Create a map of existing tag names for quick lookup
	existingTagNames := make(map[string]bool)
	for _, tag := range result.Tags {
		if tag != nil {
			existingTagNames[tag.Name] = true
		}
	}

	// Append tags that don't conflict
	for _, tag := range src.Tags {
		if tag != nil && !existingTagNames[tag.Name] {
			result.Tags = append(result.Tags, tag)
			existingTagNames[tag.Name] = true
		}
	}
}

// joinServersAndSecurity handles smart conflict resolution for servers and security
func joinServersAndSecurity(result, src *OpenAPI) {
	// Check if servers are identical (by hash)
	serversConflict := !areServersIdentical(result.Servers, src.Servers)

	// Check if security is identical (by hash)
	securityConflict := !areSecurityIdentical(result.Security, src.Security)

	// If there are conflicts, we need to push them down to operation level
	if serversConflict || securityConflict {
		// Apply source document's servers/security to its operations before joining
		applyGlobalServersSecurityToOperations(src, serversConflict, securityConflict)

		// Apply result document's servers/security to its operations if needed
		applyGlobalServersSecurityToOperations(result, serversConflict, securityConflict)

		// Clear conflicting global settings from result document
		if serversConflict {
			result.Servers = nil
		}
		if securityConflict {
			result.Security = nil
		}
	}
	// If no conflicts, keep result document's global servers/security as-is
}

// areServersIdentical checks if two server slices are identical by hash
func areServersIdentical(servers1, servers2 []*Server) bool {
	// If one is empty and the other is not, they can be merged without conflict
	// (empty servers can inherit from non-empty - just different base URLs)
	if len(servers1) == 0 || len(servers2) == 0 {
		return true
	}

	if len(servers1) != len(servers2) {
		return false
	}

	// Hash both server slices and compare
	hash1 := hashing.Hash(servers1)
	hash2 := hashing.Hash(servers2)

	return hash1 == hash2
}

// areSecurityIdentical checks if two security requirement slices are identical by hash
func areSecurityIdentical(security1, security2 []*SecurityRequirement) bool {
	// Security is different: empty security means "no auth required"
	// vs non-empty means "auth required" - these are fundamentally different
	// and must be treated as conflicts
	if len(security1) != len(security2) {
		return false
	}

	// Hash both security slices and compare
	hash1 := hashing.Hash(security1)
	hash2 := hashing.Hash(security2)

	return hash1 == hash2
}

// applyGlobalServersSecurityToOperations pushes global servers/security down to operation level
func applyGlobalServersSecurityToOperations(doc *OpenAPI, applyServers, applySecurity bool) {
	if doc.Paths == nil {
		return
	}

	// Walk through all operations and apply global settings
	for _, pathItem := range doc.Paths.All() {
		if pathItem == nil || pathItem.Object == nil {
			continue
		}

		pathItemObj := pathItem.Object

		// Apply to path-level servers if needed
		if applyServers && len(doc.Servers) > 0 && len(pathItemObj.Servers) == 0 {
			pathItemObj.Servers = make([]*Server, len(doc.Servers))
			copy(pathItemObj.Servers, doc.Servers)
		}

		// Apply to each operation in the path item
		for _, operation := range pathItemObj.All() {
			if operation == nil {
				continue
			}

			// Apply servers to operation if it doesn't have any
			if applyServers && len(doc.Servers) > 0 && len(operation.Servers) == 0 {
				operation.Servers = make([]*Server, len(doc.Servers))
				copy(operation.Servers, doc.Servers)
			}

			// Apply security to operation if it doesn't have any
			if applySecurity && len(doc.Security) > 0 && len(operation.Security) == 0 {
				operation.Security = make([]*SecurityRequirement, len(doc.Security))
				copy(operation.Security, doc.Security)
			}
		}
	}
}
