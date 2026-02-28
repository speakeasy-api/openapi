package oas3

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/references"
	"go.yaml.in/yaml/v4"
)

// GetRootNoder is an interface for objects that can return a YAML root node.
type GetRootNoder interface {
	GetRootNode() *yaml.Node
}

// resolveDefsReference handles special resolution for $defs references
// It uses the standard references.Resolve infrastructure but adjusts the target document for $defs resolution
//
// Key JSON Schema behavior: When a schema has its own $id, JSON pointer references like #/$defs/inner
// resolve relative to THAT schema (the one with the $id), not the root document.
// The fragment identifier "#" refers to the "current document", which for schemas with $id means
// the schema resource identified by that $id.
func (s *JSONSchema[Referenceable]) resolveDefsReference(ctx context.Context, ref references.Reference, opts references.ResolveOptions) (*references.ResolveResult[JSONSchemaReferenceable], []error, error) {
	jp := ref.GetJSONPointer()

	// Validate this is a $defs reference
	if !strings.HasPrefix(jp.String(), "/$defs/") {
		return nil, nil, fmt.Errorf("not a $defs reference: %s", ref)
	}

	// IMPORTANT: For external $defs references (with a URI), we need to fetch the whole document
	// first, set up its registry, then navigate to the fragment. This enables $id and $anchor
	// resolution within the fetched document.
	if ref.GetURI() != "" {
		return s.resolveExternalRefWithFragment(ctx, ref, opts)
	}

	// IMPORTANT: When a schema has its own $id, JSON pointer fragments (#/...) resolve within THAT schema.
	// First, try to resolve locally within the current schema if it has its own $id
	if result := s.tryResolveLocalDefs(ctx, ref, opts); result != nil {
		return result, nil, nil
	}

	// Next, try to resolve using the standard references.Resolve with the target document
	// This handles local $defs resolution (no URI), caching, and all standard resolution features
	result, validationErrs, err := references.Resolve(ctx, ref, unmarshaler, opts)
	if err == nil {
		return result, validationErrs, nil
	}

	// If standard resolution failed and we have a parent, try resolving with the parent as target
	if parent := s.GetParent(); parent != nil {
		parentOpts := opts
		parentOpts.TargetDocument = parent
		parentOpts.TargetLocation = opts.TargetLocation // Keep the same location for caching

		result, validationErrs, err := references.Resolve(ctx, ref, unmarshaler, parentOpts)
		if err == nil {
			return result, validationErrs, nil
		}
	}

	// Fallback: try JSON pointer navigation when no parent chain exists
	if s.GetParent() == nil && s.GetTopLevelParent() == nil {
		result, validationErrs, err := s.tryResolveDefsUsingJSONPointerNavigation(ctx, ref, opts)
		if err == nil && result != nil {
			return result, validationErrs, nil
		}
	}

	return nil, nil, fmt.Errorf("definition not found: %s", ref)
}

