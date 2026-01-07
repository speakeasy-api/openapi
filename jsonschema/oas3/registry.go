package oas3

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
)

// SchemaRegistry provides $id and $anchor resolution for a document.
// It maps canonical URIs and anchors to their corresponding schemas,
// enabling efficient lookup during reference resolution.
type SchemaRegistry interface {
	// RegisterSchema registers a schema with its $id and/or $anchor.
	// The parentBaseURI is used to resolve relative $id values.
	RegisterSchema(js *JSONSchema[Referenceable], parentBaseURI string) error

	// LookupByID finds a schema by its absolute $id URI.
	// Returns nil if not found.
	LookupByID(absoluteURI string) *JSONSchema[Referenceable]

	// LookupByAnchor finds a schema by anchor within a base URI scope.
	// The baseURI determines the scope in which to search for the anchor.
	// Returns nil if not found.
	LookupByAnchor(baseURI, anchor string) *JSONSchema[Referenceable]

	// GetBaseURI returns the effective base URI for a schema.
	// This is either the schema's own $id (if absolute) or inherited from ancestors.
	GetBaseURI(js *JSONSchema[Referenceable]) string

	// GetDocumentBaseURI returns the base URI of the owning document.
	GetDocumentBaseURI() string
}

// SchemaRegistryProvider is implemented by documents that can provide a schema registry.
// OpenAPI documents and standalone JSON Schema documents implement this interface.
type SchemaRegistryProvider interface {
	GetSchemaRegistry() SchemaRegistry
	GetDocumentBaseURI() string
}

// SchemaRegistryImpl is the default implementation of SchemaRegistry.
// It uses maps for O(1) lookups of schemas by $id and $anchor.
type SchemaRegistryImpl struct {
	mu sync.RWMutex

	// idToSchema maps absolute $id URIs to schemas
	idToSchema map[string]*JSONSchema[Referenceable]

	// anchorToSchema maps "baseURI#anchor" to schemas
	// The key format ensures anchors are scoped to their containing $id
	anchorToSchema map[string]*JSONSchema[Referenceable]

	// documentBase is the base URI of the owning document
	documentBase string
}

// NewSchemaRegistry creates a new empty registry with the given document base URI.
// The documentBaseURI is used as the default base for schemas without their own $id.
func NewSchemaRegistry(documentBaseURI string) *SchemaRegistryImpl {
	return &SchemaRegistryImpl{
		idToSchema:     make(map[string]*JSONSchema[Referenceable]),
		anchorToSchema: make(map[string]*JSONSchema[Referenceable]),
		documentBase:   normalizeURI(documentBaseURI),
	}
}

// RegisterSchema registers a schema with its $id and/or $anchor.
// If the schema has a relative $id, it is resolved against parentBaseURI.
// If the schema has an $anchor, it is registered under the schema's effective base URI.
// The computed effective base URI is stored directly on the schema.
// Returns an error if a duplicate $id or $anchor is detected.
func (r *SchemaRegistryImpl) RegisterSchema(js *JSONSchema[Referenceable], parentBaseURI string) error {
	if js == nil {
		return nil
	}

	schema := js.GetSchema()
	if schema == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Compute effective base URI for this schema
	effectiveBase := r.computeBaseURI(schema, parentBaseURI)

	// Store effective base URI directly on the schema (removes need for schemaToBase map)
	schema.SetEffectiveBaseURI(effectiveBase)

	// Register by $id if present
	schemaID := schema.GetID()
	if schemaID != "" {
		absoluteID := effectiveBase
		if absoluteID != "" {
			// Check for duplicate $id
			if existing, found := r.idToSchema[absoluteID]; found && existing != js {
				return fmt.Errorf("duplicate $id detected: %q is already registered", absoluteID)
			}
			r.idToSchema[absoluteID] = js
		}
	}

	// Register by $anchor if present
	anchor := schema.GetAnchor()
	if anchor != "" {
		anchorKey := r.buildAnchorKey(effectiveBase, anchor)
		// Check for duplicate $anchor
		if existing, found := r.anchorToSchema[anchorKey]; found && existing != js {
			return fmt.Errorf("duplicate $anchor detected: %q in scope %q is already registered", anchor, effectiveBase)
		}
		r.anchorToSchema[anchorKey] = js
	}

	return nil
}

// LookupByID finds a schema by its absolute $id URI.
func (r *SchemaRegistryImpl) LookupByID(absoluteURI string) *JSONSchema[Referenceable] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normalized := normalizeURI(absoluteURI)

	// Remove fragment if present for $id lookup
	if idx := strings.Index(normalized, "#"); idx != -1 {
		normalized = normalized[:idx]
	}

	return r.idToSchema[normalized]
}

// LookupByAnchor finds a schema by anchor within a base URI scope.
func (r *SchemaRegistryImpl) LookupByAnchor(baseURI, anchor string) *JSONSchema[Referenceable] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	anchorKey := r.buildAnchorKey(normalizeURI(baseURI), anchor)
	return r.anchorToSchema[anchorKey]
}

