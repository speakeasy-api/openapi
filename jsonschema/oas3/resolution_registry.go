package oas3

import (
	"context"

	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/references"
)

// tryResolveViaRegistry attempts to resolve a reference using the schema registry.
// This handles $id URIs and $anchor references within the same document.
// Returns nil if the reference cannot be resolved via the registry.
func (s *JSONSchema[Referenceable]) tryResolveViaRegistry(ctx context.Context, ref references.Reference, opts references.ResolveOptions) *references.ResolveResult[JSONSchemaReferenceable] {
	// Get the schema registry from the owning document
	registry := s.getSchemaRegistry(opts)
	if registry == nil {
		return nil
	}

	refStr := string(ref)
	if refStr == "" {
		return nil
	}

	// Get the effective base URI for this schema
	baseURI := s.getEffectiveBaseURI(opts)

	// Check if this is an anchor reference (e.g., "#foo" or "otherfile.json#foo" where "foo" doesn't start with "/")
	if anchor := ExtractAnchor(refStr); anchor != "" {
		// Determine the base URI for the anchor lookup
		anchorBase := baseURI
		if uri := ref.GetURI(); uri != "" {
			// If the reference has a URI part, resolve it against the base
			anchorBase = ResolveURI(baseURI, uri)
		}

		if resolved := registry.LookupByAnchor(anchorBase, anchor); resolved != nil {
			absRef := utils.BuildAbsoluteReference(anchorBase, "#"+anchor)
			return &references.ResolveResult[JSONSchemaReferenceable]{
				Object:               resolved,
				AbsoluteDocumentPath: anchorBase,
				AbsoluteReference:    references.Reference(absRef),
				ResolvedDocument:     opts.TargetDocument,
			}
		}

		// For local anchor references (no URI part), also try with empty base
		// This handles the case where anchors were registered without a document base URI
		if ref.GetURI() == "" && anchorBase != "" {
			if resolved := registry.LookupByAnchor("", anchor); resolved != nil {
				absRef := "#" + anchor
				return &references.ResolveResult[JSONSchemaReferenceable]{
					Object:               resolved,
					AbsoluteDocumentPath: "",
					AbsoluteReference:    references.Reference(absRef),
					ResolvedDocument:     opts.TargetDocument,
				}
			}
		}

		// Also try with document base URI if different from computed anchorBase
		docBase := registry.GetDocumentBaseURI()
		if docBase != "" && docBase != anchorBase {
			if resolved := registry.LookupByAnchor(docBase, anchor); resolved != nil {
				absRef := utils.BuildAbsoluteReference(docBase, "#"+anchor)
				return &references.ResolveResult[JSONSchemaReferenceable]{
					Object:               resolved,
					AbsoluteDocumentPath: docBase,
					AbsoluteReference:    references.Reference(absRef),
					ResolvedDocument:     opts.TargetDocument,
				}
			}
		}
	}

	// Check if this is an $id reference (with or without JSON pointer)
	// If the URI part matches an $id in the registry, we resolve within that schema
	if ref.GetURI() != "" {
		uri := ref.GetURI()
		jp := ref.GetJSONPointer()
		var resolvedSchema *JSONSchemaReferenceable
		var absoluteReference string

		// Try as absolute URI first
		if IsAbsoluteURI(uri) {
			if resolved := registry.LookupByID(uri); resolved != nil {
				resolvedSchema = resolved
				absoluteReference = uri
			}
		}

		// Try as relative URI resolved against base
		if resolvedSchema == nil && baseURI != "" {
			resolvedURI := ResolveURI(baseURI, uri)
			if resolved := registry.LookupByID(resolvedURI); resolved != nil {
				resolvedSchema = resolved
				absoluteReference = resolvedURI
			}
		}

		// Also try with document base URI
		if resolvedSchema == nil {
			docBase := registry.GetDocumentBaseURI()
			if docBase != "" && docBase != baseURI {
				resolvedURI := ResolveURI(docBase, uri)
				if resolved := registry.LookupByID(resolvedURI); resolved != nil {
					resolvedSchema = resolved
					absoluteReference = resolvedURI
				}
			}
		}

		// If we found a schema via $id lookup
		if resolvedSchema != nil {
			// If there's no JSON pointer, return the schema directly
			if jp == "" {
				return &references.ResolveResult[JSONSchemaReferenceable]{
					Object:               resolvedSchema,
					AbsoluteDocumentPath: absoluteReference,
					AbsoluteReference:    references.Reference(absoluteReference),
					ResolvedDocument:     opts.TargetDocument,
				}
			}

			// There's a JSON pointer - navigate within the found schema
			target, err := navigateJSONPointer(ctx, resolvedSchema, jp)
			if err == nil && target != nil {
				absRef := utils.BuildAbsoluteReference(absoluteReference, string(jp))
				return &references.ResolveResult[JSONSchemaReferenceable]{
					Object:               target,
					AbsoluteDocumentPath: absoluteReference,
					AbsoluteReference:    references.Reference(absRef),
					ResolvedDocument:     opts.TargetDocument,
				}
			}
			// If navigation failed, fall through to external resolution
		}
	}

	return nil
}

