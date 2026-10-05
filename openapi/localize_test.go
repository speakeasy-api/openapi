package openapi_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type localizeTestFS struct {
	fstest.MapFS
	openErr  error
	writeErr error
}

func (f *localizeTestFS) Open(name string) (fs.File, error) {
	if f.openErr != nil {
		return nil, f.openErr
	}
	return f.MapFS.Open(filepath.ToSlash(name))
}

func (f *localizeTestFS) MkdirAll(name string, mode fs.FileMode) error {
	f.MapFS[filepath.ToSlash(name)] = &fstest.MapFile{Mode: mode | fs.ModeDir}
	return nil
}

func (f *localizeTestFS) WriteFile(name string, content []byte, mode fs.FileMode) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	f.MapFS[filepath.ToSlash(name)] = &fstest.MapFile{Data: content, Mode: mode}
	return nil
}

func TestLocalize_MissingReusableObject_Error(t *testing.T) {
	t.Parallel()
	for _, section := range []string{"responses", "parameters", "requestBodies", "headers", "examples", "links", "callbacks", "pathItems", "securitySchemes"} {
		t.Run(section, func(t *testing.T) {
			t.Parallel()
			doc := unmarshalOpenAPI(t, t.Context(), "openapi: 3.1.0\ninfo: {title: Missing object, version: '1'}\npaths: {}\ncomponents:\n  "+section+":\n    Missing: {$ref: 'missing.yaml#/Object'}\n")
			vfs := &localizeTestFS{MapFS: fstest.MapFS{}}
			err := openapi.Localize(t.Context(), doc, openapi.LocalizeOptions{DocumentLocation: "openapi.yaml", TargetDirectory: "output", VirtualFS: vfs})
			require.ErrorContains(t, err, "missing.yaml", "missing reusable objects should report their source rather than panic")
			var output bytes.Buffer
			require.NoError(t, openapi.Marshal(t.Context(), doc, &output), "failed localization should leave a serializable document")
			assert.Contains(t, output.String(), "missing.yaml#/Object", "failed resolution should leave the original reference unchanged")
			assert.Empty(t, vfs.MapFS, "failed discovery must not create directories or copy partial files")
		})
	}
}

func TestLocalize_NamingFallbacks_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		strategy openapi.LocalizeNamingStrategy
		first    string
		second   string
	}{
		{name: "path", strategy: openapi.LocalizeNamingPathBased, first: "shared.yaml", second: "two-shared.yaml"},
		{name: "counter", strategy: openapi.LocalizeNamingCounter, first: "shared.yaml", second: "shared_1.yaml"},
		{name: "custom without function", strategy: openapi.LocalizeNamingCustom, first: "one-shared.yaml", second: "two-shared.yaml"},
		{name: "unknown strategy", strategy: openapi.LocalizeNamingStrategy(99), first: "one-shared.yaml", second: "two-shared.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			vfs := &localizeTestFS{MapFS: fstest.MapFS{
				"one/shared.yaml": {Data: []byte("Thing:\n  type: string\n  enum: [first]\n")},
				"two/shared.yaml": {Data: []byte("Thing:\n  type: string\n  enum: [second]\n")},
			}}
			doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.1.0
info:
  title: Naming conflicts
  version: 1.0.0
components:
  schemas:
    First:
      $ref: './one/shared.yaml#/Thing'
    Second:
      $ref: './two/shared.yaml#/Thing'
    Duplicate:
      $ref: './one/shared.yaml#/Thing'
    Internal:
      $ref: '#/components/schemas/First'
