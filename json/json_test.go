package json_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/json"
	"github.com/speakeasy-api/openapi/sequencedmap"
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

func TestYAMLToJSONCompatibleGoType_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		yamlInput string
		wantAny   any
	}{
		{
			name:      "simple string",
			yamlInput: `hello world`,
			wantAny:   "hello world",
		},
		{
			name:      "simple number",
			yamlInput: `42`,
			wantAny:   42,
		},
		{
			name:      "simple boolean",
			yamlInput: `true`,
			wantAny:   true,
		},
		{
			name:      "null value",
			yamlInput: `null`,
			wantAny:   nil,
		},
		{
			name: "simple object",
			yamlInput: `name: John
age: 30`,
			wantAny: sequencedmap.New(
				sequencedmap.NewElem("name", any("John")),
				sequencedmap.NewElem("age", any(30)),
			),
		},
		{
			name: "nested object",
			yamlInput: `person:
  name: John
  age: 30`,
			wantAny: sequencedmap.New(
				sequencedmap.NewElem("person", any(sequencedmap.New(
					sequencedmap.NewElem("name", any("John")),
					sequencedmap.NewElem("age", any(30)),
				))),
			),
		},
		{
			name: "simple array",
			yamlInput: `- apple
- banana
- cherry`,
			wantAny: []any{"apple", "banana", "cherry"},
		},
		{
			name: "array of objects",
			yamlInput: `- name: John
  age: 30
- name: Jane
  age: 25`,
			wantAny: []any{
				sequencedmap.New(
					sequencedmap.NewElem("name", any("John")),
					sequencedmap.NewElem("age", any(30)),
				),
				sequencedmap.New(
					sequencedmap.NewElem("name", any("Jane")),
					sequencedmap.NewElem("age", any(25)),
				),
			},
		},
		{
			name: "preserves key order",
			yamlInput: `zebra: last
apple: first
middle: second`,
			wantAny: sequencedmap.New(
				sequencedmap.NewElem("zebra", any("last")),
				sequencedmap.NewElem("apple", any("first")),
				sequencedmap.NewElem("middle", any("second")),
			),
		},
		{
			name: "numeric keys converted to strings",
			yamlInput: `1: one
2: two
3: three`,
			wantAny: sequencedmap.New(
				sequencedmap.NewElem("1", any("one")),
				sequencedmap.NewElem("2", any("two")),
				sequencedmap.NewElem("3", any("three")),
			),
		},
		{
			name:      "empty object",
			yamlInput: `{}`,
			wantAny:   sequencedmap.New[string, any](),
		},
		{
			name:      "empty array",
			yamlInput: `[]`,
			wantAny:   []any{},
		},
		{
			name: "yaml alias",
			yamlInput: `defaults: &defaults
  timeout: 30
  retries: 3
production:
  <<: *defaults
  host: prod.example.com`,
			wantAny: sequencedmap.New(
				sequencedmap.NewElem("defaults", any(sequencedmap.New(
					sequencedmap.NewElem("timeout", any(30)),
					sequencedmap.NewElem("retries", any(3)),
				))),
				sequencedmap.NewElem("production", any(sequencedmap.New(
					sequencedmap.NewElem("timeout", any(30)),
					sequencedmap.NewElem("retries", any(3)),
					sequencedmap.NewElem("host", any("prod.example.com")),
				))),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlInput), &node)
			require.NoError(t, err, "failed to parse YAML input")

			actual, err := json.YAMLToJSONCompatibleGoType(&node)
			require.NoError(t, err, "YAMLToJSONCompatibleGoType should not return error")

			assert.Equal(t, tt.wantAny, actual, "result should match expected value")
		})
	}
}

func TestYAMLToJSONCompatibleGoType_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		node      *yaml.Node
		wantError bool
	}{
		{
			name:      "nil node",
			node:      nil,
			wantError: false, // nil node returns nil, nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := json.YAMLToJSONCompatibleGoType(tt.node)

			if tt.wantError {
				assert.Error(t, err, "expected error for invalid input")
			} else {
				assert.NoError(t, err, "expected no error")
				if tt.node == nil {
					assert.Nil(t, result, "nil node should return nil result")
				}
			}
		})
	}
}
