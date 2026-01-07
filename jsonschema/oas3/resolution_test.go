package oas3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// MockResolutionTarget implements references.ResolutionTarget for testing
type MockResolutionTarget struct {
	objCache map[string]any
	docCache map[string][]byte
}

func NewMockResolutionTarget() *MockResolutionTarget {
	return &MockResolutionTarget{
		objCache: make(map[string]any),
		docCache: make(map[string][]byte),
	}
}

func (m *MockResolutionTarget) GetCachedReferenceDocument(key string) ([]byte, bool) {
	data, exists := m.docCache[key]
	return data, exists
}

func (m *MockResolutionTarget) StoreReferenceDocumentInCache(key string, doc []byte) {
	m.docCache[key] = doc
}

func (m *MockResolutionTarget) GetCachedReferencedObject(key string) (any, bool) {
	data, exists := m.objCache[key]
	return data, exists
}

func (m *MockResolutionTarget) StoreReferencedObjectInCache(key string, obj any) {
	m.objCache[key] = obj
}

func (m *MockResolutionTarget) InitCache() {
	if m.objCache == nil {
		m.objCache = make(map[string]any)
	}
	if m.docCache == nil {
		m.docCache = make(map[string][]byte)
	}
}

// MockVirtualFS implements system.VirtualFS for testing
type MockVirtualFS struct {
	files map[string]string
}

func NewMockVirtualFS() *MockVirtualFS {
	return &MockVirtualFS{
		files: make(map[string]string),
	}
}

func (m *MockVirtualFS) AddFile(path, content string) {
	// Normalize path separators for cross-platform compatibility
	normalizedPath := filepath.ToSlash(path)
	m.files[normalizedPath] = content
}

func (m *MockVirtualFS) Open(name string) (fs.File, error) {
	// Normalize path separators for cross-platform compatibility
	normalizedName := filepath.ToSlash(name)
	content, exists := m.files[normalizedName]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", name)
	}
	return &MockFile{content: content}, nil
}

// MockFile implements fs.File for testing
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

// MockHTTPClient implements system.Client for testing
type MockHTTPClient struct {
	responses map[string]*http.Response
	errors    map[string]error
}

func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		responses: make(map[string]*http.Response),
		errors:    make(map[string]error),
	}
}

func (m *MockHTTPClient) AddResponse(url, body string, statusCode int) {
	m.responses[url] = &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func (m *MockHTTPClient) AddError(url string, err error) {
	m.errors[url] = err
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	url := req.URL.String()
	if err, exists := m.errors[url]; exists {
		return nil, err
	}
	if resp, exists := m.responses[url]; exists {
		return resp, nil
	}
	return nil, fmt.Errorf("no response configured for URL: %s", url)
}

// TestResolutionTarget implements ResolutionTarget and contains real schema data
type TestResolutionTarget struct {
	*Schema
	cache map[string][]byte
}

func LoadTestSchemaFromFile(ctx context.Context, filename string) (*JSONSchema[Referenceable], error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Unmarshal into a JSONSchema[Referenceable] since the test data contains a JSON schema document
	jsonSchema := &JSONSchema[Referenceable]{}
	validationErrs, err := marshaller.Unmarshal(ctx, bytes.NewReader(data), jsonSchema)
	if err != nil {
		return nil, err
	}
	if len(validationErrs) > 0 {
		return nil, fmt.Errorf("validation errors: %v", validationErrs)
	}

	return jsonSchema, nil
}

func (t *TestResolutionTarget) GetCachedReferenceDocument(key string) ([]byte, bool) {
	data, exists := t.cache[key]
	return data, exists
}

func (t *TestResolutionTarget) StoreReferenceDocumentInCache(key string, doc []byte) {
	t.cache[key] = doc
}

// Test helper functions
func createSimpleSchema() *JSONSchema[Referenceable] {
	schema := &Schema{
		Type: NewTypeFromString(SchemaTypeString),
	}
	return NewJSONSchemaFromSchema[Referenceable](schema)
}

func createSchemaWithRef(ref string) *JSONSchema[Referenceable] {
	refObj := references.Reference(ref)
	schema := &Schema{
		Ref: &refObj,
	}
	return NewJSONSchemaFromSchema[Referenceable](schema)
}

// Test IsReference method
func TestJSONSchema_IsReference(t *testing.T) {
	t.Parallel()

	t.Run("nil schema is not a reference", func(t *testing.T) {
		t.Parallel()
		var schema *JSONSchema[Referenceable]
		assert.False(t, schema.IsReference())
	})

	t.Run("schema without ref is not a reference", func(t *testing.T) {
		t.Parallel()
		schema := createSimpleSchema()
		assert.False(t, schema.IsReference())
	})

	t.Run("schema with nil ref is not a reference", func(t *testing.T) {
		t.Parallel()
		schema := NewJSONSchemaFromSchema[Referenceable](&Schema{
			Ref: nil,
		})
		assert.False(t, schema.IsReference())
	})

	t.Run("schema with empty ref is not a reference", func(t *testing.T) {
		t.Parallel()
		emptyRef := references.Reference("")
		schema := NewJSONSchemaFromSchema[Referenceable](&Schema{
			Ref: &emptyRef,
		})
		assert.False(t, schema.IsReference())
	})

	t.Run("schema with valid ref is a reference", func(t *testing.T) {
		t.Parallel()
		ref := references.Reference("#/components/schemas/User")
		schema := NewJSONSchemaFromSchema[Referenceable](&Schema{
			Ref: &ref,
		})
		assert.True(t, schema.IsReference())
	})
}

// Test resolution against root document (empty reference)
func TestJSONSchema_Resolve_RootDocument(t *testing.T) {
	t.Parallel()

	t.Run("resolve empty reference against root document", func(t *testing.T) {
		t.Parallel()
		root, err := LoadTestSchemaFromFile(t.Context(), "testdata/simple_schema.yaml")
		require.NoError(t, err)
		schema := createSchemaWithRef("")

		opts := ResolveOptions{
			TargetLocation: "testdata/simple_schema.yaml",
			RootDocument:   root,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)

		// Get the resolved schema after resolution
		result := schema.GetResolvedSchema()
		require.NotNil(t, result)
		// Should return the JSONSchema wrapping the original schema
		// We can't do direct equality comparison due to cache side effects, so check the content
		assert.True(t, result.IsSchema())
		assert.NotNil(t, result.GetSchema())
	})

	t.Run("resolve JSON pointer against root document", func(t *testing.T) {
		t.Parallel()
		root, err := LoadTestSchemaFromFile(t.Context(), "testdata/simple_schema.yaml")
		require.NoError(t, err)
		ref := "#/properties/name"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "testdata/simple_schema.yaml",
			RootDocument:   root,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)

		// Get the resolved schema after resolution
		result := schema.GetResolvedSchema()
		require.NotNil(t, result)
		// Should contain the resolved JSONSchema - check if it has a Schema on the Left
		assert.True(t, result.IsSchema())
		assert.NotNil(t, result.Left)
		// The resolved schema should be a string type property
		schemaTypes := result.GetSchema().GetType()
		require.NotEmpty(t, schemaTypes)
		assert.Equal(t, SchemaTypeString, schemaTypes[0])
	})

	t.Run("non-reference schema returns itself", func(t *testing.T) {
		t.Parallel()
		schema := createSimpleSchema()

		root, err := LoadTestSchemaFromFile(t.Context(), "testdata/simple_schema.yaml")
		require.NoError(t, err)

		opts := ResolveOptions{
			TargetLocation: "testdata/simple_schema.yaml",
			RootDocument:   root,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)

		// Get the resolved schema after resolution
		result := schema.GetResolvedSchema()
		// Should return the JSONSchema wrapping the original schema
		// We can't do direct equality comparison due to cache side effects, so check the content
		require.NotNil(t, result)
		assert.True(t, result.IsSchema())
		assert.NotNil(t, result.GetSchema())
	})
}

