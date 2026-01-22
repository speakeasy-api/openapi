package openapi_test

import (
	"context"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func unmarshalOpenAPI(t *testing.T, ctx context.Context, yaml string) *openapi.OpenAPI {
	t.Helper()
	o, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(yaml))
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "should have no validation errors")
	return o
}

func TestBuildIndex_EmptyDoc_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
openapi: "3.1.0"
info:
  title: Empty API
  version: 1.0.0
paths: {}
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.Empty(t, idx.GetAllSchemas(), "should have no schemas")
	assert.Empty(t, idx.GetAllPathItems(), "should have no path items")
	assert.False(t, idx.HasErrors(), "should have no errors")
}

func TestBuildIndex_ComponentSchemas_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
    Pet:
      type: object
      properties:
        name:
          type: string
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Should have component schemas indexed
	assert.Len(t, idx.ComponentSchemas, 2, "should have 2 component schemas")

	// Should have inline schemas within the components
	assert.Len(t, idx.InlineSchemas, 3, "should have 3 inline schemas from properties")
}

func TestBuildIndex_InlineSchemas_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
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
                type: array
                items:
                  type: object
                  properties:
                    id:
                      type: integer
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Should have inline schemas: array, object (items), integer (id property)
	assert.Len(t, idx.InlineSchemas, 3, "should have 3 inline schemas")
	assert.Empty(t, idx.ComponentSchemas, "should have no component schemas")
	assert.Empty(t, idx.SchemaReferences, "should have no schema references")
}

func TestBuildIndex_SchemaReferences_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
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
      properties:
        id:
          type: integer
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// $ref to User schema
	assert.Len(t, idx.SchemaReferences, 1, "should have 1 schema reference")
	// User component schema
	assert.Len(t, idx.ComponentSchemas, 1, "should have 1 component schema")
	// id property inline schema
	assert.Len(t, idx.InlineSchemas, 1, "should have 1 inline schema")
}

func TestBuildIndex_BooleanSchemas_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    AnyValue:
      type: object
      additionalProperties: true
    NoAdditional:
      type: object
      additionalProperties: false
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Two boolean schemas (true and false for additionalProperties)
	assert.Len(t, idx.BooleanSchemas, 2, "should have 2 boolean schemas")
	// Two component schemas (AnyValue and NoAdditional)
	assert.Len(t, idx.ComponentSchemas, 2, "should have 2 component schemas")
	assert.Empty(t, idx.InlineSchemas, "should have no inline schemas")
}

func TestBuildIndex_Servers_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com
    description: Production
    variables:
      version:
        default: v1
        enum: [v1, v2]
  - url: https://staging.example.com
    description: Staging
paths: {}
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	assert.Len(t, idx.Servers, 2, "should have 2 servers")
	assert.Len(t, idx.ServerVariables, 1, "should have 1 server variable")
}

func TestBuildIndex_Tags_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
tags:
  - name: users
    description: User operations
  - name: pets
    description: Pet operations
paths: {}
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	assert.Len(t, idx.Tags, 2, "should have 2 tags")
}

func TestBuildIndex_ExternalDocs_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
externalDocs:
  url: https://docs.example.com
  description: API Documentation
tags:
  - name: users
    externalDocs:
      url: https://docs.example.com/users
paths: {}
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	assert.Len(t, idx.ExternalDocumentation, 2, "should have 2 external docs")
}

func TestBuildIndex_GetAllSchemas_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
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
      additionalProperties: true
      properties:
        id:
          type: integer
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	allSchemas := idx.GetAllSchemas()
	assert.NotEmpty(t, allSchemas, "should have schemas")

	// Should include boolean, inline, component, external, and reference schemas
	totalExpected := len(idx.BooleanSchemas) + len(idx.InlineSchemas) +
		len(idx.ComponentSchemas) + len(idx.ExternalSchemas) + len(idx.SchemaReferences)
	assert.Len(t, allSchemas, totalExpected, "GetAllSchemas should return all schema types")
}

func TestBuildIndex_NilIndex_Methods_Success(t *testing.T) {
	t.Parallel()

	var idx *openapi.Index

	assert.Nil(t, idx.GetAllSchemas(), "nil index GetAllSchemas should return nil")
	assert.Nil(t, idx.GetAllPathItems(), "nil index GetAllPathItems should return nil")
	assert.Nil(t, idx.GetValidationErrors(), "nil index GetValidationErrors should return nil")
	assert.Nil(t, idx.GetResolutionErrors(), "nil index GetResolutionErrors should return nil")
	assert.Nil(t, idx.GetCircularReferenceErrors(), "nil index GetCircularReferenceErrors should return nil")
	assert.Nil(t, idx.GetAllErrors(), "nil index GetAllErrors should return nil")
	assert.False(t, idx.HasErrors(), "nil index HasErrors should return false")
}

// Tests for circular reference detection

