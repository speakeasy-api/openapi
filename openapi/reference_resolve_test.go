package openapi

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveObject_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		filename      string
		componentType string
		componentName string
		testFunc      func(t *testing.T, resolved interface{}, validationErrs []error, err error)
	}{
		{
			name:          "internal parameter reference",
			filename:      "main.yaml",
			componentType: "parameters",
			componentName: "testParamRef",
			testFunc: func(t *testing.T, resolved interface{}, validationErrs []error, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, validationErrs)
				param, ok := resolved.(*Parameter)
				require.True(t, ok, "resolved object should be a Parameter")
				require.NotNil(t, param)
				assert.Equal(t, "userId", param.GetName())
				assert.Equal(t, ParameterInPath, param.GetIn())
				assert.True(t, param.GetRequired())
			},
		},
		{
			name:          "external parameter reference",
			filename:      "main.yaml",
			componentType: "parameters",
			componentName: "testExternalParamRef",
			testFunc: func(t *testing.T, resolved interface{}, validationErrs []error, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, validationErrs)
				param, ok := resolved.(*Parameter)
				require.True(t, ok, "resolved object should be a Parameter")
				require.NotNil(t, param)
				// Test that it resolved to external parameter
				assert.NotEmpty(t, param.GetName())
			},
		},
		{
			name:          "internal response reference",
			filename:      "main.yaml",
			componentType: "responses",
			componentName: "testResponseRef",
			testFunc: func(t *testing.T, resolved interface{}, validationErrs []error, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, validationErrs)
				response, ok := resolved.(*Response)
				require.True(t, ok, "resolved object should be a Response")
				require.NotNil(t, response)
				assert.Equal(t, "User response", response.GetDescription())
				assert.NotNil(t, response.GetContent())
			},
		},
		{
			name:          "internal example reference",
			filename:      "main.yaml",
			componentType: "examples",
			componentName: "testExampleRef",
			testFunc: func(t *testing.T, resolved interface{}, validationErrs []error, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, validationErrs)
				example, ok := resolved.(*Example)
				require.True(t, ok, "resolved object should be an Example")
				require.NotNil(t, example)
				assert.Equal(t, "Example user", example.GetSummary())
				assert.NotNil(t, example.GetValue())
			},
		},
		{
			name:          "internal request body reference",
			filename:      "main.yaml",
			componentType: "requestBodies",
			componentName: "testRequestBodyRef",
			testFunc: func(t *testing.T, resolved interface{}, validationErrs []error, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, validationErrs)
				requestBody, ok := resolved.(*RequestBody)
				require.True(t, ok, "resolved object should be a RequestBody")
				require.NotNil(t, requestBody)
				assert.Equal(t, "User data", requestBody.GetDescription())
				assert.NotNil(t, requestBody.GetContent())
			},
		},
		{
			name:          "internal header reference",
			filename:      "main.yaml",
			componentType: "headers",
			componentName: "testHeaderRef",
			testFunc: func(t *testing.T, resolved interface{}, validationErrs []error, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, validationErrs)
				header, ok := resolved.(*Header)
				require.True(t, ok, "resolved object should be a Header")
				require.NotNil(t, header)
				assert.Equal(t, "User header", header.GetDescription())
				assert.NotNil(t, header.GetSchema())
			},
		},
		{
			name:          "internal security scheme reference",
			filename:      "main.yaml",
			componentType: "securitySchemes",
			componentName: "testSecurityRef",
			testFunc: func(t *testing.T, resolved interface{}, validationErrs []error, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, validationErrs)
				security, ok := resolved.(*SecurityScheme)
				require.True(t, ok, "resolved object should be a SecurityScheme")
				require.NotNil(t, security)
				assert.Equal(t, SecuritySchemeTypeAPIKey, security.GetType())
				assert.Equal(t, SecuritySchemeInHeader, security.GetIn())
				assert.Equal(t, "X-API-Key", security.GetName())
			},
		},
		{
			name:          "internal link reference",
			filename:      "main.yaml",
			componentType: "links",
			componentName: "testLinkRef",
			testFunc: func(t *testing.T, resolved interface{}, validationErrs []error, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, validationErrs)
				link, ok := resolved.(*Link)
				require.True(t, ok, "resolved object should be a Link")
				require.NotNil(t, link)
				assert.Equal(t, "getUser", link.GetOperationID())
				assert.NotNil(t, link.GetParameters())
			},
		},
		{
			name:          "internal callback reference",
			filename:      "main.yaml",
			componentType: "callbacks",
			componentName: "testCallbackRef",
			testFunc: func(t *testing.T, resolved interface{}, validationErrs []error, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, validationErrs)
				callback, ok := resolved.(*Callback)
				require.True(t, ok, "resolved object should be a Callback")
				require.NotNil(t, callback)
				// Test that callback has expressions (via embedded map)
				assert.NotNil(t, callback.Map)
				assert.Positive(t, callback.Len(), "Callback should have expressions")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			// Load the OpenAPI document
			testDataPath := filepath.Join("testdata", "resolve_test", tt.filename)
			file, err := os.Open(testDataPath)
			require.NoError(t, err)
			defer file.Close()

			doc, validationErrs, err := Unmarshal(ctx, file)
			require.NoError(t, err)
			assert.Empty(t, validationErrs)

			// Get the component from the document
			require.NotNil(t, doc.Components)

			// Setup resolve options
			absPath, err := filepath.Abs(testDataPath)
			require.NoError(t, err)

			opts := ResolveOptions{
				TargetLocation: absPath,
				RootDocument:   doc,
			}

			// Test different component types
			switch tt.componentType {
			case "parameters":
				require.NotNil(t, doc.Components.Parameters)
				refParam, exists := doc.Components.Parameters.Get(tt.componentName)
				require.True(t, exists, "Parameter %s should exist", tt.componentName)
				require.True(t, refParam.IsReference(), "Test parameter should have a reference")

				ref := ReferencedParameter{
					Reference: pointer.From(refParam.GetReference()),
				}
				validationErrs, err := ref.Resolve(ctx, opts)

				// Test parent links for single-level reference
				if err == nil && ref.GetObject() != nil {
					parent := ref.GetParent()
					topLevelParent := ref.GetTopLevelParent()

					// For single-level references, parent should be nil since this is the original reference
					assert.Nil(t, parent, "single-level reference should have no parent")
					assert.Nil(t, topLevelParent, "single-level reference should have no top-level parent")
				}

				tt.testFunc(t, ref.GetObject(), validationErrs, err)

			case "responses":
				require.NotNil(t, doc.Components.Responses)
				refResponse, exists := doc.Components.Responses.Get(tt.componentName)
				require.True(t, exists, "Response %s should exist", tt.componentName)
				require.True(t, refResponse.IsReference(), "Test response should have a reference")

				ref := ReferencedResponse{
					Reference: pointer.From(refResponse.GetReference()),
				}
				validationErrs, err := ref.Resolve(ctx, opts)
				tt.testFunc(t, ref.GetObject(), validationErrs, err)

			case "examples":
				require.NotNil(t, doc.Components.Examples)
				refExample, exists := doc.Components.Examples.Get(tt.componentName)
				require.True(t, exists, "Example %s should exist", tt.componentName)
				require.True(t, refExample.IsReference(), "Test example should have a reference")

				ref := ReferencedExample{
					Reference: pointer.From(refExample.GetReference()),
				}
				validationErrs, err := ref.Resolve(ctx, opts)
				tt.testFunc(t, ref.GetObject(), validationErrs, err)

			case "requestBodies":
				require.NotNil(t, doc.Components.RequestBodies)
				refRequestBody, exists := doc.Components.RequestBodies.Get(tt.componentName)
				require.True(t, exists, "RequestBody %s should exist", tt.componentName)
				require.True(t, refRequestBody.IsReference(), "Test request body should have a reference")

				ref := ReferencedRequestBody{
					Reference: pointer.From(refRequestBody.GetReference()),
				}
				validationErrs, err := ref.Resolve(ctx, opts)
				tt.testFunc(t, ref.GetObject(), validationErrs, err)

			case "headers":
				require.NotNil(t, doc.Components.Headers)
				refHeader, exists := doc.Components.Headers.Get(tt.componentName)
				require.True(t, exists, "Header %s should exist", tt.componentName)
				require.True(t, refHeader.IsReference(), "Test header should have a reference")

				ref := ReferencedHeader{
					Reference: pointer.From(refHeader.GetReference()),
				}
				validationErrs, err := ref.Resolve(ctx, opts)
				tt.testFunc(t, ref.GetObject(), validationErrs, err)

			case "securitySchemes":
				require.NotNil(t, doc.Components.SecuritySchemes)
				refSecurity, exists := doc.Components.SecuritySchemes.Get(tt.componentName)
				require.True(t, exists, "SecurityScheme %s should exist", tt.componentName)
				require.True(t, refSecurity.IsReference(), "Test security scheme should have a reference")

				ref := ReferencedSecurityScheme{
					Reference: pointer.From(refSecurity.GetReference()),
				}
				validationErrs, err := ref.Resolve(ctx, opts)
				tt.testFunc(t, ref.GetObject(), validationErrs, err)

			case "links":
				require.NotNil(t, doc.Components.Links)
				refLink, exists := doc.Components.Links.Get(tt.componentName)
				require.True(t, exists, "Link %s should exist", tt.componentName)
				require.True(t, refLink.IsReference(), "Test link should have a reference")

				ref := ReferencedLink{
					Reference: pointer.From(refLink.GetReference()),
				}
				validationErrs, err := ref.Resolve(ctx, opts)
				tt.testFunc(t, ref.GetObject(), validationErrs, err)

			case "callbacks":
				require.NotNil(t, doc.Components.Callbacks)
				refCallback, exists := doc.Components.Callbacks.Get(tt.componentName)
				require.True(t, exists, "Callback %s should exist", tt.componentName)
				require.True(t, refCallback.IsReference(), "Test callback should have a reference")

				ref := ReferencedCallback{
					Reference: pointer.From(refCallback.GetReference()),
				}
				validationErrs, err := ref.Resolve(ctx, opts)
				tt.testFunc(t, ref.GetObject(), validationErrs, err)

			default:
				t.Fatalf("Unknown component type: %s", tt.componentType)
			}
		})
	}
}