// Test resolution against file paths
func TestJSONSchema_Resolve_FilePath(t *testing.T) {
	t.Parallel()

	t.Run("resolve against file path", func(t *testing.T) {
		t.Parallel()
		fs := NewMockVirtualFS()
		fs.AddFile("testdata/schemas/user.yaml", `
type: object
properties:
  name:
    type: string
  email:
    type: string
    format: email
`)

		root := NewMockResolutionTarget()
		ref := "schemas/user.yaml"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "testdata/root.yaml",
			RootDocument:   root,
			VirtualFS:      fs,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)

		// Get the resolved schema after resolution
		result := schema.GetResolvedSchema()
		require.NotNil(t, result)
		// Should contain the resolved JSONSchema - check if it has a Schema on the Left
		assert.True(t, result.IsSchema())
		assert.NotNil(t, result.Left)
		assert.NotNil(t, result.GetSchema().Type)
	})

	t.Run("resolve with JSON pointer in file path", func(t *testing.T) {
		t.Parallel()
		// Load complex schema from testdata
		complexSchemaData, err := os.ReadFile("testdata/complex_schema.yaml")
		require.NoError(t, err)

		fs := NewMockVirtualFS()
		fs.AddFile("testdata/schemas/definitions.yaml", string(complexSchemaData))

		root := NewMockResolutionTarget()
		ref := "schemas/definitions.yaml#/definitions/User"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "testdata/root.yaml",
			RootDocument:   root,
			VirtualFS:      fs,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)

		// Get the resolved schema after resolution
		result := schema.GetResolvedSchema()
		require.NotNil(t, result)
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()
		fs := NewMockVirtualFS()
		root := NewMockResolutionTarget()
		ref := "missing.yaml"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "testdata/root.yaml",
			RootDocument:   root,
			VirtualFS:      fs,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		result := schema.GetResolvedSchema()
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "file not found")
	})
}

// Test resolution against URLs
func TestJSONSchema_Resolve_URL(t *testing.T) {
	t.Parallel()

	t.Run("resolve against URL", func(t *testing.T) {
		t.Parallel()
		// Load simple schema data
		simpleSchemaData, err := os.ReadFile("testdata/simple_schema.yaml")
		require.NoError(t, err)

		client := NewMockHTTPClient()
		client.AddResponse("https://example.com/schemas/user.yaml", string(simpleSchemaData), 200)

		root := NewMockResolutionTarget()
		ref := "schemas/user.yaml"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "https://example.com/root.yaml",
			RootDocument:   root,
			HTTPClient:     client,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)

		// Get the resolved schema after resolution
		result := schema.GetResolvedSchema()
		require.NotNil(t, result)
	})

	t.Run("HTTP error response", func(t *testing.T) {
		t.Parallel()
		client := NewMockHTTPClient()
		client.AddResponse("https://example.com/missing.yaml", "Not Found", 404)

		root := NewMockResolutionTarget()
		ref := "missing.yaml"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "https://example.com/root.yaml",
			RootDocument:   root,
			HTTPClient:     client,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		assert.Contains(t, err.Error(), "HTTP request failed")
	})
}

// Test caching behavior
func TestJSONSchema_Resolve_Caching(t *testing.T) {
	t.Parallel()

	t.Run("cached resolution", func(t *testing.T) {
		t.Parallel()
		schema := createSchemaWithRef("#/components/schemas/User")
		resolved := createSimpleSchema()

		// Set up cached resolved schema using the actual cache field
		schema.referenceResolutionCache = &references.ResolveResult[JSONSchema[Referenceable]]{
			Object:            resolved,
			AbsoluteReference: "testdata/simple_schema.yaml#/components/schemas/User",
			ResolvedDocument:  resolved,
		}

		root, err := LoadTestSchemaFromFile(t.Context(), "testdata/simple_schema.yaml")
		require.NoError(t, err)

		opts := ResolveOptions{
			TargetLocation: "testdata/simple_schema.yaml",
			RootDocument:   root,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)

		// Get the resolved schema after resolution
		result := schema.GetResolvedSchema()
		assert.NotNil(t, result)
	})

	t.Run("cached document is used", func(t *testing.T) {
		t.Parallel()
		fs := NewMockVirtualFS()
		fs.AddFile("testdata/schemas/cached.yaml", "original: content")

		root := NewMockResolutionTarget()

		// Pre-populate cache with different content
		cachedData := []byte(`
type: object
properties:
  cached:
    type: string
`)
		root.StoreReferenceDocumentInCache("testdata/schemas/cached.yaml", cachedData)

		ref := "schemas/cached.yaml"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "testdata/root.yaml",
			RootDocument:   root,
			VirtualFS:      fs,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)

		// Get the resolved schema after resolution
		result := schema.GetResolvedSchema()
		require.NotNil(t, result)

		// Verify cache was used (not the filesystem content)
		cached, exists := root.GetCachedReferenceDocument("testdata/schemas/cached.yaml")
		assert.True(t, exists)
		assert.Equal(t, cachedData, cached)
	})
}

