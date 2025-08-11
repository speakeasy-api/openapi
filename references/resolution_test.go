package references

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// MockResolutionTarget implements ResolutionTarget for testing
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
	return nil, fmt.Errorf("not implemented")
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

// Test unmarshalers
func testComplexUnmarshaler(ctx context.Context, node *yaml.Node, skipValidation bool) (*tests.TestComplexHighModel, []error, error) {
	model := &tests.TestComplexHighModel{}
	model.ArrayField = []string{"test1", "test2", "test3"}
	return model, nil, nil
}

func testPrimitiveUnmarshaler(ctx context.Context, node *yaml.Node, skipValidation bool) (*tests.TestPrimitiveHighModel, []error, error) {
	model := &tests.TestPrimitiveHighModel{}
	model.StringField = "test-string"
	intVal := 42
	model.IntPtrField = &intVal
	return model, nil, nil
}

func testErrorUnmarshaler(ctx context.Context, node *yaml.Node, skipValidation bool) (*tests.TestComplexHighModel, []error, error) {
	return nil, nil, fmt.Errorf("unmarshaling failed")
}

func testNilUnmarshaler(ctx context.Context, node *yaml.Node, skipValidation bool) (*tests.TestComplexHighModel, []error, error) {
	return nil, nil, nil
}

// TestResolutionTarget implements ResolutionTarget and can act as test data
type TestResolutionTarget struct {
	*tests.TestComplexHighModel
	cache map[string][]byte
}

func NewTestResolutionTarget() *TestResolutionTarget {
	model := &tests.TestComplexHighModel{}
	model.ArrayField = []string{"test1", "test2", "test3"}

	nested := &tests.TestPrimitiveHighModel{}
	nested.StringField = "nested-string"
	intVal := 42
	nested.IntPtrField = &intVal
	model.NestedModel = nested

	return &TestResolutionTarget{
		TestComplexHighModel: model,
		cache:                make(map[string][]byte),
	}
}

func (t *TestResolutionTarget) GetCachedReferenceDocument(key string) ([]byte, bool) {
	data, exists := t.cache[key]
	return data, exists
}

func (t *TestResolutionTarget) StoreReferenceDocumentInCache(key string, doc []byte) {
	t.cache[key] = doc
}

// Test resolution against root document (empty reference)
func TestResolve_RootDocument(t *testing.T) {
	t.Parallel()

	t.Run("resolve empty reference against root document", func(t *testing.T) {
		t.Parallel()
		root := NewTestResolutionTarget()
		root.InitCache()
		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference(""), func(ctx context.Context, node *yaml.Node, skipValidation bool) (*TestResolutionTarget, []error, error) {
			return root, nil, nil
		}, opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)
		require.NotNil(t, result)
		require.NotNil(t, result.Object)
		assert.Equal(t, 3, len(result.Object.ArrayField))
	})

	t.Run("resolve JSON pointer against root document", func(t *testing.T) {
		t.Parallel()

		root := NewTestResolutionTarget()
		root.InitCache()

		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("#/nestedModel"), testPrimitiveUnmarshaler, opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)
		require.NotNil(t, result)
		require.NotNil(t, result.Object)
		assert.Equal(t, "nested-string", result.Object.StringField)
	})
}

// Test resolution against file paths
func TestResolve_FilePath(t *testing.T) {
	t.Parallel()

	t.Run("resolve against file path", func(t *testing.T) {
		t.Parallel()
		fs := NewMockVirtualFS()
		fs.AddFile("/test/schemas/test.yaml", "type: object\nproperties:\n  name:\n    type: string")

		root := NewMockResolutionTarget()
		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
			VirtualFS:      fs,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("schemas/test.yaml"), testComplexUnmarshaler, opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)
		require.NotNil(t, result)
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()

		fs := NewMockVirtualFS()
		root := NewMockResolutionTarget()
		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
			VirtualFS:      fs,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("missing.yaml"), testPrimitiveUnmarshaler, opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "file not found")
	})
}

// Test resolution against URLs
func TestResolve_URL(t *testing.T) {
	t.Parallel()

	t.Run("resolve against URL", func(t *testing.T) {
		t.Parallel()
		client := NewMockHTTPClient()
		client.AddResponse("https://example.com/schemas/test.yaml", "type: object", 200)

		root := NewMockResolutionTarget()
		opts := ResolveOptions{
			TargetLocation: "https://example.com/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
			HTTPClient:     client,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("schemas/test.yaml"), testPrimitiveUnmarshaler, opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)
		require.NotNil(t, result)
	})

	t.Run("HTTP error response", func(t *testing.T) {
		t.Parallel()

		client := NewMockHTTPClient()
		client.AddResponse("https://example.com/missing.yaml", "Not Found", 404)

		root := NewMockResolutionTarget()
		opts := ResolveOptions{
			TargetLocation: "https://example.com/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
			HTTPClient:     client,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("missing.yaml"), testPrimitiveUnmarshaler, opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "HTTP request failed")
	})
}

