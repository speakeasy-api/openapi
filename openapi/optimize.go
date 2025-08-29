package openapi

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/hashing"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
	walkpkg "github.com/speakeasy-api/openapi/walk"
)

// OptimizeNameCallback is a callback function that receives information about a new component
// being created and returns the name to use for that component.
//
// Parameters:
//   - suggestedName: The suggested name (e.g., "Schema_abc123def")
//   - hash: The hash of the schema content
//   - locations: Array of JSON pointers to where the inline schemas were found
//   - schema: The JSON schema that will be turned into a component
//
// Returns:
//   - The name to use for the new component
type OptimizeNameCallback func(suggestedName string, hash string, locations []string, schema *oas3.JSONSchema[oas3.Referenceable]) string

// Optimize finds all inline JSON schemas with the same content (hash) and replaces them
// with references to existing or newly created components.
//
// The optimization process:
//  1. Walks through all JSON schemas and collects hashes of inline schemas (except top-level components)
//  2. Only considers complex schemas (object schemas, enums, oneOf/allOf/anyOf, etc.) - not simple types
//  3. For schemas with multiple matches, tries to match them to existing components first
//  4. Creates new components with Schema_{hash} names for unmatched duplicates
//  5. Replaces inline schemas with references to the components
//
// The nameCallback allows customization of component names. If nil, default names are used.
//
// This function modifies the document in place.
//
// Why use Optimize?
//
//   - **Reduce document size**: Eliminate duplicate inline schema definitions
//   - **Improve maintainability**: Centralize schema definitions in components
//   - **Enhance reusability**: Make schemas available for reference throughout the document
//   - **Optimize tooling performance**: Reduce parsing overhead from duplicate schemas
//   - **Standardize structure**: Follow OpenAPI best practices for schema organization
//
// Example usage:
//
//	// Load an OpenAPI document with duplicate inline schemas
//	doc := &OpenAPI{...}
//
//	// Optimize with default naming
//	err := Optimize(ctx, doc, nil)
//	if err != nil {
//		return fmt.Errorf("failed to optimize document: %w", err)
//	}
//
//	// Optimize with custom naming
//	err = Optimize(ctx, doc, func(suggested, hash string, locations []string, schema *oas3.JSONSchema[oas3.Referenceable]) string {
//		return "CustomSchema_" + hash[:8]
//	})
//
// Parameters:
//   - ctx: Context for the operation
//   - doc: The OpenAPI document to optimize (modified in place)
//   - nameCallback: Optional callback to customize component names
//
// Returns:
//   - error: Any error that occurred during optimization
func Optimize(ctx context.Context, doc *OpenAPI, nameCallback OptimizeNameCallback) error {
	if doc == nil {
		return nil
	}

	// Initialize components if needed
	if doc.Components == nil {
		doc.Components = &Components{}
	}
	if doc.Components.Schemas == nil {
		doc.Components.Schemas = sequencedmap.New[string, *oas3.JSONSchema[oas3.Referenceable]]()
	}

	// Step 1: Collect all inline schemas and their locations
	schemaCollector := &inlineSchemaCollector{
		schemas:            sequencedmap.New[string, *schemaInfo](),
		existingComponents: sequencedmap.New[string, string](), // hash -> component name
	}

	// First, catalog existing components by their hash
	for name, schema := range doc.Components.Schemas.All() {
		if schema != nil && !schema.IsReference() && schema.GetLeft() != nil {
			if isComplexSchema(schema.GetLeft()) {
				hash := hashing.Hash(schema.GetLeft())
				schemaCollector.existingComponents.Set(hash, name)
			}
		}
	}

	// Walk through the document to collect inline schemas
	for item := range Walk(ctx, doc) {
		err := item.Match(Matcher{
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				return schemaCollector.collectSchema(schema, item.Location)
			},
		})
		if err != nil {
			return fmt.Errorf("failed to collect schemas: %w", err)
		}
	}

	// Step 2: Collect all individual schema locations and sort by depth (deepest first)
	type schemaLocationWithDepth struct {
		hash        string
		location    schemaLocation
		jsonPointer string
		depth       int
		schema      *oas3.JSONSchema[oas3.Referenceable]
	}

	var allLocations []schemaLocationWithDepth

	for hash, info := range schemaCollector.schemas.All() {
		if len(info.locations) <= 1 {
			continue // Skip schemas that appear only once
		}

		for i, location := range info.locations {
			jsonPtr := info.jsonPointers[i]
			depth := strings.Count(jsonPtr, "/")

			allLocations = append(allLocations, schemaLocationWithDepth{
				hash:        hash,
				location:    location,
				jsonPointer: jsonPtr,
				depth:       depth,
				schema:      info.schema,
			})
		}
	}

	// Sort by depth (deepest first)
	for i := 0; i < len(allLocations); i++ {
		for j := i + 1; j < len(allLocations); j++ {
			if allLocations[j].depth > allLocations[i].depth {
				allLocations[i], allLocations[j] = allLocations[j], allLocations[i]
			}
		}
	}

	// Step 3: Replace all inline occurrences with references (deepest first)
	processedHashes := sequencedmap.New[string, string]() // hash -> componentName

	for _, loc := range allLocations {
		// Check if we already have a component for this hash
		componentName, exists := processedHashes.Get(loc.hash)
		if !exists {
			// Check existing components first
			if existingName, hasExisting := schemaCollector.existingComponents.Get(loc.hash); hasExisting {
				componentName = existingName
			} else {
				// Generate component name but don't create the component yet
				suggestedName := "Schema_" + loc.hash[:8]
				if nameCallback != nil {
					// Get all locations for this hash for the callback
					var allJsonPointers []string
					for _, otherLoc := range allLocations {
						if otherLoc.hash == loc.hash {
							allJsonPointers = append(allJsonPointers, otherLoc.jsonPointer)
						}
					}
					componentName = nameCallback(suggestedName, loc.hash, allJsonPointers, loc.schema)
				} else {
					componentName = suggestedName
				}

				// Ensure the name is unique
				componentName = ensureUniqueName(componentName, doc.Components.Schemas)
			}
			processedHashes.Set(loc.hash, componentName)
		}

		// Replace this specific location
		err := replaceInlineSchema(ctx, doc, loc.location, componentName)
		if err != nil {
			return fmt.Errorf("failed to replace inline schema for hash %s at %s: %w", loc.hash, loc.jsonPointer, err)
		}
	}

	// Step 4: Create components after all replacements are done
	// Group locations by hash to get the final schema for each component
	finalSchemas := sequencedmap.New[string, *oas3.JSONSchema[oas3.Referenceable]]()
	for hash, info := range schemaCollector.schemas.All() {
		if len(info.locations) > 1 {
			finalSchemas.Set(hash, info.schema)
		}
	}

	// Add the components to the document
	for hash, componentName := range processedHashes.All() {
		if _, hasExisting := schemaCollector.existingComponents.Get(hash); !hasExisting {
			if schema, exists := finalSchemas.Get(hash); exists {
				doc.Components.Schemas.Set(componentName, schema.ShallowCopy())
			}
		}
	}

	return nil
}