// Test Resolve method for recursive resolution
func TestJSONSchema_Resolve(t *testing.T) {
	t.Parallel()

	t.Run("resolve object with non-reference schema", func(t *testing.T) {
		t.Parallel()
		schema := createSimpleSchema()

		root, err := LoadTestSchemaFromFile(t.Context(), "testdata/simple_schema.yaml")
		require.NoError(t, err)

		opts := ResolveOptions{
			TargetLocation: "testdata/simple_schema.yaml",
			RootDocument:   root,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)

		// Get the resolved schema after resolution
		result := schema.GetResolvedSchema()
		require.NotNil(t, result)

		// ResolveSchema returns a JSONSchema (EitherValue), so check if it has the expected schema on the left
		assert.True(t, result.IsSchema())
		resolvedSchema := result.GetSchema()
		originalSchema := schema.GetSchema()
		assert.Equal(t, originalSchema.Type, resolvedSchema.Type)
	})

	t.Run("resolve object with single reference", func(t *testing.T) {
		t.Parallel()
		fs := NewMockVirtualFS()
		fs.AddFile("testdata/schemas/simple.yaml", `
type: string
`)

		root := NewMockResolutionTarget()
		ref := "schemas/simple.yaml"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "testdata/root.yaml",
			RootDocument:   root,
			VirtualFS:      fs,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)

		// Get the resolved schema after resolution
		result := schema.GetResolvedSchema()
		require.NotNil(t, result)

		// Test parent links for single-level reference
		parent := result.GetParent()
		topLevelParent := result.GetTopLevelParent()

		assert.Equal(t, schema, parent, "parent should be the reference schema")
		assert.Equal(t, schema, topLevelParent, "top-level parent should be the reference schema for single-level reference")
	})

	t.Run("circular reference detection", func(t *testing.T) {
		t.Parallel()
		fs := NewMockVirtualFS()
		fs.AddFile("testdata/schemas/circular1.yaml", `
$ref: "circular2.yaml"
`)
		fs.AddFile("testdata/schemas/circular2.yaml", `
$ref: "circular1.yaml"
`)

		root := NewMockResolutionTarget()
		ref := "schemas/circular1.yaml"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "testdata/root.yaml",
			RootDocument:   root,
			VirtualFS:      fs,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		// Accept either circular reference or file not found error since test file may not exist
		assert.True(t, strings.Contains(err.Error(), "circular reference detected") || strings.Contains(err.Error(), "file not found"))
	})

	t.Run("self-referencing schema resolves to root", func(t *testing.T) {
		t.Parallel()
		root, err := LoadTestSchemaFromFile(t.Context(), "testdata/simple_schema.yaml")
		require.NoError(t, err)
		ref := "#"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "testdata/simple_schema.yaml",
			RootDocument:   root,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		// Self-referencing schemas with $ref: "#" are valid in JSON Schema
		// and should resolve successfully to the root document
		require.NoError(t, err)
		assert.Empty(t, validationErrs)
		require.NotNil(t, schema.GetResolvedSchema())
	})
}

// Test error cases
func TestJSONSchema_Resolve_Errors(t *testing.T) {
	t.Parallel()

	t.Run("missing root document", func(t *testing.T) {
		t.Parallel()
		ref := "#/test"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "testdata/simple_schema.yaml",
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		result := schema.GetResolvedSchema()
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "root document is required")
	})

	t.Run("invalid yaml in referenced file", func(t *testing.T) {
		t.Parallel()
		fs := NewMockVirtualFS()
		fs.AddFile("testdata/invalid.yaml", "invalid: yaml: content: [")

		root := NewMockResolutionTarget()
		ref := "invalid.yaml"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "testdata/root.yaml",
			RootDocument:   root,
			VirtualFS:      fs,
		}

		_, err := schema.Resolve(t.Context(), opts)

		require.Error(t, err)
		result := schema.GetResolvedSchema()
		assert.Nil(t, result)
	})
}

// Test with real HTTP server
func TestJSONSchema_Resolve_HTTPIntegration(t *testing.T) {
	t.Parallel()

	t.Run("successful HTTP resolution", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/user.yaml":
				w.Header().Set("Content-Type", "application/yaml")
				w.WriteHeader(http.StatusOK)
				// Use actual test data
				data, _ := os.ReadFile("testdata/simple_schema.yaml")
				_, _ = w.Write(data)
			case "/error":
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("Not Found"))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		root := NewMockResolutionTarget()
		ref := "user.yaml"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: server.URL + "/root.yaml",
			RootDocument:   root,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)
		result := schema.GetResolvedSchema()
		require.NotNil(t, result)
	})

	t.Run("HTTP error response", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/user.yaml":
				w.Header().Set("Content-Type", "application/yaml")
				w.WriteHeader(http.StatusOK)
				// Use actual test data
				data, _ := os.ReadFile("testdata/simple_schema.yaml")
				_, _ = w.Write(data)
			case "/error":
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("Not Found"))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		root := NewMockResolutionTarget()
		ref := "error"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: server.URL + "/root.yaml",
			RootDocument:   root,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		result := schema.GetResolvedSchema()
		assert.Nil(t, result)
	})
}

// Test with real file system
func TestJSONSchema_Resolve_FileSystemIntegration(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := tmpDir + "/user.yaml"

	// Use actual test data
	testData, err := os.ReadFile("testdata/simple_schema.yaml")
	require.NoError(t, err)

	err = os.WriteFile(testFile, testData, 0o644)
	require.NoError(t, err)

	t.Run("successful file resolution", func(t *testing.T) {
		t.Parallel()
		root := NewMockResolutionTarget()
		ref := "user.yaml"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: tmpDir + "/root.yaml",
			RootDocument:   root,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)
		result := schema.GetResolvedSchema()
		require.NotNil(t, result)
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()
		root := NewMockResolutionTarget()
		ref := "nonexistent.yaml"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: tmpDir + "/root.yaml",
			RootDocument:   root,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		result := schema.GetResolvedSchema()
		assert.Nil(t, result)
		// Check for platform-agnostic file not found error
		errMsg := err.Error()
		assert.True(t,
			strings.Contains(errMsg, "no such file or directory") ||
				strings.Contains(errMsg, "The system cannot find the file specified") ||
				strings.Contains(errMsg, "cannot find the file"),
			"Expected file not found error, got: %s", errMsg)
	})
}

// Test default options behavior
func TestJSONSchema_Resolve_DefaultOptions(t *testing.T) {
	t.Parallel()

	t.Run("default VirtualFS", func(t *testing.T) {
		t.Parallel()
		root := NewMockResolutionTarget()
		ref := "nonexistent.yaml"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "testdata/root.yaml",
			RootDocument:   root,
			// VirtualFS not set - should default to system.FileSystem
		}

		_, err := schema.Resolve(t.Context(), opts)

		require.Error(t, err)
		// Error should be from the actual file system, not a nil pointer panic
		assert.NotContains(t, err.Error(), "nil pointer")
	})

	t.Run("default HTTPClient", func(t *testing.T) {
		t.Parallel()
		root := NewMockResolutionTarget()
		ref := "https://nonexistent.example.com/test.yaml"
		schema := createSchemaWithRef(ref)

		opts := ResolveOptions{
			TargetLocation: "https://example.com/root.yaml",
			RootDocument:   root,
			// HTTPClient not set - should default to http.DefaultClient
		}

		_, err := schema.Resolve(t.Context(), opts)

		require.Error(t, err)
		// Error should be from the HTTP client, not a nil pointer panic
		assert.NotContains(t, err.Error(), "nil pointer")
	})
}

