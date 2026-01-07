package oas3

import (
	"context"
	"strings"
	"unsafe"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/values"
)

type Referenceable interface{}

type Concrete interface{}

type JSONSchema[T Referenceable | Concrete] struct {
	values.EitherValue[Schema, core.Schema, bool, bool]

	referenceResolutionCache *references.ResolveResult[JSONSchema[Referenceable]]
	validationErrsCache      []error
	circularErrorFound       bool
	resolvedSchemaCache      *JSONSchema[Concrete] // Cache for GetResolvedSchema wrapper

	// Parent reference links - private fields to avoid serialization
	// These are set when the schema was resolved via a reference chain.
	//
	// Parent links are only set if this schema was accessed through reference resolution.
	// If you access a schema directly (e.g., by iterating through a document's components),
	// these will be nil even if the schema could be referenced elsewhere.
	//
	// Example scenarios when parent links are set:
	// - Single reference: main.yaml#/components/schemas/User -> User schema
	//   parent = reference schema, topLevelParent = reference schema
	// - Chained reference: main.yaml -> external.yaml#/User -> final User schema
	//   parent = intermediate reference, topLevelParent = original reference
	parent         *JSONSchema[Referenceable] // Immediate parent reference in the chain
	topLevelParent *JSONSchema[Referenceable] // Top-level parent (root of the reference chain)

	// enclosingSchema is the Schema that contains this JSONSchema as a field.
	// This is used during population to determine the parent's effective base URI
	// for relative $id resolution. Set during PopulateWithContext.
	enclosingSchema *Schema

	// schemaRegistry stores $id and $anchor mappings for this document.
	// Used for standalone JSON Schema documents that are not embedded in an OpenAPI document.
	// For embedded schemas, the owning OpenAPI document's registry is used instead.
	schemaRegistry SchemaRegistry

	// documentBaseURI is the base URI for this standalone JSON Schema document.
	// This is typically derived from the $id keyword or empty if not specified.
	documentBaseURI string
}

var _ references.Resolvable[JSONSchema[Concrete]] = (*JSONSchema[Referenceable])(nil)

func NewJSONSchemaFromSchema[T Referenceable | Concrete](value *Schema) *JSONSchema[T] {
	return &JSONSchema[T]{
		EitherValue: values.EitherValue[Schema, core.Schema, bool, bool]{
			Left:  value,
			Right: nil,
		},
	}
}

func NewJSONSchemaFromReference(ref references.Reference) *JSONSchema[Referenceable] {
	return &JSONSchema[Referenceable]{
		EitherValue: values.EitherValue[Schema, core.Schema, bool, bool]{
			Left: &Schema{
				Ref: pointer.From(ref),
			},
			Right: nil,
		},
	}
}

func NewJSONSchemaFromBool(value bool) *JSONSchema[Referenceable] {
	return &JSONSchema[Referenceable]{
		EitherValue: values.EitherValue[Schema, core.Schema, bool, bool]{
			Left:  nil,
			Right: pointer.From(value),
		},
	}
}

// NewReferencedScheme will create a new JSONSchema with the provided reference and and optional pre-resolved schema
func NewReferencedScheme(ctx context.Context, ref references.Reference, resolvedSchema *JSONSchema[Concrete]) *JSONSchema[Referenceable] {
	var referenceResolution *references.ResolveResult[JSONSchema[Referenceable]]

	if resolvedSchema != nil {
		referenceResolution = &references.ResolveResult[JSONSchema[Referenceable]]{
			Object: &JSONSchema[Referenceable]{
				EitherValue: values.EitherValue[Schema, core.Schema, bool, bool]{
					Left:  resolvedSchema.GetSchema(),
					Right: resolvedSchema.GetBool(),
				},
			},
		}
	}

	js := &JSONSchema[Referenceable]{
		EitherValue: values.EitherValue[Schema, core.Schema, bool, bool]{
			Left: &Schema{
				Ref: &ref,
			},
			Right: nil,
		},
		referenceResolutionCache: referenceResolution,
	}

	if resolvedSchema != nil {
		js.resolvedSchemaCache = resolvedSchema
		js.SetParent(js)
		js.SetTopLevelParent(js)
	}

	return js
}