// Test caching behavior
func TestResolve_Caching(t *testing.T) {
	t.Parallel()

	t.Run("cached document is used", func(t *testing.T) {
		t.Parallel()
		fs := NewMockVirtualFS()
		fs.AddFile("/test/schemas/cached.yaml", "original: content")

		root := NewMockResolutionTarget()

		// Pre-populate cache with different content
		cachedData := []byte("cached: content")
		root.StoreReferenceDocumentInCache("/test/schemas/cached.yaml", cachedData)

		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
			VirtualFS:      fs,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("schemas/cached.yaml"), testPrimitiveUnmarshaler, opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)
		require.NotNil(t, result)

		// Verify cache was used (not the filesystem content)
		cached, exists := root.GetCachedReferenceDocument("/test/schemas/cached.yaml")
		assert.True(t, exists)
		assert.Equal(t, cachedData, cached)
	})
}

// Test error cases
func TestResolve_Errors(t *testing.T) {
	t.Parallel()

	t.Run("missing root location", func(t *testing.T) {
		t.Parallel()
		opts := ResolveOptions{
			RootDocument: NewMockResolutionTarget(),
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("#/test"), testPrimitiveUnmarshaler, opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "target location is required")
	})

	t.Run("missing root document", func(t *testing.T) {
		t.Parallel()

		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("#/test"), testPrimitiveUnmarshaler, opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "root document is required")
	})

	t.Run("missing target document", func(t *testing.T) {
		t.Parallel()

		root := NewMockResolutionTarget()

		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("#/test"), testPrimitiveUnmarshaler, opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "target document is required")
	})

	t.Run("unmarshaler error", func(t *testing.T) {
		t.Parallel()

		fs := NewMockVirtualFS()
		fs.AddFile("/test/test.yaml", "test: content")

		root := NewMockResolutionTarget()
		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
			VirtualFS:      fs,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("test.yaml"), testErrorUnmarshaler, opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "unmarshaling failed")
	})

	t.Run("unmarshaler returns nil", func(t *testing.T) {
		t.Parallel()

		fs := NewMockVirtualFS()
		fs.AddFile("/test/test.yaml", "test: content")

		root := NewMockResolutionTarget()
		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
			VirtualFS:      fs,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("test.yaml"), testNilUnmarshaler, opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("external references disabled", func(t *testing.T) {
		t.Parallel()

		fs := NewMockVirtualFS()
		fs.AddFile("/test/external.yaml", "type: object\\nproperties:\\n  test:\\n    type: string")

		root := NewMockResolutionTarget()
		opts := ResolveOptions{
			TargetLocation:      "/test/root.yaml",
			RootDocument:        root,
			TargetDocument:      root,
			VirtualFS:           fs,
			DisableExternalRefs: true,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("external.yaml"), testPrimitiveUnmarshaler, opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "external reference not allowed")
	})
}

// Test with real HTTP server
func TestResolve_HTTPIntegration(t *testing.T) {
	t.Parallel()

	t.Run("successful HTTP resolution", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/test.yaml":
				w.Header().Set("Content-Type", "application/yaml")
				w.WriteHeader(200)
				_, _ = w.Write([]byte("type: object\nproperties:\n  test: {type: string}"))
			case "/error":
				w.WriteHeader(404)
				_, _ = w.Write([]byte("Not Found"))
			default:
				w.WriteHeader(404)
			}
		}))
		defer server.Close()

		root := NewMockResolutionTarget()
		opts := ResolveOptions{
			TargetLocation: server.URL + "/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("test.yaml"), testPrimitiveUnmarshaler, opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)
		require.NotNil(t, result)
	})

	t.Run("HTTP error response", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/test.yaml":
				w.Header().Set("Content-Type", "application/yaml")
				w.WriteHeader(200)
				_, _ = w.Write([]byte("type: object\nproperties:\n  test: {type: string}"))
			case "/error":
				w.WriteHeader(404)
				_, _ = w.Write([]byte("Not Found"))
			default:
				w.WriteHeader(404)
			}
		}))
		defer server.Close()

		root := NewMockResolutionTarget()
		opts := ResolveOptions{
			TargetLocation: server.URL + "/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("error"), testPrimitiveUnmarshaler, opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
		assert.Nil(t, result)
	})
}