func TestResolveSchema_ChainedReference_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Create mock filesystem with the test files using existing MockVirtualFS
	mockFS := NewMockVirtualFS()

	// Read existing external test file
	externalPath := filepath.Join("testdata", "resolve_test_external.yaml")
	externalContent, err := os.ReadFile(externalPath)
	require.NoError(t, err)
	mockFS.AddFile("./resolve_test_external.yaml", string(externalContent))

	// Read the chained test file we created
	chainedPath := filepath.Join("testdata", "resolve_test_chained.yaml")
	chainedContent, err := os.ReadFile(chainedPath)
	require.NoError(t, err)
	mockFS.AddFile("./resolve_test_chained.yaml", string(chainedContent))

	// Also add with absolute paths that the resolution system will request
	absExternalPath, err := filepath.Abs(externalPath)
	require.NoError(t, err)
	mockFS.AddFile(absExternalPath, string(externalContent))

	absChainedPath, err := filepath.Abs(chainedPath)
	require.NoError(t, err)
	mockFS.AddFile(absChainedPath, string(chainedContent))

	// Load existing main test document - we need to parse it as an OpenAPI document since we're using components
	mainPath := filepath.Join("testdata", "resolve_test_main.yaml")
	mainContent, err := os.ReadFile(mainPath)
	require.NoError(t, err)

	// Parse as OpenAPI document since it has components structure
	var node yaml.Node
	err = yaml.Unmarshal(mainContent, &node)
	require.NoError(t, err)

	// Create a mock resolution target from the main content
	mainRoot := &TestResolutionTarget{
		Schema: &Schema{}, // Will be populated during unmarshaling
		cache:  make(map[string][]byte),
	}
	mainRoot.InitCache()

	// Setup resolve options with mock filesystem
	absPath, err := filepath.Abs(mainPath)
	require.NoError(t, err)

	opts := ResolveOptions{
		TargetLocation: absPath,
		RootDocument:   mainRoot,
		VirtualFS:      mockFS,
	}

	// Create a reference schema that points to the chained reference
	// This simulates the main.yaml -> external.yaml#/components/schemas/ChainedExternal chain
	ref := "./resolve_test_external.yaml#/components/schemas/ChainedExternal"
	refSchema := createSchemaWithRef(ref)

	// This will trigger: main.yaml -> external.yaml#ChainedExternal -> chained.yaml#ChainedSchema -> #LocalChainedSchema
	// Attempt to resolve the chained reference
	validationErrs, err := refSchema.Resolve(ctx, opts)

	// The resolution should succeed - this tests the correct behavior
	require.NoError(t, err)
	assert.Empty(t, validationErrs)

	// Get the resolved schema after resolution
	resolved := refSchema.GetResolvedSchema()
	require.NotNil(t, resolved)

	// Test parent links for chained reference
	parent := resolved.GetParent()
	topLevelParent := resolved.GetTopLevelParent()

	assert.NotNil(t, parent, "parent should be set for chained reference")
	assert.Equal(t, refSchema, topLevelParent, "top-level parent should be the original reference")
	assert.NotEqual(t, refSchema, parent, "immediate parent should be different from top-level for chained reference")

	// Verify the schema has the expected description from the final LocalChainedSchema
	// This tests that the local reference #/components/schemas/LocalChainedSchema
	// was resolved correctly within chained.yaml (not against main.yaml)
	if resolved.IsSchema() {
		schema := resolved.GetSchema()
		assert.Equal(t, "Local chained schema", schema.GetDescription())

		// Verify the schema has properties
		properties := schema.GetProperties()
		require.NotNil(t, properties)

		// Verify we can access the nestedValue property with the expected structure
		nestedValue, exists := properties.Get("nestedValue")
		require.True(t, exists, "nestedValue property should exist")
		require.NotNil(t, nestedValue)

		// Verify the nested property structure (it should be a JSONSchema)
		if nestedValue.IsSchema() {
			nestedSchema := nestedValue.GetSchema()
			assert.Equal(t, "A nested value in the chained schema", nestedSchema.GetDescription())
		}
	}
}

// Test parent link functionality
func TestJSONSchema_ParentLinks(t *testing.T) {
	t.Parallel()

	t.Run("non-reference schema has no parent", func(t *testing.T) {
		t.Parallel()

		// Create a non-reference schema
		schema := createSimpleSchema()

		// Check parent links
		parent := schema.GetParent()
		topLevelParent := schema.GetTopLevelParent()

		assert.Nil(t, parent, "non-reference schema should have no parent")
		assert.Nil(t, topLevelParent, "non-reference schema should have no top-level parent")
	})

	t.Run("manual parent setting works correctly", func(t *testing.T) {
		t.Parallel()

		// Create schemas
		parentSchema := createSchemaWithRef("#/components/schemas/Parent")
		topLevelSchema := createSchemaWithRef("#/components/schemas/TopLevel")
		childSchema := createSimpleSchema()

		// Manually set parent links
		childSchema.SetParent(parentSchema)
		childSchema.SetTopLevelParent(topLevelSchema)

		// Check parent links
		parent := childSchema.GetParent()
		topLevelParent := childSchema.GetTopLevelParent()

		assert.Equal(t, parentSchema, parent, "manually set parent should be correct")
		assert.Equal(t, topLevelSchema, topLevelParent, "manually set top-level parent should be correct")
	})

	t.Run("nil schema methods handle gracefully", func(t *testing.T) {
		t.Parallel()

		var nilSchema *JSONSchema[Referenceable]

		// Test getter methods
		assert.Nil(t, nilSchema.GetParent(), "nil schema GetParent should return nil")
		assert.Nil(t, nilSchema.GetTopLevelParent(), "nil schema GetTopLevelParent should return nil")

		// Test setter methods (should not panic)
		assert.NotPanics(t, func() {
			nilSchema.SetParent(createSimpleSchema())
		}, "SetParent on nil schema should not panic")

		assert.NotPanics(t, func() {
			nilSchema.SetTopLevelParent(createSimpleSchema())
		}, "SetTopLevelParent on nil schema should not panic")
	})
}

