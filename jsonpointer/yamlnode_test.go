package jsonpointer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGetTarget_YamlNode_Success(t *testing.T) {
	t.Parallel()

	type args struct {
		yamlContent string
		pointer     JSONPointer
	}
	tests := []struct {
		name     string
		args     args
		validate func(t *testing.T, result any)
	}{
		{
			name: "root yaml node",
			args: args{
				yamlContent: `value: test`,
				pointer:     JSONPointer("/"),
			},
			validate: func(t *testing.T, result any) {
				t.Helper()
				node, ok := result.(*yaml.Node)
				require.True(t, ok, "result should be *yaml.Node")
				// Root node should return the document's content (MappingNode)
				assert.Equal(t, yaml.MappingNode, node.Kind)
			},
		},
		{
			name: "simple key access in mapping",
			args: args{
				yamlContent: `
name: test-value
age: 25
active: true`,
				pointer: JSONPointer("/name"),
			},
			validate: func(t *testing.T, result any) {
				t.Helper()
				node, ok := result.(*yaml.Node)
				require.True(t, ok, "result should be *yaml.Node")
				assert.Equal(t, yaml.ScalarNode, node.Kind)
				assert.Equal(t, "test-value", node.Value)
			},
		},
		{
			name: "nested object access",
			args: args{
				yamlContent: `
user:
  profile:
    name: john
    settings:
      theme: dark`,
				pointer: JSONPointer("/user/profile/settings/theme"),
			},
			validate: func(t *testing.T, result any) {
				t.Helper()
				node, ok := result.(*yaml.Node)
				require.True(t, ok, "result should be *yaml.Node")
				assert.Equal(t, yaml.ScalarNode, node.Kind)
				assert.Equal(t, "dark", node.Value)
			},
		},
		{
			name: "array access by index",
			args: args{
				yamlContent: `
items:
  - first
  - second  
  - third`,
				pointer: JSONPointer("/items/1"),
			},
			validate: func(t *testing.T, result any) {
				t.Helper()
				node, ok := result.(*yaml.Node)
				require.True(t, ok, "result should be *yaml.Node")
				assert.Equal(t, yaml.ScalarNode, node.Kind)
				assert.Equal(t, "second", node.Value)
			},
		},
		{
			name: "complex nested structure",
			args: args{
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
				pointer: JSONPointer("/api/endpoints/0/methods/1"),
			},
			validate: func(t *testing.T, result any) {
				t.Helper()
				node, ok := result.(*yaml.Node)
				require.True(t, ok, "result should be *yaml.Node")
				assert.Equal(t, yaml.ScalarNode, node.Kind)
				assert.Equal(t, "POST", node.Value)
			},
		},
		{
			name: "escaped key characters",
			args: args{
				yamlContent: `
"paths":
  "/users/{id}": 
    get:
      summary: Get user`,
				pointer: JSONPointer("/paths/~1users~1{id}/get/summary"),
			},
			validate: func(t *testing.T, result any) {
				t.Helper()
				node, ok := result.(*yaml.Node)
				require.True(t, ok, "result should be *yaml.Node")
				assert.Equal(t, yaml.ScalarNode, node.Kind)
				assert.Equal(t, "Get user", node.Value)
			},
		},
		{
			name: "numeric string as key in yaml mapping",
			args: args{
				yamlContent: `responses:
  "200": "OK"
  "400": "Bad Request"
  "500": "Internal Server Error"`,
				pointer: JSONPointer("/responses/400"),
			},
			validate: func(t *testing.T, result any) {
				t.Helper()
				node, ok := result.(*yaml.Node)
				require.True(t, ok, "result should be *yaml.Node")
				assert.Equal(t, yaml.ScalarNode, node.Kind)
				assert.Equal(t, "Bad Request", node.Value)
			},
		},
		{
			name: "numeric string as key in nested yaml mapping",
			args: args{
				yamlContent: `components:
  responses:
    "400":
      description: "Bad Request"
      content:
        application/json:
          schema:
            type: object`,
				pointer: JSONPointer("/components/responses/400/description"),
			},
			validate: func(t *testing.T, result any) {
				t.Helper()
				node, ok := result.(*yaml.Node)
				require.True(t, ok, "result should be *yaml.Node")
				assert.Equal(t, yaml.ScalarNode, node.Kind)
				assert.Equal(t, "Bad Request", node.Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var yamlNode yaml.Node
			err := yaml.Unmarshal([]byte(tt.args.yamlContent), &yamlNode)
			require.NoError(t, err)

			// Test with *yaml.Node
			result, err := GetTarget(&yamlNode, tt.args.pointer)
			require.NoError(t, err)
			tt.validate(t, result)

			// Test with yaml.Node (non-pointer)
			result, err = GetTarget(yamlNode, tt.args.pointer)
			require.NoError(t, err)
			tt.validate(t, result)
		})
	}
}

func TestGetTarget_YamlNode_Error(t *testing.T) {
	t.Parallel()

	type args struct {
		yamlContent string
		pointer     JSONPointer
	}
	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{
			name: "key not found in mapping",
			args: args{
				yamlContent: `name: test`,
				pointer:     JSONPointer("/nonexistent"),
			},
			wantErr: "not found -- key nonexistent not found in yaml mapping at /nonexistent",
		},
		{
			name: "index out of range in sequence",
			args: args{
				yamlContent: `items: [a, b, c]`,
				pointer:     JSONPointer("/items/5"),
			},
			wantErr: "not found -- index 5 out of range for yaml sequence of length 3 at /items/5",
		},
		{
			name: "numeric key not found in mapping",
			args: args{
				yamlContent: `name: test`,
				pointer:     JSONPointer("/0"),
			},
			wantErr: "not found -- key 0 not found in yaml mapping at /0",
		},
		{
			name: "wrong type - using key on sequence",
			args: args{
				yamlContent: `[a, b, c]`,
				pointer:     JSONPointer("/key"),
			},
			wantErr: "invalid path -- expected index, got key at /key",
		},
		{
			name: "navigate through scalar",
			args: args{
				yamlContent: `value: test`,
				pointer:     JSONPointer("/value/invalid"),
			},
			wantErr: "invalid path -- cannot navigate through scalar yaml node at /value/invalid",
		},
		{
			name: "negative index",
			args: args{
				yamlContent: `items: [a, b, c]`,
				pointer:     JSONPointer("/items/-1"),
			},
			wantErr: "invalid path -- expected index, got key at /items/-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var yamlNode yaml.Node
			err := yaml.Unmarshal([]byte(tt.args.yamlContent), &yamlNode)
			require.NoError(t, err)

			result, err := GetTarget(&yamlNode, tt.args.pointer)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Nil(t, result)
		})
	}
}

func TestGetTarget_YamlNode_WithAliases(t *testing.T) {
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

	var yamlNode yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &yamlNode)
	require.NoError(t, err)

	// Test accessing aliased value
	result, err := GetTarget(&yamlNode, JSONPointer("/production/timeout"))
	require.NoError(t, err)

	node, ok := result.(*yaml.Node)
	require.True(t, ok)
	assert.Equal(t, yaml.ScalarNode, node.Kind)
	assert.Equal(t, "30", node.Value) // Should resolve the alias

	// Test accessing overridden value
	result, err = GetTarget(&yamlNode, JSONPointer("/development/timeout"))
	require.NoError(t, err)

	node, ok = result.(*yaml.Node)
	require.True(t, ok)
	assert.Equal(t, yaml.ScalarNode, node.Kind)
	assert.Equal(t, "10", node.Value) // Should get the overridden value
}