// Test with real file system
func TestResolve_FileSystemIntegration(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.yaml"

	err := os.WriteFile(testFile, []byte("type: object\ntest: data"), 0o644)
	require.NoError(t, err)

	t.Run("successful file resolution", func(t *testing.T) {
		t.Parallel()

		root := NewMockResolutionTarget()
		opts := ResolveOptions{
			TargetLocation: tmpDir + "/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("test.yaml"), testPrimitiveUnmarshaler, opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)
		require.NotNil(t, result)
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()

		root := NewMockResolutionTarget()
		opts := ResolveOptions{
			TargetLocation: tmpDir + "/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
		}

		result, validationErrs, err := Resolve(context.Background(), Reference("nonexistent.yaml"), testPrimitiveUnmarshaler, opts)

		require.Error(t, err)
		assert.Nil(t, validationErrs)
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
func TestResolve_DefaultOptions(t *testing.T) {
	t.Parallel()

	t.Run("default VirtualFS", func(t *testing.T) {
		t.Parallel()
		root := NewMockResolutionTarget()
		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
			// VirtualFS not set - should default to system.FileSystem
		}

		_, _, err := Resolve(context.Background(), Reference("nonexistent.yaml"), testComplexUnmarshaler, opts)

		require.Error(t, err)
		// Error should be from the actual file system, not a nil pointer panic
		assert.NotContains(t, err.Error(), "nil pointer")
	})

	t.Run("default HTTPClient", func(t *testing.T) {
		t.Parallel()

		root := NewMockResolutionTarget()
		opts := ResolveOptions{
			TargetLocation: "https://example.com/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
			// HTTPClient not set - should default to http.DefaultClient
		}

		_, _, err := Resolve(context.Background(), Reference("https://nonexistent.example.com/test.yaml"), testComplexUnmarshaler, opts)

		require.Error(t, err)
		// Error should be from the HTTP client, not a nil pointer panic
		assert.NotContains(t, err.Error(), "nil pointer")
	})
}

// TestResolve_AbsoluteVsRelativeReferenceHandling tests the core distinction that
// absolute references should NOT be resolved against the root location,
// while relative references should be resolved against the root location.
func TestResolve_AbsoluteVsRelativeReferenceHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		rootLocation        string
		referenceURI        string
		expectedAbsoluteRef string
		isAbsolute          bool
		description         string
		setupMocks          func(*MockVirtualFS, *MockHTTPClient)
	}{
		// Relative references - should be resolved against root location
		{
			name:                "relative_file_path",
			rootLocation:        "/project/api/spec.yaml",
			referenceURI:        "schemas/user.yaml",
			expectedAbsoluteRef: "/project/api/schemas/user.yaml",
			isAbsolute:          false,
			description:         "Relative file path should be resolved against root directory",
			setupMocks: func(fs *MockVirtualFS, client *MockHTTPClient) {
				fs.AddFile("/project/api/schemas/user.yaml", "type: object\nproperties:\n  name:\n    type: string")
			},
		},
		{
			name:                "relative_with_dotdot",
			rootLocation:        "/project/api/spec.yaml",
			referenceURI:        "../common/schema.yaml",
			expectedAbsoluteRef: "/project/common/schema.yaml",
			isAbsolute:          false,
			description:         "Relative path with .. should be resolved against root directory",
			setupMocks: func(fs *MockVirtualFS, client *MockHTTPClient) {
				fs.AddFile("/project/common/schema.yaml", "type: object\nproperties:\n  id:\n    type: integer")
			},
		},
		{
			name:                "relative_url_path",
			rootLocation:        "https://api.example.com/v1/spec.yaml",
			referenceURI:        "schemas/common.yaml",
			expectedAbsoluteRef: "https://api.example.com/v1/schemas/common.yaml",
			isAbsolute:          false,
			description:         "Relative URL path should be resolved against root URL",
			setupMocks: func(fs *MockVirtualFS, client *MockHTTPClient) {
				client.AddResponse("https://api.example.com/v1/schemas/common.yaml", "type: object", 200)
			},
		},

		// Absolute references - should NOT be resolved against root location
		{
			name:                "absolute_file_path",
			rootLocation:        "/project/spec.yaml",
			referenceURI:        "/external/schema.yaml",
			expectedAbsoluteRef: "/external/schema.yaml",
			isAbsolute:          true,
			description:         "Absolute file path should remain unchanged (not resolved against root)",
			setupMocks: func(fs *MockVirtualFS, client *MockHTTPClient) {
				fs.AddFile("/external/schema.yaml", "type: object\nproperties:\n  external:\n    type: boolean")
			},
		},
		{
			name:                "absolute_http_url",
			rootLocation:        "/project/spec.yaml",
			referenceURI:        "http://example.com/schema.yaml",
			expectedAbsoluteRef: "http://example.com/schema.yaml",
			isAbsolute:          true,
			description:         "Absolute HTTP URL should remain unchanged (not resolved against root)",
			setupMocks: func(fs *MockVirtualFS, client *MockHTTPClient) {
				client.AddResponse("http://example.com/schema.yaml", "type: object", 200)
			},
		},
		{
			name:                "absolute_https_url",
			rootLocation:        "https://api.example.com/spec.yaml",
			referenceURI:        "https://external.com/schema.yaml",
			expectedAbsoluteRef: "https://external.com/schema.yaml",
			isAbsolute:          true,
			description:         "Absolute HTTPS URL should remain unchanged (not resolved against root)",
			setupMocks: func(fs *MockVirtualFS, client *MockHTTPClient) {
				client.AddResponse("https://external.com/schema.yaml", "type: object", 200)
			},
		},
		{
			name:                "absolute_https_from_file_root",
			rootLocation:        "/project/spec.yaml",
			referenceURI:        "https://external.com/schema.yaml",
			expectedAbsoluteRef: "https://external.com/schema.yaml",
			isAbsolute:          true,
			description:         "Absolute HTTPS URL should remain unchanged even with file root",
			setupMocks: func(fs *MockVirtualFS, client *MockHTTPClient) {
				client.AddResponse("https://external.com/schema.yaml", "type: object", 200)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup mocks
			fs := NewMockVirtualFS()
			client := NewMockHTTPClient()
			root := NewMockResolutionTarget()

			tt.setupMocks(fs, client)

			// Setup resolve options
			opts := ResolveOptions{
				TargetLocation: tt.rootLocation,
				RootDocument:   root,
				TargetDocument: root,
				VirtualFS:      fs,
				HTTPClient:     client,
			}

			// Test resolution using the Resolve function
			result, validationErrs, err := Resolve(context.Background(), Reference(tt.referenceURI), func(ctx context.Context, node *yaml.Node, skipValidation bool) (*TestResolutionTarget, []error, error) {
				target := NewTestResolutionTarget()
				target.InitCache()
				return target, nil, nil
			}, opts)

			// Verify the resolution was successful
			require.NoError(t, err, "Failed to resolve reference %s from %s", tt.referenceURI, tt.rootLocation)
			assert.Nil(t, validationErrs)
			require.NotNil(t, result)
			require.NotNil(t, result.Object)

			// Verify the absolute reference is what we expect
			assert.Equal(t, tt.expectedAbsoluteRef, result.AbsoluteReference, tt.description)

			// Verify the behavior matches our expectation about absolute vs relative
			if tt.isAbsolute {
				// For absolute references, the result should be exactly the same as the original URI
				assert.Equal(t, tt.referenceURI, result.AbsoluteReference, "Absolute reference should remain unchanged")
			} else {
				// For relative references, the result should be different from the original URI
				assert.NotEqual(t, tt.referenceURI, result.AbsoluteReference, "Relative reference should be resolved")
			}
		})
	}
}

// TestResolve_RootDocumentDifferentFromTargetDocument tests scenarios where
// the root document is different from the target document, which happens during
// reference chains. This ensures that:
// 1. Resolution works correctly against the target document
// 2. Caching is always stored in the root document (not the target document)
// 3. Cache lookups happen against the root document
func TestResolve_RootDocumentDifferentFromTargetDocument(t *testing.T) {
	t.Parallel()

	t.Run("resolve against different target document with file cache stored in root", func(t *testing.T) {
		t.Parallel()
		// Create a root document for caching
		rootDoc := NewMockResolutionTarget()

		// Create a different target document that simulates an external document
		targetDoc := NewTestResolutionTarget()
		targetDoc.InitCache()

		// Setup a mock file system with an external schema
		fs := NewMockVirtualFS()
		fs.AddFile("/project/api/schemas/user.yaml", "type: object\nproperties:\n  name:\n    type: string")

		opts := ResolveOptions{
			TargetLocation: "/project/api/spec.yaml",
			RootDocument:   rootDoc,   // Different from target
			TargetDocument: targetDoc, // Different from root
			VirtualFS:      fs,
		}

		// Resolve a reference to an external file
		result, validationErrs, err := Resolve(context.Background(), Reference("schemas/user.yaml"), testPrimitiveUnmarshaler, opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)
		require.NotNil(t, result)
		require.NotNil(t, result.Object)
		assert.Equal(t, "/project/api/schemas/user.yaml", result.AbsoluteReference)

		// Verify the cache was stored in the ROOT document, not the target document
		cachedData, exists := rootDoc.GetCachedReferenceDocument("/project/api/schemas/user.yaml")
		assert.True(t, exists, "Cache should be stored in root document")
		assert.Contains(t, string(cachedData), "type: object", "Cached data should contain the resolved content")

		// Verify the target document does NOT have the cache
		_, existsInTarget := targetDoc.GetCachedReferenceDocument("/project/api/schemas/user.yaml")
		assert.False(t, existsInTarget, "Cache should NOT be stored in target document")
	})

	t.Run("resolve against different target document with URL cache stored in root", func(t *testing.T) {
		t.Parallel()

		// Create a root document for caching
		rootDoc := NewMockResolutionTarget()

		// Create a different target document that simulates an external document
		targetDoc := NewTestResolutionTarget()
		targetDoc.InitCache()

		// Setup a mock HTTP client
		client := NewMockHTTPClient()
		client.AddResponse("https://external.com/schemas/common.yaml", "type: object\nproperties:\n  id:\n    type: integer", 200)

		opts := ResolveOptions{
			TargetLocation: "https://api.example.com/spec.yaml",
			RootDocument:   rootDoc,   // Different from target
			TargetDocument: targetDoc, // Different from root
			HTTPClient:     client,
		}

		// Resolve a reference to an external URL
		result, validationErrs, err := Resolve(context.Background(), Reference("https://external.com/schemas/common.yaml"), testPrimitiveUnmarshaler, opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)
		require.NotNil(t, result)
		require.NotNil(t, result.Object)
		assert.Equal(t, "https://external.com/schemas/common.yaml", result.AbsoluteReference)

		// Verify the cache was stored in the ROOT document, not the target document
		cachedData, exists := rootDoc.GetCachedReferenceDocument("https://external.com/schemas/common.yaml")
		assert.True(t, exists, "Cache should be stored in root document")
		assert.Contains(t, string(cachedData), "type: object", "Cached data should contain the resolved content")

		// Verify the target document does NOT have the cache
		_, existsInTarget := targetDoc.GetCachedReferenceDocument("https://external.com/schemas/common.yaml")
		assert.False(t, existsInTarget, "Cache should NOT be stored in target document")
	})

	t.Run("cache lookup uses root document even with different target", func(t *testing.T) {
		t.Parallel()

		// Create a root document and pre-populate its cache
		rootDoc := NewMockResolutionTarget()
		cachedData := []byte("cached: content\ntype: object")
		rootDoc.StoreReferenceDocumentInCache("/project/api/schemas/cached.yaml", cachedData)

		// Create a different target document
		targetDoc := NewTestResolutionTarget()
		targetDoc.InitCache()

		// Setup a mock file system with different content than the cache
		fs := NewMockVirtualFS()
		fs.AddFile("/project/schemas/cached.yaml", "original: content\ntype: string")

		opts := ResolveOptions{
			TargetLocation: "/project/api/spec.yaml",
			RootDocument:   rootDoc,   // Has the cache
			TargetDocument: targetDoc, // Different from root
			VirtualFS:      fs,        // Has different content than cache
		}

		// Resolve - should use cache from root document, not file system
		result, validationErrs, err := Resolve(context.Background(), Reference("schemas/cached.yaml"), testPrimitiveUnmarshaler, opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)
		require.NotNil(t, result)
		require.NotNil(t, result.Object)
		assert.Equal(t, "/project/api/schemas/cached.yaml", result.AbsoluteReference)

		// Verify the cache from root document was used (not the file system)
		retrievedCache, exists := rootDoc.GetCachedReferenceDocument("/project/api/schemas/cached.yaml")
		assert.True(t, exists)
		assert.Equal(t, cachedData, retrievedCache, "Should use cache from root document")
		assert.Contains(t, string(retrievedCache), "cached: content", "Should contain cached content, not file system content")
	})

	t.Run("resolve JSON pointer against different target document", func(t *testing.T) {
		t.Parallel()

		// Create a root document for caching
		rootDoc := NewMockResolutionTarget()

		// Create a target document with specific structure
		targetDoc := NewTestResolutionTarget()
		targetDoc.InitCache()

		opts := ResolveOptions{
			TargetLocation: "/project/external.yaml",
			RootDocument:   rootDoc,   // Different from target
			TargetDocument: targetDoc, // Has the structure we want to resolve against
		}

		// Resolve a JSON pointer against the target document
		result, validationErrs, err := Resolve(context.Background(), Reference("#/nestedModel"), testPrimitiveUnmarshaler, opts)

		require.NoError(t, err)
		assert.Nil(t, validationErrs)
		require.NotNil(t, result)
		require.NotNil(t, result.Object)
		assert.Equal(t, "nested-string", result.Object.StringField)
		assert.Equal(t, "/project/external.yaml", result.AbsoluteReference)

		// Verify that the resolved document is the target document
		assert.Equal(t, targetDoc, result.ResolvedDocument)
	})

	t.Run("chained resolution scenario - external doc references another external doc", func(t *testing.T) {
		t.Parallel()

		// Simulate a chain: root.yaml -> external1.yaml -> external2.yaml
		// Cache should always be stored in the root document

		rootDoc := NewMockResolutionTarget()

		// Setup file system with a chain of references
		fs := NewMockVirtualFS()
		fs.AddFile("/project/external1.yaml", "reference: external2.yaml\ntype: object")
		fs.AddFile("/project/external2.yaml", "type: object\nproperties:\n  final:\n    type: string")

		// First resolution: root -> external1
		opts1 := ResolveOptions{
			TargetLocation: "/project/root.yaml",
			RootDocument:   rootDoc,
			TargetDocument: rootDoc,
			VirtualFS:      fs,
		}

		result1, validationErrs1, err1 := Resolve(context.Background(), Reference("external1.yaml"), testComplexUnmarshaler, opts1)
		require.NoError(t, err1)
		assert.Nil(t, validationErrs1)
		require.NotNil(t, result1)

		// Verify external1.yaml is cached in root
		cached1, exists1 := rootDoc.GetCachedReferenceDocument("/project/external1.yaml")
		assert.True(t, exists1)
		assert.Contains(t, string(cached1), "reference: external2.yaml")

		// Second resolution: external1 -> external2 (simulating a chained resolution)
		// The key point: root document stays the same for caching, but target changes
		opts2 := ResolveOptions{
			TargetLocation: "/project/external1.yaml",
			RootDocument:   rootDoc,                  // SAME root for caching
			TargetDocument: result1.ResolvedDocument, // DIFFERENT target (external1)
			VirtualFS:      fs,
		}

		result2, validationErrs2, err2 := Resolve(context.Background(), Reference("external2.yaml"), testComplexUnmarshaler, opts2)
		require.NoError(t, err2)
		assert.Nil(t, validationErrs2)
		require.NotNil(t, result2)

		// Verify external2.yaml is ALSO cached in the ROOT document (not external1)
		cached2, exists2 := rootDoc.GetCachedReferenceDocument("/project/external2.yaml")
		assert.True(t, exists2)
		assert.Contains(t, string(cached2), "type: object")
		assert.Contains(t, string(cached2), "final:")

		// Verify we now have both files cached in the root document
		assert.True(t, exists1, "external1.yaml should be cached in root")
		assert.True(t, exists2, "external2.yaml should be cached in root")
	})
}

// Test object caching functionality to ensure objects are shared and memory is not duplicated
func TestResolve_ObjectCaching_Success(t *testing.T) {
	t.Parallel()

	t.Run("same reference returns cached object instance", func(t *testing.T) {
		t.Parallel()
		root := NewMockResolutionTarget()
		fs := NewMockVirtualFS()
		fs.AddFile("/test/schema.yaml", "type: object\nproperties:\n  name:\n    type: string")

		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
			VirtualFS:      fs,
		}

		// First resolution - should cache the object
		result1, validationErrs1, err1 := Resolve(context.Background(), Reference("schema.yaml"), testComplexUnmarshaler, opts)
		require.NoError(t, err1)
		assert.Nil(t, validationErrs1)
		require.NotNil(t, result1)
		require.NotNil(t, result1.Object)

		// Second resolution - should return cached object
		result2, validationErrs2, err2 := Resolve(context.Background(), Reference("schema.yaml"), testComplexUnmarshaler, opts)
		require.NoError(t, err2)
		assert.Nil(t, validationErrs2)
		require.NotNil(t, result2)
		require.NotNil(t, result2.Object)

		// Verify they are the same object instance (not just equal)
		assert.Same(t, result1.Object, result2.Object, "same reference should return same cached object instance")

		// Verify cache contains the object
		cached, exists := root.GetCachedReferencedObject("/test/schema.yaml")
		assert.True(t, exists, "object should be cached")
		assert.Same(t, result1.Object, cached, "cached object should be same instance as resolved object")
	})

	t.Run("different references cache different objects", func(t *testing.T) {
		t.Parallel()

		root := NewMockResolutionTarget()
		fs := NewMockVirtualFS()
		fs.AddFile("/test/schema1.yaml", "type: object\nproperties:\n  name:\n    type: string")
		fs.AddFile("/test/schema2.yaml", "type: object\nproperties:\n  id:\n    type: integer")

		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
			VirtualFS:      fs,
		}

		// Resolve first schema
		result1, validationErrs1, err1 := Resolve(context.Background(), Reference("schema1.yaml"), testComplexUnmarshaler, opts)
		require.NoError(t, err1)
		assert.Nil(t, validationErrs1)
		require.NotNil(t, result1)

		// Resolve second schema
		result2, validationErrs2, err2 := Resolve(context.Background(), Reference("schema2.yaml"), testComplexUnmarshaler, opts)
		require.NoError(t, err2)
		assert.Nil(t, validationErrs2)
		require.NotNil(t, result2)

		// Verify they are different object instances
		assert.NotSame(t, result1.Object, result2.Object, "different references should cache different object instances")

		// Verify both are cached separately
		cached1, exists1 := root.GetCachedReferencedObject("/test/schema1.yaml")
		cached2, exists2 := root.GetCachedReferencedObject("/test/schema2.yaml")
		assert.True(t, exists1, "schema1 should be cached")
		assert.True(t, exists2, "schema2 should be cached")
		assert.Same(t, result1.Object, cached1, "cached schema1 should match resolved")
		assert.Same(t, result2.Object, cached2, "cached schema2 should match resolved")
	})

	t.Run("object cache with JSON pointers", func(t *testing.T) {
		t.Parallel()

		root := NewMockResolutionTarget()
		target := NewTestResolutionTarget()
		target.InitCache()

		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
			TargetDocument: target,
		}

		// First resolution with JSON pointer
		result1, validationErrs1, err1 := Resolve(context.Background(), Reference("#/nestedModel"), testPrimitiveUnmarshaler, opts)
		require.NoError(t, err1)
		assert.Nil(t, validationErrs1)
		require.NotNil(t, result1)

		// Second resolution with same JSON pointer
		result2, validationErrs2, err2 := Resolve(context.Background(), Reference("#/nestedModel"), testPrimitiveUnmarshaler, opts)
		require.NoError(t, err2)
		assert.Nil(t, validationErrs2)
		require.NotNil(t, result2)

		// Verify same object instance returned
		assert.Same(t, result1.Object, result2.Object, "same JSON pointer should return same cached object")

		// Verify cached with correct key (including JSON pointer)
		cached, exists := root.GetCachedReferencedObject("/test/root.yaml#/nestedModel")
		assert.True(t, exists, "object should be cached with JSON pointer in key")
		assert.Same(t, result1.Object, cached, "cached object should match resolved object")
	})

	t.Run("object memory sharing and modification", func(t *testing.T) {
		t.Parallel()

		root := NewMockResolutionTarget()

		// Use a custom unmarshaler that returns a modifiable object
		customUnmarshaler := func(ctx context.Context, node *yaml.Node, skipValidation bool) (*tests.TestComplexHighModel, []error, error) {
			model := &tests.TestComplexHighModel{}
			model.ArrayField = []string{"original"}
			return model, nil, nil
		}

		fs := NewMockVirtualFS()
		fs.AddFile("/test/schema.yaml", "type: object")

		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
			VirtualFS:      fs,
		}

		// First resolution
		result1, validationErrs1, err1 := Resolve(context.Background(), Reference("schema.yaml"), customUnmarshaler, opts)
		require.NoError(t, err1)
		assert.Nil(t, validationErrs1)
		require.NotNil(t, result1)
		require.NotNil(t, result1.Object)

		// Modify the first result
		result1.Object.ArrayField = append(result1.Object.ArrayField, "modified")

		// Second resolution should return the same modified object
		result2, validationErrs2, err2 := Resolve(context.Background(), Reference("schema.yaml"), customUnmarshaler, opts)
		require.NoError(t, err2)
		assert.Nil(t, validationErrs2)
		require.NotNil(t, result2)

		// Verify they share memory (modification is visible in both)
		assert.Same(t, result1.Object, result2.Object, "objects should share memory")
		assert.Equal(t, []string{"original", "modified"}, result2.Object.ArrayField, "modification should be visible in cached object")

		// Verify cached object also reflects the modification
		cached, exists := root.GetCachedReferencedObject("/test/schema.yaml")
		assert.True(t, exists, "object should be cached")
		cachedModel := cached.(*tests.TestComplexHighModel)
		assert.Equal(t, []string{"original", "modified"}, cachedModel.ArrayField, "cached object should reflect modifications")
	})

	t.Run("object cache prevents duplicate unmarshaling", func(t *testing.T) {
		t.Parallel()

		root := NewMockResolutionTarget()
		fs := NewMockVirtualFS()
		fs.AddFile("/test/schema.yaml", "type: object\nproperties:\n  name:\n    type: string")

		// Counter to track unmarshaler calls
		callCount := 0
		countingUnmarshaler := func(ctx context.Context, node *yaml.Node, skipValidation bool) (*tests.TestComplexHighModel, []error, error) {
			callCount++
			model := &tests.TestComplexHighModel{}
			model.ArrayField = []string{"test1", "test2", "test3"}
			return model, nil, nil
		}

		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
			VirtualFS:      fs,
		}

		// First resolution - should call unmarshaler
		result1, validationErrs1, err1 := Resolve(context.Background(), Reference("schema.yaml"), countingUnmarshaler, opts)
		require.NoError(t, err1)
		assert.Nil(t, validationErrs1)
		require.NotNil(t, result1)
		assert.Equal(t, 1, callCount, "unmarshaler should be called once")

		// Second resolution - should use cache, not call unmarshaler
		result2, validationErrs2, err2 := Resolve(context.Background(), Reference("schema.yaml"), countingUnmarshaler, opts)
		require.NoError(t, err2)
		assert.Nil(t, validationErrs2)
		require.NotNil(t, result2)
		assert.Equal(t, 1, callCount, "unmarshaler should not be called again")

		// Verify same object instance (proves caching works)
		assert.Same(t, result1.Object, result2.Object, "should return same cached object instance")

		// Verify object is in cache with correct key format
		cached, exists := root.GetCachedReferencedObject("/test/schema.yaml")
		assert.True(t, exists, "object should be cached")
		assert.Same(t, result1.Object, cached, "cached object should be same instance")
	})
}