// schemaInfo holds information about a collected schema
type schemaInfo struct {
	schema       *oas3.JSONSchema[oas3.Referenceable]
	locations    []schemaLocation
	jsonPointers []string
}

// schemaLocation holds location information for a schema
type schemaLocation struct {
	parent      any             // The actual parent object (extracted from MatchFunc)
	locationCtx LocationContext // The location context with MatchFunc
}

// inlineSchemaCollector collects inline schemas and their locations
type inlineSchemaCollector struct {
	schemas            *sequencedmap.Map[string, *schemaInfo] // hash -> schema info
	existingComponents *sequencedmap.Map[string, string]      // hash -> component name
}

// collectSchema collects a schema if it's an inline complex schema
func (c *inlineSchemaCollector) collectSchema(schema *oas3.JSONSchema[oas3.Referenceable], location Locations) error {
	if schema == nil || schema.IsReference() {
		return nil
	}

	schemaObj := schema.GetLeft()
	if schemaObj == nil {
		return nil
	}

	// Skip if this is a top-level component schema
	if isTopLevelComponentSchema(location) {
		return nil
	}

	// Only collect complex schemas
	if !isComplexSchema(schemaObj) {
		return nil
	}

	// Calculate hash
	hash := hashing.Hash(schemaObj)

	// Build JSON pointer for this location
	jsonPtr := buildJSONPointer(location)

	// Get parent information for replacement
	var parent any
	var locationCtx LocationContext
	if len(location) > 0 {
		lastLoc := location[len(location)-1]

		// Extract the actual parent object from the MatchFunc
		var capturedParent any
		_ = lastLoc.ParentMatchFunc(Matcher{
			Any: func(obj any) error {
				capturedParent = obj
				return nil
			},
		})

		locationCtx = lastLoc
		parent = capturedParent
	}

	// Add to collection
	if info, exists := c.schemas.Get(hash); exists {
		info.locations = append(info.locations, schemaLocation{
			parent:      parent,
			locationCtx: locationCtx,
		})
		info.jsonPointers = append(info.jsonPointers, jsonPtr)
	} else {
		c.schemas.Set(hash, &schemaInfo{
			schema: schema,
			locations: []schemaLocation{{
				parent:      parent,
				locationCtx: locationCtx,
			}},
			jsonPointers: []string{jsonPtr},
		})
	}

	return nil
}

