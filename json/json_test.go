package json_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestYAMLToJSON_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yamlInput    string
		expectedJSON string
		indentation  int
	}{
		{
			name:      "simple scalar string",
			yamlInput: `hello world`,
			expectedJSON: `"hello world"
`,
			indentation: 2,
		},
		{
			name:      "simple scalar number",
			yamlInput: `42`,
			expectedJSON: `42
`,
			indentation: 2,
		},
		{
			name:      "simple scalar boolean",
			yamlInput: `true`,
			expectedJSON: `true
`,
			indentation: 2,
		},
		{
			name: "simple object",
			yamlInput: `name: John
age: 30`,
			expectedJSON: `{
  "name": "John",
  "age": 30
}
`,
			indentation: 2,
		},
		{
			name: "nested object",
			yamlInput: `person:
  name: John
  age: 30
  address:
    city: New York
    zip: 10001`,
			expectedJSON: `{
  "person": {
    "name": "John",
    "age": 30,
    "address": {
      "city": "New York",
      "zip": 10001
    }
  }
}
`,
			indentation: 2,
		},
		{
			name: "simple array",
			yamlInput: `- apple
- banana
- cherry`,
			expectedJSON: `[
  "apple",
  "banana",
  "cherry"
]
`,
			indentation: 2,
		},
		{
			name: "array of objects",
			yamlInput: `- name: John
  age: 30
- name: Jane
  age: 25`,
			expectedJSON: `[
  {
    "name": "John",
    "age": 30
  },
  {
    "name": "Jane",
    "age": 25
  }
]
`,
			indentation: 2,
		},
		{
			name: "mixed types in object",
			yamlInput: `string: hello
number: 42
boolean: true
null_value: null
array:
  - 1
  - 2
  - 3
object:
  nested: value`,
			expectedJSON: `{
  "string": "hello",
  "number": 42,
  "boolean": true,
  "null_value": null,
  "array": [
    1,
    2,
    3
  ],
  "object": {
    "nested": "value"
  }
}
`,
			indentation: 2,
		},
		{
			name: "preserves key order",
			yamlInput: `zebra: last
apple: first
middle: second`,
			expectedJSON: `{
  "zebra": "last",
  "apple": "first",
  "middle": "second"
}
`,
			indentation: 2,
		},
		{
			name: "custom indentation - 4 spaces",
			yamlInput: `name: John
age: 30`,
			expectedJSON: `{
    "name": "John",
    "age": 30
}
`,
			indentation: 4,
		},
		{
			name: "custom indentation - 0 spaces (compact)",
			yamlInput: `name: John
age: 30`,
			expectedJSON: `{"name":"John","age":30}
`,
			indentation: 0,
		},
		{
			name:      "empty object",
			yamlInput: `{}`,
			expectedJSON: `{}
`,
			indentation: 2,
		},
		{
			name:      "empty array",
			yamlInput: `[]`,
			expectedJSON: `[]
`,
			indentation: 2,
		},
		{
			name: "numeric keys converted to strings",
			yamlInput: `1: one
2: two
3: three`,
			expectedJSON: `{
  "1": "one",
  "2": "two",
  "3": "three"
}
`,
			indentation: 2,
		},
		{
			name: "yaml alias",
			yamlInput: `defaults: &defaults
  timeout: 30
  retries: 3
production:
  <<: *defaults
  host: prod.example.com`,
			expectedJSON: `{
  "defaults": {
    "timeout": 30,
    "retries": 3
  },
  "production": {
    "timeout": 30,
    "retries": 3,
    "host": "prod.example.com"
  }
}
`,
			indentation: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlInput), &node)
			require.NoError(t, err, "failed to parse YAML input")

			var buffer bytes.Buffer
			err = json.YAMLToJSON(&node, tt.indentation, &buffer)
			require.NoError(t, err, "YAMLToJSON should not return error")

			actualJSON := buffer.String()
			assert.Equal(t, tt.expectedJSON, actualJSON, "JSON output should match expected")
		})
	}
}

func TestYAMLToJSON_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		node      *yaml.Node
		wantError bool
	}{
		{
			name:      "nil node",
			node:      nil,
			wantError: false, // nil node is handled gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			err := json.YAMLToJSON(tt.node, 2, &buffer)

			if tt.wantError {
				assert.Error(t, err, "expected error for invalid input")
			} else {
				assert.NoError(t, err, "expected no error")
			}
		})
	}
}