// tryResolveLocalDefs attempts to resolve a $defs reference locally within the current schema.
// This is needed when a schema has its own $id, making it a "schema resource" where
// JSON pointer fragments should resolve relative to that schema, not the root document.
func (s *JSONSchema[Referenceable]) tryResolveLocalDefs(_ context.Context, ref references.Reference, opts references.ResolveOptions) *references.ResolveResult[JSONSchemaReferenceable] {
	if s == nil || s.IsBool() {
		return nil
	}

	schema := s.GetSchema()
	if schema == nil {
		return nil
	}

	// Check if this schema has its own $id - if so, # references should resolve within this schema
	schemaID := schema.GetID()
	if schemaID == "" {
		// No $id, but check if we have an effective base URI different from the document base
		// This happens when we're inside a schema that was resolved via $id
		effectiveBase := schema.GetEffectiveBaseURI()
		registry := s.getSchemaRegistry(opts)
		if registry == nil || effectiveBase == "" || effectiveBase == registry.GetDocumentBaseURI() {
			return nil
		}
	}

	// This schema has its own $id (or effective base URI), so # references resolve within it
	jp := ref.GetJSONPointer()
	jpStr := jp.String()

	// Parse the JSON pointer to extract the $defs key
	// Expected format: /$defs/name or /$defs/name/path/to/nested
	if !strings.HasPrefix(jpStr, "/$defs/") {
		return nil
	}

	// Extract the definition key
	rest := jpStr[len("/$defs/"):]
	defKey := rest
	remainingPath := ""
	if idx := strings.Index(rest, "/"); idx != -1 {
		defKey = rest[:idx]
		remainingPath = rest[idx:]
	}

	// URL-decode the key (JSON pointer encoding)
	defKey = strings.ReplaceAll(defKey, "~1", "/")
	defKey = strings.ReplaceAll(defKey, "~0", "~")

	// Look up the definition in this schema's $defs
	defs := schema.GetDefs()
	if defs == nil {
		return nil
	}

	defSchema, ok := defs.Get(defKey)
	if !ok || defSchema == nil {
		return nil
	}

	// If there's remaining path, we need to navigate further (not supported in this simple case)
	if remainingPath != "" {
		// For now, return nil and let standard resolution handle complex paths
		return nil
	}

	// Found the definition - return it
	absRef := schema.GetEffectiveBaseURI()
	if absRef == "" {
		absRef = schemaID
	}

	absRefWithFragment := utils.BuildAbsoluteReference(absRef, string(ref.GetJSONPointer()))
	return &references.ResolveResult[JSONSchemaReferenceable]{
		Object:               defSchema,
		AbsoluteDocumentPath: absRef,
		AbsoluteReference:    references.Reference(absRefWithFragment),
	}
}

// tryResolveDefsUsingJSONPointerNavigation attempts to resolve $defs by walking up the JSON pointer structure
// This is used when there's no parent chain available
func (s *JSONSchema[Referenceable]) tryResolveDefsUsingJSONPointerNavigation(ctx context.Context, ref references.Reference, opts references.ResolveOptions) (*references.ResolveResult[JSONSchemaReferenceable], []error, error) {
	// When we don't have a parent chain, we need to find our location in the document
	// and walk up the JSON pointer chain to find parent schemas

	// Get the top-level root node from the target document
	var topLevelRootNode *yaml.Node
	if targetDoc, ok := opts.TargetDocument.(GetRootNoder); ok {
		topLevelRootNode = targetDoc.GetRootNode()
	}

	if topLevelRootNode == nil {
		return nil, nil, nil
	}

	// Get our JSON pointer location within the document using the CoreModel
	ourJSONPtr := s.GetCore().GetJSONPointer(topLevelRootNode)
	if ourJSONPtr == "" {
		return nil, nil, nil
	}

	// Walk up the parent JSON pointers
	parentJSONPtr := getParentJSONPointer(ourJSONPtr)
	for parentJSONPtr != "" {
		// Get the parent target using JSON pointer
		parentTarget, err := jsonpointer.GetTarget(opts.TargetDocument, jsonpointer.JSONPointer(parentJSONPtr), jsonpointer.WithStructTags("key"))
		if err == nil {
			parentOpts := opts
			parentOpts.TargetDocument = parentTarget
			parentOpts.TargetLocation = opts.TargetLocation // Keep the same location for caching

			result, validationErrs, err := references.Resolve(ctx, ref, unmarshaler, parentOpts)
			if err == nil {
				return result, validationErrs, nil
			}
		}

		// Move up to the next parent
		parentJSONPtr = getParentJSONPointer(parentJSONPtr)
	}

	return nil, nil, fmt.Errorf("definition not found: %s", ref)
}

// getParentJSONPointer returns the parent JSON pointer by removing the last segment
// e.g., "/properties/nested/properties/inner" -> "/properties/nested/properties"
// Returns empty string when reaching the root
func getParentJSONPointer(jsonPtr string) string {
	if jsonPtr == "" || jsonPtr == "/" {
		return ""
	}

	// Find the last slash
	lastSlash := strings.LastIndex(jsonPtr, "/")
	if lastSlash <= 0 {
		return ""
	}

	return jsonPtr[:lastSlash]
}