`))
			require.NoError(t, err, "unmarshal document with filename conflicts")
			require.Empty(t, validationErrs, "document should be valid")
			err = openapi.Localize(ctx, doc, openapi.LocalizeOptions{
				DocumentLocation: "openapi.yaml",
				TargetDirectory:  "output",
				VirtualFS:        vfs,
				NamingStrategy:   tt.strategy,
			})
			require.NoError(t, err, "localize distinct files sharing a basename")
			assert.Equal(t, tt.first+"#/Thing", string(doc.Components.Schemas.GetOrZero("First").GetRef()), "first file should use the strategy's filename")
			assert.Equal(t, tt.second+"#/Thing", string(doc.Components.Schemas.GetOrZero("Second").GetRef()), "second file should use a conflict-free filename")
			assert.Equal(t, tt.first+"#/Thing", string(doc.Components.Schemas.GetOrZero("Duplicate").GetRef()), "duplicate reference should reuse the localized file")
			assert.Equal(t, "#/components/schemas/First", string(doc.Components.Schemas.GetOrZero("Internal").GetRef()), "internal reference should remain unchanged")
			first, err := fs.ReadFile(vfs, "output/"+tt.first)
			require.NoError(t, err, "read first localized file")
			assert.Contains(t, string(first), "first", "first file content should be preserved")
			second, err := fs.ReadFile(vfs, "output/"+tt.second)
			require.NoError(t, err, "read second localized file")
			assert.Contains(t, string(second), "second", "conflicting file should not overwrite the first")
			entries, err := fs.ReadDir(vfs, "output")
			require.NoError(t, err, "list localized files")
			assert.Len(t, entries, 2, "duplicate and internal references should not create extra files")
		})
	}
}

func TestLocalize_NestedSourceReferences_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	vfs := &localizeTestFS{MapFS: fstest.MapFS{
		"schemas/parent.yaml": {Data: []byte(`Parent:
  type: object
  properties:
    nested:
      $ref: './child.yaml#/Child'
    internal:
      $ref: '#/Text'
  allOf:
    - $ref: './child.yaml#/Child'
Text:
  type: string
`)},
		"schemas/child.yaml": {Data: []byte("Child:\n  type: object\n  description: Nested child\n")},
	}}
	doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.1.0
info:
  title: Nested references
  version: 1.0.0
components:
  schemas:
    Parent:
      $ref: './schemas/parent.yaml#/Parent'
`))
	require.NoError(t, err, "unmarshal root document")
	require.Empty(t, validationErrs, "root document should be valid")
	err = openapi.Localize(ctx, doc, openapi.LocalizeOptions{DocumentLocation: "openapi.yaml", TargetDirectory: "output", VirtualFS: vfs})
	require.NoError(t, err, "localize nested source-relative references")
	assert.Equal(t, "parent.yaml#/Parent", string(doc.Components.Schemas.GetOrZero("Parent").GetRef()), "root schema should reference the copied file")
	parent, err := fs.ReadFile(vfs, "output/parent.yaml")
	require.NoError(t, err, "read localized parent")
	assert.Contains(t, string(parent), "./child.yaml#/Child", "source-relative child reference should remain valid beside the copied parent")
	assert.Contains(t, string(parent), "#/Text", "same-document fragment should be preserved")
	child, err := fs.ReadFile(vfs, "output/child.yaml")
	require.NoError(t, err, "read localized source-relative child")
	assert.Contains(t, string(child), "Nested child", "nested child content should be preserved")
}

func TestLocalize_Filesystem_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		files    fstest.MapFS
		openErr  error
		writeErr error
		section  string
		message  string
	}{
		{name: "missing file", files: fstest.MapFS{}, section: "responses", message: "failed to discover external references"},
		{name: "read denied", files: fstest.MapFS{}, openErr: fs.ErrPermission, section: "responses", message: "failed to discover external references"},
		{name: "write denied", files: fstest.MapFS{"source.yaml": {Data: []byte("type: string\n")}}, writeErr: fs.ErrPermission, section: "schemas", message: "failed to write localized file"},
		{name: "invalid source", files: fstest.MapFS{"source.yaml": {Data: []byte("description: [invalid\n")}}, section: "responses", message: "failed to discover external references"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			vfs := &localizeTestFS{MapFS: tt.files, openErr: tt.openErr, writeErr: tt.writeErr}
			doc, validationErrs, err := openapi.Unmarshal(ctx, strings.NewReader(`openapi: 3.1.0
info:
  title: Filesystem failure
  version: 1.0.0
components:
  `+tt.section+`:
    External:
      $ref: './source.yaml'
`))
			require.NoError(t, err, "unmarshal document before resolving")
			require.Empty(t, validationErrs, "unresolved document should be valid")
			err = openapi.Localize(ctx, doc, openapi.LocalizeOptions{DocumentLocation: "openapi.yaml", TargetDirectory: "output", VirtualFS: vfs})
			require.Error(t, err, "filesystem or source failure should prevent localization")
			assert.Contains(t, err.Error(), tt.message, "error should identify the failed localization phase")
			assert.Contains(t, err.Error(), "source.yaml", "error should identify the source file")
			var output bytes.Buffer
			require.NoError(t, openapi.Marshal(ctx, doc, &output), "marshal document after failure")
			assert.Contains(t, output.String(), "./source.yaml", "root reference should not be rewritten after failure")
		})
	}
}