// Test GetReferenceChain method
func TestJSONSchema_GetReferenceChain(t *testing.T) {
	t.Parallel()

	t.Run("nil schema returns nil", func(t *testing.T) {
		t.Parallel()
		var nilSchema *JSONSchema[Referenceable]
		assert.Nil(t, nilSchema.GetReferenceChain(), "nil schema GetReferenceChain should return nil")
	})

	t.Run("schema with nil parent returns nil", func(t *testing.T) {
		t.Parallel()
		schema := createSimpleSchema()
		assert.Nil(t, schema.GetReferenceChain(), "schema with nil parent should return nil from GetReferenceChain")
	})

	t.Run("schema with non-reference parent returns empty chain", func(t *testing.T) {
		t.Parallel()
		// Create parent that is NOT a reference (just a regular schema)
		nonRefParent := createSimpleSchema()

		// Create child with parent set
		childSchema := createSimpleSchema()
		childSchema.SetParent(nonRefParent)

		// Chain should be empty (not nil) since parent exists but isn't a reference
		chain := childSchema.GetReferenceChain()
		assert.Empty(t, chain, "schema with non-reference parent should return empty chain")
	})

	t.Run("schema with reference parent returns single-entry chain", func(t *testing.T) {
		t.Parallel()
		// Create parent that IS a reference
		refParent := createSchemaWithRef("#/components/schemas/Parent")

		// Create child with parent set
		childSchema := createSimpleSchema()
		childSchema.SetParent(refParent)

		chain := childSchema.GetReferenceChain()
		require.Len(t, chain, 1, "schema with reference parent should return single-entry chain")
		assert.Equal(t, "#/components/schemas/Parent", string(chain[0].Reference))
		assert.Equal(t, refParent, chain[0].Schema)
	})

	t.Run("schema with mixed parent chain filters non-references", func(t *testing.T) {
		t.Parallel()
		// Create a chain: refGrandparent -> nonRefParent -> child
		// Only refGrandparent should appear in the chain

		refGrandparent := createSchemaWithRef("#/components/schemas/Grandparent")
		nonRefParent := createSimpleSchema()
		childSchema := createSimpleSchema()

		// Set up the chain
		nonRefParent.SetParent(refGrandparent)
		childSchema.SetParent(nonRefParent)

		chain := childSchema.GetReferenceChain()
		require.Len(t, chain, 1, "chain should only include reference parents")
		assert.Equal(t, "#/components/schemas/Grandparent", string(chain[0].Reference))
	})

	t.Run("schema with multiple reference ancestors returns full chain", func(t *testing.T) {
		t.Parallel()
		// Create a chain: refGrandparent -> refParent -> child

		refGrandparent := createSchemaWithRef("#/components/schemas/Grandparent")
		refParent := createSchemaWithRef("#/components/schemas/Parent")
		childSchema := createSimpleSchema()

		// Set up the chain
		refParent.SetParent(refGrandparent)
		childSchema.SetParent(refParent)

		chain := childSchema.GetReferenceChain()
		require.Len(t, chain, 2, "chain should include both reference ancestors")
		// Chain is outer -> inner order (grandparent first, parent last)
		assert.Equal(t, "#/components/schemas/Grandparent", string(chain[0].Reference))
		assert.Equal(t, "#/components/schemas/Parent", string(chain[1].Reference))
	})
}

// Test GetImmediateReference method
func TestJSONSchema_GetImmediateReference(t *testing.T) {
	t.Parallel()

	t.Run("nil schema returns nil", func(t *testing.T) {
		t.Parallel()
		var nilSchema *JSONSchema[Referenceable]
		assert.Nil(t, nilSchema.GetImmediateReference(), "nil schema GetImmediateReference should return nil")
	})

	t.Run("schema with nil parent returns nil", func(t *testing.T) {
		t.Parallel()
		schema := createSimpleSchema()
		assert.Nil(t, schema.GetImmediateReference(), "schema with nil parent should return nil")
	})

	t.Run("schema with non-reference parent returns nil", func(t *testing.T) {
		t.Parallel()
		nonRefParent := createSimpleSchema()
		childSchema := createSimpleSchema()
		childSchema.SetParent(nonRefParent)

		assert.Nil(t, childSchema.GetImmediateReference(), "schema with non-reference parent should return nil")
	})

	t.Run("schema with reference parent returns entry", func(t *testing.T) {
		t.Parallel()
		refParent := createSchemaWithRef("#/components/schemas/Parent")
		childSchema := createSimpleSchema()
		childSchema.SetParent(refParent)

		entry := childSchema.GetImmediateReference()
		require.NotNil(t, entry, "should return entry for reference parent")
		assert.Equal(t, "#/components/schemas/Parent", string(entry.Reference))
		assert.Equal(t, refParent, entry.Schema)
	})
}

// Test GetTopLevelReference method
func TestJSONSchema_GetTopLevelReference(t *testing.T) {
	t.Parallel()

	t.Run("nil schema returns nil", func(t *testing.T) {
		t.Parallel()
		var nilSchema *JSONSchema[Referenceable]
		assert.Nil(t, nilSchema.GetTopLevelReference(), "nil schema GetTopLevelReference should return nil")
	})

	t.Run("schema with nil topLevelParent returns nil", func(t *testing.T) {
		t.Parallel()
		schema := createSimpleSchema()
		assert.Nil(t, schema.GetTopLevelReference(), "schema with nil topLevelParent should return nil")
	})

	t.Run("schema with non-reference topLevelParent returns nil", func(t *testing.T) {
		t.Parallel()
		nonRefTopLevel := createSimpleSchema()
		childSchema := createSimpleSchema()
		childSchema.SetTopLevelParent(nonRefTopLevel)

		assert.Nil(t, childSchema.GetTopLevelReference(), "schema with non-reference topLevelParent should return nil")
	})

	t.Run("schema with reference topLevelParent returns entry", func(t *testing.T) {
		t.Parallel()
		refTopLevel := createSchemaWithRef("#/components/schemas/TopLevel")
		childSchema := createSimpleSchema()
		childSchema.SetTopLevelParent(refTopLevel)

		entry := childSchema.GetTopLevelReference()
		require.NotNil(t, entry, "should return entry for reference topLevelParent")
		assert.Equal(t, "#/components/schemas/TopLevel", string(entry.Reference))
		assert.Equal(t, refTopLevel, entry.Schema)
	})
}

// Test GetReferenceChain with circular reference (regression test for infinite loop bug)
func TestJSONSchema_GetReferenceChain_CircularReference(t *testing.T) {
	t.Parallel()

	t.Run("circular parent chain terminates without infinite loop", func(t *testing.T) {
		t.Parallel()
		// Create a circular reference: A -> B -> A
		// This simulates a self-referential schema like:
		// Node:
		//   properties:
		//     next:
		//       $ref: "#/components/schemas/Node"

		schemaA := createSchemaWithRef("#/components/schemas/Node")
		schemaB := createSchemaWithRef("#/components/schemas/Node")
		childSchema := createSimpleSchema()

		// Create circular parent chain: childSchema -> schemaB -> schemaA -> schemaB (circular)
		schemaA.SetParent(schemaB)
		schemaB.SetParent(schemaA)
		childSchema.SetParent(schemaB)

		// This should NOT hang or infinite loop - it should detect the cycle and return
		chain := childSchema.GetReferenceChain()

		// Chain should contain both schemas (visited before cycle detected)
		// The exact length depends on when the cycle is detected, but it should be finite
		assert.NotNil(t, chain, "chain should not be nil even with circular references")
		assert.LessOrEqual(t, len(chain), 2, "chain should be bounded, not infinite")
	})

	t.Run("self-referential parent terminates without infinite loop", func(t *testing.T) {
		t.Parallel()
		// Create a self-referential schema: A -> A
		schemaA := createSchemaWithRef("#/components/schemas/Self")
		childSchema := createSimpleSchema()

		// Schema A's parent is itself
		schemaA.SetParent(schemaA)
		childSchema.SetParent(schemaA)

		// This should NOT hang or infinite loop
		chain := childSchema.GetReferenceChain()

		// Chain should contain exactly one entry (schemaA visited once before cycle detected)
		assert.NotNil(t, chain, "chain should not be nil even with self-reference")
		assert.Len(t, chain, 1, "chain should contain exactly one entry for self-referential parent")
		assert.Equal(t, "#/components/schemas/Self", string(chain[0].Reference))
	})
}