func TestBuildIndex_CircularRef_OptionalProperty_Valid(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Optional property recursion - VALID (not required means {} is valid)
	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Node:
      type: object
      properties:
        next:
          $ref: '#/components/schemas/Node'
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	// Optional property circular refs should be VALID (no error)
	circularErrs := idx.GetCircularReferenceErrors()
	assert.Empty(t, circularErrs, "optional property circular ref should be valid (no error)")
}

func TestBuildIndex_CircularRef_RequiredProperty_Invalid(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Required property recursion - INVALID (no base case)
	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    A:
      type: object
      required: [b]
      properties:
        b:
          $ref: '#/components/schemas/B'
    B:
      type: object
      required: [a]
      properties:
        a:
          $ref: '#/components/schemas/A'
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	// Required property circular refs should be INVALID
	circularErrs := idx.GetCircularReferenceErrors()
	assert.NotEmpty(t, circularErrs, "required property circular ref should be invalid")
}

func TestBuildIndex_CircularRef_ArrayMinItemsZero_Valid(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Array with default minItems (0) - VALID (empty array terminates)
	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Category:
      type: object
      required: [children]
      properties:
        children:
          type: array
          items:
            $ref: '#/components/schemas/Category'
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	// Array with minItems=0 circular refs should be VALID
	circularErrs := idx.GetCircularReferenceErrors()
	assert.Empty(t, circularErrs, "array with minItems=0 circular ref should be valid")
}

