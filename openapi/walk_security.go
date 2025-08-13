package openapi

import (
	"context"

	"github.com/speakeasy-api/openapi/pointer"
)

// walkSecurity walks through security requirements
func walkSecurity(ctx context.Context, security []*SecurityRequirement, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if len(security) == 0 {
		return true
	}

	// Get the last loc so we can set the parent index
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for i, securityRequirement := range security {
		parentLoc.ParentIndex = pointer.From(i)

		if !walkSecurityRequirement(ctx, securityRequirement, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

// walkSecurityRequirement walks through a single security requirement
func walkSecurityRequirement(_ context.Context, securityRequirement *SecurityRequirement, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if securityRequirement == nil {
		return true
	}

	securityMatchFunc := geMatchFunc(securityRequirement)

	return yield(WalkItem{Match: securityMatchFunc, Location: loc, OpenAPI: openAPI})
}

// walkReferencedSecurityScheme walks through a referenced security scheme
func walkReferencedSecurityScheme(ctx context.Context, securityScheme *ReferencedSecurityScheme, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if securityScheme == nil {
		return true
	}

	referencedSecuritySchemeMatchFunc := geMatchFunc(securityScheme)

	if !yield(WalkItem{Match: referencedSecuritySchemeMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// If it's not a reference, walk the actual SecurityScheme
	if !securityScheme.IsReference() && securityScheme.Object != nil {
		return walkSecurityScheme(ctx, securityScheme.Object, referencedSecuritySchemeMatchFunc, loc, openAPI, yield)
	}

	return true
}

// walkSecurityScheme walks through a security scheme
func walkSecurityScheme(ctx context.Context, securityScheme *SecurityScheme, parent MatchFunc, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if securityScheme == nil {
		return true
	}

	// Walk through flows if it's OAuth2
	if !walkOAuthFlows(ctx, securityScheme.Flows, append(loc, LocationContext{Parent: parent, ParentField: "flows"}), openAPI, yield) {
		return false
	}

	// Visit SecurityScheme Extensions
	return yield(WalkItem{Match: geMatchFunc(securityScheme.Extensions), Location: append(loc, LocationContext{Parent: parent, ParentField: ""}), OpenAPI: openAPI})
}

// walkOAuthFlows walks through OAuth flows
func walkOAuthFlows(ctx context.Context, flows *OAuthFlows, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if flows == nil {
		return true
	}

	flowsMatchFunc := geMatchFunc(flows)

	if !yield(WalkItem{Match: flowsMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// Walk through individual flows
	if !walkOAuthFlow(ctx, flows.Implicit, append(loc, LocationContext{Parent: flowsMatchFunc, ParentField: "implicit"}), openAPI, yield) {
		return false
	}

	if !walkOAuthFlow(ctx, flows.Password, append(loc, LocationContext{Parent: flowsMatchFunc, ParentField: "password"}), openAPI, yield) {
		return false
	}

	if !walkOAuthFlow(ctx, flows.ClientCredentials, append(loc, LocationContext{Parent: flowsMatchFunc, ParentField: "clientCredentials"}), openAPI, yield) {
		return false
	}

	if !walkOAuthFlow(ctx, flows.AuthorizationCode, append(loc, LocationContext{Parent: flowsMatchFunc, ParentField: "authorizationCode"}), openAPI, yield) {
		return false
	}

	// Visit OAuthFlows Extensions
	return yield(WalkItem{Match: geMatchFunc(flows.Extensions), Location: append(loc, LocationContext{Parent: flowsMatchFunc, ParentField: ""}), OpenAPI: openAPI})
}

// walkOAuthFlow walks through an OAuth flow
func walkOAuthFlow(_ context.Context, flow *OAuthFlow, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if flow == nil {
		return true
	}

	flowMatchFunc := geMatchFunc(flow)

	if !yield(WalkItem{Match: flowMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// Visit OAuthFlow Extensions
	return yield(WalkItem{Match: geMatchFunc(flow.Extensions), Location: append(loc, LocationContext{Parent: flowMatchFunc, ParentField: ""}), OpenAPI: openAPI})
}