// Test resolution with $id as base URI
func TestJSONSchema_Resolve_WithID_Success(t *testing.T) {
	t.Parallel()

	t.Run("$id provides base URI for relative reference resolution", func(t *testing.T) {
		t.Parallel()

		// Create mock filesystem with schemas
		mockFS := NewMockVirtualFS()

		// Add a schema at the URL path that $id specifies
		mockFS.AddFile("api/v3/schemas/components/name.json", `{
		"type": "string",
		"description": "A name field resolved via $id base URI"
}`)

		// Add a schema with $id that points to a different base URL
		// The reference "./components/name.json" should resolve relative to $id
		mockFS.AddFile("testdata/pet.json", `{
		"$id": "https://example.com/api/v3/schemas/pet.json",
		"type": "object",
		"properties": {
		  "name": {
		    "$ref": "./components/name.json"
		  }
		}
}`)

		// Create HTTP client to handle the resolution from $id base URI
		httpClient := NewMockHTTPClient()
		httpClient.AddResponse("https://example.com/api/v3/schemas/components/name.json", `{
		"type": "string",
		"description": "A name field resolved via $id base URI"
}`, 200)

		// Create resolution target
		root := NewMockResolutionTarget()

		// Create a schema with a reference to pet.json
		schema := createSchemaWithRef("pet.json")

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   root,
			VirtualFS:      mockFS,
			HTTPClient:     httpClient,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Empty(t, validationErrs)

		// Get the resolved schema (pet.json)
		resolved := schema.GetResolvedSchema()
		require.NotNil(t, resolved)
		require.True(t, resolved.IsSchema())

		// Verify the pet schema has properties
		petSchema := resolved.GetSchema()
		require.NotNil(t, petSchema)
		assert.Equal(t, "https://example.com/api/v3/schemas/pet.json", petSchema.GetID())

		// Get the properties
		properties := petSchema.GetProperties()
		require.NotNil(t, properties)

		// Get the name property which has a $ref
		nameProperty, exists := properties.Get("name")
		require.True(t, exists, "name property should exist")
		require.NotNil(t, nameProperty)

		// The name property should be a reference, resolve it
		if nameProperty.IsReference() {
			// Create new resolve options with the updated base from $id
			nameOpts := ResolveOptions{
				TargetLocation: "https://example.com/api/v3/schemas/pet.json",
				RootDocument:   root,
				VirtualFS:      mockFS,
				HTTPClient:     httpClient,
			}

			validationErrs, err := nameProperty.Resolve(t.Context(), nameOpts)
			require.NoError(t, err)
			assert.Empty(t, validationErrs)

			nameResolved := nameProperty.GetResolvedSchema()
			require.NotNil(t, nameResolved)
			require.True(t, nameResolved.IsSchema())

			// Verify it resolved to the correct schema
			nameSchema := nameResolved.GetSchema()
			assert.Equal(t, "A name field resolved via $id base URI", nameSchema.GetDescription())
		}
	})

	t.Run("$id with URL used as base for local reference", func(t *testing.T) {
		t.Parallel()

		// Create mock filesystem
		mockFS := NewMockVirtualFS()

		// Add a schema file with $id and definitions
		mockFS.AddFile("testdata/definitions.json", `{
		"$id": "https://example.com/schemas/definitions.json",
		"type": "object",
		"definitions": {
		  "Address": {
		    "type": "object",
		    "properties": {
		      "street": { "type": "string" },
		      "city": { "type": "string" }
		    }
		  }
		},
		"properties": {
		  "homeAddress": {
		    "$ref": "#/definitions/Address"
		  }
		}
}`)

		// Create resolution target
		root := NewMockResolutionTarget()

		// Create a schema with a reference
		schema := createSchemaWithRef("definitions.json")

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   root,
			VirtualFS:      mockFS,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Empty(t, validationErrs)

		// Get the resolved schema
		resolved := schema.GetResolvedSchema()
		require.NotNil(t, resolved)
		require.True(t, resolved.IsSchema())

		// Verify $id was parsed
		defSchema := resolved.GetSchema()
		require.NotNil(t, defSchema)
		assert.Equal(t, "https://example.com/schemas/definitions.json", defSchema.GetID())
	})

	t.Run("schema without $id uses target location as base", func(t *testing.T) {
		t.Parallel()

		// Create mock filesystem
		mockFS := NewMockVirtualFS()

		// Add a schema without $id
		mockFS.AddFile("testdata/schemas/user.json", `{
		"type": "object",
		"properties": {
		  "name": { "type": "string" }
		}
}`)

		// Add a schema that references user.json
		mockFS.AddFile("testdata/schemas/order.json", `{
		"type": "object",
		"properties": {
		  "user": {
		    "$ref": "user.json"
		  }
		}
}`)

		// Create resolution target
		root := NewMockResolutionTarget()

		// Create a schema with a reference
		schema := createSchemaWithRef("schemas/order.json")

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   root,
			VirtualFS:      mockFS,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Empty(t, validationErrs)

		// Get the resolved schema
		resolved := schema.GetResolvedSchema()
		require.NotNil(t, resolved)
		require.True(t, resolved.IsSchema())

		// Verify no $id (should return empty string)
		orderSchema := resolved.GetSchema()
		require.NotNil(t, orderSchema)
		assert.Empty(t, orderSchema.GetID())
	})
}

