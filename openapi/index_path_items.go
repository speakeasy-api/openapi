package openapi

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

type pathItemTargetKey struct {
	node   *yaml.Node
	object *PathItem
}

type callbackTargetKey struct {
	node   *yaml.Node
	object *Callback
}

func (i *Index) walkReferencedPathItem(ctx context.Context, location Locations, resolved *PathItem) {
	key := pathItemTargetKey{node: resolved.GetRootNode()}
	if key.node == nil {
		key.object = resolved
	}
	if i.activePathItemTargets[key] {
		return
	}
	if i.activePathItemTargets == nil {
		i.activePathItemTargets = make(map[pathItemTargetKey]bool)
	}
	i.activePathItemTargets[key] = true
	previousPrefix := i.operationLocationPrefix
	previousOperation, previousDepth := i.currentOperation, i.operationLocationDepth
	i.operationLocationPrefix = append(copyLocations(previousPrefix), location...)
	defer func() {
		delete(i.activePathItemTargets, key)
		i.operationLocationPrefix = previousPrefix
		i.currentOperation, i.operationLocationDepth = previousOperation, previousDepth
	}()

	// Keep physical locations for source-target categorisation, while operation
	// records retain their distinct mounted contexts. Do not globally deduplicate
	// operations by the target pointer: two mounted URLs are two usages.
	wrapped := &ReferencedPathItem{Object: resolved}
	_ = buildIndex(ctx, i, wrapped)
}

func (i *Index) walkReferencedCallback(ctx context.Context, resolved *Callback) {
	key := callbackTargetKey{node: resolved.GetRootNode()}
	if key.node == nil {
		key.object = resolved
	}
	if i.activeCallbackTargets[key] {
		return
	}
	if i.activeCallbackTargets == nil {
		i.activeCallbackTargets = make(map[callbackTargetKey]bool)
	}
	i.activeCallbackTargets[key] = true
	previousOperation, previousDepth := i.currentOperation, i.operationLocationDepth
	defer func() {
		delete(i.activeCallbackTargets, key)
		i.currentOperation, i.operationLocationDepth = previousOperation, previousDepth
	}()
	wrapped := &ReferencedCallback{Object: resolved}
	_ = buildIndex(ctx, i, wrapped)
}

// Root validation already checks inline operation IDs. Complete that check for
// collisions involving mounted reference targets, without re-emitting the
// inline-only collisions already supplied by Unmarshal/Validate.
func (i *Index) validateReferencedOperationIDs() {
	if len(i.referencedOperations) == 0 {
		return
	}
	type usageState struct {
		seen       bool
		inlineSeen bool
	}
	seen := make(map[string]usageState)
	for _, operation := range i.Operations {
		method, path := ExtractMethodAndPath(operation.Location)
		if method == "" || path == "" {
			continue
		}
		id := operation.Node.GetOperationID()
		if id == "" {
			continue
		}
		state := seen[id]
		referenced := i.referencedOperations[operation]
		if state.seen && (referenced || !state.inlineSeen) {
			node := getOperationIDValueNode(operation.Node)
			if node == nil {
				node = operation.Node.GetRootNode()
			}
			i.validationErrs = append(i.validationErrs, validation.NewValidationErrorWithDocumentLocation(
				validation.SeverityError,
				validation.RuleValidationOperationIdUnique,
				fmt.Errorf("the `%s` operation at path `%s` contains a duplicate operationId `%s`", method, path, id),
				node,
				i.GetDocumentLocation(node),
			))
		}
		state.seen = true
		state.inlineSeen = state.inlineSeen || !referenced
		seen[id] = state
	}
}
