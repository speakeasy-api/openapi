package openapi_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestBuildIndex_NodeToOperations_WithOption_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
    post:
      operationId: createUser
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/User'
      responses:
        "201":
          description: Created
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
`
	doc := unmarshalOpenAPI(t, ctx, yml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}, openapi.WithNodeOperationMap())

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Map should be populated
	require.NotNil(t, idx.NodeToOperations, "NodeToOperations map should be initialized")
	assert.NotEmpty(t, idx.NodeToOperations, "NodeToOperations should have entries when enabled")

	// Should have operations indexed
	assert.Len(t, idx.Operations, 2, "should have 2 operations")
}

func TestBuildIndex_NodeToOperations_Disabled_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "200":
          description: Success
`
	doc := unmarshalOpenAPI(t, ctx, yml)
	// Don't pass WithNodeOperationMap() option
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Map should be nil when disabled (default)
	assert.Nil(t, idx.NodeToOperations, "NodeToOperations should be nil when disabled")

	// GetNodeOperations should return nil for any node
	assert.Nil(t, idx.GetNodeOperations(nil), "GetNodeOperations should return nil when disabled")
}

func TestBuildIndex_NodeToOperations_SharedSchema_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
  /admin/users:
    get:
      operationId: getAdminUsers
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
`
	doc := unmarshalOpenAPI(t, ctx, yml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}, openapi.WithNodeOperationMap())

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Should have 2 operations
	assert.Len(t, idx.Operations, 2, "should have 2 operations")

	// The User schema is referenced by both operations
	// Get the User schema node
	require.Len(t, idx.ComponentSchemas, 1, "should have 1 component schema")
	userSchema := idx.ComponentSchemas[0]
	require.NotNil(t, userSchema, "User schema should exist")

	userNode := userSchema.Node.GetRootNode()
	require.NotNil(t, userNode, "User schema should have a root node")

	// Check that the User schema is mapped to both operations
	ops := idx.GetNodeOperations(userNode)
	assert.Len(t, ops, 2, "User schema should be referenced by 2 operations")
}

func TestBuildIndex_NodeToOperations_SharedSchemaNestedRefs_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Two operations reference Parent, which has a nested $ref to Child.
	// The first operation walks Parent fully (including Child).
	// The second operation hits the visitedRefs shortcut for Parent.
	// Child's nodes must still be associated with BOTH operations.
	yml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /first:
    get:
      operationId: firstOp
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Parent'
  /second:
    get:
      operationId: secondOp
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Parent'
components:
  schemas:
    Parent:
      type: object
      properties:
        child:
          $ref: '#/components/schemas/Child'
    Child:
      type: object
      properties:
        name:
          type: string
`
	doc := unmarshalOpenAPI(t, ctx, yml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}, openapi.WithNodeOperationMap())

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")
	assert.Len(t, idx.Operations, 2, "should have 2 operations")

	// Find the Child component schema
	var childNode *yaml.Node
	for _, schema := range idx.ComponentSchemas {
		jp := schema.Location.ToJSONPointer()
		if strings.Contains(jp.String(), "Child") {
			childNode = schema.Node.GetRootNode()
			break
		}
	}
	require.NotNil(t, childNode, "Child schema root node should exist")

	// Child must be associated with both operations — not just the first.
	// Before the fix, the visitedRefs shortcut only registered immediate
	// leaf nodes of the resolved schema, missing nested refs like Child.
	ops := idx.GetNodeOperations(childNode)
	require.Len(t, ops, 2, "Child schema should be mapped to both operations via cached ref nodes")

	opIDs := make([]string, len(ops))
	for i, op := range ops {
		opIDs[i] = *op.Node.OperationID
	}
	assert.ElementsMatch(t, []string{"firstOp", "secondOp"}, opIDs, "both operations should reference Child")
}

func TestBuildIndex_NodeToOperations_Webhooks_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
webhooks:
  newUser:
    post:
      operationId: userCreatedWebhook
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                userId:
                  type: string
      responses:
        "200":
          description: OK
