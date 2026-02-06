package openapi_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
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

	// Should include boolean, inline, component, and external schemas (not references)
	totalExpected := len(idx.BooleanSchemas) + len(idx.InlineSchemas) +
		len(idx.ComponentSchemas) + len(idx.ExternalSchemas)
	assert.Len(t, allSchemas, totalExpected, "GetAllSchemas should return all schema types")
}

func TestBuildIndex_GetAllParameters_Success(t *testing.T) {
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
      parameters:
        - $ref: '#/components/parameters/PageSize'
      responses:
        "200":
          description: Success
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

	allParameters := idx.GetAllParameters()
	assert.NotEmpty(t, allParameters, "should have parameters")

	// Should include inline, component, and external parameters (not references)
	totalExpected := len(idx.InlineParameters) + len(idx.ComponentParameters) +
		len(idx.ExternalParameters)
	assert.Len(t, allParameters, totalExpected, "GetAllParameters should return all parameter types")
}

func TestBuildIndex_GetAllResponses_Success(t *testing.T) {
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

	allResponses := idx.GetAllResponses()
	assert.NotEmpty(t, allResponses, "should have responses")

	// Should include inline, component, and external responses (not references)
	totalExpected := len(idx.InlineResponses) + len(idx.ComponentResponses) +
		len(idx.ExternalResponses)
	assert.Len(t, allResponses, totalExpected, "GetAllResponses should return all response types")
}

func TestBuildIndex_GetAllRequestBodies_Success(t *testing.T) {
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

	allRequestBodies := idx.GetAllRequestBodies()
	assert.NotEmpty(t, allRequestBodies, "should have request bodies")

	// Should include inline, component, and external request bodies (not references)
	totalExpected := len(idx.InlineRequestBodies) + len(idx.ComponentRequestBodies) +
		len(idx.ExternalRequestBodies)
	assert.Len(t, allRequestBodies, totalExpected, "GetAllRequestBodies should return all request body types")
}

func TestBuildIndex_GetAllHeaders_Success(t *testing.T) {
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
          headers:
            X-Rate-Limit:
              description: Rate limit
              schema:
                type: integer
            X-Custom:
              $ref: '#/components/headers/CustomHeader'
components:
  headers:
    CustomHeader:
      description: Custom header
      schema:
        type: string
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	allHeaders := idx.GetAllHeaders()
	assert.NotEmpty(t, allHeaders, "should have headers")

	// Should include inline, component, and external headers (not references)
	totalExpected := len(idx.InlineHeaders) + len(idx.ComponentHeaders) +
		len(idx.ExternalHeaders)
	assert.Len(t, allHeaders, totalExpected, "GetAllHeaders should return all header types")
}

func TestBuildIndex_GetAllExamples_Success(t *testing.T) {
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
              examples:
                inline:
                  value: { id: 1 }
                referenced:
                  $ref: '#/components/examples/UserExample'
components:
  examples:
    UserExample:
      value: { id: 2 }
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	allExamples := idx.GetAllExamples()
	assert.NotEmpty(t, allExamples, "should have examples")

	// Should include inline, component, and external examples (not references)
	totalExpected := len(idx.InlineExamples) + len(idx.ComponentExamples) +
		len(idx.ExternalExamples)
	assert.Len(t, allExamples, totalExpected, "GetAllExamples should return all example types")
}

func TestBuildIndex_GetAllLinks_Success(t *testing.T) {
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
          links:
            GetUserById:
              operationId: getUsers
            ReferencedLink:
              $ref: '#/components/links/CustomLink'
  /products:
    get:
      operationId: getProducts
      responses:
        "200":
          description: Success
components:
  links:
    CustomLink:
      operationId: getProducts
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	allLinks := idx.GetAllLinks()
	assert.NotEmpty(t, allLinks, "should have links")

	// Should include inline, component, and external links (not references)
	totalExpected := len(idx.InlineLinks) + len(idx.ComponentLinks) +
		len(idx.ExternalLinks)
	assert.Len(t, allLinks, totalExpected, "GetAllLinks should return all link types")
}

func TestBuildIndex_GetAllCallbacks_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /subscribe:
    post:
      operationId: subscribe
      callbacks:
        onData:
          '{$request.body#/callbackUrl}':
            post:
              requestBody:
                description: Callback
                content:
                  application/json:
                    schema:
                      type: object
              responses:
                "200":
                  description: OK
        onComplete:
          $ref: '#/components/callbacks/CompleteCallback'
      responses:
        "201":
          description: Created