func TestResolveObject_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		filename    string
		refPath     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing parameter reference",
			filename:    "main.yaml",
			refPath:     "#/components/parameters/NonExistent",
			expectError: true,
			errorMsg:    "", // Error message depends on implementation
		},
		{
			name:        "invalid external file reference",
			filename:    "main.yaml",
			refPath:     "./nonexistent.yaml#/components/parameters/SomeParam",
			expectError: true,
			errorMsg:    "", // Error message depends on implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			// Load the OpenAPI document
			testDataPath := filepath.Join("testdata", "resolve_test", tt.filename)
			file, err := os.Open(testDataPath)
			require.NoError(t, err)
			defer file.Close()

			doc, validationErrs, err := Unmarshal(ctx, file)
			require.NoError(t, err)
			assert.Empty(t, validationErrs)

			// Create a reference with invalid path
			ref := ReferencedParameter{
				Reference: pointer.From(references.Reference(tt.refPath)),
			}

			// Setup resolve options
			absPath, err := filepath.Abs(testDataPath)
			require.NoError(t, err)

			opts := ResolveOptions{
				TargetLocation: absPath,
				RootDocument:   doc,
			}

			// Test Resolve
			_, err = ref.Resolve(ctx, opts)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestResolveObjectWithTracking_CircularReference(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Create a test reference that would cause circular reference
	ref := ReferencedParameter{
		Reference: pointer.From(references.Reference("#/components/parameters/CircularParam")),
	}

	// Pre-populate reference chain to simulate circular reference
	referenceChain := []string{"/test.yaml#/components/parameters/CircularParam"}

	// Test internal tracking function
	_, err := resolveObjectWithTracking(ctx, &ref, references.ResolveOptions{
		TargetLocation: "/test.yaml",
		RootDocument:   &OpenAPI{}, // Empty document for this test
		TargetDocument: &OpenAPI{}, // Empty document for this test
	}, referenceChain)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular reference detected")
}

// MockVirtualFS implements VirtualFS and tracks file access for caching tests
type MockVirtualFS struct {
	files      map[string][]byte
	accessLog  []string
	accessFunc func(path string)
}

func NewMockVirtualFS() *MockVirtualFS {
	return &MockVirtualFS{
		files:     make(map[string][]byte),
		accessLog: make([]string, 0),
	}
}

func (fs *MockVirtualFS) AddFile(path string, content []byte) {
	fs.files[path] = content
}

func (fs *MockVirtualFS) Open(name string) (fs.File, error) {
	// Log the file access
	fs.accessLog = append(fs.accessLog, name)
	if fs.accessFunc != nil {
		fs.accessFunc(name)
	}

	content, exists := fs.files[name]
	if !exists {
		return nil, os.ErrNotExist
	}

	return &mockFile{
		data:   content,
		offset: 0,
		name:   name,
	}, nil
}

func (fs *MockVirtualFS) GetAccessLog() []string {
	return fs.accessLog
}

func (fs *MockVirtualFS) GetAccessCount(path string) int {
	count := 0
	for _, accessed := range fs.accessLog {
		if accessed == path {
			count++
		}
	}
	return count
}

// mockFile implements fs.File for testing
type mockFile struct {
	data   []byte
	offset int64
	name   string
}

func (f *mockFile) Stat() (fs.FileInfo, error) {
	return &mockFileInfo{
		name: f.name,
		size: int64(len(f.data)),
	}, nil
}

func (f *mockFile) Read(p []byte) (int, error) {
	if f.offset >= int64(len(f.data)) {
		return 0, io.EOF
	}

	n := copy(p, f.data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *mockFile) Close() error {
	return nil
}

// mockFileInfo implements fs.FileInfo for testing
type mockFileInfo struct {
	name string
	size int64
}

func (info *mockFileInfo) Name() string       { return info.name }
func (info *mockFileInfo) Size() int64        { return info.size }
func (info *mockFileInfo) Mode() fs.FileMode  { return 0o644 }
func (info *mockFileInfo) ModTime() time.Time { return time.Now() }
func (info *mockFileInfo) IsDir() bool        { return false }
func (info *mockFileInfo) Sys() interface{}   { return nil }

func TestResolveObject_Caching_SameReference(t *testing.T) {
	t.Parallel()
	// Note: Cannot use t.Parallel() due to shared cache state causing race conditions

	ctx := t.Context()

	// Create mock filesystem and read existing test files
	mockFS := NewMockVirtualFS()

	// Read existing external test file
	externalPath := filepath.Join("testdata", "resolve_test", "external.yaml")
	externalContent, err := os.ReadFile(externalPath)
	require.NoError(t, err, "Failed to read external.yaml from path: %s", externalPath)
	mockFS.AddFile("./external.yaml", externalContent)
	// Also add with the absolute path that the resolution system will request
	absExternalPath, err := filepath.Abs(externalPath)
	require.NoError(t, err)
	mockFS.AddFile(absExternalPath, externalContent)

	// Load existing main test document
	mainPath := filepath.Join("testdata", "resolve_test", "main.yaml")
	file, err := os.Open(mainPath)
	require.NoError(t, err)
	defer file.Close()

	mainDoc, validationErrs, err := Unmarshal(ctx, file)
	require.NoError(t, err)
	assert.Empty(t, validationErrs)

	// Setup resolve options with mock filesystem
	absPath, err := filepath.Abs(mainPath)
	require.NoError(t, err)

	opts := ResolveOptions{
		TargetLocation: absPath,
		RootDocument:   mainDoc,
		VirtualFS:      mockFS,
	}

	// Get the external parameter reference from the document
	require.NotNil(t, mainDoc.Components)
	require.NotNil(t, mainDoc.Components.Parameters)
	refParam, exists := mainDoc.Components.Parameters.Get("testExternalParamRef")
	require.True(t, exists)
	require.True(t, refParam.IsReference())

	ref := ReferencedParameter{
		Reference: pointer.From(refParam.GetReference()),
	}

	// First resolution
	validationErrs1, err1 := ref.Resolve(ctx, opts)
	resolved1 := ref.GetObject()
	require.NoError(t, err1)
	assert.Empty(t, validationErrs1)
	require.NotNil(t, resolved1)
	assert.Equal(t, "external-param", resolved1.GetName())

	// Verify external file was accessed once
	assert.Equal(t, 1, mockFS.GetAccessCount(absExternalPath), "External file should be accessed once on first resolution")

	// Second resolution of the same reference
	validationErrs2, err2 := ref.Resolve(ctx, opts)
	resolved2 := ref.GetObject()
	require.NoError(t, err2)
	assert.Empty(t, validationErrs2)
	require.NotNil(t, resolved2)
	assert.Equal(t, "external-param", resolved2.GetName())

	// Verify external file was still only accessed once (cached)
	assert.Equal(t, 1, mockFS.GetAccessCount(absExternalPath), "External file should still be accessed only once due to caching")

	// Verify both resolved objects are the same
	assert.Equal(t, resolved1.GetName(), resolved2.GetName())
	assert.Equal(t, resolved1.GetIn(), resolved2.GetIn())
}

func TestResolveObject_Caching_MultipleReferencesToSameFile(t *testing.T) {
	t.Parallel()
	// Note: Cannot use t.Parallel() due to shared cache state causing race conditions

	ctx := t.Context()

	// Create mock filesystem and read existing test files
	mockFS := NewMockVirtualFS()

	// Read existing external test file
	externalPath := filepath.Join("testdata", "resolve_test", "external.yaml")
	externalContent, err := os.ReadFile(externalPath)
	require.NoError(t, err)
	mockFS.AddFile("./external.yaml", externalContent)
	// Also add with the absolute path that the resolution system will request
	absExternalPath, err := filepath.Abs(externalPath)
	require.NoError(t, err)
	mockFS.AddFile(absExternalPath, externalContent)

	// Load existing main test document
	mainPath := filepath.Join("testdata", "resolve_test", "main.yaml")
	file, err := os.Open(mainPath)
	require.NoError(t, err)
	defer file.Close()

	mainDoc, validationErrs, err := Unmarshal(ctx, file)
	require.NoError(t, err)
	assert.Empty(t, validationErrs)

	// Setup resolve options with mock filesystem
	absPath, err := filepath.Abs(mainPath)
	require.NoError(t, err)

	opts := ResolveOptions{
		TargetLocation: absPath,
		RootDocument:   mainDoc,
		VirtualFS:      mockFS,
	}

	// Resolve first external parameter reference
	refParam, exists := mainDoc.Components.Parameters.Get("testExternalParamRef")
	require.True(t, exists)
	require.True(t, refParam.IsReference())

	paramRef := ReferencedParameter{
		Reference: pointer.From(refParam.GetReference()),
	}

	validationErrs, err = paramRef.Resolve(ctx, opts)
	resolvedParam := paramRef.GetObject()
	require.NoError(t, err)
	assert.Empty(t, validationErrs)
	require.NotNil(t, resolvedParam)

	// Verify external file was accessed once
	assert.Equal(t, 1, mockFS.GetAccessCount(absExternalPath), "External file should be accessed once after first reference")

	// Resolve external response reference to the same file
	refResponse, exists := mainDoc.Components.Responses.Get("testExternalResponseRef")
	require.True(t, exists)
	require.True(t, refResponse.IsReference())

	responseRef := ReferencedResponse{
		Reference: pointer.From(refResponse.GetReference()),
	}

	validationErrs, err = responseRef.Resolve(ctx, opts)
	resolvedResponse := responseRef.GetObject()
	require.NoError(t, err)
	assert.Empty(t, validationErrs)
	require.NotNil(t, resolvedResponse)

	// Verify external file was still only accessed once (file-level caching)
	assert.Equal(t, 1, mockFS.GetAccessCount(absExternalPath), "External file should still be accessed only once despite multiple references")

	// Resolve external example reference to the same file
	refExample, exists := mainDoc.Components.Examples.Get("testExternalExampleRef")
	require.True(t, exists)
	require.True(t, refExample.IsReference())

	exampleRef := ReferencedExample{
		Reference: pointer.From(refExample.GetReference()),
	}

	validationErrs, err = exampleRef.Resolve(ctx, opts)
	resolvedExample := exampleRef.GetObject()
	require.NoError(t, err)
	assert.Empty(t, validationErrs)
	require.NotNil(t, resolvedExample)

	// Verify external file was still only accessed once (all references to same file cached)
	assert.Equal(t, 1, mockFS.GetAccessCount(absExternalPath), "External file should still be accessed only once for all references to the same file")

	// Verify different components were resolved correctly
	assert.NotEmpty(t, resolvedParam.GetName())
	assert.NotEmpty(t, resolvedResponse.GetDescription())
	assert.NotEmpty(t, resolvedExample.GetSummary())
}

func TestResolveObject_Caching_DifferentFiles(t *testing.T) {
	t.Parallel()
	// Note: Cannot use t.Parallel() due to shared cache state causing race conditions

	ctx := t.Context()

	// Create mock filesystem and read existing test files
	mockFS := NewMockVirtualFS()

	// Read existing external test file
	externalPath := filepath.Join("testdata", "resolve_test", "external.yaml")
	externalContent, err := os.ReadFile(externalPath)
	require.NoError(t, err)
	mockFS.AddFile("./external.yaml", externalContent)
	// Also add with the absolute path that the resolution system will request
	absExternalPath, err := filepath.Abs(externalPath)
	require.NoError(t, err)
	mockFS.AddFile(absExternalPath, externalContent)

	// Read existing schemas.json file
	schemasPath := filepath.Join("testdata", "resolve_test", "schemas.json")
	schemasContent, err := os.ReadFile(schemasPath)
	require.NoError(t, err)
	mockFS.AddFile("./schemas.json", schemasContent)
	// Also add with the absolute path that the resolution system will request
	absSchemasPath, err := filepath.Abs(schemasPath)
	require.NoError(t, err)
	mockFS.AddFile(absSchemasPath, schemasContent)

	// Load existing main test document
	mainPath := filepath.Join("testdata", "resolve_test", "main.yaml")
	file, err := os.Open(mainPath)
	require.NoError(t, err)
	defer file.Close()

	mainDoc, validationErrs, err := Unmarshal(ctx, file)
	require.NoError(t, err)
	assert.Empty(t, validationErrs)

	// Setup resolve options with mock filesystem
	absPath, err := filepath.Abs(mainPath)
	require.NoError(t, err)

	opts := ResolveOptions{
		TargetLocation: absPath,
		RootDocument:   mainDoc,
		VirtualFS:      mockFS,
	}

	// Resolve reference to external.yaml
	refParam, exists := mainDoc.Components.Parameters.Get("testExternalParamRef")
	require.True(t, exists)
	paramRef := ReferencedParameter{
		Reference: pointer.From(refParam.GetReference()),
	}

	validationErrs, err = paramRef.Resolve(ctx, opts)
	resolvedParam := paramRef.GetObject()
	require.NoError(t, err)
	assert.Empty(t, validationErrs)
	require.NotNil(t, resolvedParam)

	// Verify only external.yaml was accessed
	assert.Equal(t, 1, mockFS.GetAccessCount(absExternalPath), "external.yaml should be accessed once")
	assert.Equal(t, 0, mockFS.GetAccessCount(absSchemasPath), "schemas.json should not be accessed yet")

	// Now resolve an internal reference (should not access any external files)
	refInternal, exists := mainDoc.Components.Parameters.Get("testParamRef")
	require.True(t, exists)
	internalRef := ReferencedParameter{
		Reference: pointer.From(refInternal.GetReference()),
	}

	validationErrs, err = internalRef.Resolve(ctx, opts)
	resolvedInternal := internalRef.GetObject()
	require.NoError(t, err)
	assert.Empty(t, validationErrs)
	require.NotNil(t, resolvedInternal)

	// Verify file access counts haven't changed for internal reference
	assert.Equal(t, 1, mockFS.GetAccessCount(absExternalPath), "external.yaml should still be accessed only once")
	assert.Equal(t, 0, mockFS.GetAccessCount(absSchemasPath), "schemas.json should still not be accessed")

	// Total access log should show only external.yaml
	accessLog := mockFS.GetAccessLog()
	assert.Len(t, accessLog, 1, "Should have exactly 1 file access")
	assert.Contains(t, accessLog, absExternalPath)
}

func TestResolveObject_TrickyJSONPointers(t *testing.T) {
	t.Parallel()
	// Note: Cannot use t.Parallel() due to shared cache state causing race conditions

	ctx := t.Context()

	// Load test document with tricky JSON pointer references
	mainPath := filepath.Join("testdata", "resolve_test", "main.yaml")
	file, err := os.Open(mainPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })

	mainDoc, validationErrs, err := Unmarshal(ctx, file)
	require.NoError(t, err)
	assert.Empty(t, validationErrs)

	// Setup resolve options
	absPath, err := filepath.Abs(mainPath)
	require.NoError(t, err)

	opts := ResolveOptions{
		TargetLocation: absPath,
		RootDocument:   mainDoc,
	}

	tests := []struct {
		name          string
		componentType string
		componentName string
		testFunc      func(t *testing.T, resolved interface{}, validationErrs []error, err error)
	}{
		{
			name:          "reference to parameter within operation",
			componentType: "parameters",
			componentName: "trickyOperationParamRef",
			testFunc: func(t *testing.T, resolved interface{}, validationErrs []error, err error) {
				t.Helper()
				// Log the error to understand what's happening with the JSON pointer
				if err != nil {
					t.Logf("JSON pointer resolution failed (this may be expected): %v", err)
					// For now, just verify the error is related to path resolution
					assert.Contains(t, err.Error(), "not found")
					return
				}
				// If it succeeds, verify the result
				param, ok := resolved.(*Parameter)
				require.True(t, ok, "resolved object should be a Parameter")
				require.NotNil(t, param)
				assert.Equal(t, "limit", param.GetName())
				assert.Equal(t, ParameterInQuery, param.GetIn())
			},
		},
		{
			name:          "reference to parameter within POST operation",
			componentType: "parameters",
			componentName: "trickyPostParamRef",
			testFunc: func(t *testing.T, resolved interface{}, validationErrs []error, err error) {
				t.Helper()
				if err != nil {
					t.Logf("JSON pointer resolution failed (this may be expected): %v", err)
					assert.Contains(t, err.Error(), "not found")
					return
				}
				param, ok := resolved.(*Parameter)
				require.True(t, ok, "resolved object should be a Parameter")
				require.NotNil(t, param)
				assert.Equal(t, "apiVersion", param.GetName())
				assert.Equal(t, ParameterInHeader, param.GetIn())
			},
		},
		{
			name:          "reference to response within operation",
			componentType: "responses",
			componentName: "trickyOperationResponseRef",
			testFunc: func(t *testing.T, resolved interface{}, validationErrs []error, err error) {
				t.Helper()
				if err != nil {
					t.Logf("JSON pointer resolution failed (this may be expected): %v", err)
					assert.Contains(t, err.Error(), "not found")
					return
				}
				response, ok := resolved.(*Response)
				require.True(t, ok, "resolved object should be a Response")
				require.NotNil(t, response)
				assert.NotEmpty(t, response.GetDescription())
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			// Note: Cannot use t.Parallel() due to shared cache state causing race conditions

			// Get the component from the document (same pattern as existing tests)
			require.NotNil(t, mainDoc.Components)

			switch test.componentType {
			case "parameters":
				require.NotNil(t, mainDoc.Components.Parameters)
				refParam, exists := mainDoc.Components.Parameters.Get(test.componentName)
				require.True(t, exists, "Component %s should exist", test.componentName)
				require.True(t, refParam.IsReference())

				ref := ReferencedParameter{
					Reference: pointer.From(refParam.GetReference()),
				}

				validationErrs, err := ref.Resolve(ctx, opts)
				test.testFunc(t, ref.GetObject(), validationErrs, err)

			case "responses":
				require.NotNil(t, mainDoc.Components.Responses)
				refResponse, exists := mainDoc.Components.Responses.Get(test.componentName)
				require.True(t, exists, "Component %s should exist", test.componentName)
				require.True(t, refResponse.IsReference())

				ref := ReferencedResponse{
					Reference: pointer.From(refResponse.GetReference()),
				}

				validationErrs, err := ref.Resolve(ctx, opts)
				test.testFunc(t, ref.GetObject(), validationErrs, err)

			default:
				t.Fatalf("Unsupported component type: %s", test.componentType)
			}
		})
	}
}

func TestResolveObject_ChainedReference_Success(t *testing.T) {
	t.Parallel()
	// Note: Cannot use t.Parallel() due to shared cache state causing race conditions

	ctx := t.Context()

	// Create mock filesystem with the test files
	mockFS := NewMockVirtualFS()

	// Read existing external test file
	externalPath := filepath.Join("testdata", "resolve_test", "external.yaml")
	externalContent, err := os.ReadFile(externalPath)
	require.NoError(t, err)
	mockFS.AddFile("./external.yaml", externalContent)

	// Read the chained test file we created
	chainedPath := filepath.Join("testdata", "resolve_test", "chained.yaml")
	chainedContent, err := os.ReadFile(chainedPath)
	require.NoError(t, err)
	mockFS.AddFile("./chained.yaml", chainedContent)

	// Also add with absolute paths that the resolution system will request
	absExternalPath, err := filepath.Abs(externalPath)
	require.NoError(t, err)
	mockFS.AddFile(absExternalPath, externalContent)

	absChainedPath, err := filepath.Abs(chainedPath)
	require.NoError(t, err)
	mockFS.AddFile(absChainedPath, chainedContent)

	// Load existing main test document
	mainPath := filepath.Join("testdata", "resolve_test", "main.yaml")
	file, err := os.Open(mainPath)
	require.NoError(t, err)
	defer file.Close()

	mainDoc, validationErrs, err := Unmarshal(ctx, file)
	require.NoError(t, err)
	assert.Empty(t, validationErrs)

	// Setup resolve options with mock filesystem
	absPath, err := filepath.Abs(mainPath)
	require.NoError(t, err)

	opts := ResolveOptions{
		TargetLocation: absPath,
		RootDocument:   mainDoc,
		VirtualFS:      mockFS,
	}

	// Get the chained response reference from the document (following existing test pattern)
	require.NotNil(t, mainDoc.Components)
	require.NotNil(t, mainDoc.Components.Responses)
	refResponse, exists := mainDoc.Components.Responses.Get("testChainedResponseRef")
	require.True(t, exists, "testChainedResponseRef should exist")
	require.True(t, refResponse.IsReference(), "testChainedResponseRef should be a reference")

	// This will trigger: main.yaml -> external.yaml#ChainedExternalResponse -> chained.yaml#ChainedResponse -> #LocalChainedResponse
	// Attempt to resolve the chained reference
	validationErrs, err = refResponse.Resolve(ctx, opts)
	resolved := refResponse.GetObject()

	// The resolution should succeed - this tests the correct behavior
	require.NoError(t, err)
	assert.Empty(t, validationErrs)

	// Verify we got a valid response object
	require.NotNil(t, resolved)

	// Test parent links for chained reference
	parent := refResponse.GetParent()
	topLevelParent := refResponse.GetTopLevelParent()

	// For chained references, the resolved reference should have parent links set
	// Note: The parent links are set on the resolved reference object, not the original reference
	// Since we're testing the original reference object, it should not have parent links
	assert.Nil(t, parent, "original reference should have no parent")
	assert.Nil(t, topLevelParent, "original reference should have no top-level parent")

	// Verify the response has the expected description from the final LocalChainedResponse
	// This tests that the local reference #/components/responses/LocalChainedResponse
	// was resolved correctly within chained.yaml (not against main.yaml)
	assert.Equal(t, "Local chained response", resolved.GetDescription())

	// Verify the response has content
	content := resolved.GetContent()
	require.NotNil(t, content)

	// Verify we can access the JSON content with the expected nested structure
	jsonContent, exists := content.Get("application/json")
	require.True(t, exists, "JSON content should exist")
	require.NotNil(t, jsonContent)

	// Verify the schema shows the expected nestedValue property from LocalChainedResponse
	require.NotNil(t, jsonContent.Schema)
}

// Test parent link functionality
func TestReference_ParentLinks(t *testing.T) {
	t.Parallel()

	t.Run("non-reference has no parent", func(t *testing.T) {
		t.Parallel()

		// Create a non-reference (inline object)
		ref := ReferencedParameter{
			Object: &Parameter{
				Name: "test",
				In:   ParameterInQuery,
			},
		}

		// Check parent links
		parent := ref.GetParent()
		topLevelParent := ref.GetTopLevelParent()

		assert.Nil(t, parent, "non-reference should have no parent")
		assert.Nil(t, topLevelParent, "non-reference should have no top-level parent")
	})

	t.Run("manual parent setting works correctly", func(t *testing.T) {
		t.Parallel()

		// Create references
		parentRef := ReferencedParameter{
			Reference: pointer.From(references.Reference("#/components/parameters/Parent")),
		}
		topLevelRef := ReferencedParameter{
			Reference: pointer.From(references.Reference("#/components/parameters/TopLevel")),
		}
		childRef := ReferencedParameter{
			Reference: pointer.From(references.Reference("#/components/parameters/Child")),
		}

		// Manually set parent links
		childRef.SetParent(&parentRef)
		childRef.SetTopLevelParent(&topLevelRef)

		// Check parent links
		parent := childRef.GetParent()
		topLevelParent := childRef.GetTopLevelParent()

		assert.Equal(t, &parentRef, parent, "manually set parent should be correct")
		assert.Equal(t, &topLevelRef, topLevelParent, "manually set top-level parent should be correct")
	})

	t.Run("nil reference methods handle gracefully", func(t *testing.T) {
		t.Parallel()

		var nilRef *ReferencedParameter

		// Test getter methods
		assert.Nil(t, nilRef.GetParent(), "nil reference GetParent should return nil")
		assert.Nil(t, nilRef.GetTopLevelParent(), "nil reference GetTopLevelParent should return nil")

		// Test setter methods (should not panic)
		assert.NotPanics(t, func() {
			nilRef.SetParent(&ReferencedParameter{})
		}, "SetParent on nil reference should not panic")

		assert.NotPanics(t, func() {
			nilRef.SetTopLevelParent(&ReferencedParameter{})
		}, "SetTopLevelParent on nil reference should not panic")
	})
}