// GetBaseURI returns the effective base URI for a schema.
// It retrieves the value stored on the schema during registration.
func (r *SchemaRegistryImpl) GetBaseURI(js *JSONSchema[Referenceable]) string {
	if js == nil {
		return r.documentBase
	}

	schema := js.GetSchema()
	if schema == nil {
		return r.documentBase
	}

	// Retrieve from the schema where it was stored during registration
	if baseURI := schema.GetEffectiveBaseURI(); baseURI != "" {
		return baseURI
	}

	// Fall back to document base
	return r.documentBase
}

// GetDocumentBaseURI returns the base URI of the owning document.
func (r *SchemaRegistryImpl) GetDocumentBaseURI() string {
	return r.documentBase
}

// computeBaseURI calculates the effective base URI for a schema.
// If the schema has an absolute $id, that becomes the base.
// If the schema has a relative $id, it's resolved against the parent base.
// Otherwise, the parent base is inherited.
// Per JSON Schema spec, $id values should not contain fragments. Any fragment
// is stripped to ensure consistent lookup behavior.
func (r *SchemaRegistryImpl) computeBaseURI(schema *Schema, parentBaseURI string) string {
	schemaID := schema.GetID()
	if schemaID == "" {
		// No $id, inherit parent base
		if parentBaseURI != "" {
			return parentBaseURI
		}
		return r.documentBase
	}

	// Strip any fragment from $id - per JSON Schema spec, $id should not contain fragments
	// We strip them here for robustness and to ensure consistent lookup behavior
	if idx := strings.Index(schemaID, "#"); idx != -1 {
		schemaID = schemaID[:idx]
	}

	// Check if $id is absolute
	if IsAbsoluteURI(schemaID) {
		return normalizeURI(schemaID)
	}

	// Relative $id - resolve against parent base
	base := parentBaseURI
	if base == "" {
		base = r.documentBase
	}

	return ResolveURI(base, schemaID)
}

// buildAnchorKey creates a unique key for an anchor within a base URI scope.
func (r *SchemaRegistryImpl) buildAnchorKey(baseURI, anchor string) string {
	// Remove any existing fragment from base URI
	if idx := strings.Index(baseURI, "#"); idx != -1 {
		baseURI = baseURI[:idx]
	}

	return baseURI + "#" + anchor
}

// IsAbsoluteURI returns true if the URI has a scheme component.
func IsAbsoluteURI(uri string) bool {
	if uri == "" {
		return false
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}

	return parsed.Scheme != ""
}

// IsAnchorReference returns true for fragment references that are anchors
// (e.g., "#foo") but not JSON pointers (e.g., "#/path/to/schema").
func IsAnchorReference(ref string) bool {
	if !strings.HasPrefix(ref, "#") {
		return false
	}

	fragment := ref[1:]
	if fragment == "" {
		return false
	}

	// JSON pointers start with /
	if strings.HasPrefix(fragment, "/") {
		return false
	}

	return true
}

// ExtractAnchor extracts the anchor name from a reference.
// For "#foo" returns "foo", for "uri#foo" returns "foo".
// Returns empty string if no anchor is present.
func ExtractAnchor(ref string) string {
	idx := strings.Index(ref, "#")
	if idx == -1 {
		return ""
	}

	fragment := ref[idx+1:]

	// Ensure it's not a JSON pointer
	if strings.HasPrefix(fragment, "/") {
		return ""
	}

	return fragment
}

// ResolveURI resolves a relative URI against a base URI.
// Uses standard RFC 3986 resolution.
func ResolveURI(base, ref string) string {
	if ref == "" {
		return base
	}

	// If ref is already absolute, return as-is
	if IsAbsoluteURI(ref) {
		return normalizeURI(ref)
	}

	if base == "" {
		return normalizeURI(ref)
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return normalizeURI(ref)
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		return normalizeURI(ref)
	}

	resolved := baseURL.ResolveReference(refURL)
	return normalizeURI(resolved.String())
}

// normalizeURI normalizes a URI for consistent comparison.
// Preserves trailing slashes as they are significant per RFC 3986.
// A trailing slash indicates a directory/collection, which affects relative URI resolution.
func normalizeURI(uri string) string {
	if uri == "" {
		return ""
	}

	// Parse and re-serialize to normalize
	parsed, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	// IMPORTANT: Do NOT strip trailing slashes!
	// Per RFC 3986, trailing slashes are significant:
	// - "http://example.com/foo/" is a directory
	// - "http://example.com/foo" is a file
	// Resolving "bar/" against them gives different results:
	// - "http://example.com/foo/" + "bar/" = "http://example.com/foo/bar/"
	// - "http://example.com/foo" + "bar/" = "http://example.com/bar/"

	return parsed.String()
}