// getSchemaRegistry attempts to get a schema registry from the schema's owning document or resolve options.
// IMPORTANT: We check the schema's own registry first because when resolving a reference within a remote
// document, we want to use the registry from THAT document, not the root document of the resolution chain.
func (s *JSONSchema[Referenceable]) getSchemaRegistry(opts references.ResolveOptions) SchemaRegistry {
	// First, try from the schema's owning document
	// This is the most specific registry - the one for the document containing this schema
	if s.GetSchema() != nil {
		if registry := s.GetSchema().GetSchemaRegistry(); registry != nil {
			return registry
		}
	}

	// Try from target document
	if opts.TargetDocument != nil {
		if provider, ok := opts.TargetDocument.(SchemaRegistryProvider); ok {
			if registry := provider.GetSchemaRegistry(); registry != nil {
				return registry
			}
		}
	}

	// Fall back to root document
	if opts.RootDocument != nil {
		if provider, ok := opts.RootDocument.(SchemaRegistryProvider); ok {
			return provider.GetSchemaRegistry()
		}
	}

	return nil
}

// getEffectiveBaseURI returns the effective base URI for this schema.
func (s *JSONSchema[Referenceable]) getEffectiveBaseURI(opts references.ResolveOptions) string {
	// First, check if the schema has its own effective base URI
	if schema := s.GetSchema(); schema != nil {
		if baseURI := schema.GetEffectiveBaseURI(); baseURI != "" {
			return baseURI
		}
	}

	// Check if we have a cached absolute reference
	if s.referenceResolutionCache != nil && s.referenceResolutionCache.AbsoluteDocumentPath != "" {
		return s.referenceResolutionCache.AbsoluteDocumentPath
	}

	// Fall back to target location
	if opts.TargetLocation != "" {
		return opts.TargetLocation
	}

	// Try to get from the root document
	if opts.RootDocument != nil {
		if provider, ok := opts.RootDocument.(SchemaRegistryProvider); ok {
			return provider.GetDocumentBaseURI()
		}
	}

	return ""
}

// setupRemoteSchemaRegistry sets up the schema registry for a remotely fetched schema.
// This enables $id and $anchor resolution within the fetched document by:
// 1. Setting the document base URI on the root schema
// 2. Creating a registry for the remote document
// 3. Walking all schemas to register their $id and $anchor values
func setupRemoteSchemaRegistry(ctx context.Context, schema *JSONSchemaReferenceable, documentBaseURI string) {
	if schema == nil || schema.IsBool() {
		return
	}

	// Check if this schema already has a properly configured registry
	// This happens when we're resolving a fragment from a document that was already set up.
	// We check if the registry has a non-empty document base URI, which indicates it was
	// explicitly configured (not just lazily created during unmarshalling).
	if schema.GetSchema() != nil {
		if existingRegistry := schema.GetSchema().GetSchemaRegistry(); existingRegistry != nil {
			if existingRegistry.GetDocumentBaseURI() != "" {
				// Already has a properly configured registry - don't overwrite
				// This preserves anchors and $ids from the whole document
				return
			}
		}
	}

	// Set the document base URI on the root schema
	schema.SetDocumentBaseURI(documentBaseURI)

	// Get or create the registry for this remote schema
	registry := schema.GetSchemaRegistry()
	if registry == nil {
		return
	}

	// Walk the schema tree to register all $id and $anchor values
	registerSchemaInRegistry(ctx, schema, registry, documentBaseURI)
}

// registerSchemaInRegistry walks a schema tree and registers all $id and $anchor values.
func registerSchemaInRegistry(ctx context.Context, rootSchema *JSONSchemaReferenceable, registry SchemaRegistry, documentBaseURI string) {
	if rootSchema == nil || registry == nil {
		return
	}

	// Walk all schemas and register them with the registry
	for item := range Walk(ctx, rootSchema) {
		_ = item.Match(SchemaMatcher{
			Schema: func(js *JSONSchemaReferenceable) error {
				if js == nil || js.IsBool() {
					return nil
				}

				schema := js.GetSchema()
				if schema == nil {
					return nil
				}

				// Compute parent base URI for relative $id resolution
				// Use enclosingSchema (document tree parent) not GetParent (reference chain parent)
				parentBaseURI := documentBaseURI
				if enclosingSchema := js.GetEnclosingSchema(); enclosingSchema != nil {
					if parentEffectiveURI := enclosingSchema.GetEffectiveBaseURI(); parentEffectiveURI != "" {
						parentBaseURI = parentEffectiveURI
					}
				}

				// Register the schema with the registry
				// Note: RegisterSchema() already sets effectiveBaseURI on the schema,
				// so we don't need to call SetEffectiveBaseURI() again here.
				if err := registry.RegisterSchema(js, parentBaseURI); err != nil {
					return nil //nolint:nilerr // Continue walk on error - registration is best-effort
				}

				// Set the owning document to the root schema so nested schemas can find the registry
				schema.SetOwningDocument(rootSchema)

				return nil
			},
		})
	}
}