func TestLocalize_EmptyDocument_Success(t *testing.T) {
	t.Parallel()

	require.NoError(t, openapi.Localize(t.Context(), nil, openapi.LocalizeOptions{}), "nil document should be a no-op")
	doc := &openapi.OpenAPI{OpenAPI: openapi.Version, Info: openapi.Info{Title: "No references", Version: "1.0.0"}}
	require.NoError(t, openapi.Localize(t.Context(), doc, openapi.LocalizeOptions{TargetDirectory: t.TempDir()}), "document without external references should succeed with the default filesystem")
	assert.Equal(t, "No references", doc.Info.Title, "document content should remain unchanged")
}

func TestLocalize_TargetDirectory_Error(t *testing.T) {
	t.Parallel()

	err := openapi.Localize(t.Context(), &openapi.OpenAPI{}, openapi.LocalizeOptions{})
	require.Error(t, err, "localization requires a target directory")
	assert.EqualError(t, err, "target directory is required", "missing target should produce a clear error")
}

func TestLocalize_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Create a mock HTTP server to serve remote schemas
	server := createMockRemoteServer(t)
	defer server.Close()

	// Load the input document
	inputFile, err := os.Open("testdata/localize/input/spec.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Create a temporary directory for output
	tempDir := t.TempDir()

	// Create custom HTTP client that redirects api.example.com to our test server
	httpClient := createRedirectHTTPClient(server.URL)

	// Configure localization options
	opts := openapi.LocalizeOptions{
		DocumentLocation: "testdata/localize/input/spec.yaml",
		TargetDirectory:  tempDir,
		VirtualFS:        &system.FileSystem{},
		HTTPClient:       httpClient,
		NamingStrategy:   openapi.LocalizeNamingPathBased,
	}

	// Localize all external references
	err = openapi.Localize(ctx, inputDoc, opts)
	require.NoError(t, err)

	// Marshal the localized main document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualMainYAML := buf.Bytes()

	// Load the expected main document output
	expectedMainBytes, err := os.ReadFile("testdata/localize/output_pathbased/openapi.yaml")
	require.NoError(t, err)

	// Compare the main document with expected output
	assert.Equal(t, string(expectedMainBytes), string(actualMainYAML), "Localized main document should match expected output")

	// Verify that the expected files were created in the target directory
	expectedFiles := []string{
		"components.yaml",       // from ./components.yaml (first conflict file gets simple name)
		"api-components.yaml",   // from ./api/components.yaml (subsequent conflict file gets path prefix)
		"address.yaml",          // from ./schemas/address.yaml (first conflict file gets simple name)
		"shared-address.yaml",   // from ./shared/address.yaml (subsequent conflict file gets path prefix)
		"category.yaml",         // from ./schemas/category.yaml (no conflict)
		"geo.yaml",              // from ./schemas/geo.yaml (no conflict, referenced by address.yaml)
		"user-profile.yaml",     // from remote URL
		"user-preferences.yaml", // from remote URL (referenced by user-profile.yaml)
		"metadata.yaml",         // from remote URL (referenced by user-profile.yaml)
	}

	for _, expectedFile := range expectedFiles {
		// Check that the file exists
		actualFilePath := filepath.Join(tempDir, expectedFile)
		_, err := os.Stat(actualFilePath)
		require.NoError(t, err, "Expected file %s should exist in target directory", expectedFile)

		// Read the actual file content
		actualContent, err := os.ReadFile(actualFilePath)
		require.NoError(t, err)

		// Read the expected file content
		expectedFilePath := filepath.Join("testdata/localize/output_pathbased", expectedFile)
		expectedContent, err := os.ReadFile(expectedFilePath)
		require.NoError(t, err)

		// Compare the content
		assert.Equal(t, string(expectedContent), string(actualContent), "Localized file %s should match expected content", expectedFile)
	}
}

