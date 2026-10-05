package openapi_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestJoin_Counter_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the main document
	mainFile, err := os.Open("testdata/join/main.yaml")
	require.NoError(t, err)
	defer mainFile.Close()

	mainDoc, validationErrs, err := openapi.Unmarshal(ctx, mainFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Main document should be valid")

	// Load the second document
	secondFile, err := os.Open("testdata/join/subdir/second.yaml")
	require.NoError(t, err)
	defer secondFile.Close()

	secondDoc, validationErrs, err := openapi.Unmarshal(ctx, secondFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Second document should be valid")

	// Load the third document
	thirdFile, err := os.Open("testdata/join/third.yaml")
	require.NoError(t, err)
	defer thirdFile.Close()

	thirdDoc, validationErrs, err := openapi.Unmarshal(ctx, thirdFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Third document should be valid")

	// Configure join options with counter strategy
	documents := []openapi.JoinDocumentInfo{
		{
			Document: secondDoc,
			FilePath: "subdir/second.yaml",
		},
		{
			Document: thirdDoc,
			FilePath: "third.yaml",
		},
	}

	opts := openapi.JoinOptions{
		ConflictStrategy: openapi.JoinConflictCounter,
	}

	// Join documents
	err = openapi.Join(ctx, mainDoc, documents, opts)
	require.NoError(t, err)

	// Marshal the joined document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, mainDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/join/joined_counter_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Joined document with counter strategy should match expected output")
}

func TestJoin_FilePath_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the main document
	mainFile, err := os.Open("testdata/join/main.yaml")
	require.NoError(t, err)
	defer mainFile.Close()

	mainDoc, validationErrs, err := openapi.Unmarshal(ctx, mainFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Main document should be valid")

	// Load the second document
	secondFile, err := os.Open("testdata/join/subdir/second.yaml")
	require.NoError(t, err)
	defer secondFile.Close()

	secondDoc, validationErrs, err := openapi.Unmarshal(ctx, secondFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Second document should be valid")

	// Load the third document
	thirdFile, err := os.Open("testdata/join/third.yaml")
	require.NoError(t, err)
	defer thirdFile.Close()

	thirdDoc, validationErrs, err := openapi.Unmarshal(ctx, thirdFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Third document should be valid")

	// Configure join options with filepath strategy
	documents := []openapi.JoinDocumentInfo{
		{
			Document: secondDoc,
			FilePath: "subdir/second.yaml",
		},
		{
			Document: thirdDoc,
			FilePath: "third.yaml",
		},
	}

	opts := openapi.JoinOptions{
		ConflictStrategy: openapi.JoinConflictFilePath,
	}

	// Join documents
	err = openapi.Join(ctx, mainDoc, documents, opts)
	require.NoError(t, err)

	// Marshal the joined document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, mainDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/join/joined_filepath_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Joined document with filepath strategy should match expected output")
}

func TestJoin_EmptyDocuments_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the main document
	mainFile, err := os.Open("testdata/join/main.yaml")
	require.NoError(t, err)
	defer mainFile.Close()

	mainDoc, validationErrs, err := openapi.Unmarshal(ctx, mainFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Main document should be valid")

	// Store original values for comparison
	originalTitle := mainDoc.Info.Title
	originalVersion := mainDoc.Info.Version

	// Join with empty documents slice
	documents := []openapi.JoinDocumentInfo{}

	opts := openapi.JoinOptions{
		ConflictStrategy: openapi.JoinConflictCounter,
	}

	err = openapi.Join(ctx, mainDoc, documents, opts)
	require.NoError(t, err)

	// Main document should remain unchanged
	assert.Equal(t, originalTitle, mainDoc.Info.Title)
	assert.Equal(t, originalVersion, mainDoc.Info.Version)
}

func TestJoin_NilMainDocument_Error(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	documents := []openapi.JoinDocumentInfo{}
	opts := openapi.JoinOptions{}

	err := openapi.Join(ctx, nil, documents, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "main document is nil")
}

func TestJoin_NilDocumentInSlice_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the main document
	mainFile, err := os.Open("testdata/join/main.yaml")
	require.NoError(t, err)
	defer mainFile.Close()

	mainDoc, validationErrs, err := openapi.Unmarshal(ctx, mainFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Main document should be valid")

	// Include nil document in slice
	documents := []openapi.JoinDocumentInfo{
		{
			Document: nil,
			FilePath: "nil.yaml",
		},
	}

	opts := openapi.JoinOptions{
		ConflictStrategy: openapi.JoinConflictCounter,
	}

	// Should not error, just skip nil documents
	err = openapi.Join(ctx, mainDoc, documents, opts)
	assert.NoError(t, err)
}

func TestJoin_NoFilePath_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the main document
	mainFile, err := os.Open("testdata/join/main.yaml")
	require.NoError(t, err)
	defer mainFile.Close()

	mainDoc, validationErrs, err := openapi.Unmarshal(ctx, mainFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Main document should be valid")

	// Load the second document
	secondFile, err := os.Open("testdata/join/subdir/second.yaml")
	require.NoError(t, err)
	defer secondFile.Close()

	secondDoc, validationErrs, err := openapi.Unmarshal(ctx, secondFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Second document should be valid")

	// Join documents without file path
	documents := []openapi.JoinDocumentInfo{
		{
			Document: secondDoc,
			FilePath: "", // Empty file path
		},
	}

	opts := openapi.JoinOptions{
		ConflictStrategy: openapi.JoinConflictCounter,
	}

	err = openapi.Join(ctx, mainDoc, documents, opts)
	require.NoError(t, err)

	// Verify original /users path exists and contains both operations
	assert.True(t, mainDoc.Paths.Has("/users"))
	usersPath, exists := mainDoc.Paths.Get("/users")
	require.True(t, exists, "Users path should exist")
	require.NotNil(t, usersPath)
	require.NotNil(t, usersPath.Object)

	// Should have both GET (from main) and POST (from second)
	assert.NotNil(t, usersPath.Object.Get, "Should have GET operation from main document")
	assert.NotNil(t, usersPath.Object.Post, "Should have POST operation from second document")

	// Should NOT have a duplicated path since methods are different
	assert.False(t, mainDoc.Paths.Has("/users#document_0"), "Should not create duplicate path for different methods")
}

func TestJoin_ServersSecurityConflicts_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the main document
	mainFile, err := os.Open("testdata/join/main.yaml")
	require.NoError(t, err)
	defer mainFile.Close()

	mainDoc, validationErrs, err := openapi.Unmarshal(ctx, mainFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Main document should be valid")

	// Load the conflict servers document
	conflictServersFile, err := os.Open("testdata/join/conflict_servers.yaml")
	require.NoError(t, err)
	defer conflictServersFile.Close()

	conflictServersDoc, validationErrs, err := openapi.Unmarshal(ctx, conflictServersFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Conflict servers document should be valid")

	// Load the conflict security document
	conflictSecurityFile, err := os.Open("testdata/join/conflict_security.yaml")
	require.NoError(t, err)
	defer conflictSecurityFile.Close()

	conflictSecurityDoc, validationErrs, err := openapi.Unmarshal(ctx, conflictSecurityFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Conflict security document should be valid")

	// Configure join options with counter strategy
	documents := []openapi.JoinDocumentInfo{
		{
			Document: conflictServersDoc,
			FilePath: "conflict_servers.yaml",
		},
		{
			Document: conflictSecurityDoc,
			FilePath: "conflict_security.yaml",
		},
	}

	opts := openapi.JoinOptions{
		ConflictStrategy: openapi.JoinConflictCounter,
	}

	err = openapi.Join(ctx, mainDoc, documents, opts)
	require.NoError(t, err)

	// Marshal the result to YAML for comparison
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, mainDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Read expected output
	expectedBytes, err := os.ReadFile("testdata/join/joined_conflicts_expected.yaml")
	require.NoError(t, err)

	assert.Equal(t, string(expectedBytes), string(actualYAML), "Joined document with server/security conflicts should match expected output")
}

func TestJoin_ComponentConflictsRewriteLocalReferences_Success(t *testing.T) {
	t.Parallel()

	components := []struct {
		section    string
		definition string
	}{
		{section: "schemas", definition: "{type: string, description: source}"},
		{section: "responses", definition: "{description: source}"},
		{section: "parameters", definition: "{name: id, in: query, description: source, schema: {type: string}}"},
		{section: "examples", definition: "{description: source, value: example}"},
		{section: "requestBodies", definition: "{description: source, content: {application/json: {schema: {type: string}}}}"},
		{section: "headers", definition: "{description: source, schema: {type: string}}"},
		{section: "links", definition: "{description: source, operationId: getItem}"},
		{section: "callbacks", definition: "{'{$request.query.callbackUrl}': {post: {responses: {'200': {description: source}}}}}"},
		{section: "pathItems", definition: "{description: source, get: {responses: {'200': {description: source}}}}"},
	}
	strategies := []struct {
		name     string
		strategy openapi.JoinConflictStrategy
		reserved string
		renamed  string
	}{
		{name: "counter", strategy: openapi.JoinConflictCounter, reserved: "Shared_1", renamed: "Shared_2"},
		{name: "filepath", strategy: openapi.JoinConflictFilePath, reserved: "subdir_second_yaml__Shared", renamed: "subdir_second_yaml__Shared_1"},
		{name: "unknown strategy falls back to counter", strategy: openapi.JoinConflictStrategy(99), reserved: "Shared_1", renamed: "Shared_2"},
	}

	for _, component := range components {
		for _, strategy := range strategies {
			t.Run(component.section+"/"+strategy.name, func(t *testing.T) {
				t.Parallel()
				ctx := t.Context()
				prefix := "openapi: 3.1.0\ninfo: {title: Join API, version: 1.0.0}\npaths: {}\ncomponents:\n  " + component.section + ":\n"
				mainDefinition := strings.ReplaceAll(component.definition, "source", "main")
				main := unmarshalOpenAPI(t, ctx, prefix+"    Shared: "+mainDefinition+"\n    "+strategy.reserved+": "+mainDefinition+"\n")
				source := unmarshalOpenAPI(t, ctx, prefix+"    Shared: "+component.definition+"\n    Alias: {$ref: '#/components/"+component.section+"/Shared'}\n    External: {$ref: 'external.yaml#/Shared'}\n")

				err := openapi.Join(ctx, main, []openapi.JoinDocumentInfo{{Document: source, FilePath: "subdir/second.yaml"}}, openapi.JoinOptions{ConflictStrategy: strategy.strategy})
				require.NoError(t, err, "join conflicting components")
				var output bytes.Buffer
				require.NoError(t, openapi.Marshal(ctx, main, &output), "marshal joined document")
				var joined struct {
					Components map[string]map[string]map[string]any `yaml:"components"`
				}
				require.NoError(t, yaml.Unmarshal(output.Bytes(), &joined), "read joined component values")
				section := joined.Components[component.section]
				assert.Len(t, section, 5, "retain both originals, renamed source, and aliases")
				assert.Contains(t, section, strategy.renamed, "source component should have a unique conflict name")
				assert.Equal(t, "#/components/"+component.section+"/"+strategy.renamed, section["Alias"]["$ref"], "local alias should target the renamed source")
				assert.Equal(t, "external.yaml#/Shared", section["External"]["$ref"], "external reference should remain untouched")
				assert.NotEqual(t, section["Shared"], section[strategy.renamed], "main and source component contents should remain distinct")
			})
		}
	}
}
