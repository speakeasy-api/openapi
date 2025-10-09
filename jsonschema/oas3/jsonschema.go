package oas3

import (
	"context"
	"unsafe"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3/core"
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
// A convenience method equivalent to calling IsSchema().
func (j *JSONSchema[T]) IsSchema() bool {
	if j == nil {
		return false
	}
	return j.IsLeft()
}

// GetSchema returns the schema object if the JSONSchema is a schema object.
// Returns nil if the JSONSchema is a boolean value.
// A convenience method equivalent to calling GetSchema().
func (j *JSONSchema[T]) GetSchema() *Schema {
	if j == nil {
		return nil
	}
	return j.GetLeft()
}

// IsBool returns true if the JSONSchema is a boolean value.
// Returns false if the JSONSchema is a schema object.
// A convenience method equivalent to calling IsBool().
func (j *JSONSchema[T]) IsBool() bool {
	if j == nil {
		return false
	}
	return j.IsRight()
}

// GetBool returns the boolean value if the JSONSchema is a boolean value.
// Returns nil if the JSONSchema is a schema object.
// A convenience method equivalent to calling GetBool().
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

// PopulateWithParent implements the ParentAwarePopulator interface to establish parent relationships during population
func (j *JSONSchema[T]) PopulateWithParent(source any, parent any) error {
	// If we have a parent that is also a JSONSchema, establish the parent relationship
	if parent != nil {
		if parentSchema, ok := parent.(*Schema); ok {
			j.SetParent(parentSchema.GetParent())
			// If the parent has a top-level parent, inherit it; otherwise, the parent is the top-level
			if parentSchema.GetParent().GetTopLevelParent() != nil {
				j.SetTopLevelParent(parentSchema.GetParent().GetTopLevelParent())
			} else {
				j.SetTopLevelParent(parentSchema.GetParent())
			}
		}
	}

	// First, perform the standard population
	if err := j.EitherValue.PopulateWithParent(source, j); err != nil {
		return err
	}

	return nil
}
