package marshaller_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCoreModel_GetJSONPointer_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		yamlContent     string
		expectedPointer string // The JSON pointer we expect to get back
	}{
		{
			name: "root node",
			yamlContent: `
name: test
age: 25`,
			expectedPointer: "/",
		},
		{
			name: "simple key in mapping",
			yamlContent: `
name: test-value
age: 25
active: true`,
			expectedPointer: "/name",
		},
		{
			name: "nested object access",
			yamlContent: `
user:
  profile:
    name: john
    settings:
      theme: dark`,
			expectedPointer: "/user/profile/settings/theme",
		},
		{
			name: "array element access",
			yamlContent: `
items:
  - first
  - second
  - third`,
			expectedPointer: "/items/1",
		},
		{
			name: "complex nested structure",
			yamlContent: `
api:
  endpoints:
    - path: /users
      methods:
        - GET
        - POST
    - path: /posts
      methods:
        - GET`,
			expectedPointer: "/api/endpoints/0/methods/1",
		},
		{
			name: "key with special characters",
			yamlContent: `paths:
  "/users/{id}":
    get:
      summary: Get user`,
			expectedPointer: "/paths/~1users~1{id}/get/summary",
		},
		{
			name: "key with tilde character",
			yamlContent: `config:
  "~temp": temporary
  "normal": value`,
			expectedPointer: "/config/~0temp",
		},
		{
			name: "key with both tilde and slash",
			yamlContent: `special:
  "~/path": value`,
			expectedPointer: "/special/~0~1path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Parse the YAML content
			var rootNode yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlContent), &rootNode)
			require.NoError(t, err)

			// Get the target node using the expected JSON pointer (demonstrating reversible operation)
			targetNode, err := jsonpointer.GetTarget(&rootNode, jsonpointer.JSONPointer(tt.expectedPointer))
			require.NoError(t, err, "should be able to get target node at path: %s", tt.expectedPointer)

			// Convert to yaml.Node if needed
			yamlTargetNode, ok := targetNode.(*yaml.Node)
			require.True(t, ok, "target should be a yaml.Node")

			// Create CoreModel with the target node
			coreModel := &marshaller.CoreModel{
				RootNode: yamlTargetNode,
			}

			// Get the JSON pointer - this should return the same pointer we used to get the node
			pointer := coreModel.GetJSONPointer(&rootNode)
			assert.Equal(t, tt.expectedPointer, pointer, "JSON pointer should match the pointer used to get the node (reversible operation)")

			// Validate that the returned pointer is a valid JSON pointer
			err = jsonpointer.JSONPointer(pointer).Validate()
			require.NoError(t, err, "returned pointer should be a valid JSON pointer")

			// Verify reversibility: use the returned pointer to retrieve the same node
			retrievedNode, err := jsonpointer.GetTarget(&rootNode, jsonpointer.JSONPointer(pointer))
			require.NoError(t, err, "should be able to retrieve node using returned pointer (reversible operation)")
			assert.Equal(t, yamlTargetNode, retrievedNode, "retrieved node should match original target node (reversible operation)")
		})
	}
}

func TestCoreModel_GetJSONPointer_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		coreModel    *marshaller.CoreModel
		topLevelNode *yaml.Node
		expected     string
	}{
		{
			name: "nil CoreModel RootNode",
			coreModel: &marshaller.CoreModel{
				RootNode: nil,
			},
			topLevelNode: &yaml.Node{},
			expected:     "",
		},
		{
			name: "nil topLevelNode",
			coreModel: &marshaller.CoreModel{
				RootNode: &yaml.Node{},
			},
			topLevelNode: nil,
			expected:     "",
		},
		{
			name: "both nodes nil",
			coreModel: &marshaller.CoreModel{
				RootNode: nil,
			},
			topLevelNode: nil,
			expected:     "",
		},
		{
			name: "target node not found in top level",
			coreModel: &marshaller.CoreModel{
				RootNode: &yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: "not-found",
				},
			},
			topLevelNode: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "key"},
					{Kind: yaml.ScalarNode, Value: "value"},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pointer := tt.coreModel.GetJSONPointer(tt.topLevelNode)
			assert.Equal(t, tt.expected, pointer)
		})
	}
}