func TestResolve_ObjectCaching_Integration_Success(t *testing.T) {
	t.Parallel()

	t.Run("complete cache integration with both document and object caching", func(t *testing.T) {
		t.Parallel()
		root := NewMockResolutionTarget()
		fs := NewMockVirtualFS()

		// Add multiple files to test comprehensive caching
		schemas := map[string]string{
			"/test/user.yaml":    "type: object\nproperties:\n  name:\n    type: string",
			"/test/product.yaml": "type: object\nproperties:\n  id:\n    type: integer",
			"/test/order.yaml":   "type: object\nproperties:\n  total:\n    type: number",
		}

		for path, content := range schemas {
			fs.AddFile(path, content)
		}

		opts := ResolveOptions{
			TargetLocation: "/test/root.yaml",
			RootDocument:   root,
			TargetDocument: root,
			VirtualFS:      fs,
		}

		// Resolve all schemas multiple times
		results := make(map[string][]*ResolveResult[tests.TestComplexHighModel])
		references := []string{"user.yaml", "product.yaml", "order.yaml"}

		// First round of resolutions
		for _, ref := range references {
			result, validationErrs, err := Resolve(context.Background(), Reference(ref), testComplexUnmarshaler, opts)
			require.NoError(t, err, "Failed to resolve %s", ref)
			assert.Nil(t, validationErrs)
			require.NotNil(t, result)
			results[ref] = append(results[ref], result)
		}

		// Second round of resolutions (should use cache)
		for _, ref := range references {
			result, validationErrs, err := Resolve(context.Background(), Reference(ref), testComplexUnmarshaler, opts)
			require.NoError(t, err, "Failed to resolve %s on second attempt", ref)
			assert.Nil(t, validationErrs)
			require.NotNil(t, result)
			results[ref] = append(results[ref], result)
		}

		// Verify all objects are cached and shared
		for _, ref := range references {
			absRef := "/test/" + ref

			// Verify object caching
			cachedObj, objExists := root.GetCachedReferencedObject(absRef)
			assert.True(t, objExists, "Object should be cached for %s", ref)

			// Verify document caching
			cachedDoc, docExists := root.GetCachedReferenceDocument(absRef)
			assert.True(t, docExists, "Document should be cached for %s", ref)
			assert.Contains(t, string(cachedDoc), "type: object", "Cached document should contain expected content for %s", ref)

			// Verify same object instances across resolutions
			first := results[ref][0]
			second := results[ref][1]
			assert.Same(t, first.Object, second.Object, "Same reference should return same object instance for %s", ref)
			assert.Same(t, first.Object, cachedObj, "Resolved object should match cached object for %s", ref)
		}

		// Verify different references have different cached objects
		userObj, _ := root.GetCachedReferencedObject("/test/user.yaml")
		productObj, _ := root.GetCachedReferencedObject("/test/product.yaml")
		orderObj, _ := root.GetCachedReferencedObject("/test/order.yaml")

		assert.NotSame(t, userObj, productObj, "Different references should have different objects")
		assert.NotSame(t, userObj, orderObj, "Different references should have different objects")
		assert.NotSame(t, productObj, orderObj, "Different references should have different objects")
	})
}
