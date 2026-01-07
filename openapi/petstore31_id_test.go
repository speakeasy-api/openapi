package openapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isAnchorRef checks if the reference contains an anchor fragment (non-JSON-pointer fragment).
// Anchor fragments don't start with / after the #, e.g., "#pet_details_id"
func isAnchorRef(ref string) bool {
	hashIdx := strings.LastIndex(ref, "#")
	if hashIdx == -1 {
		return false
	}
	fragment := ref[hashIdx+1:]
	// JSON pointer fragments start with /, anchor fragments don't
	return len(fragment) > 0 && !strings.HasPrefix(fragment, "/")
}

func TestPetstore31_WalkAndResolveAllSchemas_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Create mock filesystem to serve external schemas by their $id paths
	mockFS := NewMockVirtualFS()

	// Read the external schema files and add them at their $id paths
	categoryContent, err := os.ReadFile(filepath.Join("testdata", "petstore31.category.json"))
	require.NoError(t, err)
	mockFS.AddFile("/api/v31/components/schemas/category", categoryContent)

	petdetailsContent, err := os.ReadFile(filepath.Join("testdata", "petstore31.petdetails.json"))
	require.NoError(t, err)
	mockFS.AddFile("/api/v31/components/schemas/petdetails", petdetailsContent)

	tagContent, err := os.ReadFile(filepath.Join("testdata", "petstore31.tag.json"))
	require.NoError(t, err)
	mockFS.AddFile("/api/v31/components/schemas/tag", tagContent)

	// Load the petstore31 OpenAPI document
	testDataPath := filepath.Join("testdata", "petstore31.openapi.json")
	file, err := os.Open(testDataPath)
	require.NoError(t, err)
	defer file.Close()

	doc, validationErrs, err := Unmarshal(ctx, file)
	require.NoError(t, err)
	// Note: We ignore validation errors about $vocabulary being a string (test data issue, not related to $id)
	for _, vErr := range validationErrs {
		if !strings.Contains(vErr.Error(), "$vocabulary") {
			t.Errorf("Unexpected validation error: %v", vErr)
		}
	}

	// Setup resolve options with mock filesystem
	absPath, err := filepath.Abs(testDataPath)
	require.NoError(t, err)

	resolveOpts := ResolveOptions{
		TargetLocation: absPath,
		RootDocument:   doc,
		VirtualFS:      mockFS,
	}

	// Walk through the entire document and resolve all schema references
	schemasVisited := 0
	refsResolved := 0
	anchorRefsSkipped := 0
	schemasWithID := make(map[string]string) // location -> $id

	for item := range Walk(ctx, doc) {
		err := item.Match(Matcher{
			Schema: func(schema *oas3.JSONSchema[oas3.Referenceable]) error {
				schemaLoc := string(item.Location.ToJSONPointer())
				schemasVisited++

				// If it's a reference, try to resolve it
				if schema.IsReference() {
					ref := string(schema.GetReference())
					if ref != "" {
						// Skip anchor refs ($anchor support is not yet implemented)
						if isAnchorRef(ref) {
							t.Logf("Skipping anchor reference at %s: %s", schemaLoc, ref)
							anchorRefsSkipped++
							return nil
						}

						resolveErrs, resolveErr := schema.Resolve(ctx, resolveOpts)
						require.NoError(t, resolveErr, "Failed to resolve reference at %s: %s", schemaLoc, ref)
						assert.Empty(t, resolveErrs, "Validation errors resolving reference at %s: %s", schemaLoc, ref)
						refsResolved++
					}
				}

				// Track schemas with $id
				if schema.IsSchema() && schema.GetSchema() != nil {
					schemaID := schema.GetSchema().GetID()
					if schemaID != "" {
						schemasWithID[schemaLoc] = schemaID
					}
				}

				return nil
			},
		})
		require.NoError(t, err)
	}

	// Log summary
	t.Logf("Schemas visited: %d", schemasVisited)
	t.Logf("References resolved: %d", refsResolved)
	t.Logf("Anchor refs skipped: %d", anchorRefsSkipped)
	t.Logf("Schemas with $id: %d", len(schemasWithID))
	for loc, id := range schemasWithID {
		t.Logf("  %s: $id=%s", loc, id)
	}

	// Verify we found and resolved content
	assert.Positive(t, schemasVisited, "Should have visited at least some schemas")
	assert.NotEmpty(t, schemasWithID, "Should have found schemas with $id")
	assert.Positive(t, refsResolved, "Should have resolved at least some references")

	// Verify expected schemas with $id were found
	expectedIDs := map[string]string{
		"/components/schemas/Category":   "/api/v31/components/schemas/category",
		"/components/schemas/PetDetails": "/api/v31/components/schemas/petdetails",
		"/components/schemas/Tag":        "/api/v31/components/schemas/tag",
	}

	for loc, expectedID := range expectedIDs {
		actualID, found := schemasWithID[loc]
		assert.True(t, found, "Should have found schema with $id at %s", loc)
		assert.Equal(t, expectedID, actualID, "Schema at %s should have correct $id", loc)
	}

	// Verify that MockVirtualFS was accessed for external $id-based references
	accessLog := mockFS.GetAccessLog()
	t.Logf("Mock FS access log: %v", accessLog)
	assert.Contains(t, accessLog, "/api/v31/components/schemas/petdetails", "Should have accessed petdetails via $id path")
	assert.Contains(t, accessLog, "/api/v31/components/schemas/category", "Should have accessed category via $id path")
	assert.Contains(t, accessLog, "/api/v31/components/schemas/tag", "Should have accessed tag via $id path")
}