// IsSchema returns true if the JSONSchema is a schema object.
// Returns false if the JSONSchema is a boolean value.
// A convenience method equivalent to calling IsLeft().
func (j *JSONSchema[T]) IsSchema() bool {
	if j == nil {
		return false
	}
	return j.IsLeft()
}

// GetSchema returns the schema object if the JSONSchema is a schema object.
// Returns nil if the JSONSchema is a boolean value.
// A convenience method equivalent to calling GetLeft().
func (j *JSONSchema[T]) GetSchema() *Schema {
	if j == nil {
		return nil
	}
	return j.GetLeft()
}

// IsBool returns true if the JSONSchema is a boolean value.
// Returns false if the JSONSchema is a schema object.
// A convenience method equivalent to calling IsRight().
func (j *JSONSchema[T]) IsBool() bool {
	if j == nil {
		return false
	}
	return j.IsRight()
}

// GetBool returns the boolean value if the JSONSchema is a boolean value.
// Returns nil if the JSONSchema is a schema object.
// A convenience method equivalent to calling GetRight().
func (j *JSONSchema[T]) GetBool() *bool {
	if j == nil {
		return nil
	}
	return j.GetRight()
}

// GetExtensions returns the extensions object if the JSONSchema is a schema object or an empty extensions object if the JSONSchema is a boolean value.
func (j *JSONSchema[Concrete]) GetExtensions() *extensions.Extensions {
	if j == nil || j.IsBool() {
		return extensions.New()
	}

	return j.GetSchema().GetExtensions()
}

// GetParent returns the immediate parent reference if this schema was resolved via a reference chain.
//
// Returns nil if:
// - This schema was not resolved via a reference (accessed directly)
// - This schema is the top-level reference in a chain
// - The schema was accessed by iterating through document components rather than reference resolution
//
// Example: main.yaml -> external.yaml#/User -> User schema
// The resolved User schema's GetParent() returns the external.yaml reference.
func (j *JSONSchema[T]) GetParent() *JSONSchema[Referenceable] {
	if j == nil {
		return nil
	}
	return j.parent
}

// GetTopLevelParent returns the top-level parent reference if this schema was resolved via a reference chain.
//
// Returns nil if:
// - This schema was not resolved via a reference (accessed directly)
// - This schema is already the top-level reference
// - The schema was accessed by iterating through document components rather than reference resolution
//
// Example: main.yaml -> external.yaml#/User -> chained.yaml#/User -> final User schema
// The final User schema's GetTopLevelParent() returns the original main.yaml reference.
func (j *JSONSchema[T]) GetTopLevelParent() *JSONSchema[Referenceable] {
	if j == nil {
		return nil
	}
	return j.topLevelParent
}

// SetParent sets the immediate parent reference for this schema.
// This is a public API for manually constructing reference chains.
//
// Use this when you need to manually establish parent-child relationships
// between references, typically when creating reference chains programmatically
// rather than through the normal resolution process.
func (j *JSONSchema[T]) SetParent(parent *JSONSchema[Referenceable]) {
	if j == nil {
		return
	}
	j.parent = parent
}

// SetTopLevelParent sets the top-level parent reference for this schema.
// This is a public API for manually constructing reference chains.
//
// Use this when you need to manually establish the root of a reference chain,
// typically when creating reference chains programmatically rather than
// through the normal resolution process.
func (j *JSONSchema[T]) SetTopLevelParent(topLevelParent *JSONSchema[Referenceable]) {
	if j == nil {
		return
	}
	j.topLevelParent = topLevelParent
}