func TestYAMLToJSONWithConfig_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yamlInput    string
		indent       string
		indentCount  int
		expectedJSON string
	}{
		{
			name: "2 spaces indentation",
			yamlInput: `name: John
age: 30`,
			indent:      " ",
			indentCount: 2,
			expectedJSON: `{
  "name": "John",
  "age": 30
}
`,
		},
		{
			name: "4 spaces indentation",
			yamlInput: `name: John
age: 30`,
			indent:      " ",
			indentCount: 4,
			expectedJSON: `{
    "name": "John",
    "age": 30
}
`,
		},
		{
			name: "single tab indentation",
			yamlInput: `name: John
age: 30`,
			indent:       "\t",
			indentCount:  1,
			expectedJSON: "{\n\t\"name\": \"John\",\n\t\"age\": 30\n}\n",
		},
		{
			name: "double tab indentation",
			yamlInput: `name: John
age: 30`,
			indent:       "\t",
			indentCount:  2,
			expectedJSON: "{\n\t\t\"name\": \"John\",\n\t\t\"age\": 30\n}\n",
		},
		{
			name: "tabs with nested object",
			yamlInput: `person:
  name: John
  age: 30
  address:
    city: New York`,
			indent:       "\t",
			indentCount:  1,
			expectedJSON: "{\n\t\"person\": {\n\t\t\"name\": \"John\",\n\t\t\"age\": 30,\n\t\t\"address\": {\n\t\t\t\"city\": \"New York\"\n\t\t}\n\t}\n}\n",
		},
		{
			name: "tabs with array",
			yamlInput: `items:
  - apple
  - banana
  - cherry`,
			indent:       "\t",
			indentCount:  1,
			expectedJSON: "{\n\t\"items\": [\n\t\t\"apple\",\n\t\t\"banana\",\n\t\t\"cherry\"\n\t]\n}\n",
		},
		{
			name: "zero indentation (compact)",
			yamlInput: `name: John
age: 30`,
			indent:      " ",
			indentCount: 0,
			expectedJSON: `{"name":"John","age":30}
`,
		},
		{
			name: "3 spaces indentation",
			yamlInput: `name: John
age: 30`,
			indent:      " ",
			indentCount: 3,
			expectedJSON: `{
   "name": "John",
   "age": 30
}
`,
		},
		{
			name: "tabs with array of objects",
			yamlInput: `- name: John
  age: 30
- name: Jane
  age: 25`,
			indent:       "\t",
			indentCount:  1,
			expectedJSON: "[\n\t{\n\t\t\"name\": \"John\",\n\t\t\"age\": 30\n\t},\n\t{\n\t\t\"name\": \"Jane\",\n\t\t\"age\": 25\n\t}\n]\n",
		},
		{
			name:         "scalar string with tabs",
			yamlInput:    `hello world`,
			indent:       "\t",
			indentCount:  1,
			expectedJSON: "\"hello world\"\n",
		},
		{
			name: "tabs preserving key order",
			yamlInput: `zebra: last
apple: first
middle: second`,
			indent:       "\t",
			indentCount:  1,
			expectedJSON: "{\n\t\"zebra\": \"last\",\n\t\"apple\": \"first\",\n\t\"middle\": \"second\"\n}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlInput), &node)
			require.NoError(t, err, "failed to parse YAML input")

			var buffer bytes.Buffer
			err = json.YAMLToJSONWithConfig(&node, tt.indent, tt.indentCount, true, &buffer)
			require.NoError(t, err, "YAMLToJSONWithIndentation should not return error")

			actualJSON := buffer.String()
			assert.Equal(t, tt.expectedJSON, actualJSON, "JSON output should match expected")
		})
	}
}

