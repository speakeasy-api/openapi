package oas3

import (
	"context"
	"iter"
	"reflect"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/walk"
)

// SchemaWalkItem represents a single item yielded by the WalkSchema iterator.
type SchemaWalkItem struct {
	Match    SchemaMatchFunc
	Location walk.Locations[SchemaMatchFunc]
	Schema   *JSONSchemaReferenceable // The root schema being walked
}

// SchemaMatchFunc represents a particular model in the JSON schema that can be matched.
// Pass it a SchemaMatcher with the appropriate functions populated to match the model type(s) you are interested in.
type SchemaMatchFunc func(SchemaMatcher) error

// SchemaMatcher is a struct that can be used to match specific nodes in the JSON schema.
type SchemaMatcher struct {
	Schema        func(*JSONSchemaReferenceable) error
	Discriminator func(*Discriminator) error
	XML           func(*XML) error
	ExternalDocs  func(*ExternalDocumentation) error
	Extensions    func(*extensions.Extensions) error
	Any           func(any) error // Any will be called along with the other functions above on a match of a model
}

// WalkExternalDocs returns an iterator that yields items for external documentation and its extensions.
func WalkExternalDocs(ctx context.Context, externalDocs *ExternalDocumentation) iter.Seq[SchemaWalkItem] {
	return func(yield func(SchemaWalkItem) bool) {
		if externalDocs == nil {
			return
		}
		walkExternalDocs(ctx, externalDocs, walk.Locations[SchemaMatchFunc]{}, nil, yield)
	}
}

// Walk returns an iterator that yields SchemaMatchFunc items for each model in the JSON schema.
// Users can iterate over the results using a for loop and break out at any time.
func Walk(ctx context.Context, schema *JSONSchemaReferenceable) iter.Seq[SchemaWalkItem] {
	return func(yield func(SchemaWalkItem) bool) {
		if schema == nil {
			return
		}
		walkSchema(ctx, schema, walk.Locations[SchemaMatchFunc]{}, schema, yield)
	}
}

func walkSchema(ctx context.Context, schema *JSONSchema[Referenceable], loc walk.Locations[SchemaMatchFunc], rootSchema *JSONSchema[Referenceable], yield func(SchemaWalkItem) bool) bool {
	if schema == nil {
		return true
	}

	schemaMatchFunc := getSchemaMatchFunc(schema)

	// Visit self schema first
	if !yield(SchemaWalkItem{Match: schemaMatchFunc, Location: loc, Schema: rootSchema}) {
		return false
	}

	if schema.IsLeft() {
		js := schema.Left

		// Walk through allOf schemas
		for i, schema := range js.AllOf {
			if !walkSchema(ctx, schema, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "allOf", ParentIndex: pointer.From(i)}), rootSchema, yield) {
				return false
			}
		}

		// Walk through oneOf schemas
		for i, schema := range js.OneOf {
			if !walkSchema(ctx, schema, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "oneOf", ParentIndex: pointer.From(i)}), rootSchema, yield) {
				return false
			}
		}

		// Walk through anyOf schemas
		for i, schema := range js.AnyOf {
			if !walkSchema(ctx, schema, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "anyOf", ParentIndex: pointer.From(i)}), rootSchema, yield) {
				return false
			}
		}

		// Visit discriminator
		if js.Discriminator != nil {
			discriminatorMatchFunc := getSchemaMatchFunc(js.Discriminator)

			discriminatorLoc := loc
			discriminatorLoc = append(discriminatorLoc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "discriminator"})

			if !yield(SchemaWalkItem{Match: discriminatorMatchFunc, Location: discriminatorLoc, Schema: rootSchema}) {
				return false
			}

			// Visit discriminator Extensions
			if !yield(SchemaWalkItem{Match: getSchemaMatchFunc(js.Discriminator.Extensions), Location: append(discriminatorLoc, walk.LocationContext[SchemaMatchFunc]{Parent: discriminatorMatchFunc, ParentField: ""}), Schema: rootSchema}) {
				return false
			}
		}

		// Walk through prefixItems schemas
		for i, schema := range js.PrefixItems {
			if !walkSchema(ctx, schema, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "prefixItems", ParentIndex: pointer.From(i)}), rootSchema, yield) {
				return false
			}
		}

		// Visit contains schema
		if !walkSchema(ctx, js.Contains, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "contains"}), rootSchema, yield) {
			return false
		}

		// Visit if schema
		if !walkSchema(ctx, js.If, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "if"}), rootSchema, yield) {
			return false
		}

		// Visit then schema
		if !walkSchema(ctx, js.Then, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "then"}), rootSchema, yield) {
			return false
		}

		// Visit else schema
		if !walkSchema(ctx, js.Else, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "else"}), rootSchema, yield) {
			return false
		}

		// Walk through dependentSchemas schemas
		for property, schema := range js.DependentSchemas.All() {
			if !walkSchema(ctx, schema, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "dependentSchemas", ParentKey: pointer.From(property)}), rootSchema, yield) {
				return false
			}
		}

		// Walk through patternProperties schemas
		for property, schema := range js.PatternProperties.All() {
			if !walkSchema(ctx, schema, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "patternProperties", ParentKey: pointer.From(property)}), rootSchema, yield) {
				return false
			}
		}

		// Visit propertyNames schema
		if !walkSchema(ctx, js.PropertyNames, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "propertyNames"}), rootSchema, yield) {
			return false
		}

		// Visit unevaluatedItems schema
		if !walkSchema(ctx, js.UnevaluatedItems, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "unevaluatedItems"}), rootSchema, yield) {
			return false
		}

		// Visit unevaluatedProperties schema
		if !walkSchema(ctx, js.UnevaluatedProperties, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "unevaluatedProperties"}), rootSchema, yield) {
			return false
		}

		// Visit items schema
		if !walkSchema(ctx, js.Items, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "items"}), rootSchema, yield) {
			return false
		}

		// Visit not schema
		if !walkSchema(ctx, js.Not, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "not"}), rootSchema, yield) {
			return false
		}

		// Walk through properties schemas
		for property, schema := range js.Properties.All() {
			if !walkSchema(ctx, schema, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "properties", ParentKey: pointer.From(property)}), rootSchema, yield) {
				return false
			}
		}

		// Walk through $defs schemas
		for property, schema := range js.Defs.All() {
			if !walkSchema(ctx, schema, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "$defs", ParentKey: pointer.From(property)}), rootSchema, yield) {
				return false
			}
		}

		// Visit additionalProperties schema
		if !walkSchema(ctx, js.AdditionalProperties, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "additionalProperties"}), rootSchema, yield) {
			return false
		}

		// Visit externalDocs
		if !walkExternalDocs(ctx, js.ExternalDocs, append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "externalDocs"}), rootSchema, yield) {
			return false
		}

		if js.XML != nil {
			xmlMatchFunc := getSchemaMatchFunc(js.XML)

			xmlLoc := loc
			xmlLoc = append(xmlLoc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: "xml"})

			if !yield(SchemaWalkItem{Match: xmlMatchFunc, Location: xmlLoc, Schema: rootSchema}) {
				return false
			}

			// Visit xml Extensions
			if !yield(SchemaWalkItem{Match: getSchemaMatchFunc(js.XML.Extensions), Location: append(xmlLoc, walk.LocationContext[SchemaMatchFunc]{Parent: xmlMatchFunc, ParentField: ""}), Schema: rootSchema}) {
				return false
			}
		}

		// Visit extensions
		if !yield(SchemaWalkItem{Match: getSchemaMatchFunc(js.Extensions), Location: append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: schemaMatchFunc, ParentField: ""}), Schema: rootSchema}) {
			return false
		}
	}

	return true
}

