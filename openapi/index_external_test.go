package openapi_test

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockVirtualFS implements system.VirtualFS for testing external file references
type MockVirtualFS struct {
	files map[string]string
}

func NewMockVirtualFS() *MockVirtualFS {
	return &MockVirtualFS{
		files: make(map[string]string),
	}
}

func (m *MockVirtualFS) AddFile(path, content string) {
	m.files[path] = content
}

func (m *MockVirtualFS) Open(name string) (fs.File, error) {
	content, exists := m.files[name]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", name)
	}
	return &MockFile{content: content}, nil
}

// MockFile implements fs.File
type MockFile struct {
	content string
	pos     int
}

func (m *MockFile) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.content) {
		return 0, io.EOF
	}
	n = copy(p, m.content[m.pos:])
	m.pos += n
	return n, nil
}

func (m *MockFile) Close() error {
	return nil
}

func (m *MockFile) Stat() (fs.FileInfo, error) {
	return nil, errors.New("not implemented")
}

// MockHTTPClient implements system.Client for testing external HTTP references
type MockHTTPClient struct {
	responses map[string]string
}

func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		responses: make(map[string]string),
	}
}

func (m *MockHTTPClient) AddResponse(url, body string) {
	m.responses[url] = body
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	url := req.URL.String()
	body, exists := m.responses[url]
	if !exists {
		return nil, fmt.Errorf("no response configured for URL: %s", url)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}, nil
}

// setupComprehensiveExternalRefs creates a complete test environment with:
// - File-based external references
// - HTTP-based external references
// - Valid and invalid circular references
// - Referenced and unreferenced schemas
func setupComprehensiveExternalRefs(t *testing.T) (*openapi.Index, *MockVirtualFS, *MockHTTPClient) {
	t.Helper()
	ctx := t.Context()

	vfs := NewMockVirtualFS()
	httpClient := NewMockHTTPClient()

	// Expected index counts (verified by tests):
	// ExternalDocumentation: 2 (main doc + users tag)
	// Tags: 2 (users, products)
	// Servers: 2 (production, staging)
	// ServerVariables: 1 (version variable)
	// BooleanSchemas: 2 (true, false from additionalProperties)
	// InlineSchemas: 10 (9 from external + 1 from LocalSchema.id property)
	// ComponentSchemas: 2 (LocalSchema, AnotherLocal)
	// ExternalSchemas: 6 (UserResponse, User, Address, Product, Category, TreeNode)
	// SchemaReferences: 9 (all $ref pointers including circulars)
	// CircularErrors: 1 (Product<->Category invalid circular)

	// TODO: PathItems indexing (currently marked TODO in buildIndex)

	// Main API document
	vfs.AddFile("/api/openapi.yaml", `
openapi: "3.1.0"
info:
  title: Comprehensive API
  version: 1.0.0
externalDocs:
  url: https://docs.example.com
  description: Main API Documentation
tags:
  - name: users
    description: User operations
    externalDocs:
      url: https://docs.example.com/users
  - name: products
    description: Product operations
servers:
  - url: https://api.example.com/{version}
    description: Production server
    variables:
      version:
        default: v1
        enum: [v1, v2]
  - url: https://staging.example.com
    description: Staging server
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "200":
          description: Users response
          content:
            application/json:
              schema:
                $ref: 'schemas/user.yaml#/UserResponse'
  /products:
    get:
      operationId: getProducts
      responses:
        "200":
          description: Products response
          content:
            application/json:
              schema:
                $ref: 'https://schemas.example.com/product.yaml#/Product'
  /trees:
    get:
      operationId: getTrees
      responses:
        "200":
          description: Trees response
          content:
            application/json:
              schema:
                $ref: 'schemas/tree.yaml#/TreeNode'
components:
  schemas:
    LocalSchema:
      type: object
      additionalProperties: true
      properties:
        id:
          type: integer
    AnotherLocal:
      type: object
      additionalProperties: false
`)

	// External file: User schemas with valid circular (optional property)
	vfs.AddFile("/api/schemas/user.yaml", `
UserResponse:
  type: object
  properties:
    user:
      $ref: '#/User'
User:
  type: object
  required: [id, name]
  properties:
    id:
      type: integer
    name:
      type: string
    address:
      $ref: '#/Address'
Address:
  type: object
  properties:
    street:
      type: string
    user:
      $ref: '#/User'
# Unreferenced schema in external file
UnreferencedUser:
  type: object
  properties:
    neverUsed:
      type: string
`)

	// External file: Tree with valid self-reference (array with minItems=0)
	vfs.AddFile("/api/schemas/tree.yaml", `
TreeNode:
  type: object
  properties:
    value:
      type: string
    children:
      type: array
      items:
        $ref: '#/TreeNode'
# Another unreferenced schema
UnusedTreeType:
  type: object
  properties:
    unusedProp:
      type: boolean
`)

	// Unreferenced file - nothing from here should appear in index
	vfs.AddFile("/api/schemas/completely-unreferenced.yaml", `
TotallyUnused:
  type: object
  properties:
    shouldNotAppear:
      type: string
`)

	// External HTTP: Product with invalid circular (required + minItems)
	httpClient.AddResponse("https://schemas.example.com/product.yaml", `
Product:
  type: object
  required: [id, category]
  properties:
    id:
      type: integer
    name:
      type: string
    category:
      $ref: '#/Category'
Category:
  type: object
  required: [products]
  properties:
    name:
      type: string
    products:
      type: array
      minItems: 1
      items:
        $ref: '#/Product'
# Unreferenced in HTTP document
UnreferencedCategory:
  type: object
  properties:
    alsoNeverUsed:
      type: integer
`)

	// Unmarshal and build index
	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(vfs.files["/api/openapi.yaml"]))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	resolveOpts := references.ResolveOptions{
		TargetLocation: "/api/openapi.yaml",
		RootDocument:   doc,
		TargetDocument: doc,
		VirtualFS:      vfs,
		HTTPClient:     httpClient,
	}
	idx := openapi.BuildIndex(ctx, doc, resolveOpts)
	require.NotNil(t, idx)

	return idx, vfs, httpClient
}

