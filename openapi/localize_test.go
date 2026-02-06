package openapi_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