func TestLocalize_CounterBased_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Create a mock HTTP server to serve remote schemas
	server := createMockRemoteServer(t)
	defer server.Close()

	// Load the input document
	inputFile, err := os.Open("testdata/localize/input/spec.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Create a temporary directory for output
	tempDir := t.TempDir()

	// Create custom HTTP client that redirects api.example.com to our test server
	httpClient := createRedirectHTTPClient(server.URL)

	// Configure localization options with counter-based naming
	opts := openapi.LocalizeOptions{
		DocumentLocation: "testdata/localize/input/spec.yaml",
		TargetDirectory:  tempDir,
		VirtualFS:        &system.FileSystem{},
		HTTPClient:       httpClient,
		NamingStrategy:   openapi.LocalizeNamingCounter,
	}

	// Localize all external references
	err = openapi.Localize(ctx, inputDoc, opts)
	require.NoError(t, err)

	// Marshal the localized main document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualMainYAML := buf.Bytes()

	// Load the expected main document output
	expectedMainBytes, err := os.ReadFile("testdata/localize/output_counter/openapi.yaml")
	require.NoError(t, err)

	// Compare the main document with expected output
	assert.Equal(t, string(expectedMainBytes), string(actualMainYAML), "Localized main document should match expected output")

	// Verify that the expected files were created in the target directory
	expectedFiles := []string{
		"components.yaml",       // from ./components.yaml (first conflict file gets simple name)
		"components_1.yaml",     // from ./api/components.yaml (subsequent conflict file gets counter suffix)
		"address.yaml",          // from ./schemas/address.yaml (first conflict file gets simple name)
		"address_1.yaml",        // from ./shared/address.yaml (subsequent conflict file gets counter suffix)
		"category.yaml",         // from ./schemas/category.yaml (no conflict)
		"geo.yaml",              // from ./schemas/geo.yaml (no conflict, referenced by address.yaml)
		"user-profile.yaml",     // from remote URL
		"user-preferences.yaml", // from remote URL (referenced by user-profile.yaml)
		"metadata.yaml",         // from remote URL (referenced by user-profile.yaml)
	}

	for _, expectedFile := range expectedFiles {
		// Check that the file exists
		actualFilePath := filepath.Join(tempDir, expectedFile)
		_, err := os.Stat(actualFilePath)
		require.NoError(t, err, "Expected file %s should exist in target directory", expectedFile)

		// Read the actual file content
		actualContent, err := os.ReadFile(actualFilePath)
		require.NoError(t, err)

		// Read the expected file content
		expectedFilePath := filepath.Join("testdata/localize/output_counter", expectedFile)
		expectedContent, err := os.ReadFile(expectedFilePath)
		require.NoError(t, err)

		// Compare the content
		assert.Equal(t, string(expectedContent), string(actualContent), "Localized file %s should match expected content", expectedFile)
	}
}