func TestCoreModel_GetJSONPointer_WithAliases(t *testing.T) {
	t.Parallel()

	yamlContent := `
defaults: &defaults
  timeout: 30
  retries: 3

production:
  <<: *defaults
  host: prod.example.com

development:
  <<: *defaults  
  host: dev.example.com
  timeout: 10`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &rootNode)
	require.NoError(t, err)

	// Test accessing aliased value using jsonpointer
	targetNode, err := jsonpointer.GetTarget(&rootNode, jsonpointer.JSONPointer("/production/timeout"))
	require.NoError(t, err)

	// Convert to yaml.Node if needed
	yamlTargetNode, ok := targetNode.(*yaml.Node)
	require.True(t, ok, "target should be a yaml.Node")

	coreModel := &marshaller.CoreModel{
		RootNode: yamlTargetNode,
	}

	pointer := coreModel.GetJSONPointer(&rootNode)
	// Note: The jsonpointer package resolves aliases, so it finds the original node in defaults
	assert.Equal(t, "/defaults/timeout", pointer)

	// Validate that the returned pointer is a valid JSON pointer
	err = jsonpointer.JSONPointer(pointer).Validate()
	require.NoError(t, err, "returned pointer should be a valid JSON pointer")
}

func TestCoreModel_GetJSONPointer_KeyAndValueNodes_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		yamlContent     string
		nodeType        string // "key" or "value"
		targetPath      string // JSON pointer to get the target node
		expectedPointer string // The JSON pointer we expect GetJSONPointer to return
	}{
		{
			name:            "simple key node",
			yamlContent:     `name: test-value`,
			nodeType:        "key",
			targetPath:      "/name",
			expectedPointer: "/name",
		},
		{
			name:            "simple value node",
			yamlContent:     `name: test-value`,
			nodeType:        "value",
			targetPath:      "/name",
			expectedPointer: "/name",
		},
		{
			name: "nested key node",
			yamlContent: `user:
  profile:
    name: john`,
			nodeType:        "key",
			targetPath:      "/user/profile/name",
			expectedPointer: "/user/profile/name",
		},
		{
			name: "nested value node",
			yamlContent: `user:
  profile:
    name: john`,
			nodeType:        "value",
			targetPath:      "/user/profile/name",
			expectedPointer: "/user/profile/name",
		},
		{
			name: "intermediate key node",
			yamlContent: `user:
  profile:
    name: john
    age: 25`,
			nodeType:        "key",
			targetPath:      "/user/profile",
			expectedPointer: "/user/profile",
		},
		{
			name: "intermediate value node (object)",
			yamlContent: `user:
  profile:
    name: john
    age: 25`,
			nodeType:        "value",
			targetPath:      "/user/profile",
			expectedPointer: "/user/profile",
		},
		{
			name: "key with special characters",
			yamlContent: `paths:
  "/users/{id}":
    get:
      summary: Get user`,
			nodeType:        "key",
			targetPath:      "/paths/~1users~1{id}",
			expectedPointer: "/paths/~1users~1{id}",
		},
		{
			name: "value with special characters in key",
			yamlContent: `paths:
  "/users/{id}":
    get:
      summary: Get user`,
			nodeType:        "value",
			targetPath:      "/paths/~1users~1{id}",
			expectedPointer: "/paths/~1users~1{id}",
		},
		{
			name: "array element key access",
			yamlContent: `items:
  - name: first
    value: 1
  - name: second
    value: 2`,
			nodeType:        "key",
			targetPath:      "/items/0/name",
			expectedPointer: "/items/0/name",
		},
		{
			name: "array element value access",
			yamlContent: `items:
  - name: first
    value: 1
  - name: second
    value: 2`,
			nodeType:        "value",
			targetPath:      "/items/0/name",
			expectedPointer: "/items/0/name",
		},
		{
			name: "root level key",
			yamlContent: `name: test
age: 25
active: true`,
			nodeType:        "key",
			targetPath:      "/age",
			expectedPointer: "/age",
		},
		{
			name: "root level value",
			yamlContent: `name: test
age: 25
active: true`,
			nodeType:        "value",
			targetPath:      "/age",
			expectedPointer: "/age",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_"+tt.nodeType, func(t *testing.T) {
			t.Parallel()

			// Parse the YAML content
			var rootNode yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlContent), &rootNode)
			require.NoError(t, err)

			var targetNode *yaml.Node

			if tt.nodeType == "key" {
				// For key nodes, we need to manually traverse to find the key node
				targetNode = findKeyNodeAtPath(&rootNode, tt.targetPath)
				require.NotNil(t, targetNode, "should find key node at path: %s", tt.targetPath)
			} else {
				// For value nodes, use the existing jsonpointer functionality
				valueNode, err := jsonpointer.GetTarget(&rootNode, jsonpointer.JSONPointer(tt.targetPath))
				require.NoError(t, err, "should be able to get value node at path: %s", tt.targetPath)

				yamlValueNode, ok := valueNode.(*yaml.Node)
				require.True(t, ok, "target should be a yaml.Node")
				targetNode = yamlValueNode
			}

			// Create CoreModel with the target node
			coreModel := &marshaller.CoreModel{
				RootNode: targetNode,
			}

			// Get the JSON pointer
			pointer := coreModel.GetJSONPointer(&rootNode)
			assert.Equal(t, tt.expectedPointer, pointer, "JSON pointer should match expected for %s node", tt.nodeType)

			// Validate that the returned pointer is a valid JSON pointer
			err = jsonpointer.JSONPointer(pointer).Validate()
			require.NoError(t, err, "returned pointer should be a valid JSON pointer")

			// For both key and value nodes, the pointer should resolve to the value
			// This demonstrates the expected behavior: key nodes produce pointers that resolve to their values
			if pointer != "" && pointer != "/" {
				retrievedNode, err := jsonpointer.GetTarget(&rootNode, jsonpointer.JSONPointer(pointer))
				require.NoError(t, err, "should be able to retrieve node using returned pointer")

				if tt.nodeType == "key" {
					// For key nodes, we need to get the actual value node to compare against
					expectedValueNode, err := jsonpointer.GetTarget(&rootNode, jsonpointer.JSONPointer(tt.targetPath))
					require.NoError(t, err, "should be able to get expected value node")

					// The pointer should resolve to the value node associated with the key
					assert.Equal(t, expectedValueNode, retrievedNode, "key node pointer should resolve to its associated value node")
				} else {
					// For value nodes, the pointer should resolve to the same node
					assert.Equal(t, targetNode, retrievedNode, "value node pointer should resolve to same node")
				}
			}
		})
	}
}