components:
  callbacks:
    CompleteCallback:
      '{$request.body#/callbackUrl}':
        post:
          requestBody:
            description: Complete callback
            content:
              application/json:
                schema:
                  type: object
          responses:
            "200":
              description: OK
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	allCallbacks := idx.GetAllCallbacks()
	assert.NotEmpty(t, allCallbacks, "should have callbacks")

	// Should include inline, component, and external callbacks (not references)
	totalExpected := len(idx.InlineCallbacks) + len(idx.ComponentCallbacks) +
		len(idx.ExternalCallbacks)
	assert.Len(t, allCallbacks, totalExpected, "GetAllCallbacks should return all callback types")
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

func TestBuildIndex_UnknownProperties_DetectedAsWarnings(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	tests := []struct {
		name                  string
		yaml                  string
		expectedWarningCount  int
		expectedWarningSubstr string
	}{
		{
			name: "MediaType with $ref property in OpenAPI 3.1",
			yaml: `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /vehicles:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              $ref: '#/components/schemas/VehiclesResponse'
components:
  schemas:
    VehiclesResponse:
      type: object
      properties:
        vehicles:
          type: array
`,
			expectedWarningCount:  1,
			expectedWarningSubstr: "unknown property `$ref`",
		},
		{
			name: "MediaType with schema property (valid)",
			yaml: `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /vehicles:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/VehiclesResponse'
components:
  schemas:
    VehiclesResponse:
      type: object
`,
			expectedWarningCount:  0,
			expectedWarningSubstr: "",
		},
		{
			name: "Operation with unknown property",
			yaml: `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      unknownField: value
      responses:
        "200":
          description: Success
`,
			expectedWarningCount:  1,
			expectedWarningSubstr: "unknown property `unknownField`",
		},
		{
			name: "Schema property with unknown keyword",
			yaml: `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              example: foobar
              properties:
                test:
                  type: string
                  description: Test
                  name: foo
      responses:
        "204":
          description: No content
`,
			expectedWarningCount:  1,
			expectedWarningSubstr: "unknown property `name`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := unmarshalOpenAPI(t, ctx, tt.yaml)
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})

			require.NotNil(t, idx, "index should not be nil")

			allErrors := idx.GetAllErrors()
			warnings := []error{}
			for _, err := range allErrors {
				var vErr *validation.Error
				if errors.As(err, &vErr) && vErr.Severity == validation.SeverityWarning {
					warnings = append(warnings, err)
				}
			}

			assert.Len(t, warnings, tt.expectedWarningCount, "should have expected number of warnings")

			if tt.expectedWarningCount > 0 {
				found := false
				for _, w := range warnings {
					if strings.Contains(w.Error(), tt.expectedWarningSubstr) {
						found = true
						break
					}
				}
				assert.True(t, found, "should have warning containing '%s'", tt.expectedWarningSubstr)
			}
		})
	}
}

func TestBuildIndex_UnknownProperties_Deduplicated_WhenComponentReferencedMultipleTimes(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Create a schema with an unknown property that is referenced from multiple operations
	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        "200":
          description: Get users
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/User'
      responses:
        "201":
          description: Created
  /admin/users:
    get:
      responses:
        "200":
          description: Get admin users
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/User'
components:
  schemas:
    User:
      type: object
      unknownField: this-should-trigger-warning
      properties:
        id:
          type: string
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

	// Get all warnings
	allErrors := idx.GetAllErrors()
	unknownPropWarnings := []error{}
	for _, err := range allErrors {
		var vErr *validation.Error
		if errors.As(err, &vErr) && vErr.Severity == validation.SeverityWarning {
			if strings.Contains(err.Error(), "unknown property `unknownField`") {
				unknownPropWarnings = append(unknownPropWarnings, err)
			}
		}
	}

	// Despite the User schema being referenced 3 times (in 3 different operations),
	// we should only get 1 warning for the unknown property
	assert.Len(t, unknownPropWarnings, 1, "should only have 1 warning for unknownField despite multiple references")
	assert.Contains(t, unknownPropWarnings[0].Error(), "unknown property `unknownField`", "warning should mention the unknown field")
}

func TestBuildIndex_CircularReferenceCounts_ValidCircular_Success(t *testing.T) {
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
    Node:
      type: object
      properties:
        value:
          type: string
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
	assert.Equal(t, 1, idx.GetValidCircularRefCount(), "should have 1 valid circular reference")
	assert.Equal(t, 0, idx.GetInvalidCircularRefCount(), "should have 0 invalid circular references")
	assert.Empty(t, idx.GetCircularReferenceErrors(), "should have no circular reference errors")
}