func TestYAMLToJSON_ArrayFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yamlInput    string
		expectedJSON string
		indentation  int
		description  string
	}{
		{
			name:      "flow style array - single line compact",
			yamlInput: `items: [apple, banana, cherry]`,
			expectedJSON: `{
  "items": ["apple", "banana", "cherry"]
}
`,
			indentation: 2,
			description: "Flow style arrays remain compact in JSON (preserves source formatting)",
		},
		{
			name: "block style array - already multi-line",
			yamlInput: `items:
    - apple
    - banana
    - cherry`,
			expectedJSON: `{
  "items": [
    "apple",
    "banana",
    "cherry"
  ]
}
`,
			indentation: 2,
			description: "Block style arrays remain multi-line in JSON",
		},
		{
			name:      "nested flow arrays",
			yamlInput: `matrix: [[1, 2], [3, 4], [5, 6]]`,
			expectedJSON: `{
  "matrix": [[1, 2], [3, 4], [5, 6]]
}
`,
			indentation: 2,
			description: "Nested flow arrays remain compact (preserves source formatting)",
		},
		{
			name: "mixed flow and block arrays",
			yamlInput: `config:
  inline: [1, 2, 3]
  block:
    - a
    - b
    - c`,
			expectedJSON: `{
  "config": {
    "inline": [1, 2, 3],
    "block": [
      "a",
      "b",
      "c"
    ]
  }
}
`,
			indentation: 2,
			description: "Mixed flow and block style arrays - flow stays compact, block expands",
		},
		{
			name:      "empty array flow style",
			yamlInput: `empty: []`,
			expectedJSON: `{
  "empty": []
}
`,
			indentation: 2,
			description: "Empty flow style arrays - root expands, value stays compact",
		},
		{
			name:      "single element flow array",
			yamlInput: `single: [one]`,
			expectedJSON: `{
  "single": ["one"]
}
`,
			indentation: 2,
			description: "Single element flow arrays remain compact",
		},
		{
			name:      "array of objects in flow style",
			yamlInput: `users: [{name: John, age: 30}, {name: Jane, age: 25}]`,
			expectedJSON: `{
  "users": [{"name": "John", "age": 30}, {"name": "Jane", "age": 25}]
}
`,
			indentation: 2,
			description: "Flow style array of objects remains compact",
		},
		{
			name:      "compact indentation with arrays",
			yamlInput: `data: [1, 2, 3]`,
			expectedJSON: `{"data":[1,2,3]}
`,
			indentation: 0,
			description: "Compact mode (indent=0) produces single-line arrays",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlInput), &node)
			require.NoError(t, err, "failed to parse YAML input")

			var buffer bytes.Buffer
			err = json.YAMLToJSON(&node, tt.indentation, &buffer)
			require.NoError(t, err, "YAMLToJSON should not return error")

			actualJSON := buffer.String()
			assert.Equal(t, tt.expectedJSON, actualJSON, tt.description)
		})
	}
}

func TestYAMLToJSON_ObjectFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yamlInput    string
		expectedJSON string
		indentation  int
		description  string
	}{
		{
			name:      "flow style object - single line compact",
			yamlInput: `person: {name: John, age: 30, city: NYC}`,
			expectedJSON: `{
  "person": {"name": "John", "age": 30, "city": "NYC"}
}
`,
			indentation: 2,
			description: "Flow style objects remain compact (preserves source formatting)",
		},
		{
			name: "block style object - already multi-line",
			yamlInput: `person:
    name: John
    age: 30
    city: NYC`,
			expectedJSON: `{
  "person": {
    "name": "John",
    "age": 30,
    "city": "NYC"
  }
}
`,
			indentation: 2,
			description: "Block style objects remain multi-line in JSON",
		},
		{
			name:      "nested flow objects",
			yamlInput: `data: {user: {name: John, email: john@example.com}, meta: {version: 1}}`,
			expectedJSON: `{
  "data": {"user": {"name": "John", "email": "john@example.com"}, "meta": {"version": 1}}
}
`,
			indentation: 2,
			description: "Nested flow objects remain compact (preserves source formatting)",
		},
		{
			name: "mixed flow and block objects",
			yamlInput: `config:
  inline: {a: 1, b: 2}
  block:
    c: 3
    d: 4`,
			expectedJSON: `{
  "config": {
    "inline": {"a": 1, "b": 2},
    "block": {
      "c": 3,
      "d": 4
    }
  }
}
`,
			indentation: 2,
			description: "Mixed flow and block style objects - flow stays compact, block expands",
		},
		{
			name:      "empty object flow style",
			yamlInput: `empty: {}`,
			expectedJSON: `{
  "empty": {}
}
`,
			indentation: 2,
			description: "Empty flow style objects - root expands, value stays compact",
		},
		{
			name:      "single property flow object",
			yamlInput: `config: {key: value}`,
			expectedJSON: `{
  "config": {"key": "value"}
}
`,
			indentation: 2,
			description: "Single property flow objects remain compact",
		},
		{
			name:      "compact indentation with objects",
			yamlInput: `data: {a: 1, b: 2}`,
			expectedJSON: `{"data":{"a":1,"b":2}}
`,
			indentation: 0,
			description: "Compact mode (indent=0) produces single-line objects",
		},
		{
			name: "deeply nested mixed styles",
			yamlInput: `root:
  level1: {a: 1, b: [1, 2, 3]}
  level2:
    c: {x: 10, y: 20}
    d:
      - {id: 1}
      - {id: 2}`,
			expectedJSON: `{
  "root": {
    "level1": {"a": 1, "b": [1, 2, 3]},
    "level2": {
      "c": {"x": 10, "y": 20},
      "d": [
        {"id": 1},
        {"id": 2}
      ]
    }
  }
}
`,
			indentation: 2,
			description: "Deeply nested mixed flow and block styles - flow stays compact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlInput), &node)
			require.NoError(t, err, "failed to parse YAML input")

			var buffer bytes.Buffer
			err = json.YAMLToJSON(&node, tt.indentation, &buffer)
			require.NoError(t, err, "YAMLToJSON should not return error")

			actualJSON := buffer.String()
			assert.Equal(t, tt.expectedJSON, actualJSON, tt.description)
		})
	}
}

