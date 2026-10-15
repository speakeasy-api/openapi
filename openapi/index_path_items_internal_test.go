package openapi

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Programmatically built targets have no YAML root node, so the active-target
// guards key them by object identity instead.
func TestIndex_WalkReferencedTargets_WithoutRootNode_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc, _, err := Unmarshal(ctx, strings.NewReader("openapi: 3.1.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}\n"))
	require.NoError(t, err, "main document should parse")
	idx := BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument: doc, TargetDocument: doc, TargetLocation: "/spec/main.yaml",
	})

	pathItem := NewPathItem()
	pathItem.Set(HTTPMethodGet, &Operation{OperationID: pointer.From("getItem")})
	callback := NewCallback()

	idx.walkReferencedPathItem(ctx, nil, pathItem)
	idx.walkReferencedCallback(ctx, nil, callback)

	assert.Len(t, idx.Operations, 1, "the programmatic path item's operation should be indexed")
	assert.Empty(t, idx.activePathItemTargets, "path item guard should be released after the walk")
	assert.Empty(t, idx.activeCallbackTargets, "callback guard should be released after the walk")
	assert.Empty(t, idx.operationLocationPrefix, "operation location prefix should be restored after the walk")
}