func TestBuildIndex_ExternalReferences_Comprehensive(t *testing.T) {
	t.Parallel()

	idx, _, _ := setupComprehensiveExternalRefs(t)

	tests := []struct {
		name      string
		assertion func(t *testing.T, idx *openapi.Index)
	}{
		{
			name: "external schemas count correct",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				// External schemas: UserResponse, User, Address, Product, Category, TreeNode (6)
				assert.Len(t, idx.ExternalSchemas, 6, "should have exactly 6 external schemas")
			},
		},
		{
			name: "external documentation count correct",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				// ExternalDocs: main doc + users tag
				assert.Len(t, idx.ExternalDocumentation, 2, "should have exactly 2 external documentation")
			},
		},
		{
			name: "tags count correct",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				// Tags: users, products
				assert.Len(t, idx.Tags, 2, "should have exactly 2 tags")
			},
		},
		{
			name: "servers count correct",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				// Servers: production, staging
				assert.Len(t, idx.Servers, 2, "should have exactly 2 servers")
			},
		},
		{
			name: "server variables count correct",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				// ServerVariables: version
				assert.Len(t, idx.ServerVariables, 1, "should have exactly 1 server variable")
			},
		},
		{
			name: "boolean schemas count correct",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				// BooleanSchemas: true, false from additionalProperties
				assert.Len(t, idx.BooleanSchemas, 2, "should have exactly 2 boolean schemas")
			},
		},
		{
			name: "component schemas count correct",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				// ComponentSchemas: LocalSchema, AnotherLocal
				assert.Len(t, idx.ComponentSchemas, 2, "should have exactly 2 component schemas")
			},
		},
		{
			name: "schema references count correct",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				// Schema references: 9 $ref pointers total
				assert.Len(t, idx.SchemaReferences, 9, "should have exactly 9 schema references")
			},
		},
		{
			name: "inline property schemas count correct",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				// Inline schemas: 9 from external + 1 from LocalSchema.id
				assert.Len(t, idx.InlineSchemas, 10, "should have exactly 10 inline schemas")
			},
		},
		{
			name: "inline path items count correct",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				// InlinePathItems: /users, /products, /trees
				assert.Len(t, idx.InlinePathItems, 3, "should have exactly 3 inline path items")
			},
		},
		{
			name: "operations count correct",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				// Operations: getUsers, getProducts, getTrees
				assert.Len(t, idx.Operations, 3, "should have exactly 3 operations")
			},
		},
		{
			name: "inline responses count correct",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				// InlineResponses: 200 response for each operation
				assert.Len(t, idx.InlineResponses, 3, "should have exactly 3 inline responses")
			},
		},
		{
			name: "circular error count correct",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				// Should detect 1 invalid circular: Product<->Category
				assert.Len(t, idx.GetCircularReferenceErrors(), 1, "should have exactly 1 circular error")
			},
		},
		{
			name: "no errors for valid references",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				// Should have no resolution errors
				assert.Empty(t, idx.GetResolutionErrors(), "should have no resolution errors")
				assert.Empty(t, idx.GetValidationErrors(), "should have no validation errors")
			},
		},
		{
			name: "unreferenced schemas in external files not indexed",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				for _, schema := range idx.GetAllSchemas() {
					loc := string(schema.Location.ToJSONPointer())
					assert.NotContains(t, loc, "UnreferencedUser", "UnreferencedUser should not be indexed")
					assert.NotContains(t, loc, "UnusedTreeType", "UnusedTreeType should not be indexed")
					assert.NotContains(t, loc, "TotallyUnused", "TotallyUnused should not be indexed")
					assert.NotContains(t, loc, "UnreferencedCategory", "UnreferencedCategory should not be indexed")
				}
			},
		},
		{
			name: "valid circular reference via optional property",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				circularErrs := idx.GetCircularReferenceErrors()
				for _, err := range circularErrs {
					errStr := err.Error()
					// User<->Address should not have circular error (address is optional)
					if strings.Contains(errStr, "User") && strings.Contains(errStr, "Address") {
						t.Errorf("User<->Address circular via optional property should be valid, got error: %v", err)
					}
				}
			},
		},
		{
			name: "valid circular reference via array minItems=0",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				circularErrs := idx.GetCircularReferenceErrors()
				for _, err := range circularErrs {
					errStr := err.Error()
					// TreeNode self-reference should not have circular error
					if strings.Contains(errStr, "TreeNode") {
						t.Errorf("TreeNode self-reference via array should be valid, got error: %v", err)
					}
				}
			},
		},
		{
			name: "schema references tracked with locations",
			assertion: func(t *testing.T, idx *openapi.Index) {
				t.Helper()
				assert.NotEmpty(t, idx.SchemaReferences, "should have schema references")
				for _, ref := range idx.SchemaReferences {
					assert.NotNil(t, ref.Location, "reference should have location")
					assert.NotNil(t, ref.Node, "reference should have node")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.assertion(t, idx)
		})
	}
}