func walkExternalDocs(_ context.Context, externalDocs *ExternalDocumentation, loc walk.Locations[SchemaMatchFunc], rootSchema *JSONSchema[Referenceable], yield func(SchemaWalkItem) bool) bool {
	if externalDocs == nil {
		return true
	}

	externalDocsMatchFunc := getSchemaMatchFunc(externalDocs)

	if !yield(SchemaWalkItem{Match: externalDocsMatchFunc, Location: loc, Schema: rootSchema}) {
		return false
	}

	return yield(SchemaWalkItem{Match: getSchemaMatchFunc(externalDocs.Extensions), Location: append(loc, walk.LocationContext[SchemaMatchFunc]{Parent: externalDocsMatchFunc, ParentField: ""}), Schema: rootSchema})
}

type schemaMatchHandler[T any] struct {
	GetSpecific func(m SchemaMatcher) func(*T) error
}

var schemaMatchRegistry = map[reflect.Type]any{
	reflect.TypeOf((*JSONSchema[Referenceable])(nil)): schemaMatchHandler[JSONSchema[Referenceable]]{
		GetSpecific: func(m SchemaMatcher) func(*JSONSchema[Referenceable]) error { return m.Schema },
	},
	reflect.TypeOf((*Discriminator)(nil)): schemaMatchHandler[Discriminator]{
		GetSpecific: func(m SchemaMatcher) func(*Discriminator) error { return m.Discriminator },
	},
	reflect.TypeOf((*XML)(nil)): schemaMatchHandler[XML]{
		GetSpecific: func(m SchemaMatcher) func(*XML) error { return m.XML },
	},
	reflect.TypeOf((*ExternalDocumentation)(nil)): schemaMatchHandler[ExternalDocumentation]{
		GetSpecific: func(m SchemaMatcher) func(*ExternalDocumentation) error { return m.ExternalDocs },
	},
	reflect.TypeOf((*extensions.Extensions)(nil)): schemaMatchHandler[extensions.Extensions]{
		GetSpecific: func(m SchemaMatcher) func(*extensions.Extensions) error { return m.Extensions },
	},
}

func getSchemaMatchFunc[T any](target *T) SchemaMatchFunc {
	t := reflect.TypeOf(target)

	h, ok := schemaMatchRegistry[t]
	if !ok {
		// For unknown types, just use the Any matcher
		return func(m SchemaMatcher) error {
			if m.Any != nil {
				return m.Any(target)
			}
			return nil
		}
	}

	handler := h.(schemaMatchHandler[T])
	return func(m SchemaMatcher) error {
		if m.Any != nil {
			if err := m.Any(target); err != nil {
				return err
			}
		}
		if specific := handler.GetSpecific(m); specific != nil {
			return specific(target)
		}
		return nil
	}
}