func TestYAMLToJSON_ComplexMixedFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		yamlInput    string
		expectedJSON string
		indentation  int
		description  string
	}{
		{
			name: "swagger-like structure with mixed formats",
			yamlInput: `swagger: "2.0"
info: {title: API, version: 1.0.0}
paths:
    /users:
       get:
         tags: [users]
         responses:
           "200":
             description: Success`,
			expectedJSON: `{
  "swagger": "2.0",
  "info": {"title": "API", "version": "1.0.0"},
  "paths": {
    "/users": {
      "get": {
        "tags": ["users"],
        "responses": {
          "200": {
            "description": "Success"
          }
        }
      }
    }
  }
}
`,
			indentation: 2,
			description: "Real-world API spec with mixed flow/block styles",
		},
		{
			name: "configuration file with inline arrays",
			yamlInput: `server:
    ports: [80, 443, 8080]
    hosts: [localhost, example.com]
    options: {timeout: 30, retries: 3}`,
			expectedJSON: `{
  "server": {
    "ports": [80, 443, 8080],
    "hosts": ["localhost", "example.com"],
    "options": {"timeout": 30, "retries": 3}
  }
}
`,
			indentation: 2,
			description: "Config file with inline arrays and objects",
		},
		{
			name: "matrix data structure",
			yamlInput: `matrix:
    - [1, 0, 0]
    - [0, 1, 0]
    - [0, 0, 1]`,
			expectedJSON: `{
  "matrix": [
    [1, 0, 0],
    [0, 1, 0],
    [0, 0, 1]
  ]
}
`,
			indentation: 2,
			description: "Matrix represented as array of flow arrays",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlInput), &node)
			require.NoError(t, err, "failed to parse YAML input")

			var buffer bytes.Buffer
			err = json.YAMLToJSON(&node, tt.indentation, &buffer)
			require.NoError(t, err, "YAMLToJSON should not return error")

			actualJSON := buffer.String()
			assert.Equal(t, tt.expectedJSON, actualJSON, tt.description)
		})
	}
}

func TestYAMLToJSONWithConfig_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		node        *yaml.Node
		indent      string
		indentCount int
		wantError   bool
	}{
		{
			name:        "nil node",
			node:        nil,
			indent:      " ",
			indentCount: 2,
			wantError:   false, // nil node is handled gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			err := json.YAMLToJSONWithConfig(tt.node, tt.indent, tt.indentCount, true, &buffer)

			if tt.wantError {
				assert.Error(t, err, "expected error for invalid input")
			} else {
				assert.NoError(t, err, "expected no error")
			}
		})
	}
}