// findKeyNodeAtPath manually traverses the YAML structure to find the key node at the given JSON pointer path
func findKeyNodeAtPath(rootNode *yaml.Node, jsonPointerPath string) *yaml.Node {
	if jsonPointerPath == "/" {
		return rootNode
	}

	// Parse the JSON pointer path
	parts := strings.Split(strings.TrimPrefix(jsonPointerPath, "/"), "/")

	// Start from the document root
	currentNode := rootNode
	if currentNode.Kind == yaml.DocumentNode && len(currentNode.Content) > 0 {
		currentNode = currentNode.Content[0]
	}

	// Traverse to find the key node
	for i, part := range parts {
		// Unescape JSON pointer token
		unescapedPart := strings.ReplaceAll(part, "~1", "/")
		unescapedPart = strings.ReplaceAll(unescapedPart, "~0", "~")

		switch currentNode.Kind {
		case yaml.MappingNode:
			// Look for the key in the mapping
			for j := 0; j < len(currentNode.Content); j += 2 {
				if j+1 >= len(currentNode.Content) {
					break
				}

				keyNode := currentNode.Content[j]
				valueNode := currentNode.Content[j+1]

				if keyNode.Kind == yaml.ScalarNode && keyNode.Value == unescapedPart {
					// If this is the last part, return the key node
					if i == len(parts)-1 {
						return keyNode
					}
					// Otherwise, continue with the value node
					currentNode = valueNode
					break
				}
			}
		case yaml.SequenceNode:
			// Handle array index
			index, err := strconv.Atoi(unescapedPart)
			if err != nil || index < 0 || index >= len(currentNode.Content) {
				return nil
			}
			currentNode = currentNode.Content[index]
		default:
			return nil
		}
	}

	return nil
}
