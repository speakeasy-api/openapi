package openapi

import (
	"context"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// walkComponents walks through components
func walkComponents(ctx context.Context, components *Components, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if components == nil {
		return true
	}

	componentsMatchFunc := getMatchFunc(components)

	if !yield(WalkItem{Match: componentsMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// Walk through schemas
	if !walkComponentSchemas(ctx, components.Schemas, append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: "schemas"}), openAPI, yield) {
		return false
	}

	// Walk through responses
	if !walkComponentResponses(ctx, components.Responses, append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: "responses"}), openAPI, yield) {
		return false
	}

	// Walk through parameters
	if !walkComponentParameters(ctx, components.Parameters, append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: "parameters"}), openAPI, yield) {
		return false
	}

	// Walk through examples
	if !walkComponentExamples(ctx, components.Examples, append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: "examples"}), openAPI, yield) {
		return false
	}

	// Walk through request bodies
	if !walkComponentRequestBodies(ctx, components.RequestBodies, append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: "requestBodies"}), openAPI, yield) {
		return false
	}

	// Walk through headers
	if !walkComponentHeaders(ctx, components.Headers, append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: "headers"}), openAPI, yield) {
		return false
	}

	// Walk through security schemes
	if !walkComponentSecuritySchemes(ctx, components.SecuritySchemes, append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: "securitySchemes"}), openAPI, yield) {
		return false
	}

	// Walk through links
	if !walkComponentLinks(ctx, components.Links, append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: "links"}), openAPI, yield) {
		return false
	}

	// Walk through callbacks
	if !walkComponentCallbacks(ctx, components.Callbacks, append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: "callbacks"}), openAPI, yield) {
		return false
	}

	// Walk through path items
	if !walkComponentPathItems(ctx, components.PathItems, append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: "pathItems"}), openAPI, yield) {
		return false
	}

	// Visit Components Extensions
	return yield(WalkItem{Match: getMatchFunc(components.Extensions), Location: append(loc, LocationContext{Parent: componentsMatchFunc, ParentField: ""}), OpenAPI: openAPI})
}

// walkComponentSchemas walks through component schemas
func walkComponentSchemas(ctx context.Context, schemas *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if schemas == nil || schemas.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, schema := range schemas.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkSchema(ctx, schema, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

// walkComponentResponses walks through component responses
func walkComponentResponses(ctx context.Context, responses *sequencedmap.Map[string, *ReferencedResponse], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if responses == nil || responses.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, response := range responses.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkReferencedResponse(ctx, response, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

// walkComponentParameters walks through component parameters
func walkComponentParameters(ctx context.Context, parameters *sequencedmap.Map[string, *ReferencedParameter], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if parameters == nil || parameters.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, parameter := range parameters.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkReferencedParameter(ctx, parameter, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

// walkComponentExamples walks through component examples
func walkComponentExamples(ctx context.Context, examples *sequencedmap.Map[string, *ReferencedExample], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if examples == nil || examples.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, example := range examples.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkReferencedExample(ctx, example, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

// walkComponentRequestBodies walks through component request bodies
func walkComponentRequestBodies(ctx context.Context, requestBodies *sequencedmap.Map[string, *ReferencedRequestBody], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if requestBodies == nil || requestBodies.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, requestBody := range requestBodies.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkReferencedRequestBody(ctx, requestBody, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

// walkComponentHeaders walks through component headers
func walkComponentHeaders(ctx context.Context, headers *sequencedmap.Map[string, *ReferencedHeader], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if headers == nil || headers.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, header := range headers.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkReferencedHeader(ctx, header, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

// walkComponentSecuritySchemes walks through component security schemes
func walkComponentSecuritySchemes(ctx context.Context, securitySchemes *sequencedmap.Map[string, *ReferencedSecurityScheme], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if securitySchemes == nil || securitySchemes.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, securityScheme := range securitySchemes.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkReferencedSecurityScheme(ctx, securityScheme, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

// walkComponentLinks walks through component links
func walkComponentLinks(ctx context.Context, links *sequencedmap.Map[string, *ReferencedLink], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if links == nil || links.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, link := range links.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkReferencedLink(ctx, link, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

// walkComponentCallbacks walks through component callbacks
func walkComponentCallbacks(ctx context.Context, callbacks *sequencedmap.Map[string, *ReferencedCallback], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if callbacks == nil || callbacks.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, callback := range callbacks.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkReferencedCallback(ctx, callback, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

// walkComponentPathItems walks through component path items
func walkComponentPathItems(ctx context.Context, pathItems *sequencedmap.Map[string, *ReferencedPathItem], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if pathItems == nil || pathItems.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, pathItem := range pathItems.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkReferencedPathItem(ctx, pathItem, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}
