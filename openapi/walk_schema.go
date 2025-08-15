package openapi

import (
	"context"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	walkpkg "github.com/speakeasy-api/openapi/walk"
)

// walkSchema walks through a schema using the oas3 package's walking functionality
func walkSchema(ctx context.Context, schema *oas3.JSONSchema[oas3.Referenceable], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if schema == nil {
		return true
	}

	// Use the oas3 package's walking functionality
	for item := range oas3.Walk(ctx, schema) {
		// Convert the oas3 walk item to an openapi walk item
		openAPIMatchFunc := convertSchemaMatchFunc(item.Match)
		openAPILocation := convertSchemaLocation(item.Location, loc)

		if !yield(WalkItem{Match: openAPIMatchFunc, Location: openAPILocation, OpenAPI: openAPI}) {
			return false
		}
	}

	return true
}

// convertSchemaMatchFunc converts an oas3.SchemaMatchFunc to an openapi.MatchFunc
func convertSchemaMatchFunc(schemaMatchFunc oas3.SchemaMatchFunc) MatchFunc {
	return func(m Matcher) error {
		return schemaMatchFunc(oas3.SchemaMatcher{
			Schema:        m.Schema,
			Discriminator: m.Discriminator,
			XML:           m.XML,
			ExternalDocs:  m.ExternalDocs,
			Extensions:    m.Extensions,
			Any:           m.Any,
		})
	}
}

// convertSchemaLocation converts oas3 schema locations to openapi locations
func convertSchemaLocation(schemaLoc walkpkg.Locations[oas3.SchemaMatchFunc], baseLoc []LocationContext) []LocationContext {
	// Start with the base location (where the schema is located in the OpenAPI document)
	result := make([]LocationContext, len(baseLoc)+len(schemaLoc))
	copy(result, baseLoc)

	// Convert each oas3 location context to openapi location context
	for i, schemaLocCtx := range schemaLoc {
		result[len(baseLoc)+i] = LocationContext{
			Parent:      convertSchemaMatchFunc(schemaLocCtx.Parent),
			ParentField: schemaLocCtx.ParentField,
			ParentKey:   schemaLocCtx.ParentKey,
			ParentIndex: schemaLocCtx.ParentIndex,
		}
	}

	return result
}

func walkExternalDocs(ctx context.Context, externalDocs *oas3.ExternalDocumentation, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if externalDocs == nil {
		return true
	}

	// Use the oas3 package's external docs walking functionality
	for item := range oas3.WalkExternalDocs(ctx, externalDocs) {
		// Convert the oas3 walk item to an openapi walk item
		openAPIMatchFunc := convertSchemaMatchFunc(item.Match)
		openAPILocation := convertSchemaLocation(item.Location, loc)

		if !yield(WalkItem{Match: openAPIMatchFunc, Location: openAPILocation, OpenAPI: openAPI}) {
			return false
		}
	}

	return true
}