func TestBuildIndex_CircularReferenceCounts_InvalidCircular_Success(t *testing.T) {
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
    BadNode:
      type: object
      required:
        - next
      properties:
        next:
          $ref: '#/components/schemas/BadNode'
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.Equal(t, 0, idx.GetValidCircularRefCount(), "should have 0 valid circular references")
	assert.Equal(t, 1, idx.GetInvalidCircularRefCount(), "should have 1 invalid circular reference")
	assert.Len(t, idx.GetCircularReferenceErrors(), 1, "should have 1 circular reference error")
}

func TestBuildIndex_CircularReferenceCounts_MixedCirculars_Success(t *testing.T) {
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
    GoodNode:
      type: object
      properties:
        value:
          type: string
        next:
          $ref: '#/components/schemas/GoodNode'
    BadNode:
      type: object
      required:
        - next
      properties:
        next:
          $ref: '#/components/schemas/BadNode'
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.Equal(t, 1, idx.GetValidCircularRefCount(), "should have 1 valid circular reference")
	assert.Equal(t, 1, idx.GetInvalidCircularRefCount(), "should have 1 invalid circular reference")
	assert.Len(t, idx.GetCircularReferenceErrors(), 1, "should have 1 circular reference error")
}

func TestBuildIndex_CircularReferenceCounts_ArrayWithMinItems_Success(t *testing.T) {
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
    TreeNode:
      type: object
      properties:
        children:
          type: array
          items:
            $ref: '#/components/schemas/TreeNode'
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.Equal(t, 1, idx.GetValidCircularRefCount(), "should have 1 valid circular reference (empty array terminates)")
	assert.Equal(t, 0, idx.GetInvalidCircularRefCount(), "should have 0 invalid circular references")
	assert.Empty(t, idx.GetCircularReferenceErrors(), "should have no circular reference errors")
}

func TestBuildIndex_CircularReferenceCounts_NullableSchema_Success(t *testing.T) {
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
    NullableNode:
      type: object
      nullable: true
      required:
        - next
      properties:
        next:
          $ref: '#/components/schemas/NullableNode'
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.Equal(t, 1, idx.GetValidCircularRefCount(), "should have 1 valid circular reference (nullable terminates)")
	assert.Equal(t, 0, idx.GetInvalidCircularRefCount(), "should have 0 invalid circular references")
	assert.Empty(t, idx.GetCircularReferenceErrors(), "should have no circular reference errors")
}

func TestBuildIndex_CircularReferenceCounts_OneOfValid_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	tests := []struct {
		name                  string
		yaml                  string
		expectedValidCircular int
	}{
		{
			name: "oneOf with referenced schema",
			yaml: `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    PolyNode:
      oneOf:
        - type: string
        - $ref: '#/components/schemas/PolyNodeObject'
    PolyNodeObject:
      type: object
      properties:
        next:
          $ref: '#/components/schemas/PolyNode'
`,
			// 2 circular refs detected: one starting from PolyNode, one from PolyNodeObject
			// Both are part of the same cycle but detected at different entry points
			expectedValidCircular: 2,
		},
		{
			name: "oneOf with inline schema",
			yaml: `
openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    PolyNode:
      oneOf:
        - type: string
        - type: object
          properties:
            next:
              $ref: '#/components/schemas/PolyNode'
`,
			// 1 circular ref: PolyNode referencing itself
			expectedValidCircular: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := unmarshalOpenAPI(t, ctx, tt.yaml)
			idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
				RootDocument:   doc,
				TargetDocument: doc,
				TargetLocation: "test.yaml",
			})

			require.NotNil(t, idx, "index should not be nil")
			assert.Equal(t, tt.expectedValidCircular, idx.GetValidCircularRefCount(), "should have expected valid circular references")
			assert.Equal(t, 0, idx.GetInvalidCircularRefCount(), "should have 0 invalid circular references")
			assert.Empty(t, idx.GetCircularReferenceErrors(), "should have no circular reference errors")
		})
	}
}

func TestBuildIndex_CircularReferenceCounts_GettersWithNilIndex_Success(t *testing.T) {
	t.Parallel()

	var idx *openapi.Index = nil

	assert.Equal(t, 0, idx.GetValidCircularRefCount(), "should return 0 for nil index")
	assert.Equal(t, 0, idx.GetInvalidCircularRefCount(), "should return 0 for nil index")
}