func TestJSONRoundTrip_PreservesFormatting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		inputJSON   string
		indent      string
		indentCount int
		description string
	}{
		{
			name:        "compact single-line object",
			inputJSON:   `{"name":"John","age":30,"active":true}` + "\n",
			indent:      " ",
			indentCount: 2,
			description: "Compact JSON should stay compact",
		},
		{
			name:        "compact single-line array",
			inputJSON:   `[1,2,3,4,5]` + "\n",
			indent:      " ",
			indentCount: 2,
			description: "Compact arrays stay compact",
		},
		{
			name:        "compact nested structures",
			inputJSON:   `{"user":{"name":"John","address":{"city":"NYC","zip":10001}},"tags":["dev","admin"]}` + "\n",
			indent:      " ",
			indentCount: 2,
			description: "Deeply nested compact JSON stays compact",
		},
		{
			name: "pretty 2-space indentation",
			inputJSON: `{
  "name": "John",
  "age": 30,
  "address": {
    "city": "New York",
    "zip": 10001
  },
  "tags": [
    "developer",
    "admin"
  ]
}
`,
			indent:      " ",
			indentCount: 2,
			description: "Pretty JSON with 2-space indent preserved",
		},
		{
			name: "pretty 4-space indentation",
			inputJSON: `{
    "name": "John",
    "age": 30,
    "nested": {
        "level1": {
            "level2": "value"
        }
    }
}
`,
			indent:      " ",
			indentCount: 4,
			description: "Pretty JSON with 4-space indent preserved",
		},
		{
			name:        "tab indentation",
			inputJSON:   "{\n\t\"name\": \"John\",\n\t\"age\": 30,\n\t\"address\": {\n\t\t\"city\": \"NYC\"\n\t}\n}\n",
			indent:      "\t",
			indentCount: 1,
			description: "Tab-indented JSON preserved",
		},
		{
			name: "mixed formatting - compact and pretty",
			inputJSON: `{
  "compact": {"a": 1, "b": 2},
  "pretty": {
    "c": 3,
    "d": 4
  },
  "array": [1, 2, 3],
  "prettyArray": [
    "one",
    "two"
  ]
}
`,
			indent:      " ",
			indentCount: 2,
			description: "Mixed compact and pretty formatting preserved",
		},
		{
			name: "array of compact objects",
			inputJSON: `[
  {"id": 1, "name": "Alice"},
  {"id": 2, "name": "Bob"},
  {"id": 3, "name": "Charlie"}
]
`,
			indent:      " ",
			indentCount: 2,
			description: "Array of compact objects in pretty array",
		},
		{
			name:        "compact array of pretty objects",
			inputJSON:   `[{"name": "Alice", "age": 30}, {"name": "Bob", "age": 25}]` + "\n",
			indent:      " ",
			indentCount: 2,
			description: "Compact root with inline objects",
		},
		{
			name: "empty structures",
			inputJSON: `{
  "emptyObject": {},
  "emptyArray": [],
  "nestedEmpty": {
    "obj": {},
    "arr": []
  }
}
`,
			indent:      " ",
			indentCount: 2,
			description: "Empty objects and arrays preserved",
		},
		{
			name: "all JSON data types",
			inputJSON: `{
  "string": "hello",
  "number": 42,
  "float": 3.14,
  "boolean": true,
  "null": null,
  "array": [1, 2, 3],
  "object": {"nested": "value"}
}
`,
			indent:      " ",
			indentCount: 2,
			description: "All JSON data types with formatting",
		},
		{
			name: "deeply nested mixed formatting",
			inputJSON: `{
  "level1": {
    "compact": {"a": 1, "b": 2},
    "level2": {
      "array": [1, 2, 3],
      "prettyArray": [
        {"id": 1},
        {"id": 2}
      ],
      "level3": {
        "compact": {"x": 10},
        "pretty": {
          "y": 20
        }
      }
    }
  }
}
`,
			indent:      " ",
			indentCount: 2,
			description: "Complex nested structure with mixed formatting",
		},
		{
			name:        "single-line pretty root with nested structures",
			inputJSON:   `{"users": [{"name": "John", "age": 30}], "count": 1}` + "\n",
			indent:      " ",
			indentCount: 2,
			description: "Compact root with inline nested structures",
		},
		{
			name: "string escaping preserved",
			inputJSON: `{
  "quote": "He said \"hello\"",
  "newline": "line1\nline2",
  "tab": "col1\tcol2",
  "backslash": "path\\to\\file"
}
`,
			indent:      " ",
			indentCount: 2,
			description: "String escaping maintained through roundtrip",
		},
		{
			name: "numeric precision",
			inputJSON: `{
  "int": 42,
  "float": 3.14159,
  "scientific": 1.23e-4,
  "negative": -273.15,
  "zero": 0
}
`,
			indent:      " ",
			indentCount: 2,
			description: "Numeric values preserved exactly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Parse JSON as YAML (yaml parser handles JSON)
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.inputJSON), &node)
			require.NoError(t, err, "failed to parse input JSON")

			// Convert back to JSON with specified indentation
			var buffer bytes.Buffer
			err = json.YAMLToJSONWithConfig(&node, tt.indent, tt.indentCount, true, &buffer)
			require.NoError(t, err, "YAMLToJSONWithConfig should not return error")

			actualJSON := buffer.String()

			// The output should exactly match the input
			assert.Equal(t, tt.inputJSON, actualJSON, tt.description)
		})
	}
}