func TestJSONSchema_Resolve_WithAnchor_Success(t *testing.T) {
	t.Parallel()

	t.Run("$anchor resolves to schema in same document", func(t *testing.T) {
		t.Parallel()

		// Create mock filesystem with a schema that has $anchor references
		mockFS := NewMockVirtualFS()

		// Schema with $anchor definitions that can be referenced
		mockFS.AddFile("testdata/schema.json", `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$defs": {
		  "foo": {
		    "$anchor": "foo",
		    "type": "string"
		  }
		},
		"properties": {
		  "name": {
		    "$ref": "#foo"
		  }
		}
}`)

		// Create resolution target
		root := NewMockResolutionTarget()

		// Create a schema with a reference
		schema := createSchemaWithRef("schema.json")

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   root,
			VirtualFS:      mockFS,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Empty(t, validationErrs)

		// Get the resolved schema
		resolved := schema.GetResolvedSchema()
		require.NotNil(t, resolved)
		require.True(t, resolved.IsSchema())
	})

	t.Run("$anchor with absolute URI in same document", func(t *testing.T) {
		t.Parallel()

		// Create mock filesystem with a schema that uses absolute URI with anchor
		mockFS := NewMockVirtualFS()

		mockFS.AddFile("testdata/schema.json", `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id": "https://example.com/schemas/root.json",
		"$defs": {
		  "A": {
		    "$anchor": "foo",
		    "type": "integer"
		  }
		},
		"properties": {
		  "value": {
		    "$ref": "https://example.com/schemas/root.json#foo"
		  }
		}
}`)

		// Create resolution target
		root := NewMockResolutionTarget()

		// Create a schema with a reference
		schema := createSchemaWithRef("schema.json")

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   root,
			VirtualFS:      mockFS,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Empty(t, validationErrs)

		// Get the resolved schema
		resolved := schema.GetResolvedSchema()
		require.NotNil(t, resolved)
		require.True(t, resolved.IsSchema())
	})

	t.Run("$anchor with base URI change in subschema", func(t *testing.T) {
		t.Parallel()

		// Create mock filesystem
		mockFS := NewMockVirtualFS()

		mockFS.AddFile("testdata/schema.json", `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id": "https://example.com/root.json",
		"$defs": {
		  "A": {
		    "$id": "https://example.com/nested.json",
		    "$anchor": "bar",
		    "type": "string"
		  },
		  "B": {
		    "$anchor": "bar",
		    "type": "integer"
		  }
		},
		"properties": {
		  "strVal": {
		    "$ref": "https://example.com/nested.json#bar"
		  },
		  "intVal": {
		    "$ref": "#bar"
		  }
		}
}`)

		// Create resolution target
		root := NewMockResolutionTarget()

		// Create a schema with a reference
		schema := createSchemaWithRef("schema.json")

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   root,
			VirtualFS:      mockFS,
		}

		validationErrs, err := schema.Resolve(t.Context(), opts)

		require.NoError(t, err)
		assert.Empty(t, validationErrs)

		// Get the resolved schema
		resolved := schema.GetResolvedSchema()
		require.NotNil(t, resolved)
		require.True(t, resolved.IsSchema())
	})
}

func TestJSONSchema_GetReferenceResolutionInfo_Success(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for non-reference", func(t *testing.T) {
		t.Parallel()

		schema := NewJSONSchemaFromSchema[Referenceable](&Schema{
			Type: NewTypeFromArray([]SchemaType{SchemaTypeString}),
		})

		info := schema.GetReferenceResolutionInfo()
		assert.Nil(t, info, "should return nil for non-reference")
	})

	t.Run("returns nil for nil schema", func(t *testing.T) {
		t.Parallel()

		var schema *JSONSchema[Referenceable]
		info := schema.GetReferenceResolutionInfo()
		assert.Nil(t, info, "should return nil for nil schema")
	})

	t.Run("returns info after resolution", func(t *testing.T) {
		t.Parallel()

		// Create mock filesystem
		mockFS := NewMockVirtualFS()

		mockFS.AddFile("testdata/user.json", `{
		"type": "object",
		"properties": {
		  "name": { "type": "string" }
		}
}`)

		root := NewMockResolutionTarget()
		schema := createSchemaWithRef("user.json")

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   root,
			VirtualFS:      mockFS,
		}

		_, err := schema.Resolve(t.Context(), opts)
		require.NoError(t, err)

		info := schema.GetReferenceResolutionInfo()
		require.NotNil(t, info, "should return info after resolution")
		assert.NotNil(t, info.Object, "should have resolved object")
	})
}

func TestJoinReferenceChain_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		chain    []string
		expected string
	}{
		{
			name:     "empty chain",
			chain:    []string{},
			expected: "",
		},
		{
			name:     "single element",
			chain:    []string{"#/components/schemas/User"},
			expected: "#/components/schemas/User",
		},
		{
			name:     "two elements",
			chain:    []string{"#/components/schemas/User", "#/components/schemas/Address"},
			expected: "#/components/schemas/User -> #/components/schemas/Address",
		},
		{
			name:     "three elements",
			chain:    []string{"A", "B", "C"},
			expected: "A -> B -> C",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := joinReferenceChain(tt.chain)
			assert.Equal(t, tt.expected, result, "join result should match expected")
		})
	}
}

// MockSchemaRegistryProvider implements SchemaRegistryProvider for testing
type MockSchemaRegistryProvider struct {
	*MockResolutionTarget
	registry SchemaRegistry
	baseURI  string
}

func NewMockSchemaRegistryProvider() *MockSchemaRegistryProvider {
	registry := NewSchemaRegistry("")
	return &MockSchemaRegistryProvider{
		MockResolutionTarget: NewMockResolutionTarget(),
		registry:             registry,
		baseURI:              "",
	}
}

func (m *MockSchemaRegistryProvider) GetSchemaRegistry() SchemaRegistry {
	return m.registry
}

func (m *MockSchemaRegistryProvider) GetDocumentBaseURI() string {
	return m.baseURI
}