func TestBuildIndex_CircularRef_ArrayMinItemsOne_Invalid(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Array with minItems=1 - INVALID (can't have empty array)
	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Node:
      type: object
      required: [children]
      properties:
        children:
          type: array
          minItems: 1
          items:
            $ref: '#/components/schemas/Node'
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	// Array with minItems>=1 circular refs should be INVALID
	circularErrs := idx.GetCircularReferenceErrors()
	assert.NotEmpty(t, circularErrs, "array with minItems>=1 circular ref should be invalid")
}

func TestBuildIndex_CircularRef_Nullable_Valid(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Nullable type union - VALID (null is a base case)
	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Node:
      type: [object, "null"]
      required: [next]
      properties:
        next:
          $ref: '#/components/schemas/Node'
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	// Nullable circular refs should be VALID
	circularErrs := idx.GetCircularReferenceErrors()
	assert.Empty(t, circularErrs, "nullable circular ref should be valid")
}

func TestBuildIndex_CircularRef_AdditionalPropertiesMinZero_Valid(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// AdditionalProperties with default minProperties (0) - VALID
	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    TrieNode:
      type: object
      required: [children]
      properties:
        children:
          type: object
          additionalProperties:
            $ref: '#/components/schemas/TrieNode'
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	// AdditionalProperties with minProperties=0 should be VALID
	circularErrs := idx.GetCircularReferenceErrors()
	assert.Empty(t, circularErrs, "additionalProperties with minProperties=0 should be valid")
}

func TestBuildIndex_CircularRef_AdditionalPropertiesMinOne_Invalid(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// AdditionalProperties with minProperties>=1 - INVALID
	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Node:
      type: object
      required: [children]
      properties:
        children:
          type: object
          minProperties: 1
          additionalProperties:
            $ref: '#/components/schemas/Node'
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	// AdditionalProperties with minProperties>=1 should be INVALID
	circularErrs := idx.GetCircularReferenceErrors()
	assert.NotEmpty(t, circularErrs, "additionalProperties with minProperties>=1 should be invalid")
}

func TestBuildIndex_CircularRef_OneOfWithNonRecursiveBranch_Valid(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// oneOf with at least one non-recursive branch - VALID
	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Expr:
      oneOf:
        - $ref: '#/components/schemas/Literal'
        - $ref: '#/components/schemas/BinaryExpr'
    Literal:
      type: object
      properties:
        value:
          type: string
    BinaryExpr:
      type: object
      required: [left, right]
      properties:
        left:
          $ref: '#/components/schemas/Expr'
        right:
          $ref: '#/components/schemas/Expr'
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	// oneOf with a non-recursive branch should be VALID
	circularErrs := idx.GetCircularReferenceErrors()
	assert.Empty(t, circularErrs, "oneOf with non-recursive branch should be valid")
}

func TestBuildIndex_CircularRef_DirectSelfRef_Optional_Valid(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Direct self-reference through optional property - VALID
	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    LinkedNode:
      type: object
      properties:
        value:
          type: string
        next:
          $ref: '#/components/schemas/LinkedNode'
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	circularErrs := idx.GetCircularReferenceErrors()
	assert.Empty(t, circularErrs, "direct self-ref through optional should be valid")
}

func TestBuildIndex_CircularRef_DirectSelfRef_Required_Invalid(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Direct self-reference through required property - INVALID
	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    InfiniteNode:
      type: object
      required: [self]
      properties:
        self:
          $ref: '#/components/schemas/InfiniteNode'
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	circularErrs := idx.GetCircularReferenceErrors()
	assert.NotEmpty(t, circularErrs, "direct self-ref through required should be invalid")
}

func TestBuildIndex_NoCircularRef_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// No circular reference - just regular refs
	yaml := `
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
      properties:
        id:
          type: integer
        address:
          $ref: '#/components/schemas/Address'
    Address:
      type: object
      properties:
        street:
          type: string
        city:
          type: string
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")
	assert.Empty(t, idx.GetCircularReferenceErrors(), "should have no circular reference errors")
}

func TestBuildIndex_LocationInfo_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	// Check that schemas have location information
	for _, schema := range idx.ComponentSchemas {
		assert.NotNil(t, schema.Location, "schema should have location")
		jp := schema.Location.ToJSONPointer()
		assert.NotEmpty(t, jp, "location should produce JSON pointer")
	}
}

func TestBuildIndex_Operations_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      summary: Get users
      responses:
        "200":
          description: Success
    post:
      operationId: createUser
      summary: Create user
      responses:
        "201":
          description: Created
  /products:
    get:
      operationId: getProducts
      summary: Get products
      responses:
        "200":
          description: Success
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Should have 3 operations indexed
	assert.Len(t, idx.Operations, 3, "should have 3 operations")
	// Should have 2 inline path items
	assert.Len(t, idx.InlinePathItems, 2, "should have 2 inline path items")
}

func TestBuildIndex_Parameters_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{id}:
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: integer
    get:
      operationId: getUser
      responses:
        "200":
          description: Success
      parameters:
        - $ref: '#/components/parameters/PageSize'
components:
  parameters:
    PageSize:
      name: pageSize
      in: query
      schema:
        type: integer
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Should have 1 component parameter (PageSize)
	assert.Len(t, idx.ComponentParameters, 1, "should have 1 component parameter")
	// Should have 1 inline parameter (id in path)
	assert.Len(t, idx.InlineParameters, 1, "should have 1 inline parameter")
	// Should have 1 parameter reference ($ref to PageSize)
	assert.Len(t, idx.ParameterReferences, 1, "should have 1 parameter reference")
}

func TestBuildIndex_Responses_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
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
                type: array
        "404":
          $ref: '#/components/responses/NotFound'
components:
  responses:
    NotFound:
      description: Not found
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Should have 1 component response (NotFound)
	assert.Len(t, idx.ComponentResponses, 1, "should have 1 component response")
	// Should have 1 inline response (200)
	assert.Len(t, idx.InlineResponses, 1, "should have 1 inline response")
	// Should have 1 response reference ($ref to NotFound)
	assert.Len(t, idx.ResponseReferences, 1, "should have 1 response reference")
}

func TestBuildIndex_RequestBodies_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    post:
      operationId: createUser
      requestBody:
        description: User to create
        content:
          application/json:
            schema:
              type: object
      responses:
        "201":
          description: Created
    put:
      operationId: updateUser
      requestBody:
        $ref: '#/components/requestBodies/UserBody'
      responses:
        "200":
          description: Updated
components:
  requestBodies:
    UserBody:
      description: User body
      content:
        application/json:
          schema:
            type: object
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Should have 1 component request body (UserBody)
	assert.Len(t, idx.ComponentRequestBodies, 1, "should have 1 component request body")
	// Should have 1 inline request body (POST)
	assert.Len(t, idx.InlineRequestBodies, 1, "should have 1 inline request body")
	// Should have 1 request body reference ($ref to UserBody)
	assert.Len(t, idx.RequestBodyReferences, 1, "should have 1 request body reference")
}

func TestBuildIndex_MediaTypes_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    post:
      operationId: createUser
      requestBody:
        content:
          application/json:
            schema:
              type: object
          application/xml:
            schema:
              type: object
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                type: object
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Should have 3 media types (2 in request, 1 in response)
	assert.Len(t, idx.MediaTypes, 3, "should have 3 media types")
}

func TestBuildIndex_Discriminator_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: petType
        mapping:
          dog: '#/components/schemas/Dog'
          cat: '#/components/schemas/Cat'
      properties:
        petType:
          type: string
    Dog:
      type: object
    Cat:
      type: object
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Should have 1 discriminator
	assert.Len(t, idx.Discriminators, 1, "should have 1 discriminator")
}

func TestBuildIndex_SecuritySchemes_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
    oauth2:
      type: oauth2
      flows:
        implicit:
          authorizationUrl: https://example.com/oauth/authorize
          scopes:
            read: Read access
            write: Write access
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Should have 2 component security schemes
	assert.Len(t, idx.ComponentSecuritySchemes, 2, "should have 2 component security schemes")
	// Should have 1 OAuth flows container
	assert.Len(t, idx.OAuthFlows, 1, "should have 1 OAuth flows")
	// Should have 1 OAuth flow item (implicit)
	assert.Len(t, idx.OAuthFlowItems, 1, "should have 1 OAuth flow item")
}
