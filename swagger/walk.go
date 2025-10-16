package swagger

import (
	"context"
	"iter"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// WalkItem represents a single item yielded by the Walk iterator.
type WalkItem struct {
	Match    MatchFunc
	Location Locations
	Swagger  *Swagger
}

// Walk returns an iterator that yields MatchFunc items for each model in the provided Swagger model.
// Users can iterate over the results using a for loop and break out at any time.
// When called with *Swagger, it walks the entire document. When called with other types,
// it walks from that specific component.
func Walk[T any](ctx context.Context, start *T) iter.Seq[WalkItem] {
	return func(yield func(WalkItem) bool) {
		if start == nil {
			return
		}
		walkFrom(ctx, start, yield)
	}
}

// walkFrom handles walking from different starting points using type switching
func walkFrom[T any](ctx context.Context, start *T, yield func(WalkItem) bool) {
	switch v := any(start).(type) {
	case *Swagger:
		walk(ctx, v, yield)
	case *oas3.JSONSchema[oas3.Concrete]:
		walkSchemaConcrete(v, []LocationContext{}, nil, yield)
	case *oas3.JSONSchema[oas3.Referenceable]:
		walkSchemaReferenceable(v, []LocationContext{}, nil, yield)
	case *ExternalDocumentation:
		walkExternalDocs(ctx, v, []LocationContext{}, nil, yield)
	case *Info:
		walkInfo(ctx, v, []LocationContext{}, nil, yield)
	case *Contact:
		yield(WalkItem{Match: getMatchFunc(v), Location: []LocationContext{}, Swagger: nil})
	case *License:
		yield(WalkItem{Match: getMatchFunc(v), Location: []LocationContext{}, Swagger: nil})
	case *Tag:
		walkTag(ctx, v, []LocationContext{}, nil, yield)
	case *Paths:
		walkPaths(ctx, v, []LocationContext{}, nil, yield)
	case *PathItem:
		walkPathItem(ctx, v, []LocationContext{}, nil, yield)
	case *Operation:
		walkOperation(ctx, v, []LocationContext{}, nil, yield)
	case *ReferencedParameter:
		walkReferencedParameter(ctx, v, []LocationContext{}, nil, yield)
	case *Parameter:
		walkParameter(ctx, v, []LocationContext{}, nil, yield)
	case *Items:
		walkItems(v, getMatchFunc(v), []LocationContext{}, nil, yield)
	case *ReferencedResponse:
		walkReferencedResponse(ctx, v, []LocationContext{}, nil, yield)
	case *Response:
		walkResponse(ctx, v, []LocationContext{}, nil, yield)
	case *Header:
		walkHeader(v, []LocationContext{}, nil, yield)
	case *SecurityRequirement:
		walkSecurityRequirement(ctx, v, []LocationContext{}, nil, yield)
	case *SecurityScheme:
		walkSecurityScheme(v, []LocationContext{}, nil, yield)
	case *extensions.Extensions:
		yield(WalkItem{Match: getMatchFunc(v), Location: []LocationContext{}, Swagger: nil})
	default:
		yield(WalkItem{Match: getMatchFunc(start), Location: []LocationContext{}, Swagger: nil})
	}
}

func walk(ctx context.Context, swagger *Swagger, yield func(WalkItem) bool) {
	swaggerMatchFunc := getMatchFunc(swagger)

	// Visit the root Swagger document first, location nil to specify the root
	if !yield(WalkItem{Match: swaggerMatchFunc, Location: nil, Swagger: swagger}) {
		return
	}

	// Visit each of the top level fields in turn populating their location context with field and any key/index information
	loc := []LocationContext{}

	if !walkInfo(ctx, &swagger.Info, append(loc, LocationContext{ParentMatchFunc: swaggerMatchFunc, ParentField: "info"}), swagger, yield) {
		return
	}

	if !walkExternalDocs(ctx, swagger.ExternalDocs, append(loc, LocationContext{ParentMatchFunc: swaggerMatchFunc, ParentField: "externalDocs"}), swagger, yield) {
		return
	}

	if !walkTags(ctx, swagger.Tags, append(loc, LocationContext{ParentMatchFunc: swaggerMatchFunc, ParentField: "tags"}), swagger, yield) {
		return
	}

	if !walkPaths(ctx, swagger.Paths, append(loc, LocationContext{ParentMatchFunc: swaggerMatchFunc, ParentField: "paths"}), swagger, yield) {
		return
	}

	if !walkDefinitions(ctx, swagger.Definitions, append(loc, LocationContext{ParentMatchFunc: swaggerMatchFunc, ParentField: "definitions"}), swagger, yield) {
		return
	}

	if !walkParameters(ctx, swagger.Parameters, append(loc, LocationContext{ParentMatchFunc: swaggerMatchFunc, ParentField: "parameters"}), swagger, yield) {
		return
	}

	if !walkGlobalResponses(ctx, swagger.Responses, append(loc, LocationContext{ParentMatchFunc: swaggerMatchFunc, ParentField: "responses"}), swagger, yield) {
		return
	}

	if !walkSecurityDefinitions(ctx, swagger.SecurityDefinitions, append(loc, LocationContext{ParentMatchFunc: swaggerMatchFunc, ParentField: "securityDefinitions"}), swagger, yield) {
		return
	}

	if !walkSecurity(ctx, swagger.Security, append(loc, LocationContext{ParentMatchFunc: swaggerMatchFunc, ParentField: "security"}), swagger, yield) {
		return
	}

	// Visit Swagger Extensions
	yield(WalkItem{Match: getMatchFunc(swagger.Extensions), Location: append(loc, LocationContext{ParentMatchFunc: swaggerMatchFunc, ParentField: ""}), Swagger: swagger})
}

func walkInfo(_ context.Context, info *Info, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if info == nil {
		return true
	}

	infoMatchFunc := getMatchFunc(info)

	if !yield(WalkItem{Match: infoMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	// Visit Contact and its Extensions
	if info.Contact != nil {
		contactMatchFunc := getMatchFunc(info.Contact)

		contactLoc := loc
		contactLoc = append(contactLoc, LocationContext{ParentMatchFunc: infoMatchFunc, ParentField: "contact"})

		if !yield(WalkItem{Match: contactMatchFunc, Location: contactLoc, Swagger: swagger}) {
			return false
		}

		if !yield(WalkItem{Match: getMatchFunc(info.Contact.Extensions), Location: append(contactLoc, LocationContext{ParentMatchFunc: contactMatchFunc, ParentField: ""}), Swagger: swagger}) {
			return false
		}
	}

	// Visit License and its Extensions
	if info.License != nil {
		licenseMatchFunc := getMatchFunc(info.License)

		licenseLoc := loc
		licenseLoc = append(licenseLoc, LocationContext{ParentMatchFunc: infoMatchFunc, ParentField: "license"})

		if !yield(WalkItem{Match: licenseMatchFunc, Location: licenseLoc, Swagger: swagger}) {
			return false
		}

		if !yield(WalkItem{Match: getMatchFunc(info.License.Extensions), Location: append(licenseLoc, LocationContext{ParentMatchFunc: licenseMatchFunc, ParentField: ""}), Swagger: swagger}) {
			return false
		}
	}

	// Visit Info Extensions
	return yield(WalkItem{Match: getMatchFunc(info.Extensions), Location: append(loc, LocationContext{ParentMatchFunc: infoMatchFunc, ParentField: ""}), Swagger: swagger})
}

func walkExternalDocs(_ context.Context, externalDocs *ExternalDocumentation, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if externalDocs == nil {
		return true
	}

	externalDocsMatchFunc := getMatchFunc(externalDocs)

	if !yield(WalkItem{Match: externalDocsMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	// Visit ExternalDocs Extensions
	return yield(WalkItem{Match: getMatchFunc(externalDocs.Extensions), Location: append(loc, LocationContext{ParentMatchFunc: externalDocsMatchFunc, ParentField: ""}), Swagger: swagger})
}

func walkTags(ctx context.Context, tags []*Tag, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if len(tags) == 0 {
		return true
	}

	// Get the last loc so we can set the parent index
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for i, tag := range tags {
		parentLoc.ParentIndex = pointer.From(i)

		if !walkTag(ctx, tag, append(loc, parentLoc), swagger, yield) {
			return false
		}
	}
	return true
}

func walkTag(ctx context.Context, tag *Tag, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if tag == nil {
		return true
	}

	tagMatchFunc := getMatchFunc(tag)

	if !yield(WalkItem{Match: tagMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	// Walk through external docs
	if !walkExternalDocs(ctx, tag.ExternalDocs, append(loc, LocationContext{ParentMatchFunc: tagMatchFunc, ParentField: "externalDocs"}), swagger, yield) {
		return false
	}

	// Visit Tag Extensions
	return yield(WalkItem{Match: getMatchFunc(tag.Extensions), Location: append(loc, LocationContext{ParentMatchFunc: tagMatchFunc, ParentField: ""}), Swagger: swagger})
}

// walkPaths walks through the paths object
func walkPaths(ctx context.Context, paths *Paths, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if paths == nil {
		return true
	}

	pathsMatchFunc := getMatchFunc(paths)

	if !yield(WalkItem{Match: pathsMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	for path, pathItem := range paths.All() {
		if !walkPathItem(ctx, pathItem, append(loc, LocationContext{ParentMatchFunc: pathsMatchFunc, ParentKey: pointer.From(path)}), swagger, yield) {
			return false
		}
	}

	// Visit Paths Extensions
	return yield(WalkItem{Match: getMatchFunc(paths.Extensions), Location: append(loc, LocationContext{ParentMatchFunc: pathsMatchFunc, ParentField: ""}), Swagger: swagger})
}

// walkPathItem walks through a path item
func walkPathItem(ctx context.Context, pathItem *PathItem, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if pathItem == nil {
		return true
	}

	pathItemMatchFunc := getMatchFunc(pathItem)

	if !yield(WalkItem{Match: pathItemMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	// Walk through parameters
	if !walkReferencedParameters(ctx, pathItem.Parameters, append(loc, LocationContext{ParentMatchFunc: pathItemMatchFunc, ParentField: "parameters"}), swagger, yield) {
		return false
	}

	// Walk through operations
	for method, operation := range pathItem.All() {
		if !walkOperation(ctx, operation, append(loc, LocationContext{ParentMatchFunc: pathItemMatchFunc, ParentKey: pointer.From(string(method))}), swagger, yield) {
			return false
		}
	}

	// Visit PathItem Extensions
	return yield(WalkItem{Match: getMatchFunc(pathItem.Extensions), Location: append(loc, LocationContext{ParentMatchFunc: pathItemMatchFunc, ParentField: ""}), Swagger: swagger})
}

// walkOperation walks through an operation
func walkOperation(ctx context.Context, operation *Operation, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if operation == nil {
		return true
	}

	operationMatchFunc := getMatchFunc(operation)

	if !yield(WalkItem{Match: operationMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	// Walk through security
	if !walkSecurity(ctx, operation.Security, append(loc, LocationContext{ParentMatchFunc: operationMatchFunc, ParentField: "security"}), swagger, yield) {
		return false
	}

	// Walk through parameters
	if !walkReferencedParameters(ctx, operation.Parameters, append(loc, LocationContext{ParentMatchFunc: operationMatchFunc, ParentField: "parameters"}), swagger, yield) {
		return false
	}

	// Walk through responses
	if !walkOperationResponses(ctx, operation.Responses, append(loc, LocationContext{ParentMatchFunc: operationMatchFunc, ParentField: "responses"}), swagger, yield) {
		return false
	}

	// Walk through external docs
	if !walkExternalDocs(ctx, operation.ExternalDocs, append(loc, LocationContext{ParentMatchFunc: operationMatchFunc, ParentField: "externalDocs"}), swagger, yield) {
		return false
	}

	// Visit Operation Extensions
	return yield(WalkItem{Match: getMatchFunc(operation.Extensions), Location: append(loc, LocationContext{ParentMatchFunc: operationMatchFunc, ParentField: ""}), Swagger: swagger})
}

// walkReferencedParameters walks through referenced parameters
func walkReferencedParameters(ctx context.Context, parameters []*ReferencedParameter, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if len(parameters) == 0 {
		return true
	}

	// Get the last loc so we can set the parent index
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for i, parameter := range parameters {
		parentLoc.ParentIndex = pointer.From(i)

		if !walkReferencedParameter(ctx, parameter, append(loc, parentLoc), swagger, yield) {
			return false
		}
	}
	return true
}

// walkReferencedParameter walks through a referenced parameter
func walkReferencedParameter(ctx context.Context, parameter *ReferencedParameter, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if parameter == nil {
		return true
	}

	referencedParameterMatchFunc := getMatchFunc(parameter)

	if !yield(WalkItem{Match: referencedParameterMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	// If it's not a reference, walk the actual Parameter
	if !parameter.IsReference() && parameter.Object != nil {
		return walkParameter(ctx, parameter.Object, loc, swagger, yield)
	}

	return true
}

// walkParameter walks through a parameter
func walkParameter(_ context.Context, parameter *Parameter, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if parameter == nil {
		return true
	}

	parameterMatchFunc := getMatchFunc(parameter)

	if !yield(WalkItem{Match: parameterMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	// Walk through schema
	if !walkSchemaReferenceable(parameter.Schema, append(loc, LocationContext{ParentMatchFunc: parameterMatchFunc, ParentField: "schema"}), swagger, yield) {
		return false
	}

	// Walk through items
	if !walkItems(parameter.Items, parameterMatchFunc, append(loc, LocationContext{ParentMatchFunc: parameterMatchFunc, ParentField: "items"}), swagger, yield) {
		return false
	}

	// Visit Parameter Extensions
	return yield(WalkItem{Match: getMatchFunc(parameter.Extensions), Location: append(loc, LocationContext{ParentMatchFunc: parameterMatchFunc, ParentField: ""}), Swagger: swagger})
}

// walkItems walks through items
func walkItems(items *Items, _ MatchFunc, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if items == nil {
		return true
	}

	itemsMatchFunc := getMatchFunc(items)

	if !yield(WalkItem{Match: itemsMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	// Walk through nested items
	if !walkItems(items.Items, itemsMatchFunc, append(loc, LocationContext{ParentMatchFunc: itemsMatchFunc, ParentField: "items"}), swagger, yield) {
		return false
	}

	// Visit Items Extensions
	return yield(WalkItem{Match: getMatchFunc(items.Extensions), Location: append(loc, LocationContext{ParentMatchFunc: itemsMatchFunc, ParentField: ""}), Swagger: swagger})
}

// walkOperationResponses walks through operation responses
func walkOperationResponses(ctx context.Context, responses *Responses, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if responses == nil {
		return true
	}

	responsesMatchFunc := getMatchFunc(responses)

	if !yield(WalkItem{Match: responsesMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	// Walk through default response
	if !walkReferencedResponse(ctx, responses.Default, append(loc, LocationContext{ParentMatchFunc: responsesMatchFunc, ParentField: "default"}), swagger, yield) {
		return false
	}

	// Walk through status code responses
	for statusCode, response := range responses.All() {
		if !walkReferencedResponse(ctx, response, append(loc, LocationContext{ParentMatchFunc: responsesMatchFunc, ParentKey: pointer.From(statusCode)}), swagger, yield) {
			return false
		}
	}

	// Visit Responses Extensions
	return yield(WalkItem{Match: getMatchFunc(responses.Extensions), Location: append(loc, LocationContext{ParentMatchFunc: responsesMatchFunc, ParentField: ""}), Swagger: swagger})
}

// walkReferencedResponse walks through a referenced response
func walkReferencedResponse(ctx context.Context, response *ReferencedResponse, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if response == nil {
		return true
	}

	referencedResponseMatchFunc := getMatchFunc(response)

	if !yield(WalkItem{Match: referencedResponseMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	// If it's not a reference, walk the actual Response
	if !response.IsReference() && response.Object != nil {
		return walkResponse(ctx, response.Object, loc, swagger, yield)
	}

	return true
}

// walkResponse walks through a response
func walkResponse(ctx context.Context, response *Response, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if response == nil {
		return true
	}

	responseMatchFunc := getMatchFunc(response)

	if !yield(WalkItem{Match: responseMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	// Walk through schema
	if !walkSchemaReferenceable(response.Schema, append(loc, LocationContext{ParentMatchFunc: responseMatchFunc, ParentField: "schema"}), swagger, yield) {
		return false
	}

	// Walk through headers
	if !walkHeaders(ctx, response.Headers, append(loc, LocationContext{ParentMatchFunc: responseMatchFunc, ParentField: "headers"}), swagger, yield) {
		return false
	}

	// Visit Response Extensions
	return yield(WalkItem{Match: getMatchFunc(response.Extensions), Location: append(loc, LocationContext{ParentMatchFunc: responseMatchFunc, ParentField: ""}), Swagger: swagger})
}

// walkHeaders walks through headers
func walkHeaders(_ context.Context, headers *sequencedmap.Map[string, *Header], loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if headers == nil || headers.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, header := range headers.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkHeader(header, append(loc, parentLoc), swagger, yield) {
			return false
		}
	}
	return true
}

// walkHeader walks through a header
func walkHeader(header *Header, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if header == nil {
		return true
	}

	headerMatchFunc := getMatchFunc(header)

	if !yield(WalkItem{Match: headerMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	// Walk through items
	if !walkItems(header.Items, headerMatchFunc, append(loc, LocationContext{ParentMatchFunc: headerMatchFunc, ParentField: "items"}), swagger, yield) {
		return false
	}

	// Visit Header Extensions
	return yield(WalkItem{Match: getMatchFunc(header.Extensions), Location: append(loc, LocationContext{ParentMatchFunc: headerMatchFunc, ParentField: ""}), Swagger: swagger})
}

// walkDefinitions walks through schema definitions
func walkDefinitions(_ context.Context, definitions *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Concrete]], loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if definitions == nil || definitions.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, schema := range definitions.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkSchemaConcrete(schema, append(loc, parentLoc), swagger, yield) {
			return false
		}
	}
	return true
}

// walkSchemaConcrete walks through a concrete schema
func walkSchemaConcrete(schema *oas3.JSONSchema[oas3.Concrete], loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if schema == nil {
		return true
	}

	schemaMatchFunc := getMatchFunc(schema)

	if !yield(WalkItem{Match: schemaMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	// For Swagger, we just visit the schema itself without walking nested schemas
	// since schema walking is specific to the JSON Schema implementation
	return true
}

// walkSchemaReferenceable walks through a referenceable schema
func walkSchemaReferenceable(schema *oas3.JSONSchema[oas3.Referenceable], loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if schema == nil {
		return true
	}

	// Convert to match func for referenceable schema
	// Note: We can't use getMatchFunc directly because we only have oas3.JSONSchema[oas3.Concrete] in the registry
	// For referenceable schemas, we just yield without a specific match function
	if !yield(WalkItem{Match: func(m Matcher) error {
		// No specific matcher for referenceable schemas
		if m.Any != nil {
			return m.Any(schema)
		}
		return nil
	}, Location: loc, Swagger: swagger}) {
		return false
	}

	return true
}

// walkParameters walks through global parameters
func walkParameters(ctx context.Context, parameters *sequencedmap.Map[string, *Parameter], loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if parameters == nil || parameters.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, parameter := range parameters.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkParameter(ctx, parameter, append(loc, parentLoc), swagger, yield) {
			return false
		}
	}
	return true
}

// walkGlobalResponses walks through global responses
func walkGlobalResponses(ctx context.Context, responses *sequencedmap.Map[string, *Response], loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if responses == nil || responses.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, response := range responses.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkResponse(ctx, response, append(loc, parentLoc), swagger, yield) {
			return false
		}
	}
	return true
}

// walkSecurityDefinitions walks through security definitions
func walkSecurityDefinitions(_ context.Context, securityDefinitions *sequencedmap.Map[string, *SecurityScheme], loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if securityDefinitions == nil || securityDefinitions.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, securityScheme := range securityDefinitions.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkSecurityScheme(securityScheme, append(loc, parentLoc), swagger, yield) {
			return false
		}
	}
	return true
}

// walkSecurityScheme walks through a security scheme
func walkSecurityScheme(securityScheme *SecurityScheme, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if securityScheme == nil {
		return true
	}

	securitySchemeMatchFunc := getMatchFunc(securityScheme)

	if !yield(WalkItem{Match: securitySchemeMatchFunc, Location: loc, Swagger: swagger}) {
		return false
	}

	// Visit SecurityScheme Extensions
	return yield(WalkItem{Match: getMatchFunc(securityScheme.Extensions), Location: append(loc, LocationContext{ParentMatchFunc: securitySchemeMatchFunc, ParentField: ""}), Swagger: swagger})
}

// walkSecurity walks through security requirements
func walkSecurity(ctx context.Context, security []*SecurityRequirement, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if len(security) == 0 {
		return true
	}

	// Get the last loc so we can set the parent index
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for i, secReq := range security {
		parentLoc.ParentIndex = pointer.From(i)

		if !walkSecurityRequirement(ctx, secReq, append(loc, parentLoc), swagger, yield) {
			return false
		}
	}
	return true
}

// walkSecurityRequirement walks through a security requirement
func walkSecurityRequirement(_ context.Context, securityRequirement *SecurityRequirement, loc []LocationContext, swagger *Swagger, yield func(WalkItem) bool) bool {
	if securityRequirement == nil {
		return true
	}

	securityRequirementMatchFunc := getMatchFunc(securityRequirement)

	return yield(WalkItem{Match: securityRequirementMatchFunc, Location: loc, Swagger: swagger})
}