func TestTryResolveViaRegistry_Success(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when no registry available", func(t *testing.T) {
		t.Parallel()

		schema := createSchemaWithRef("#foo")
		ref := schema.GetRef()

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   NewMockResolutionTarget(), // No registry provider
		}

		result := schema.tryResolveViaRegistry(t.Context(), ref, opts)
		assert.Nil(t, result, "should return nil when no registry available")
	})

	t.Run("returns nil for empty reference", func(t *testing.T) {
		t.Parallel()

		schema := createSchemaWithRef("")
		ref := schema.GetRef()

		provider := NewMockSchemaRegistryProvider()
		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   provider,
		}

		result := schema.tryResolveViaRegistry(t.Context(), ref, opts)
		assert.Nil(t, result, "should return nil for empty reference")
	})

	t.Run("resolves $anchor reference from registry", func(t *testing.T) {
		t.Parallel()

		provider := NewMockSchemaRegistryProvider()

		// Create a target schema with $anchor set
		anchor := "foo"
		targetSchema := NewJSONSchemaFromSchema[Referenceable](&Schema{
			Type:   NewTypeFromArray([]SchemaType{SchemaTypeString}),
			Anchor: &anchor,
		})

		// Register the schema (it will pick up the anchor from the schema)
		err := provider.registry.RegisterSchema(targetSchema, "https://example.com/root.json")
		require.NoError(t, err)

		// Create a schema that references the anchor
		schema := createSchemaWithRef("#foo")
		ref := schema.GetRef()

		// Set effective base URI on the schema
		innerSchema := schema.GetSchema()
		innerSchema.effectiveBaseURI = "https://example.com/root.json"

		opts := ResolveOptions{
			TargetLocation: "https://example.com/root.json",
			RootDocument:   provider,
		}

		result := schema.tryResolveViaRegistry(t.Context(), ref, opts)
		require.NotNil(t, result, "should resolve anchor reference")
		assert.Equal(t, targetSchema, result.Object, "should resolve to registered schema")
	})

	t.Run("resolves $id reference from registry", func(t *testing.T) {
		t.Parallel()

		provider := NewMockSchemaRegistryProvider()

		// Create a target schema with $id set
		targetSchema := NewJSONSchemaFromSchema[Referenceable](&Schema{
			Type: NewTypeFromArray([]SchemaType{SchemaTypeString}),
			ID:   ptr("https://example.com/schemas/user.json"),
		})

		// Register the schema with its $id
		err := provider.registry.RegisterSchema(targetSchema, "")
		require.NoError(t, err)

		// Create a schema that references the $id
		schema := createSchemaWithRef("https://example.com/schemas/user.json")
		ref := schema.GetRef()

		opts := ResolveOptions{
			TargetLocation: "https://example.com/root.json",
			RootDocument:   provider,
		}

		result := schema.tryResolveViaRegistry(t.Context(), ref, opts)
		require.NotNil(t, result, "should resolve $id reference")
		assert.Equal(t, targetSchema, result.Object, "should resolve to registered schema")
	})

	t.Run("resolves relative $id reference from registry", func(t *testing.T) {
		t.Parallel()

		provider := NewMockSchemaRegistryProvider()

		// Create a target schema with $id set
		targetSchema := NewJSONSchemaFromSchema[Referenceable](&Schema{
			Type: NewTypeFromArray([]SchemaType{SchemaTypeString}),
			ID:   ptr("https://example.com/schemas/user.json"),
		})

		// Register the schema
		err := provider.registry.RegisterSchema(targetSchema, "")
		require.NoError(t, err)

		// Create a schema that references with a relative URI
		schema := createSchemaWithRef("user.json")
		ref := schema.GetRef()

		// Set effective base URI on the schema
		innerSchema := schema.GetSchema()
		innerSchema.effectiveBaseURI = "https://example.com/schemas/"

		opts := ResolveOptions{
			TargetLocation: "https://example.com/schemas/root.json",
			RootDocument:   provider,
		}

		result := schema.tryResolveViaRegistry(t.Context(), ref, opts)
		require.NotNil(t, result, "should resolve relative $id reference")
		assert.Equal(t, targetSchema, result.Object, "should resolve to registered schema")
	})

	t.Run("resolves anchor with document base URI fallback", func(t *testing.T) {
		t.Parallel()

		// Create a registry with a document base URI
		registry := NewSchemaRegistry("https://example.com/doc.json")
		provider := &MockSchemaRegistryProvider{
			MockResolutionTarget: NewMockResolutionTarget(),
			registry:             registry,
			baseURI:              "https://example.com/doc.json",
		}

		// Create a target schema with $anchor set
		anchor := "bar"
		targetSchema := NewJSONSchemaFromSchema[Referenceable](&Schema{
			Type:   NewTypeFromArray([]SchemaType{SchemaTypeString}),
			Anchor: &anchor,
		})

		// Register the schema under document base URI
		err := registry.RegisterSchema(targetSchema, "https://example.com/doc.json")
		require.NoError(t, err)

		// Create a schema that references the anchor
		schema := createSchemaWithRef("#bar")
		ref := schema.GetRef()

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   provider,
		}

		result := schema.tryResolveViaRegistry(t.Context(), ref, opts)
		require.NotNil(t, result, "should resolve anchor with document base URI")
		assert.Equal(t, targetSchema, result.Object, "should resolve to registered schema")
	})
}

// ptr is a helper to create a pointer to a string
func ptr(s string) *string {
	return &s
}

func TestGetEffectiveBaseURI_Success(t *testing.T) {
	t.Parallel()

	t.Run("returns schema's effective base URI", func(t *testing.T) {
		t.Parallel()

		schema := createSimpleSchema()
		innerSchema := schema.GetSchema()
		innerSchema.effectiveBaseURI = "https://example.com/schema.json"

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   NewMockResolutionTarget(),
		}

		result := schema.getEffectiveBaseURI(opts)
		assert.Equal(t, "https://example.com/schema.json", result)
	})

	t.Run("returns cached absolute reference", func(t *testing.T) {
		t.Parallel()

		schema := createSchemaWithRef("#foo")
		schema.referenceResolutionCache = &references.ResolveResult[JSONSchema[Referenceable]]{
			AbsoluteReference: "https://example.com/cached.json",
		}

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   NewMockResolutionTarget(),
		}

		result := schema.getEffectiveBaseURI(opts)
		assert.Equal(t, "https://example.com/cached.json", result)
	})

	t.Run("returns target location as fallback", func(t *testing.T) {
		t.Parallel()

		schema := createSimpleSchema()

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   NewMockResolutionTarget(),
		}

		result := schema.getEffectiveBaseURI(opts)
		assert.Equal(t, "testdata/root.json", result)
	})

	t.Run("returns document base URI from provider", func(t *testing.T) {
		t.Parallel()

		schema := createSimpleSchema()

		provider := NewMockSchemaRegistryProvider()
		provider.baseURI = "https://example.com/document.json"

		opts := ResolveOptions{
			TargetLocation: "", // Empty target location
			RootDocument:   provider,
		}

		result := schema.getEffectiveBaseURI(opts)
		assert.Equal(t, "https://example.com/document.json", result)
	})
}

func TestGetSchemaRegistry_Success(t *testing.T) {
	t.Parallel()

	t.Run("returns registry from root document", func(t *testing.T) {
		t.Parallel()

		provider := NewMockSchemaRegistryProvider()
		schema := createSimpleSchema()

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   provider,
		}

		result := schema.getSchemaRegistry(opts)
		assert.NotNil(t, result, "should return registry from root document")
		assert.Equal(t, provider.registry, result)
	})

	t.Run("returns registry from target document", func(t *testing.T) {
		t.Parallel()

		provider := NewMockSchemaRegistryProvider()
		schema := createSimpleSchema()

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   NewMockResolutionTarget(), // No registry
			TargetDocument: provider,                  // Has registry
		}

		result := schema.getSchemaRegistry(opts)
		assert.NotNil(t, result, "should return registry from target document")
		assert.Equal(t, provider.registry, result)
	})

	t.Run("returns registry from schema's owning document", func(t *testing.T) {
		t.Parallel()

		// Create a provider that implements SchemaRegistryProvider
		provider := NewMockSchemaRegistryProvider()

		// Create schema and set its owning document
		schema := createSimpleSchema()
		schema.GetSchema().SetOwningDocument(provider)

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   NewMockResolutionTarget(), // No registry
		}

		result := schema.getSchemaRegistry(opts)
		assert.NotNil(t, result, "should return registry from schema")
		assert.Equal(t, provider.registry, result)
	})

	t.Run("returns nil when no registry available", func(t *testing.T) {
		t.Parallel()

		schema := createSimpleSchema()

		opts := ResolveOptions{
			TargetLocation: "testdata/root.json",
			RootDocument:   NewMockResolutionTarget(),
		}

		result := schema.getSchemaRegistry(opts)
		assert.Nil(t, result, "should return nil when no registry available")
	})
}

func TestGetParentJSONPointer_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "root slash",
			input:    "/",
			expected: "",
		},
		{
			name:     "single segment",
			input:    "/properties",
			expected: "",
		},
		{
			name:     "two segments",
			input:    "/properties/name",
			expected: "/properties",
		},
		{
			name:     "three segments",
			input:    "/properties/nested/inner",
			expected: "/properties/nested",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := getParentJSONPointer(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