// IsEqual compares two JSONSchema instances for equality.
func (j *JSONSchema[T]) IsEqual(other *JSONSchema[T]) bool {
	if j == nil && other == nil {
		return true
	}
	if j == nil || other == nil {
		return false
	}

	// Use the EitherValue's IsEqual method which will handle calling
	// IsEqual on the contained Schema or bool values appropriately
	return j.EitherValue.IsEqual(&other.EitherValue)
}

// Validate validates the JSONSchema against the JSON Schema specification.
// This is a wrapper around calling GetSchema().Validate() for schema objects.
func (j *JSONSchema[T]) Validate(ctx context.Context, opts ...validation.Option) []error {
	if j == nil {
		return []error{}
	}

	// If it's a boolean schema, no validation needed
	if j.IsBool() {
		return []error{}
	}

	// If it's a schema object, validate it
	if j.IsSchema() {
		schema := j.GetSchema()
		if schema != nil {
			// Convert opts to the expected validation options type
			// For now, we'll call without options since the Schema.Validate method
			// signature may vary
			return schema.Validate(ctx)
		}
	}

	return []error{}
}

// ConcreteToReferenceable converts a JSONSchema[Concrete] to JSONSchema[Referenceable] using unsafe pointer casting.
// This is safe because the underlying structure is identical, only the type parameter differs.
// This allows for efficient conversion without allocation when you need to walk a concrete schema
// as if it were a referenceable schema.
func ConcreteToReferenceable(concrete *JSONSchema[Concrete]) *JSONSchema[Referenceable] {
	return (*JSONSchema[Referenceable])(unsafe.Pointer(concrete)) //nolint:gosec
}

// ReferenceableToConcrete converts a JSONSchema[Referenceable] to JSONSchema[Concrete] using unsafe pointer casting.
// This is safe because the underlying structure is identical, only the type parameter differs.
// This allows for efficient conversion without allocation when you need to walk a referenceable schema
// as if it were a concrete schema.
func ReferenceableToConcrete(referenceable *JSONSchema[Referenceable]) *JSONSchema[Concrete] {
	return (*JSONSchema[Concrete])(unsafe.Pointer(referenceable)) //nolint:gosec
}

// ShallowCopy creates a shallow copy of the JSONSchema.
func (j *JSONSchema[T]) ShallowCopy() *JSONSchema[T] {
	if j == nil {
		return nil
	}

	result := &JSONSchema[T]{
		referenceResolutionCache: j.referenceResolutionCache,
		validationErrsCache:      j.validationErrsCache,
		circularErrorFound:       j.circularErrorFound,
		resolvedSchemaCache:      j.resolvedSchemaCache,
		parent:                   j.parent,
		topLevelParent:           j.topLevelParent,
		enclosingSchema:          j.enclosingSchema,
		schemaRegistry:           j.schemaRegistry,
		documentBaseURI:          j.documentBaseURI,
	}

	// Shallow copy the EitherValue contents
	if j.IsSchema() && j.GetSchema() != nil {
		result.Left = j.GetSchema().ShallowCopy()
	}
	if j.IsBool() && j.GetBool() != nil {
		rightVal := *j.GetBool()
		result.Right = &rightVal
	}

	return result
}

// GetSchemaRegistry returns the schema registry for this standalone JSON Schema document.
// The registry stores $id and $anchor mappings for efficient schema resolution.
// If the registry has not been initialized, it creates one with the document's base URI.
// This implements the SchemaRegistryProvider interface.
func (j *JSONSchema[T]) GetSchemaRegistry() SchemaRegistry {
	if j == nil {
		return nil
	}

	// Lazily initialize the registry if needed
	if j.schemaRegistry == nil {
		j.schemaRegistry = NewSchemaRegistry(j.GetDocumentBaseURI())
	}

	return j.schemaRegistry
}

