package openapi_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndex_ReferencedPathItem_MountsAndNodeOperations(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /first:
    $ref: './alias.yaml'
  /second:
    $ref: './path.yaml'
components:
  schemas:
    Unused:
      type: string
`))
	require.NoError(t, err, "main document should parse")
	fs := NewMockVirtualFS()
	fs.AddFile("/spec/alias.yaml", "$ref: './path.yaml'\n")
	fs.AddFile("/spec/path.yaml", `get:
  responses:
    '200':
      description: ok
      content:
        application/json:
          schema:
            type: string
`)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument: doc, TargetDocument: doc, TargetLocation: "/spec/main.yaml", VirtualFS: fs,
	}, openapi.WithNodeOperationMap())
	require.False(t, idx.HasErrors(), "valid referenced path items should resolve")
	require.Len(t, idx.Operations, 2, "the same source operation mounted twice must retain both usages")
	paths := make([]string, 0, len(idx.Operations))
	for _, operation := range idx.Operations {
		method, path := openapi.ExtractMethodAndPath(operation.Location)
		assert.Equal(t, "get", method, "referenced operation should retain its method")
		paths = append(paths, path)
	}
	assert.Equal(t, []string{"/first", "/second"}, paths, "each operation should retain its mounted URL")

	response, ok := idx.Operations[0].Node.Responses.Get("200")
	require.True(t, ok, "the referenced operation should contain its response")
	mediaType, ok := response.GetObject().Content.Get("application/json")
	require.True(t, ok, "the response should contain its schema")
	typeNode := mediaType.Schema.GetSchema().GetCore().Type.ValueNode
	assert.Len(t, idx.GetNodeOperations(typeNode), 2, "shared external schema nodes must remain associated with both operation usages")
	assert.Equal(t, "/spec/path.yaml", idx.GetDocumentLocation(typeNode), "an alias must not replace the final target's source filename")
	unused, ok := doc.Components.Schemas.Get("Unused")
	require.True(t, ok, "the unrelated component should exist")
	assert.Empty(t, idx.GetNodeOperations(unused.GetSchema().GetCore().Type.ValueNode), "reference traversal must not leak its current operation into unrelated components")
}

func TestIndex_CallbackOperationInfo_Success(t *testing.T) {
	t.Parallel()

	const callbackPathItem = `'{$request.body#/url}':
  put:
    responses:
      '200':
        description: ok
`
	tests := []struct {
		name  string
		hooks string
	}{
		{
			name: "inline callback",
			hooks: `    post:
      responses:
        '200':
          description: ok
      callbacks:
        onEvent:
          '{$request.body#/url}':
            put:
              responses:
                '200':
                  description: ok
`,
		},
		{
			name: "referenced callback",
			hooks: `    post:
      responses:
        '200':
          description: ok
      callbacks:
        onEvent:
          $ref: './callback.yaml'
`,
		},
		{
			name:  "referenced callback in referenced path item",
			hooks: "    $ref: './path.yaml'\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			fs := NewMockVirtualFS()
			fs.AddFile("/spec/callback.yaml", callbackPathItem)
			fs.AddFile("/spec/path.yaml", `post:
  responses:
    '200':
      description: ok
  callbacks:
    onEvent:
      $ref: './callback.yaml'
`)
			main := "openapi: 3.1.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths:\n  /hooks:\n" + tt.hooks
			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(main))
			require.NoError(t, err, "main document should parse")

			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument: doc, TargetDocument: doc, TargetLocation: "/spec/main.yaml", VirtualFS: fs,
			})
			require.False(t, idx.HasErrors(), "callbacks should resolve")
			require.Len(t, idx.Operations, 2, "should index the mounting operation and the callback operation")

			type info struct{ path, method string }
			actual := make([]info, 0, len(idx.Operations))
			for _, operation := range idx.Operations {
				path, method, isWebhook := openapi.ExtractOperationInfo(operation.Location)
				assert.False(t, isWebhook, "callback operations are not webhooks")
				actual = append(actual, info{path: path, method: method})
			}
			assert.ElementsMatch(t, []info{
				{path: "/hooks", method: "post"},
				{path: "{$request.body#/url}", method: "put"},
			}, actual, "callback operations should report their expression and own method")
		})
	}
}

func TestIndex_ReferencedPathItem_CallbackCyclesTerminate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		pathItem   string
		callback   string
		operations int
	}{
		{
			name: "callback references its containing path item",
			pathItem: `get:
  operationId: getItem
  callbacks:
    event:
      '{$request.body#/url}':
        $ref: './path.yaml'
  responses:
    '200':
      description: ok
`,
			operations: 1,
		},
		{
			name: "referenced callback references itself through an inline path item",
			pathItem: `get:
  operationId: getItem
  callbacks:
    event:
      $ref: './callback.yaml'
  responses:
    '200':
      description: ok
`,
			callback: `'{$request.body#/url}':
  post:
    callbacks:
      next:
        $ref: './callback.yaml'
    responses:
      '200':
        description: ok
`,
			operations: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /items:
    $ref: './path.yaml'
`))
			require.NoError(t, err, "main document should parse")
			fs := NewMockVirtualFS()
			fs.AddFile("/spec/path.yaml", tt.pathItem)
			fs.AddFile("/spec/callback.yaml", tt.callback)
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument: doc, TargetDocument: doc, TargetLocation: "/spec/main.yaml", VirtualFS: fs,
			})
			assert.False(t, idx.HasErrors(), "a finite reference graph should not acquire a false schema-cycle error")
			assert.Len(t, idx.Operations, tt.operations, "active callback cycles should stop before re-expanding the same source target")
		})
	}
}
