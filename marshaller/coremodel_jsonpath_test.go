package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/jsonpath/pkg/jsonpath"
	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestCoreModel_GetJSONPath_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yamlContent  string
		expectedPath string // The JSONPath we expect to get back
	}{
		{
			name: "root node",
			yamlContent: `
name: test
age: 25`,
			expectedPath: "$",
		},
		{
			name: "simple key in mapping",
			yamlContent: `
name: test-value
age: 25
active: true`,
			expectedPath: "$.name",
		},
		{
			name: "nested object access",
			yamlContent: `
user:
  name: john
  age: 30
settings:
  theme: dark`,
			expectedPath: "$.user.name",
		},
		{
			name: "array element access",
			yamlContent: `
users:
  - name: alice
    age: 25
  - name: bob
    age: 30
settings:
  theme: dark`,
			expectedPath: "$.users[0]",
		},
		{
			name: "nested array element property",
			yamlContent: `
users:
  - name: alice
    age: 25
  - name: bob
    age: 30`,
			expectedPath: "$.users[1].name",
		},
		{
			name: "complex nested path",
			yamlContent: `
api:
  endpoints:
    - path: /users
      methods:
        - GET
        - POST
    - path: /orders
      methods:
        - GET`,
			expectedPath: "$.api.endpoints[0].methods[1]",
		},
		{
			name: "property with special characters",
			yamlContent: `
paths:
  "/users/{id}":
    get:
      summary: Get user
  "/orders":
    post:
      summary: Create order`,
			expectedPath: "$.paths['/users/{id}'].get.summary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var rootNode yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlContent), &rootNode)
			require.NoError(t, err, "failed to unmarshal YAML")

			// Find the target node using the expected JSONPath
			targetNode := findNodeByJSONPath(t, &rootNode, tt.expectedPath)
			require.NotNil(t, targetNode, "target node should be found for path: %s", tt.expectedPath)

			// Create CoreModel with the target node
			coreModel := &marshaller.CoreModel{}
			coreModel.SetRootNode(targetNode)

			// Test GetJSONPath
			result := coreModel.GetJSONPath(&rootNode)
			assert.Equal(t, tt.expectedPath, result, "JSONPath should match expected value")

			// Verify the JSONPath actually works by using it to query the document
			verifyJSONPathWorks(t, &rootNode, result, targetNode)
		})
	}
}

func TestCoreModel_GetJSONPath_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		coreModelNode    *yaml.Node
		topLevelRootNode *yaml.Node
		expected         string
	}{
		{
			name:             "nil core model root node",
			coreModelNode:    nil,
			topLevelRootNode: &yaml.Node{},
			expected:         "",
		},
		{
			name:             "nil top level root node",
			coreModelNode:    &yaml.Node{},
			topLevelRootNode: nil,
			expected:         "",
		},
		{
			name:             "both nodes nil",
			coreModelNode:    nil,
			topLevelRootNode: nil,
			expected:         "",
		},
		{
			name:             "node not found in tree",
			coreModelNode:    &yaml.Node{Kind: yaml.ScalarNode, Value: "not found"},
			topLevelRootNode: &yaml.Node{Kind: yaml.ScalarNode, Value: "different"},
			expected:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			coreModel := &marshaller.CoreModel{}
			coreModel.SetRootNode(tt.coreModelNode)
			result := coreModel.GetJSONPath(tt.topLevelRootNode)
			assert.Equal(t, tt.expected, result, "should return empty string for error cases")
		})
	}
}

// findNodeByJSONPath finds a node in the YAML tree using a JSONPath expression
func findNodeByJSONPath(t *testing.T, rootNode *yaml.Node, jsonPath string) *yaml.Node {
	t.Helper()

	// Handle root case
	if jsonPath == "$" {
		return rootNode
	}

	// Use jsonpath library to find the target
	path, err := jsonpath.NewPath(jsonPath)
	require.NoError(t, err, "JSONPath should be valid: %s", jsonPath)

	// Query the YAML node directly
	result := testutils.QueryV4(path, rootNode)
	require.NotEmpty(t, result, "JSONPath query should return results: %s", jsonPath)

	// The result should be a slice of *yaml.Node, so return the first one
	if len(result) > 0 {
		return result[0]
	}
	return nil
}

// verifyJSONPathWorks verifies that the generated JSONPath actually works
func verifyJSONPathWorks(t *testing.T, rootNode *yaml.Node, jsonPath string, expectedNode *yaml.Node) {
	t.Helper()

	// Use the JSONPath to query the YAML node directly
	path, err := jsonpath.NewPath(jsonPath)
	require.NoError(t, err, "generated JSONPath should be valid: %s", jsonPath)

	result := testutils.QueryV4(path, rootNode)
	require.NotEmpty(t, result, "JSONPath query should return results: %s", jsonPath)

	// Special case for root node: the query might return the document node
	// but our expected node might be the root node itself
	if jsonPath == "$" {
		// For root queries, we just verify that we got a result
		assert.NotEmpty(t, result, "Root JSONPath query should return results")
		return
	}

	// Verify the result contains our expected node
	// The result is a slice of nodes, so we check if our target node is in there
	found := false
	for _, node := range result {
		if node == expectedNode {
			found = true
			break
		}
	}
	assert.True(t, found, "JSONPath result should contain the expected node")
}
