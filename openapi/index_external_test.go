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
