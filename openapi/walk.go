package openapi

import (
	"context"
	"iter"

	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// WalkItem represents a single item yielded by the Walk iterator.
type WalkItem struct {
	Match    MatchFunc
	Location Locations
	OpenAPI  *OpenAPI
}

// Walk returns an iterator that yields MatchFunc items for each model in the OpenAPI document.
// Users can iterate over the results using a for loop and break out at any time.
func Walk(ctx context.Context, openAPI *OpenAPI) iter.Seq[WalkItem] {
	return func(yield func(WalkItem) bool) {
		if openAPI == nil {
			return
		}
		walk(ctx, openAPI, yield)
	}
}

func walk(ctx context.Context, openAPI *OpenAPI, yield func(WalkItem) bool) {
	openAPIMatchFunc := geMatchFunc(openAPI)

	// Visit the root OpenAPI document first, location nil to specify the root
	if !yield(WalkItem{Match: openAPIMatchFunc, Location: nil, OpenAPI: openAPI}) {
		return
	}

	// Visit each of the top level fields in turn populating their location context with field and any key/index information
	loc := []LocationContext{}

	if !walkInfo(ctx, &openAPI.Info, append(loc, LocationContext{Parent: openAPIMatchFunc, ParentField: "info"}), openAPI, yield) {
		return
	}

	if !walkExternalDocs(ctx, openAPI.ExternalDocs, append(loc, LocationContext{Parent: openAPIMatchFunc, ParentField: "externalDocs"}), openAPI, yield) {
		return
	}

	if !walkTags(ctx, openAPI.Tags, append(loc, LocationContext{Parent: openAPIMatchFunc, ParentField: "tags"}), openAPI, yield) {
		return
	}

	if !walkServers(ctx, openAPI.Servers, append(loc, LocationContext{Parent: openAPIMatchFunc, ParentField: "servers"}), openAPI, yield) {
		return
	}

	if !walkSecurity(ctx, openAPI.Security, append(loc, LocationContext{Parent: openAPIMatchFunc, ParentField: "security"}), openAPI, yield) {
		return
	}

	if !walkPaths(ctx, openAPI.Paths, append(loc, LocationContext{Parent: openAPIMatchFunc, ParentField: "paths"}), openAPI, yield) {
		return
	}

	if !walkWebhooks(ctx, openAPI.Webhooks, append(loc, LocationContext{Parent: openAPIMatchFunc, ParentField: "webhooks"}), openAPI, yield) {
		return
	}

	if !walkComponents(ctx, openAPI.Components, append(loc, LocationContext{Parent: openAPIMatchFunc, ParentField: "components"}), openAPI, yield) {
		return
	}

	// Visit OpenAPI Extensions
	yield(WalkItem{Match: geMatchFunc(openAPI.Extensions), Location: append(loc, LocationContext{Parent: openAPIMatchFunc, ParentField: ""}), OpenAPI: openAPI})
}

func walkInfo(_ context.Context, info *Info, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if info == nil {
		return true
	}

	infoMatchFunc := geMatchFunc(info)

	if !yield(WalkItem{Match: infoMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// Visit Contact and its Extensions
	if info.Contact != nil {
		contactMatchFunc := geMatchFunc(info.Contact)

		contactLoc := loc
		contactLoc = append(contactLoc, LocationContext{Parent: infoMatchFunc, ParentField: "contact"})

		if !yield(WalkItem{Match: contactMatchFunc, Location: contactLoc, OpenAPI: openAPI}) {
			return false
		}

		if !yield(WalkItem{Match: geMatchFunc(info.Contact.Extensions), Location: append(contactLoc, LocationContext{Parent: contactMatchFunc, ParentField: ""}), OpenAPI: openAPI}) {
			return false
		}
	}

	// Visit License and its Extensions
	if info.License != nil {
		licenseMatchFunc := geMatchFunc(info.License)

		licenseLoc := loc
		licenseLoc = append(licenseLoc, LocationContext{Parent: infoMatchFunc, ParentField: "license"})

		if !yield(WalkItem{Match: licenseMatchFunc, Location: licenseLoc, OpenAPI: openAPI}) {
			return false
		}

		if !yield(WalkItem{Match: geMatchFunc(info.License.Extensions), Location: append(licenseLoc, LocationContext{Parent: licenseMatchFunc, ParentField: ""}), OpenAPI: openAPI}) {
			return false
		}
	}

	// Visit Info Extensions
	return yield(WalkItem{Match: geMatchFunc(info.Extensions), Location: append(loc, LocationContext{Parent: infoMatchFunc, ParentField: ""}), OpenAPI: openAPI})
}

// walkPaths walks through the paths object
func walkPaths(ctx context.Context, paths *Paths, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if paths == nil {
		return true
	}

	pathsMatchFunc := geMatchFunc(paths)

	if !yield(WalkItem{Match: pathsMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	for path, pathItem := range paths.All() {
		if !walkReferencedPathItem(ctx, pathItem, append(loc, LocationContext{Parent: pathsMatchFunc, ParentKey: pointer.From(path)}), openAPI, yield) {
			return false
		}
	}

	// Visit Paths Extensions
	return yield(WalkItem{Match: geMatchFunc(paths.Extensions), Location: append(loc, LocationContext{Parent: pathsMatchFunc, ParentField: ""}), OpenAPI: openAPI})
}

// walkReferencedPathItem walks through a referenced path item
func walkReferencedPathItem(ctx context.Context, pathItem *ReferencedPathItem, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if pathItem == nil {
		return true
	}

	referencedPathItemMatchFunc := geMatchFunc(pathItem)

	if !yield(WalkItem{Match: referencedPathItemMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// If it's not a reference, walk the actual PathItem
	if !pathItem.IsReference() && pathItem.Object != nil {
		return walkPathItem(ctx, pathItem.Object, referencedPathItemMatchFunc, loc, openAPI, yield)
	}

	return true
}

// walkPathItem walks through a path item
func walkPathItem(ctx context.Context, pathItem *PathItem, parent MatchFunc, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if pathItem == nil {
		return true
	}

	// Walk through servers
	if !walkServers(ctx, pathItem.Servers, append(loc, LocationContext{Parent: parent, ParentField: "servers"}), openAPI, yield) {
		return false
	}

	// Walk through parameters
	if !walkReferencedParameters(ctx, pathItem.Parameters, append(loc, LocationContext{Parent: parent, ParentField: "parameters"}), openAPI, yield) {
		return false
	}

	// Walk through operations
	for method, operation := range pathItem.All() {
		if !walkOperation(ctx, operation, append(loc, LocationContext{Parent: parent, ParentKey: pointer.From(string(method))}), openAPI, yield) {
			return false
		}
	}

	// Visit PathItem Extensions
	return yield(WalkItem{Match: geMatchFunc(pathItem.Extensions), Location: append(loc, LocationContext{Parent: parent, ParentField: ""}), OpenAPI: openAPI})
}

// walkOperation walks through an operation
func walkOperation(ctx context.Context, operation *Operation, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if operation == nil {
		return true
	}

	operationMatchFunc := geMatchFunc(operation)

	if !yield(WalkItem{Match: operationMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// Walk through servers
	if !walkServers(ctx, operation.Servers, append(loc, LocationContext{Parent: operationMatchFunc, ParentField: "servers"}), openAPI, yield) {
		return false
	}

	// Walk through security
	if !walkSecurity(ctx, operation.Security, append(loc, LocationContext{Parent: operationMatchFunc, ParentField: "security"}), openAPI, yield) {
		return false
	}

	// Walk through parameters
	if !walkReferencedParameters(ctx, operation.Parameters, append(loc, LocationContext{Parent: operationMatchFunc, ParentField: "parameters"}), openAPI, yield) {
		return false
	}

	// Walk through request body
	if !walkReferencedRequestBody(ctx, operation.RequestBody, append(loc, LocationContext{Parent: operationMatchFunc, ParentField: "requestBody"}), openAPI, yield) {
		return false
	}

	// Walk through responses
	if !walkResponses(ctx, operation.Responses, append(loc, LocationContext{Parent: operationMatchFunc, ParentField: "responses"}), openAPI, yield) {
		return false
	}

	// Walk through callbacks
	if !walkReferencedCallbacks(ctx, operation.Callbacks, append(loc, LocationContext{Parent: operationMatchFunc, ParentField: "callbacks"}), openAPI, yield) {
		return false
	}

	// Walk through external docs
	if !walkExternalDocs(ctx, operation.ExternalDocs, append(loc, LocationContext{Parent: operationMatchFunc, ParentField: "externalDocs"}), openAPI, yield) {
		return false
	}

	// Visit Operation Extensions
	return yield(WalkItem{Match: geMatchFunc(operation.Extensions), Location: append(loc, LocationContext{Parent: operationMatchFunc, ParentField: ""}), OpenAPI: openAPI})
}

// walkReferencedParameters walks through referenced parameters
func walkReferencedParameters(ctx context.Context, parameters []*ReferencedParameter, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if len(parameters) == 0 {
		return true
	}

	// Get the last loc so we can set the parent index
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for i, parameter := range parameters {
		parentLoc.ParentIndex = pointer.From(i)

		if !walkReferencedParameter(ctx, parameter, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

// walkReferencedParameter walks through a referenced parameter
func walkReferencedParameter(ctx context.Context, parameter *ReferencedParameter, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if parameter == nil {
		return true
	}

	referencedParameterMatchFunc := geMatchFunc(parameter)

	if !yield(WalkItem{Match: referencedParameterMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// If it's not a reference, walk the actual Parameter
	if !parameter.IsReference() && parameter.Object != nil {
		return walkParameter(ctx, parameter.Object, referencedParameterMatchFunc, loc, openAPI, yield)
	}

	return true
}

// walkParameter walks through a parameter
func walkParameter(ctx context.Context, parameter *Parameter, parent MatchFunc, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if parameter == nil {
		return true
	}

	// Walk through schema
	if !walkSchema(ctx, parameter.Schema, append(loc, LocationContext{Parent: parent, ParentField: "schema"}), openAPI, yield) {
		return false
	}

	// Walk through content
	if !walkMediaTypes(ctx, parameter.Content, append(loc, LocationContext{Parent: parent, ParentField: "content"}), openAPI, yield) {
		return false
	}

	// Walk through examples
	if !walkReferencedExamples(ctx, parameter.Examples, append(loc, LocationContext{Parent: parent, ParentField: "examples"}), openAPI, yield) {
		return false
	}

	// Visit Parameter Extensions
	return yield(WalkItem{Match: geMatchFunc(parameter.Extensions), Location: append(loc, LocationContext{Parent: parent, ParentField: ""}), OpenAPI: openAPI})
}

// walkReferencedRequestBody walks through a referenced request body
func walkReferencedRequestBody(ctx context.Context, requestBody *ReferencedRequestBody, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if requestBody == nil {
		return true
	}

	referencedRequestBodyMatchFunc := geMatchFunc(requestBody)

	if !yield(WalkItem{Match: referencedRequestBodyMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// If it's not a reference, walk the actual RequestBody
	if !requestBody.IsReference() && requestBody.Object != nil {
		return walkRequestBody(ctx, requestBody.Object, referencedRequestBodyMatchFunc, loc, openAPI, yield)
	}

	return true
}

// walkRequestBody walks through a request body
func walkRequestBody(ctx context.Context, requestBody *RequestBody, parent MatchFunc, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if requestBody == nil {
		return true
	}

	// Walk through content
	if !walkMediaTypes(ctx, requestBody.Content, append(loc, LocationContext{Parent: parent, ParentField: "content"}), openAPI, yield) {
		return false
	}

	// Visit RequestBody Extensions
	return yield(WalkItem{Match: geMatchFunc(requestBody.Extensions), Location: append(loc, LocationContext{Parent: parent, ParentField: ""}), OpenAPI: openAPI})
}

// walkResponses walks through responses
func walkResponses(ctx context.Context, responses *Responses, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if responses == nil {
		return true
	}

	responsesMatchFunc := geMatchFunc(responses)

	if !yield(WalkItem{Match: responsesMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// Walk through default response
	if !walkReferencedResponse(ctx, responses.Default, append(loc, LocationContext{Parent: responsesMatchFunc, ParentField: "default"}), openAPI, yield) {
		return false
	}

	// Walk through status code responses
	for statusCode, response := range responses.All() {
		if !walkReferencedResponse(ctx, response, append(loc, LocationContext{Parent: responsesMatchFunc, ParentKey: pointer.From(statusCode)}), openAPI, yield) {
			return false
		}
	}

	// Visit Responses Extensions
	return yield(WalkItem{Match: geMatchFunc(responses.Extensions), Location: append(loc, LocationContext{Parent: responsesMatchFunc, ParentField: ""}), OpenAPI: openAPI})
}

// walkReferencedResponse walks through a referenced response
func walkReferencedResponse(ctx context.Context, response *ReferencedResponse, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if response == nil {
		return true
	}

	referencedResponseMatchFunc := geMatchFunc(response)

	if !yield(WalkItem{Match: referencedResponseMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// If it's not a reference, walk the actual Response
	if !response.IsReference() && response.Object != nil {
		return walkResponse(ctx, response.Object, referencedResponseMatchFunc, loc, openAPI, yield)
	}

	return true
}

// walkResponse walks through a response
func walkResponse(ctx context.Context, response *Response, parent MatchFunc, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if response == nil {
		return true
	}

	// Walk through headers
	if !walkReferencedHeaders(ctx, response.Headers, append(loc, LocationContext{Parent: parent, ParentField: "headers"}), openAPI, yield) {
		return false
	}

	// Walk through content
	if !walkMediaTypes(ctx, response.Content, append(loc, LocationContext{Parent: parent, ParentField: "content"}), openAPI, yield) {
		return false
	}

	// Walk through links
	if !walkReferencedLinks(ctx, response.Links, append(loc, LocationContext{Parent: parent, ParentField: "links"}), openAPI, yield) {
		return false
	}

	// Visit Response Extensions
	return yield(WalkItem{Match: geMatchFunc(response.Extensions), Location: append(loc, LocationContext{Parent: parent, ParentField: ""}), OpenAPI: openAPI})
}

// walkMediaTypes walks through media types
func walkMediaTypes(ctx context.Context, mediaTypes *sequencedmap.Map[string, *MediaType], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if mediaTypes == nil || mediaTypes.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for mediaType, mt := range mediaTypes.All() {
		parentLoc.ParentKey = pointer.From(mediaType)

		if !walkMediaType(ctx, mt, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

// walkMediaType walks through a media type
func walkMediaType(ctx context.Context, mediaType *MediaType, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if mediaType == nil {
		return true
	}

	mediaTypeMatchFunc := geMatchFunc(mediaType)

	if !yield(WalkItem{Match: mediaTypeMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// Walk through schema
	if !walkSchema(ctx, mediaType.Schema, append(loc, LocationContext{Parent: mediaTypeMatchFunc, ParentField: "schema"}), openAPI, yield) {
		return false
	}

	// Walk through encoding
	if !walkEncodings(ctx, mediaType.Encoding, append(loc, LocationContext{Parent: mediaTypeMatchFunc, ParentField: "encoding"}), openAPI, yield) {
		return false
	}

	// Walk through examples
	if !walkReferencedExamples(ctx, mediaType.Examples, append(loc, LocationContext{Parent: mediaTypeMatchFunc, ParentField: "examples"}), openAPI, yield) {
		return false
	}

	// Visit MediaType Extensions
	return yield(WalkItem{Match: geMatchFunc(mediaType.Extensions), Location: append(loc, LocationContext{Parent: mediaTypeMatchFunc, ParentField: ""}), OpenAPI: openAPI})
}

// walkEncodings walks through encodings
func walkEncodings(ctx context.Context, encodings *sequencedmap.Map[string, *Encoding], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if encodings == nil || encodings.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for property, encoding := range encodings.All() {
		parentLoc.ParentKey = pointer.From(property)

		if !walkEncoding(ctx, encoding, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

// walkEncoding walks through an encoding
func walkEncoding(ctx context.Context, encoding *Encoding, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if encoding == nil {
		return true
	}

	encodingMatchFunc := geMatchFunc(encoding)

	if !yield(WalkItem{Match: encodingMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// Walk through headers
	if !walkReferencedHeaders(ctx, encoding.Headers, append(loc, LocationContext{Parent: encodingMatchFunc, ParentField: "headers"}), openAPI, yield) {
		return false
	}

	// Visit Encoding Extensions
	return yield(WalkItem{Match: geMatchFunc(encoding.Extensions), Location: append(loc, LocationContext{Parent: encodingMatchFunc, ParentField: ""}), OpenAPI: openAPI})
}

// walkReferencedHeaders walks through referenced headers
func walkReferencedHeaders(ctx context.Context, headers *sequencedmap.Map[string, *ReferencedHeader], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
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

// walkReferencedHeader walks through a referenced header
func walkReferencedHeader(ctx context.Context, header *ReferencedHeader, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if header == nil {
		return true
	}

	referencedHeaderMatchFunc := geMatchFunc(header)

	if !yield(WalkItem{Match: referencedHeaderMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// If it's not a reference, walk the actual Header
	if !header.IsReference() && header.Object != nil {
		return walkHeader(ctx, header.Object, referencedHeaderMatchFunc, loc, openAPI, yield)
	}

	return true
}

// walkHeader walks through a header
func walkHeader(ctx context.Context, header *Header, parent MatchFunc, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if header == nil {
		return true
	}

	// Walk through schema
	if !walkSchema(ctx, header.Schema, append(loc, LocationContext{Parent: parent, ParentField: "schema"}), openAPI, yield) {
		return false
	}

	// Walk through content
	if !walkMediaTypes(ctx, header.Content, append(loc, LocationContext{Parent: parent, ParentField: "content"}), openAPI, yield) {
		return false
	}

	// Walk through examples
	if !walkReferencedExamples(ctx, header.Examples, append(loc, LocationContext{Parent: parent, ParentField: "examples"}), openAPI, yield) {
		return false
	}

	// Visit Header Extensions
	return yield(WalkItem{Match: geMatchFunc(header.Extensions), Location: append(loc, LocationContext{Parent: parent, ParentField: ""}), OpenAPI: openAPI})
}

// walkReferencedExamples walks through referenced examples
func walkReferencedExamples(ctx context.Context, examples *sequencedmap.Map[string, *ReferencedExample], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
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

// walkReferencedExample walks through a referenced example
func walkReferencedExample(ctx context.Context, example *ReferencedExample, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if example == nil {
		return true
	}

	referencedExampleMatchFunc := geMatchFunc(example)

	if !yield(WalkItem{Match: referencedExampleMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// If it's not a reference, walk the actual Example
	if !example.IsReference() && example.Object != nil {
		return walkExample(ctx, example.Object, referencedExampleMatchFunc, loc, openAPI, yield)
	}

	return true
}

// walkExample walks through an example
func walkExample(_ context.Context, example *Example, parent MatchFunc, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if example == nil {
		return true
	}

	// Visit Example Extensions
	return yield(WalkItem{Match: geMatchFunc(example.Extensions), Location: append(loc, LocationContext{Parent: parent, ParentField: ""}), OpenAPI: openAPI})
}