`
	doc := unmarshalOpenAPI(t, ctx, yml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}, openapi.WithNodeOperationMap())

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Should have 1 webhook operation
	assert.Len(t, idx.Operations, 1, "should have 1 operation from webhook")

	// Check that nodes are mapped to the webhook operation
	assert.NotEmpty(t, idx.NodeToOperations, "NodeToOperations should have entries")

	// Verify the operation location indicates it's a webhook
	op := idx.Operations[0]
	require.NotNil(t, op, "operation should exist")
	assert.True(t, openapi.IsWebhookLocation(op.Location), "operation should be identified as webhook")
}

func TestBuildIndex_GetNodeOperations_NilCases_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "nil index",
			test: func(t *testing.T) {
				t.Helper()
				var idx *openapi.Index
				result := idx.GetNodeOperations(nil)
				assert.Nil(t, result, "should return nil for nil index")
			},
		},
		{
			name: "nil node",
			test: func(t *testing.T) {
				t.Helper()
				ctx := t.Context()
				yml := `
openapi: "3.1.0"
info:
  title: Test
  version: 1.0.0
paths: {}
`
				doc := unmarshalOpenAPI(t, ctx, yml)
				idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
					RootDocument:   doc,
					TargetDocument: doc,
					TargetLocation: "test.yaml",
				}, openapi.WithNodeOperationMap())

				result := idx.GetNodeOperations(nil)
				assert.Nil(t, result, "should return nil for nil node")
			},
		},
		{
			name: "node not found",
			test: func(t *testing.T) {
				t.Helper()
				ctx := t.Context()
				yml := `
openapi: "3.1.0"
info:
  title: Test
  version: 1.0.0
paths: {}
`
				doc := unmarshalOpenAPI(t, ctx, yml)
				idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
					RootDocument:   doc,
					TargetDocument: doc,
					TargetLocation: "test.yaml",
				}, openapi.WithNodeOperationMap())

				// Create a node that's not in the document
				unknownNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "unknown"}
				result := idx.GetNodeOperations(unknownNode)
				assert.Nil(t, result, "should return nil for unknown node")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.test(t)
		})
	}
}

func TestIsWebhookLocation_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	tests := []struct {
		name      string
		yml       string
		isWebhook bool
		opId      string
	}{
		{
			name: "path operation is not webhook",
			yml: `
openapi: "3.1.0"
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "200":
          description: OK
`,
			isWebhook: false,
			opId:      "getUsers",
		},
		{
			name: "webhook operation is webhook",
			yml: `
openapi: "3.1.0"
info:
  title: Test
  version: 1.0.0
paths: {}
webhooks:
  userCreated:
    post:
      operationId: userCreatedWebhook
      responses:
        "200":
          description: OK
`,
			isWebhook: true,
			opId:      "userCreatedWebhook",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := unmarshalOpenAPI(t, ctx, tt.yml)
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})

			require.NotNil(t, idx, "index should not be nil")
			require.Len(t, idx.Operations, 1, "should have 1 operation")

			op := idx.Operations[0]
			assert.Equal(t, tt.isWebhook, openapi.IsWebhookLocation(op.Location),
				"IsWebhookLocation should return %v for %s", tt.isWebhook, tt.opId)
		})
	}
}

func TestExtractOperationInfo_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	tests := []struct {
		name           string
		yml            string
		expectedPath   string
		expectedMethod string
		isWebhook      bool
	}{
		{
			name: "path operation",
			yml: `
openapi: "3.1.0"
info:
  title: Test
  version: 1.0.0
paths:
  /users/{id}:
    get:
      operationId: getUser
      responses:
        "200":
          description: OK
`,
			expectedPath:   "/users/{id}",
			expectedMethod: "get",
			isWebhook:      false,
		},
		{
			name: "webhook operation",
			yml: `
openapi: "3.1.0"
info:
  title: Test
  version: 1.0.0
paths: {}
webhooks:
  orderCreated:
    post:
      operationId: orderWebhook
      responses:
        "200":
          description: OK
