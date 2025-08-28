package openapi

import (
	"context"

	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// walkWebhooks walks through webhooks
func walkWebhooks(ctx context.Context, webhooks *sequencedmap.Map[string, *ReferencedPathItem], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if webhooks == nil || webhooks.Len() == 0 {
		return true
	}

	// Get the last loc so we can set the parent key
	parentLoc := loc[len(loc)-1]
	loc = loc[:len(loc)-1]

	for name, pathItem := range webhooks.All() {
		parentLoc.ParentKey = pointer.From(name)

		if !walkReferencedPathItem(ctx, pathItem, append(loc, parentLoc), openAPI, yield) {
			return false
		}
	}
	return true
}

// walkReferencedLinks walks through referenced links
func walkReferencedLinks(ctx context.Context, links *sequencedmap.Map[string, *ReferencedLink], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
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

// walkReferencedLink walks through a referenced link
func walkReferencedLink(ctx context.Context, link *ReferencedLink, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if link == nil {
		return true
	}

	referencedLinkMatchFunc := getMatchFunc(link)

	if !yield(WalkItem{Match: referencedLinkMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// If it's not a reference, walk the actual Link
	if !link.IsReference() && link.Object != nil {
		return walkLink(ctx, link.Object, referencedLinkMatchFunc, loc, openAPI, yield)
	}

	return true
}

// walkLink walks through a link
func walkLink(ctx context.Context, link *Link, parent MatchFunc, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if link == nil {
		return true
	}

	// Walk through server
	if !walkServer(ctx, link.Server, append(loc, LocationContext{Parent: parent, ParentField: "server"}), openAPI, yield) {
		return false
	}

	// Visit Link Extensions
	return yield(WalkItem{Match: getMatchFunc(link.Extensions), Location: append(loc, LocationContext{Parent: parent, ParentField: ""}), OpenAPI: openAPI})
}

// walkReferencedCallbacks walks through referenced callbacks
func walkReferencedCallbacks(ctx context.Context, callbacks *sequencedmap.Map[string, *ReferencedCallback], loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
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

// walkReferencedCallback walks through a referenced callback
func walkReferencedCallback(ctx context.Context, callback *ReferencedCallback, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if callback == nil {
		return true
	}

	referencedCallbackMatchFunc := getMatchFunc(callback)

	if !yield(WalkItem{Match: referencedCallbackMatchFunc, Location: loc, OpenAPI: openAPI}) {
		return false
	}

	// If it's not a reference, walk the actual Callback
	if !callback.IsReference() && callback.Object != nil {
		return walkCallback(ctx, callback.Object, referencedCallbackMatchFunc, loc, openAPI, yield)
	}

	return true
}

// walkCallback walks through a callback
func walkCallback(ctx context.Context, callback *Callback, parent MatchFunc, loc []LocationContext, openAPI *OpenAPI, yield func(WalkItem) bool) bool {
	if callback == nil {
		return true
	}

	// Walk through callback path items
	for expression, pathItem := range callback.All() {
		if !walkReferencedPathItem(ctx, pathItem, append(loc, LocationContext{Parent: parent, ParentKey: pointer.From(string(expression))}), openAPI, yield) {
			return false
		}
	}

	// Visit Callback Extensions
	return yield(WalkItem{Match: getMatchFunc(callback.Extensions), Location: append(loc, LocationContext{Parent: parent, ParentField: ""}), OpenAPI: openAPI})
}
