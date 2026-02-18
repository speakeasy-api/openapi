package oas3

import (
	"context"
	"errors"
	"fmt"

	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
	"go.yaml.in/yaml/v4"
)

// resolveExternalAnchorReference handles resolution of anchor references (e.g., "file.json#anchor")
// in external documents. It fetches the external document, sets up its registry, and looks up the anchor.
func (s *JSONSchema[Referenceable]) resolveExternalAnchorReference(ctx context.Context, ref references.Reference, opts references.ResolveOptions) (*references.ResolveResult[JSONSchemaReferenceable], []error, error) {
	// Create a reference to just the URI (without the anchor) to fetch the document
	uriRef := references.Reference(ref.GetURI())

	// Resolve the external document first (fetch without any fragment)
	docResult, validationErrs, err := references.Resolve(ctx, uriRef, unmarshaler, opts)
	if err != nil {
		return nil, validationErrs, err
	}

	if docResult == nil || docResult.Object == nil {
		return nil, validationErrs, fmt.Errorf("failed to resolve external document: %s", ref.GetURI())
	}

	// Set up the registry for the external document
	externalDoc := docResult.Object

	// Use $id as base URI if present in the resolved schema (JSON Schema spec)
	// The $id keyword identifies a schema resource with its canonical URI
	// and serves as the base URI for anchor lookups within that schema
	baseURI := docResult.AbsoluteDocumentPath
	if !externalDoc.IsBool() && externalDoc.GetSchema() != nil {
		if schemaID := externalDoc.GetSchema().GetID(); schemaID != "" {
			baseURI = schemaID
		}
	}

	setupRemoteSchemaRegistry(ctx, externalDoc, baseURI)

	// Now look up the anchor in the external document's registry
	anchor := ExtractAnchor(string(ref))
	if anchor == "" {
		return nil, validationErrs, fmt.Errorf("no anchor found in reference: %s", ref)
	}

	registry := externalDoc.GetSchemaRegistry()
	if registry == nil {
		return nil, validationErrs, fmt.Errorf("no registry available for external document: %s", ref.GetURI())
	}

	// Look up the anchor in the registry using the canonical base URI ($id)
	resolved := registry.LookupByAnchor(baseURI, anchor)

	// Fallback: try with fetch URL if different from canonical $id
	// This handles the case where the reference uses the retrieval URL instead of the canonical $id
	// Example: fetch https://example.com/a.json, but $id is https://cdn.example.com/canonical.json
	// A reference to "https://example.com/a.json#foo" should still resolve
	if resolved == nil && docResult.AbsoluteDocumentPath != "" && docResult.AbsoluteDocumentPath != baseURI {
		resolved = registry.LookupByAnchor(docResult.AbsoluteDocumentPath, anchor)
	}

	// Fallback: try with empty base URI
	if resolved == nil {
		resolved = registry.LookupByAnchor("", anchor)
	}

	if resolved == nil {
		return nil, validationErrs, fmt.Errorf("anchor not found in external document: %s#%s", ref.GetURI(), anchor)
	}

	absRef := utils.BuildAbsoluteReference(baseURI, "#"+anchor)
	return &references.ResolveResult[JSONSchemaReferenceable]{
		Object:               resolved,
		AbsoluteDocumentPath: baseURI,
		AbsoluteReference:    references.Reference(absRef),
		ResolvedDocument:     docResult.ResolvedDocument,
	}, validationErrs, nil
}