`,
			expectedPath:   "orderCreated",
			expectedMethod: "post",
			isWebhook:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := unmarshalOpenAPI(t, ctx, tt.yml)
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})

			require.NotNil(t, idx, "index should not be nil")
			require.Len(t, idx.Operations, 1, "should have 1 operation")

			op := idx.Operations[0]
			path, method, isWebhook := openapi.ExtractOperationInfo(op.Location)

			assert.Equal(t, tt.expectedPath, path, "path should match")
			assert.Equal(t, tt.expectedMethod, method, "method should match")
			assert.Equal(t, tt.isWebhook, isWebhook, "isWebhook should match")
		})
	}
}

func TestBuildIndex_NodeToOperations_ComponentsNotMapped_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// This test verifies that schemas defined in components but not referenced
	// by any operation are NOT in the NodeToOperations map
	yml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
components:
  schemas:
    User:
      type: object
    UnusedSchema:
      type: object
      properties:
        unused:
          type: string
`
	doc := unmarshalOpenAPI(t, ctx, yml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}, openapi.WithNodeOperationMap())

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Get the User schema - should be mapped to the operation
	require.Len(t, idx.ComponentSchemas, 2, "should have 2 component schemas")

	// Find the User and UnusedSchema
	var userOps, unusedOps []*openapi.IndexNode[*openapi.Operation]
	for _, schema := range idx.ComponentSchemas {
		if schema == nil || schema.Node == nil {
			continue
		}
		node := schema.Node.GetRootNode()
		if node == nil {
			continue
		}
		ops := idx.GetNodeOperations(node)
		// Check location to identify which schema this is
		jp := schema.Location.ToJSONPointer()
		if strings.Contains(jp.String(), "User") {
			userOps = ops
		} else if strings.Contains(jp.String(), "UnusedSchema") {
			unusedOps = ops
		}
	}

	// User should be mapped to 1 operation
	assert.Len(t, userOps, 1, "User schema should be mapped to 1 operation")

	// UnusedSchema should NOT be mapped to any operations
	// (it's after paths in the walk order, so currentOperation is nil)
	assert.Empty(t, unusedOps, "UnusedSchema should not be mapped to any operations")
}

func TestBuildIndex_NodeToOperations_NestedSchemaNodes_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// This test verifies that nested nodes WITHIN a component schema
	// are also mapped to operations that reference the parent schema via $ref
	yml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      operationId: getTest
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/MySchema'
components:
  schemas:
    MySchema:
      type: array
      items:
        type: object
        properties:
          id:
            type: integer
`
	doc := unmarshalOpenAPI(t, ctx, yml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}, openapi.WithNodeOperationMap())

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Should have 1 operation
	require.Len(t, idx.Operations, 1, "should have 1 operation")

	// Get the component schema (MySchema)
	require.Len(t, idx.ComponentSchemas, 1, "should have 1 component schema")
	mySchema := idx.ComponentSchemas[0]
	require.NotNil(t, mySchema, "MySchema should exist")

	// The root node of MySchema should be mapped to the operation
	mySchemaNode := mySchema.Node.GetRootNode()
	require.NotNil(t, mySchemaNode, "MySchema should have a root node")

	rootOps := idx.GetNodeOperations(mySchemaNode)
	require.Len(t, rootOps, 1, "MySchema root should be mapped to 1 operation")
	assert.Equal(t, "getTest", *rootOps[0].Node.OperationID, "should be getTest operation")

	// Now check nested nodes - the items schema should also be mapped
	// Find an inline schema that's within MySchema (like the items schema)
	var itemsSchemaOps []*openapi.IndexNode[*openapi.Operation]
	for _, schema := range idx.InlineSchemas {
		if schema == nil || schema.Node == nil {
			continue
		}
		node := schema.Node.GetRootNode()
		if node == nil {
			continue
		}
		ops := idx.GetNodeOperations(node)
		if len(ops) > 0 {
			// This is an inline schema that's mapped to operations
			itemsSchemaOps = ops
			break
		}
	}

	// At least one inline schema (like items or id property) should be mapped
	assert.NotEmpty(t, itemsSchemaOps, "nested inline schemas should be mapped to operations")
}

func TestBuildIndex_NodeToOperations_BooleanSchema_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// This test reproduces the exact scenario the user described:
	// - Operation references a component schema via $ref
	// - The component schema has `items: true` (boolean schema)
	// - We need to verify that the items node is mapped to the operation
	yml := `
openapi: "3.1.0"
info:
  title: Test
  version: 1.0.0
paths:
  /test:
    get:
      operationId: getTest
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/MySchema'
components:
  schemas:
    MySchema:
      type: array
      items: true