// GetDocumentBaseURI returns the base URI for this standalone JSON Schema document.
// This is derived from the $id keyword of the root schema, if present.
// The returned URI is normalized and stripped of any fragment to align with registry behavior.
// This implements the SchemaRegistryProvider interface.
func (j *JSONSchema[T]) GetDocumentBaseURI() string {
	if j == nil {
		return ""
	}

	var uri string

	// If we have an explicit document base URI set, use it
	if j.documentBaseURI != "" {
		uri = j.documentBaseURI
	} else if j.IsSchema() && j.GetSchema() != nil {
		// Try to get from the root schema's $id
		uri = j.GetSchema().GetID()
	}

	if uri == "" {
		return ""
	}

	// Strip fragment and normalize to align with registry behavior
	// Per JSON Schema spec, $id should not contain fragments, but we strip for robustness
	return normalizeDocumentBaseURI(uri)
}

// normalizeDocumentBaseURI strips fragments and normalizes a URI for use as a document base.
func normalizeDocumentBaseURI(uri string) string {
	if uri == "" {
		return ""
	}

	// Strip fragment if present
	if idx := strings.Index(uri, "#"); idx != -1 {
		uri = uri[:idx]
	}

	// Use the same normalization as the registry
	return normalizeURI(uri)
}

// SetSchemaRegistry sets the schema registry for this document.
// This is primarily used during unmarshalling to set a pre-created registry.
func (j *JSONSchema[T]) SetSchemaRegistry(registry SchemaRegistry) {
	if j == nil {
		return
	}
	j.schemaRegistry = registry
}

// SetDocumentBaseURI sets the document base URI for this standalone JSON Schema.
func (j *JSONSchema[T]) SetDocumentBaseURI(uri string) {
	if j == nil {
		return
	}
	j.documentBaseURI = uri
}

// GetEnclosingSchema returns the Schema that contains this JSONSchema as a field.
// This is used during population to access the parent schema's effective base URI
// for relative $id resolution. Returns nil if not set.
func (j *JSONSchema[T]) GetEnclosingSchema() *Schema {
	if j == nil {
		return nil
	}
	return j.enclosingSchema
}

// SetEnclosingSchema sets the Schema that contains this JSONSchema as a field.
// This is used during population to establish parent-child relationships
// for effective base URI computation.
func (j *JSONSchema[T]) SetEnclosingSchema(schema *Schema) {
	if j == nil {
		return
	}
	j.enclosingSchema = schema
}

// PopulateWithContext implements the ContextAwarePopulator interface for full context-aware population.
// This method receives the owning document and propagates it to the contained Schema.
func (j *JSONSchema[T]) PopulateWithContext(source any, ctx *marshaller.PopulationContext) error {
	// If we have a parent that is a Schema, store it as enclosingSchema.
	// This is critical for relative $id resolution - children need to access parent's effective base URI.
	//
	// Note: We only set enclosingSchema here (document tree relationship).
	// The parent/topLevelParent fields are for reference chains and are set during
	// reference resolution, not population. See documentation on those fields (lines 28-41).
	if ctx != nil && ctx.Parent != nil {
		if parentSchema, ok := ctx.Parent.(*Schema); ok {
			j.enclosingSchema = parentSchema
		}
	}

	// Determine the owning document for context propagation
	// If we're not nested in any other document, this JSONSchema becomes its own owning document
	var owningDoc any
	if ctx != nil && ctx.OwningDocument != nil {
		owningDoc = ctx.OwningDocument
	} else {
		// This is a standalone JSON Schema document - use itself as the owning document
		owningDoc = j
	}

	// Create a new context with this JSONSchema as the parent for nested schemas
	// Note: The Schema's PopulateWithContext will use 'j' (this JSONSchema) as its parent reference
	childCtx := &marshaller.PopulationContext{
		Parent:         j,
		OwningDocument: owningDoc,
	}

	// Perform the standard population with context through the EitherValue
	if err := j.EitherValue.PopulateWithContext(source, childCtx); err != nil {
		return err
	}

	return nil
}