// resolveExternalRefWithFragment handles resolution of external references that include a JSON pointer fragment.
// For example: "http://example.com/schema.json#/$defs/foo"
// This fetches the whole document, sets up its registry (enabling $id and $anchor resolution within it),
// and then navigates to the specific fragment.
func (s *JSONSchema[Referenceable]) resolveExternalRefWithFragment(ctx context.Context, ref references.Reference, opts references.ResolveOptions) (*references.ResolveResult[JSONSchemaReferenceable], []error, error) {
	// Create a reference to just the URI (without the fragment) to fetch the whole document
	uriRef := references.Reference(ref.GetURI())

	// Resolve the external document first (fetch without any fragment)
	docResult, validationErrs, err := references.Resolve(ctx, uriRef, unmarshaler, opts)
	if err != nil {
		return nil, validationErrs, err
	}

	if docResult == nil || docResult.Object == nil {
		return nil, validationErrs, fmt.Errorf("failed to resolve external document: %s", ref.GetURI())
	}

	// Set up the registry for the external document
	// This enables $id and $anchor resolution within the fetched document
	externalDoc := docResult.Object

	// Use $id as base URI if present in the resolved schema (JSON Schema spec)
	// The $id keyword identifies a schema resource with its canonical URI
	// and serves as the base URI for relative references within that schema
	baseURI := docResult.AbsoluteDocumentPath
	if !externalDoc.IsBool() && externalDoc.GetSchema() != nil {
		if schemaID := externalDoc.GetSchema().GetID(); schemaID != "" {
			baseURI = schemaID
		}
	}

	setupRemoteSchemaRegistry(ctx, externalDoc, baseURI)

	// Now navigate to the specific fragment using JSON pointer
	jp := ref.GetJSONPointer()
	if jp == "" {
		// No fragment, return the whole document with canonical base URI
		return &references.ResolveResult[JSONSchemaReferenceable]{
			Object:               externalDoc,
			AbsoluteDocumentPath: baseURI,
			AbsoluteReference:    references.Reference(baseURI),
			ResolvedDocument:     docResult.ResolvedDocument,
		}, validationErrs, nil
	}

	// Try to navigate to the target using JSON pointer on the typed schema
	target, err := navigateJSONPointer(ctx, externalDoc, jp)
	if err != nil {
		return nil, validationErrs, fmt.Errorf("failed to navigate to fragment %s: %w", jp, err)
	}

	if target == nil {
		return nil, validationErrs, fmt.Errorf("fragment not found: %s", jp)
	}

	// IMPORTANT: Ensure the navigated target has access to the parent document's registry.
	// When navigateJSONPointer unmarshals a YAML node, the resulting schema doesn't have
	// owningDocument set, so it can't find anchors registered in the parent document.
	// We need to connect it to the parent document's registry.
	if target.GetSchema() != nil && externalDoc.GetSchema() != nil {
		// Get the registry from the external document and set it on the target
		if registry := externalDoc.GetSchemaRegistry(); registry != nil {
			target.SetSchemaRegistry(registry)
		}
		// Set the owning document so nested references can find the registry
		target.GetSchema().SetOwningDocument(externalDoc)
		// Also set the effective base URI so relative references resolve correctly
		target.GetSchema().SetEffectiveBaseURI(baseURI)
	}

	absRef := utils.BuildAbsoluteReference(baseURI, string(jp))
	return &references.ResolveResult[JSONSchemaReferenceable]{
		Object:               target,
		AbsoluteDocumentPath: baseURI,
		AbsoluteReference:    references.Reference(absRef),
		ResolvedDocument:     docResult.ResolvedDocument,
	}, validationErrs, nil
}

// navigateJSONPointer navigates a JSON pointer path within a JSONSchema.
// It delegates to jsonpointer.GetTarget which handles maps, slices, structs, and YAML nodes.
func navigateJSONPointer(ctx context.Context, schema *JSONSchemaReferenceable, jp jsonpointer.JSONPointer) (*JSONSchemaReferenceable, error) {
	if schema == nil || schema.IsBool() {
		return nil, errors.New("cannot navigate within nil or boolean schema")
	}

	jpStr := jp.String()
	if jpStr == "" || jpStr == "/" {
		return schema, nil
	}

	// Use jsonpointer.GetTarget for navigation - it handles:
	// - Maps (including sequencedmap.Map via KeyNavigable interface)
	// - Slices (for allOf, anyOf, oneOf, prefixItems)
	// - Structs (via struct tags)
	// - YAML nodes (via GetRootNode fallback)
	result, err := jsonpointer.GetTarget(schema, jp, jsonpointer.WithStructTags("key"))
	if err != nil {
		return nil, fmt.Errorf("failed to navigate JSON pointer %s: %w", jp, err)
	}

	// Check if result is already a JSONSchemaReferenceable
	if js, ok := result.(*JSONSchemaReferenceable); ok {
		return js, nil
	}

	// If we got a YAML node, we need to unmarshal it as a JSONSchema
	if yamlNode, ok := result.(*yaml.Node); ok {
		resultSchema := &JSONSchemaReferenceable{}
		_, unmarshalErr := marshaller.UnmarshalNode(ctx, "", yamlNode, resultSchema)
		if unmarshalErr != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML node at %s: %w", jp, unmarshalErr)
		}
		return resultSchema, nil
	}

	return nil, fmt.Errorf("navigation result at %s is not a JSONSchema: %T", jp, result)
}