`
	doc := unmarshalOpenAPI(t, ctx, yml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}, openapi.WithNodeOperationMap())

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Should have 1 operation
	require.Len(t, idx.Operations, 1, "should have 1 operation")

	// Find the boolean schema (items: true)
	require.NotEmpty(t, idx.BooleanSchemas, "should have boolean schemas")

	// Check if the boolean schema is mapped to the operation
	var boolSchemaOps []*openapi.IndexNode[*openapi.Operation]
	for _, boolSchema := range idx.BooleanSchemas {
		if boolSchema == nil || boolSchema.Node == nil {
			continue
		}
		node := boolSchema.Node.GetRootNode()
		if node != nil {
			ops := idx.GetNodeOperations(node)
			if len(ops) > 0 {
				boolSchemaOps = ops
				break
			}
		}
	}

	// The boolean schema should be mapped to the getTest operation
	require.Len(t, boolSchemaOps, 1, "boolean schema should be mapped to 1 operation")
	assert.Equal(t, "getTest", *boolSchemaOps[0].Node.OperationID, "should be getTest operation")
}

func TestBuildIndex_NodeToOperations_LeafValueNode_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// This test verifies that GetNodeOperations works for leaf VALUE nodes,
	// not just root nodes. For example, when a linter finds an issue on
	// the `true` value node in `items: true`, GetNodeOperations should
	// return the operations that reference the parent schema.
	yml := `
openapi: "3.1.0"
info:
  title: Test
  version: 1.0.0
paths:
  /test:
    get:
      operationId: getTest
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/MyArray'
components:
  schemas:
    MyArray:
      type: array
      items: true
`
	doc := unmarshalOpenAPI(t, ctx, yml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}, openapi.WithNodeOperationMap())

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")
	require.Len(t, idx.Operations, 1, "should have 1 operation")

	// Find the boolean schema (representing items: true)
	// This is the scenario where a linter gets a value node
	require.NotEmpty(t, idx.BooleanSchemas, "should have boolean schemas")

	// The boolean schema's root node is the actual value node (`true`)
	boolSchema := idx.BooleanSchemas[0]
	require.NotNil(t, boolSchema, "boolean schema should exist")
	require.NotNil(t, boolSchema.Node, "boolean schema node should not be nil")

	// Get the boolean value node - this is what a linter would get
	// when it finds an issue on `items: true`
	boolValueNode := boolSchema.Node.GetRootNode()
	require.NotNil(t, boolValueNode, "boolean value node should not be nil")

	// Verify this is actually the `true` value node
	assert.Equal(t, yaml.ScalarNode, boolValueNode.Kind, "should be a scalar node")
	assert.Equal(t, "true", boolValueNode.Value, "should have value 'true'")

	// Now verify GetNodeOperations works for this leaf value node
	ops := idx.GetNodeOperations(boolValueNode)
	require.Len(t, ops, 1, "leaf value node should be mapped to 1 operation")
	assert.Equal(t, "getTest", *ops[0].Node.OperationID, "should be getTest operation")
}

func TestBuildIndex_NodeToOperations_LeafKeyNode_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// This test verifies that GetNodeOperations works for leaf KEY nodes.
	// For example, when a linter reports an issue on the key `type` in
	// a schema, GetNodeOperations should return the associated operations.
	yml := `
openapi: "3.1.0"
info:
  title: Test
  version: 1.0.0
paths:
  /pets:
    get:
      operationId: getPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string
`
	doc := unmarshalOpenAPI(t, ctx, yml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	}, openapi.WithNodeOperationMap())

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")
	require.Len(t, idx.Operations, 1, "should have 1 operation")

	// Find the component schema (Pet)
	require.Len(t, idx.ComponentSchemas, 1, "should have 1 component schema")
	petSchema := idx.ComponentSchemas[0]
	require.NotNil(t, petSchema, "Pet schema should exist")

	// Get the actual schema to access the core model's Type field
	schema := petSchema.Node.GetSchema()
	require.NotNil(t, schema, "schema should not be nil")

	core := schema.GetCore()
	require.NotNil(t, core, "core should not be nil")

	// Access the Type field's key node directly
	// This tests that leaf key nodes are registered
	typeKeyNode := core.Type.KeyNode
	if typeKeyNode != nil {
		ops := idx.GetNodeOperations(typeKeyNode)
		require.Len(t, ops, 1, "type key node should be mapped to 1 operation")
		assert.Equal(t, "getPets", *ops[0].Node.OperationID, "should be getPets operation")
	}

	// Also test the value node of the Type field
	typeValueNode := core.Type.ValueNode
	if typeValueNode != nil {
		ops := idx.GetNodeOperations(typeValueNode)
		require.Len(t, ops, 1, "type value node should be mapped to 1 operation")
		assert.Equal(t, "getPets", *ops[0].Node.OperationID, "should be getPets operation")
	}
}
