package openapi

import (
	"context"

	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

func walkTags(ctx context.Context, tags []*Tag, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	// Get the last loc so we can set the parent index
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for i, tag := range tags {
		parentLoc.ParentIndex = pointer.From(i)

		if !walkTag(ctx, tag, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

func walkTag(ctx context.Context, tag *Tag, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if tag == nil {
		return true
	}

	tagMatchFunc := getMatchFunc(tag)

	if !yield(WalkItem{Match: tagMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	if !walkExternalDocs(ctx, tag.ExternalDocs, append(loc, LocationContext{Parent: tagMatchFunc, ParentField: "externalDocs"}), openAPI, yield) {
		return false
	}

	// Visit Tag Extensions
	return yield(WalkItem{Match: getMatchFunc(tag.Extensions), Location: append(loc, LocationContext{Parent: tagMatchFunc, ParentField: ""}), OpenAPI: openAPI})
}

func walkServers(ctx context.Context, servers []*Server, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	// Get the last loc so we can set the parent index
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for i, server := range servers {
		parentLoc.ParentIndex = pointer.From(i)

		if !walkServer(ctx, server, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

func walkServer(ctx context.Context, server *Server, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if server == nil {
		return true
	}

	serverMatchFunc := getMatchFunc(server)

	if !yield(WalkItem{Match: serverMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	if !walkVariables(ctx, server.Variables, append(loc, LocationContext{Parent: serverMatchFunc, ParentField: "variables"}), openAPI, yield) {
		return false
	}

	return yield(WalkItem{Match: getMatchFunc(server.Extensions), Location: append(loc, LocationContext{Parent: serverMatchFunc, ParentField: ""}), OpenAPI: openAPI})
}

func walkVariables(ctx context.Context, variables *sequencedmap.Map[string, *ServerVariable], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for key, variable := range variables.All() {
		parentLoc.ParentKey = pointer.From(key)

		if !walkVariable(ctx, variable, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

func walkVariable(_ context.Context, variable *ServerVariable, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	variableMatchFunc := getMatchFunc(variable)

	if !yield(WalkItem{Match: variableMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	return yield(WalkItem{Match: getMatchFunc(variable.Extensions), Location: append(loc, LocationContext{Parent: variableMatchFunc, ParentField: ""}), OpenAPI: openAPI})
}