// TestExternalPathItemReferencesWithOperations verifies that:
// 1. External path item references are resolved correctly
// 2. Operations within external path items are indexed
// 3. Walk descends into resolved external path items
func TestExternalPathItemReferencesWithOperations(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Create external file with path items containing operations
	externalSpec := `
a:
  get:
    operationId: op-a
    responses:
      '200':
        description: OK
  post:
    operationId: op-a-post
    responses:
      '201':
        description: Created
b:
  get:
    operationId: op-b
    responses:
      '200':
        description: OK
`

	// Create main spec that references the external path items
	mainSpec := `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /a:
    $ref: "./external.yaml#/a"
  /b:
    $ref: "./external.yaml#/b"
`

	// Parse main document
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(mainSpec))
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Setup virtual filesystem with external file
	// Use absolute path and matching reference in spec
	vfs := NewMockVirtualFS()
	vfs.AddFile("/test/external.yaml", externalSpec)

	// Build index with external reference resolution
	resolveOpts := references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "/test/main.yaml", // Absolute path so relative refs resolve correctly
		VirtualFS:      vfs,
	}

	idx := openapi.BuildIndex(ctx, doc, resolveOpts)
	require.NotNil(t, idx)

	// Verify external path item references were resolved
	assert.Len(t, idx.PathItemReferences, 2, "should have 2 external path item references")

	// Verify operations from external path items are indexed
	assert.Len(t, idx.Operations, 3, "should have 3 operations (2 from /a, 1 from /b)")

	// Verify operation IDs are correct
	operationIDs := make([]string, len(idx.Operations))
	for i, op := range idx.Operations {
		operationIDs[i] = op.Node.GetOperationID()
	}
	assert.Contains(t, operationIDs, "op-a", "should contain op-a")
	assert.Contains(t, operationIDs, "op-a-post", "should contain op-a-post")
	assert.Contains(t, operationIDs, "op-b", "should contain op-b")

	// Verify no resolution errors
	assert.Empty(t, idx.GetResolutionErrors(), "should have no resolution errors")
}