func TestLocalize_CustomNaming_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Create a mock HTTP server to serve remote schemas
	server := createMockRemoteServer(t)
	defer server.Close()

	// Load the input document
	inputFile, err := os.Open("testdata/localize/input/spec.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Create a temporary directory for output
	tempDir := t.TempDir()

	// Create custom HTTP client that redirects api.example.com to our test server
	httpClient := createRedirectHTTPClient(server.URL)

	// Track which refs the custom naming function is called with
	var calledRefs []string

	// Custom naming function that uses a content SHA prefix (similar to speakeasy bundler)
	customNaming := func(originalRef string, content []byte) string {
		calledRefs = append(calledRefs, originalRef)

		base := filepath.Base(originalRef)
		ext := filepath.Ext(base)
		if ext == "" {
			ext = ".yaml"
		}
		name := strings.TrimSuffix(base, ext)
		sha := sha256.Sum256(content)
		return fmt.Sprintf("%s-%x%s", name, sha[:4], ext)
	}

	opts := openapi.LocalizeOptions{
		DocumentLocation: "testdata/localize/input/spec.yaml",
		TargetDirectory:  tempDir,
		VirtualFS:        &system.FileSystem{},
		HTTPClient:       httpClient,
		NamingStrategy:   openapi.LocalizeNamingCustom,
		CustomNamingFunc: customNaming,
	}

	err = openapi.Localize(ctx, inputDoc, opts)
	require.NoError(t, err)

	// Verify the custom naming function was called for each external reference
	assert.NotEmpty(t, calledRefs, "Custom naming function should have been called")

	// Verify that files with custom names exist in the target directory
	entries, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.NotEmpty(t, entries, "Target directory should contain localized files")

	// All output filenames should contain a hex SHA suffix
	for _, entry := range entries {
		assert.Regexp(t, `-[0-9a-f]{8}\.yaml$`, entry.Name(),
			"File %s should match custom naming pattern", entry.Name())
	}

	// Verify the document references were rewritten to use the custom filenames
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	output := buf.String()

	// The output should not contain any of the original external references
	assert.NotContains(t, output, "./components.yaml")
	assert.NotContains(t, output, "./api/components.yaml")
	assert.NotContains(t, output, "./shared/address.yaml")
	assert.NotContains(t, output, "https://api.example.com/schemas/")

	// The main document should reference custom-named files (only direct refs, not transitive ones)
	// Transitive refs (category, geo, metadata) live inside the localized files, not the main doc
	assert.Contains(t, output, "components-")
	assert.Contains(t, output, "address-")
	assert.Contains(t, output, "UserProfile-")
	assert.Contains(t, output, "UserPreferences-")
}

// createMockRemoteServer creates a mock HTTP server that serves remote schema files
func createMockRemoteServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// Serve user-profile.yaml
	mux.HandleFunc("/schemas/user-profile.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		content, err := os.ReadFile("testdata/localize/remote/schemas/user-profile.yaml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(content)
	})

	// Serve user-preferences.yaml
	mux.HandleFunc("/schemas/user-preferences.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		content, err := os.ReadFile("testdata/localize/remote/schemas/user-preferences.yaml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(content)
	})

	// Serve metadata.yaml
	mux.HandleFunc("/schemas/metadata.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		content, err := os.ReadFile("testdata/localize/remote/schemas/metadata.yaml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(content)
	})

	return httptest.NewServer(mux)
}

// createRedirectHTTPClient creates an HTTP client that redirects api.example.com requests to the test server
func createRedirectHTTPClient(testServerURL string) *http.Client {
	return &http.Client{
		Transport: &redirectTransport{
			testServerURL: testServerURL,
			base:          http.DefaultTransport,
		},
	}
}

// redirectTransport redirects api.example.com requests to the test server
type redirectTransport struct {
	testServerURL string
	base          http.RoundTripper
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if this is an api.example.com request
	if req.URL.Host == "api.example.com" {
		// Replace the host with our test server
		newURL := *req.URL
		testURL := strings.TrimPrefix(rt.testServerURL, "http://")
		newURL.Host = testURL
		newURL.Scheme = "http"

		// Clone the request with the new URL
		newReq := req.Clone(req.Context())
		newReq.URL = &newURL

		return rt.base.RoundTrip(newReq)
	}

	// For all other requests, use the base transport
	return rt.base.RoundTrip(req)
}