// isComplexSchema determines if a schema is complex enough to warrant extraction
func isComplexSchema(schema *oas3.Schema) bool {
	if schema == nil {
		return false
	}

	// Check for complex schema patterns
	if len(schema.GetAllOf()) > 0 ||
		len(schema.GetOneOf()) > 0 ||
		len(schema.GetAnyOf()) > 0 ||
		schema.GetNot() != nil {
		return true
	}

	// Check for enum
	if len(schema.GetEnum()) > 0 {
		return true
	}

	// Check for object schemas with properties
	types := schema.GetType()
	for _, schemaType := range types {
		if schemaType == oas3.SchemaTypeObject {
			return true
		}
	}

	// Check for schemas with multiple types (complex)
	// Only consider it complex if there are multiple non-null types
	nonNullTypes := 0
	for _, schemaType := range types {
		if schemaType != oas3.SchemaTypeNull {
			nonNullTypes++
		}
	}
	if nonNullTypes > 1 {
		return true
	}

	// Check for schemas with properties (even without explicit object type)
	if schema.GetProperties() != nil && schema.GetProperties().Len() > 0 {
		return true
	}

	// Check for additional complex patterns
	if schema.GetAdditionalProperties() != nil ||
		schema.GetPatternProperties() != nil && schema.GetPatternProperties().Len() > 0 ||
		schema.GetDependentSchemas() != nil && schema.GetDependentSchemas().Len() > 0 {
		return true
	}

	// Check for conditional schemas
	if schema.GetIf() != nil || schema.GetThen() != nil || schema.GetElse() != nil {
		return true
	}

	return false
}

// isTopLevelComponentSchema checks if the location represents a top-level component schema
func isTopLevelComponentSchema(location Locations) bool {
	if len(location) < 2 {
		return false
	}

	// Check if this location is within a top-level component schema
	// Pattern: /components/schemas/{name}/...
	// Location structure: [components] -> [schemas, key={name}] -> [properties/etc...]
	if len(location) >= 2 {
		if location[0].ParentField == "components" &&
			location[1].ParentField == "schemas" &&
			location[1].ParentKey != nil {
			// This is within a top-level component schema
			return true
		}
	}

	return false
}

// buildJSONPointer builds a JSON pointer string from the location context
func buildJSONPointer(location Locations) string {
	var parts []string

	for _, loc := range location {
		if loc.ParentField != "" {
			parts = append(parts, loc.ParentField)
		}
		if loc.ParentKey != nil {
			parts = append(parts, *loc.ParentKey)
		}
		if loc.ParentIndex != nil {
			parts = append(parts, strconv.Itoa(*loc.ParentIndex))
		}
	}

	return string(jsonpointer.PartsToJSONPointer(parts))
}

// ensureUniqueName ensures the component name is unique
func ensureUniqueName(baseName string, schemas *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]]) string {
	name := baseName
	counter := 1

	for {
		if _, exists := schemas.Get(name); !exists {
			return name
		}
		name = fmt.Sprintf("%s_%d", baseName, counter)
		counter++
	}
}

// replaceInlineSchema replaces a single inline schema occurrence with a reference
func replaceInlineSchema(_ context.Context, _ *OpenAPI, location schemaLocation, componentName string) error {
	// Create the reference
	ref := references.Reference("#/components/schemas/" + componentName)
	refSchema := oas3.NewJSONSchemaFromReference(ref)

	if location.parent == nil {
		return errors.New("parent is nil for location")
	}

	// Extract the underlying schema if the parent is a JSONSchema wrapper
	parent := location.parent
	if jsonSchema, ok := location.parent.(*oas3.JSONSchema[oas3.Referenceable]); ok {
		if !jsonSchema.IsLeft() {
			return errors.New("expected left side of JSONSchema but got reference")
		}
		parent = jsonSchema.GetLeft()
	}

	// Handle Reference wrapper types that contain inline objects
	if refWrapper, ok := parent.(interface {
		IsReference() bool
		GetObjectAny() any
	}); ok {
		if !refWrapper.IsReference() {
			// This is an inline object, not an actual reference
			parent = refWrapper.GetObjectAny()
		}
	}

	err := walkpkg.SetAtLocation(parent, location.locationCtx, refSchema)
	if err != nil {
		return fmt.Errorf("failed to set reference at location: %w", err)
	}

	return nil
}