func TestJSONRoundTrip_SwaggerLikeDocuments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		inputJSON   string
		description string
	}{
		{
			name: "minimal swagger with mixed formatting",
			inputJSON: `{
  "swagger": "2.0",
  "info": {"title": "API", "version": "1.0.0"},
  "paths": {
    "/users": {
      "get": {
        "responses": {
          "200": {"description": "Success"}
        }
      }
    }
  }
}
`,
			description: "Swagger-like doc with mixed compact/pretty",
		},
		{
			name: "fully compact swagger",
			inputJSON: `{"swagger":"2.0","info":{"title":"API","version":"1.0.0"},"paths":{"/users":{"get":{"responses":{"200":{"description":"OK"}}}}}}
`,
			description: "Fully compact Swagger stays compact",
		},
		{
			name: "pretty swagger with compact inline objects",
			inputJSON: `{
  "swagger": "2.0",
  "info": {
    "title": "My API",
    "version": "1.0.0"
  },
  "paths": {
    "/users": {
      "get": {
        "tags": ["users"],
        "parameters": [
          {"name": "id", "in": "query", "type": "string"},
          {"name": "limit", "in": "query", "type": "integer"}
        ],
        "responses": {
          "200": {
            "description": "Success",
            "schema": {"type": "array", "items": {"$ref": "#/definitions/User"}}
          }
        }
      }
    }
  }
}
`,
			description: "Realistic API spec with inline parameter objects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Parse JSON
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.inputJSON), &node)
			require.NoError(t, err, "failed to parse input JSON")

			// Convert back to JSON
			var buffer bytes.Buffer
			err = json.YAMLToJSON(&node, 2, &buffer)
			require.NoError(t, err, "YAMLToJSON should not return error")

			actualJSON := buffer.String()

			// Should match exactly
			assert.Equal(t, tt.inputJSON, actualJSON, tt.description)
		})
	}
}

func TestJSONRoundTrip_DifferentIndentations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		inputJSON   string
		indent      string
		indentCount int
		description string
	}{
		{
			name: "2-space to 2-space",
			inputJSON: `{
  "key": "value",
  "nested": {
    "inner": "data"
  }
}
`,
			indent:      " ",
			indentCount: 2,
			description: "2-space indentation preserved",
		},
		{
			name: "4-space to 4-space",
			inputJSON: `{
    "key": "value",
    "nested": {
        "inner": "data"
    }
}
`,
			indent:      " ",
			indentCount: 4,
			description: "4-space indentation preserved",
		},
		{
			name:        "tabs to tabs",
			inputJSON:   "{\n\t\"key\": \"value\",\n\t\"nested\": {\n\t\t\"inner\": \"data\"\n\t}\n}\n",
			indent:      "\t",
			indentCount: 1,
			description: "Tab indentation preserved",
		},
		{
			name:        "compact to compact",
			inputJSON:   `{"key":"value","nested":{"inner":"data"}}` + "\n",
			indent:      " ",
			indentCount: 0,
			description: "Compact JSON preserved with indent=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Parse JSON
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.inputJSON), &node)
			require.NoError(t, err, "failed to parse input JSON")

			// Convert with specified indentation
			var buffer bytes.Buffer
			err = json.YAMLToJSONWithConfig(&node, tt.indent, tt.indentCount, true, &buffer)
			require.NoError(t, err, "conversion should succeed")

			actualJSON := buffer.String()

			// Should match exactly
			assert.Equal(t, tt.inputJSON, actualJSON, tt.description)
		})
	}
}