func TestIndex_GetAllReferences_Success(t *testing.T) {
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
      parameters:
        - $ref: '#/components/parameters/UserIdParam'
      responses:
        '200':
          $ref: '#/components/responses/UserResponse'
      callbacks:
        statusUpdate:
          $ref: '#/components/callbacks/StatusCallback'
components:
  parameters:
    UserIdParam:
      name: userId
      in: query
      schema:
        type: string
  responses:
    UserResponse:
      description: User response
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/User'
          examples:
            user1:
              $ref: '#/components/examples/UserExample'
      headers:
        X-Custom:
          $ref: '#/components/headers/CustomHeader'
      links:
        self:
          $ref: '#/components/links/SelfLink'
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
        profile:
          $ref: '#/components/schemas/Profile'
    Profile:
      type: object
      properties:
        name:
          type: string
  examples:
    UserExample:
      value:
        id: "123"
  headers:
    CustomHeader:
      schema:
        type: string
  links:
    SelfLink:
      operationId: getUsers
  requestBodies:
    UserBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/User'
  callbacks:
    StatusCallback:
      '{$request.body#/callbackUrl}':
        post:
          requestBody:
            $ref: '#/components/requestBodies/UserBody'
          responses:
            '200':
              description: Callback response
  securitySchemes:
    ApiKey:
      type: apiKey
      in: header
      name: X-API-Key
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")
	assert.False(t, idx.HasErrors(), "should have no errors")

	// Get all references
	allRefs := idx.GetAllReferences()
	require.NotNil(t, allRefs, "GetAllReferences should not return nil")

	expectedRefCount := 10
	assert.Len(t, allRefs, expectedRefCount, "should have expected number of references")

	// Verify all returned nodes implement ReferenceNode interface
	for i, ref := range allRefs {
		assert.NotNil(t, ref, "reference at index %d should not be nil", i)
		assert.NotNil(t, ref.Node, "reference node at index %d should not be nil", i)

		// Verify it's actually a reference
		assert.True(t, ref.Node.IsReference(), "node at index %d should be a reference", i)

		// Verify it has a reference value
		refVal := ref.Node.GetReference()
		assert.NotEmpty(t, refVal, "node at index %d should have a reference value", i)
	}

	// Verify specific reference counts
	assert.Len(t, idx.SchemaReferences, 3, "should have 3 schema references")
	assert.Len(t, idx.ParameterReferences, 1, "should have 1 parameter reference")
	assert.Len(t, idx.ResponseReferences, 1, "should have 1 response reference")
	assert.Len(t, idx.ExampleReferences, 1, "should have 1 example reference")
	assert.Len(t, idx.HeaderReferences, 1, "should have 1 header reference")
	assert.Len(t, idx.LinkReferences, 1, "should have 1 link reference")
	assert.Len(t, idx.RequestBodyReferences, 1, "should have 1 request body reference")
	assert.Len(t, idx.CallbackReferences, 1, "should have 1 callback reference")
}

func TestIndex_GetAllReferences_EmptyDoc_Success(t *testing.T) {
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

	allRefs := idx.GetAllReferences()
	assert.Empty(t, allRefs, "should have no references in empty doc")
}

func TestIndex_GetAllReferences_NilIndex_Success(t *testing.T) {
	t.Parallel()

	var idx *openapi.Index = nil
	allRefs := idx.GetAllReferences()
	assert.Nil(t, allRefs, "should return nil for nil index")
}

func TestBuildIndex_CircularRef_OneOfSelfRefWithBaseCases_Valid(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// A recursive JSON-value-like type: oneOf with self-referencing branches (object/array)
	// AND non-recursive base-case branches (string/number/boolean).
	// Referenced from within an inline oneOf in a path response.
	// This should be VALID because the oneOf has non-recursive branches.
	yaml := `
openapi: "3.0.3"
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
                oneOf:
                  - type: object
                    properties:
                      data:
                        $ref: '#/components/schemas/JsonValue'
                  - type: object
                    properties:
                      items:
                        $ref: '#/components/schemas/JsonValue'
components:
  schemas:
    JsonValue:
      nullable: true
      oneOf:
        - type: string
        - type: number
        - type: object
          additionalProperties:
            $ref: '#/components/schemas/JsonValue'
        - type: array
          items:
            $ref: '#/components/schemas/JsonValue'
        - type: boolean
`
	doc := unmarshalOpenAPI(t, ctx, yaml)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "test.yaml",
	})

	require.NotNil(t, idx, "index should not be nil")

	circularErrs := idx.GetCircularReferenceErrors()
	assert.Empty(t, circularErrs, "oneOf with non-recursive base-case branches should be valid")
}