// TestExternalReferencedComponentsWithinOperations verifies that:
// 1. External parameter, requestBody, response, header, and example references are resolved
// 2. Walk descends into resolved external references within operations
// 3. Referenced components are properly indexed
func TestExternalReferencedComponentsWithinOperations(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Create external file with reusable components
	componentsSpec := `
UserParam:
  name: userId
  in: path
  required: true
  schema:
    type: string

CreateRequest:
  required: true
  content:
    application/json:
      schema:
        type: object
        properties:
          name:
            type: string

SuccessResponse:
  description: Success
  headers:
    X-Request-ID:
      description: Request ID header
      schema:
        type: string
  content:
    application/json:
      schema:
        type: object
      examples:
        example1:
          value:
            status: success
`

	// Create main spec with operations that reference external components
	mainSpec := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    parameters:
      - $ref: "./components.yaml#/UserParam"
    post:
      operationId: createUser
      requestBody:
        $ref: "./components.yaml#/CreateRequest"
      responses:
        '200':
          $ref: "./components.yaml#/SuccessResponse"
`

	// Parse main document
	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(mainSpec))
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Setup virtual filesystem
	vfs := NewMockVirtualFS()
	vfs.AddFile("/test/components.yaml", componentsSpec)

	// Build index
	resolveOpts := references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: "/test/main.yaml",
		VirtualFS:      vfs,
	}

	idx := openapi.BuildIndex(ctx, doc, resolveOpts)
	require.NotNil(t, idx)

	// Verify operations were indexed
	assert.Len(t, idx.Operations, 1, "should have 1 operation")

	// Verify external parameter reference was resolved and indexed
	assert.NotEmpty(t, idx.ParameterReferences, "should have parameter references")

	// Verify external request body reference was resolved
	assert.NotEmpty(t, idx.RequestBodyReferences, "should have request body references")

	// Verify external response reference was resolved
	assert.NotEmpty(t, idx.ResponseReferences, "should have response references")

	// Verify headers within resolved response are indexed (inline headers, not references)
	assert.NotEmpty(t, idx.InlineHeaders, "should have inline headers from resolved response")

	// Verify examples within resolved response are indexed (inline examples, not references)
	assert.NotEmpty(t, idx.InlineExamples, "should have inline examples from resolved response")

	// Verify no resolution errors
	assert.Empty(t, idx.GetResolutionErrors(), "should have no resolution errors")
}
func TestBuildIndex_ExternalReferencesForAllTypes_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	vfs := NewMockVirtualFS()

	// Main API document with references to external components
	vfs.AddFile("/api/openapi.yaml", `
openapi: "3.1.0"
info:
  title: External Components Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      parameters:
        - $ref: 'components.yaml#/PageSize'
      responses:
        "200":
          $ref: 'components.yaml#/UsersResponse'
      callbacks:
        onUpdate:
          $ref: 'components.yaml#/UpdateCallback'
    post:
      operationId: createUser
      requestBody:
        $ref: 'components.yaml#/UserRequestBody'
      responses:
        "201":
          description: Created
`)

	// External components file with all types at top level
	vfs.AddFile("/api/components.yaml", `
PageSize:
  name: pageSize
  in: query
  schema:
    type: integer

UsersResponse:
  description: Users response
  headers:
    X-Total-Count:
      $ref: '#/TotalCountHeader'
  content:
    application/json:
      schema:
        type: array
        items:
          type: object
      examples:
        singleUser:
          $ref: '#/SingleUserExample'
  links:
    GetUserById:
      $ref: '#/UserLink'

UserRequestBody:
  description: User request body
  content:
    application/json:
      schema:
        type: object

TotalCountHeader:
  description: Total count header
  schema:
    type: integer

SingleUserExample:
  value:
    id: 1
    name: John Doe

UserLink:
  operationId: getUsers
  description: Link to get users

UpdateCallback:
  '{$request.body#/callbackUrl}':
    post:
      requestBody:
        description: Update notification
        content:
          application/json:
            schema:
              type: object
      responses:
        "200":
          description: OK
`)

	// Unmarshal and build index
	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(vfs.files["/api/openapi.yaml"]))
	require.NoError(t, err)
	require.Empty(t, validationErrs)

	resolveOpts := references.ResolveOptions{
		TargetLocation: "/api/openapi.yaml",
		RootDocument:   doc,
		TargetDocument: doc,
		VirtualFS:      vfs,
	}
	idx := openapi.BuildIndex(ctx, doc, resolveOpts)
	require.NotNil(t, idx)

	// Test External Parameters
	assert.Len(t, idx.ExternalParameters, 1, "should have 1 external parameter (PageSize)")
	assert.Len(t, idx.ParameterReferences, 1, "should have 1 parameter reference")
	assert.Empty(t, idx.ComponentParameters, "should have 0 component parameters (PageSize is external)")
	assert.Empty(t, idx.InlineParameters, "should have 0 inline parameters")

	// Test External Responses
	assert.Len(t, idx.ExternalResponses, 1, "should have 1 external response (UsersResponse)")
	assert.Len(t, idx.ResponseReferences, 1, "should have 1 response reference")
	assert.Empty(t, idx.ComponentResponses, "should have 0 component responses (UsersResponse is external)")
	assert.Len(t, idx.InlineResponses, 2, "should have 2 inline responses (201 Created + default from callback)")

	// Test External RequestBodies
	assert.Len(t, idx.ExternalRequestBodies, 1, "should have 1 external request body (UserRequestBody)")
	assert.Len(t, idx.RequestBodyReferences, 1, "should have 1 request body reference")
	assert.Empty(t, idx.ComponentRequestBodies, "should have 0 component request bodies")
	assert.Len(t, idx.InlineRequestBodies, 1, "should have 1 inline request body (from callback)")

	// Test External Headers
	// FIXED: Header references inside external files CAN now be resolved!
	assert.Len(t, idx.ExternalHeaders, 1, "should have 1 external header (TotalCountHeader)")
	assert.Len(t, idx.HeaderReferences, 1, "should have 1 header reference")
	assert.Empty(t, idx.ComponentHeaders, "should have 0 component headers")
	assert.Empty(t, idx.InlineHeaders, "should have 0 inline headers")

	// Test External Examples
	// FIXED: Example references inside external files CAN now be resolved!
	assert.Len(t, idx.ExternalExamples, 1, "should have 1 external example (SingleUserExample)")
	assert.Len(t, idx.ExampleReferences, 1, "should have 1 example reference")
	assert.Empty(t, idx.ComponentExamples, "should have 0 component examples")
	assert.Empty(t, idx.InlineExamples, "should have 0 inline examples")

	// Test External Links
	// FIXED: Link references inside external files CAN now be resolved!
	assert.Len(t, idx.ExternalLinks, 1, "should have 1 external link (UserLink)")
	assert.Len(t, idx.LinkReferences, 1, "should have 1 link reference")
	assert.Empty(t, idx.ComponentLinks, "should have 0 component links")
	assert.Empty(t, idx.InlineLinks, "should have 0 inline links")

	// Test External Callbacks
	assert.Len(t, idx.ExternalCallbacks, 1, "should have 1 external callback (UpdateCallback)")
	assert.Len(t, idx.CallbackReferences, 1, "should have 1 callback reference")
	assert.Empty(t, idx.ComponentCallbacks, "should have 0 component callbacks")
	assert.Empty(t, idx.InlineCallbacks, "should have 0 inline callbacks")

	// Test GetAll* methods include external items (but not references)
	allParameters := idx.GetAllParameters()
	assert.Len(t, allParameters, 1, "GetAllParameters should return external (not reference)")

	allResponses := idx.GetAllResponses()
	assert.Len(t, allResponses, 3, "GetAllResponses should return external + 2 inline (not reference)")

	allRequestBodies := idx.GetAllRequestBodies()
	assert.Len(t, allRequestBodies, 2, "GetAllRequestBodies should return external + inline (not reference)")

	allHeaders := idx.GetAllHeaders()
	assert.Len(t, allHeaders, 1, "GetAllHeaders should have 1 (TotalCountHeader - internal refs now work!)")

	allExamples := idx.GetAllExamples()
	assert.Len(t, allExamples, 1, "GetAllExamples should have 1 (SingleUserExample - internal refs now work!)")

	allLinks := idx.GetAllLinks()
	assert.Len(t, allLinks, 1, "GetAllLinks should have 1 (UserLink - internal refs now work!)")

	allCallbacks := idx.GetAllCallbacks()
	assert.Len(t, allCallbacks, 1, "GetAllCallbacks should return external (not reference)")

	// FIXED: No more resolution errors! Internal references in external files now work correctly
	assert.False(t, idx.HasErrors(), "should have no errors after multi-file reference fix")
	assert.Empty(t, idx.GetResolutionErrors(), "should have 0 resolution errors (bug is fixed!)")
}
func TestDebugExternalParameter(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	vfs := NewMockVirtualFS()

	// Main document
	vfs.AddFile("/api/main.yaml", `
openapi: "3.1.0"
info:
  title: Test
  version: 1.0.0
paths:
  /test:
    get:
      operationId: test
      parameters:
        - $ref: 'external.yaml#/PageSize'
      responses:
        "200":
          description: OK
`)

	// External parameter
	vfs.AddFile("/api/external.yaml", `
PageSize:
  name: pageSize
  in: query
  schema:
    type: integer
`)

	doc, _, err := openapi.Unmarshal(ctx, strings.NewReader(vfs.files["/api/main.yaml"]))
	require.NoError(t, err)

	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		TargetLocation: "/api/main.yaml",
		RootDocument:   doc,
		TargetDocument: doc,
		VirtualFS:      vfs,
	})

	t.Logf("ComponentParameters: %d", len(idx.ComponentParameters))
	t.Logf("ExternalParameters: %d", len(idx.ExternalParameters))
	t.Logf("InlineParameters: %d", len(idx.InlineParameters))
	t.Logf("ParameterReferences: %d", len(idx.ParameterReferences))

	if len(idx.ExternalParameters) > 0 {
		t.Logf("External parameter location: %s", idx.ExternalParameters[0].Location.ToJSONPointer())
	}
	if len(idx.InlineParameters) > 0 {
		t.Logf("Inline parameter location: %s", idx.InlineParameters[0].Location.ToJSONPointer())
	}

	t.Logf("Errors: %v", idx.HasErrors())
	for _, err := range idx.GetAllErrors() {
		t.Logf("Error: %v", err)
	}
}
