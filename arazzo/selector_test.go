package arazzo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// selectorTestDoc is an Arazzo 1.1 document, the %s verb is replaced by the value of the `pet` step output.
const selectorTestDoc = `arazzo: 1.1.0
info:
  title: Selectors
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: main
    steps:
      - stepId: first
        operationId: getPet
        outputs:
          pet:
%s
`

func TestSelector_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "runtime expression",
			value: `            $response.body#/pet`,
		},
		{
			name: "jsonpath selector",
			value: `            context: $response.body
            selector: $.pets[?@.status == 'available'].id
            type: jsonpath`,
		},
		{
			name: "jsonpointer selector",
			value: `            context: $response.body
            selector: /pets/0/id
            type: jsonpointer`,
		},
		{
			name: "xpath selector with version",
			value: `            context: $response.body
            selector: //pet/id
            type:
              type: xpath
              version: xpath-31`,
		},
		{
			name: "jsonpath selector with goessner version",
			value: `            context: $response.body
            selector: $.pets[0].id
            type:
              type: jsonpath
              version: draft-goessner-dispatch-jsonpath-00`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			messages := validationMessages(t, sprintfDoc(selectorTestDoc, tt.value))
			assert.Empty(t, messages, "selector should be valid")
		})
	}
}

func TestSelector_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected []string
	}{
		{
			name: "invalid jsonpath",
			value: `            context: $response.body
            selector: $[
            type: jsonpath`,
			expected: []string{"selector.selector is invalid: invalid jsonpath expression"},
		},
		{
			name: "invalid json pointer",
			value: `            context: $response.body
            selector: pets/0
            type: jsonpointer`,
			expected: []string{"selector.selector is invalid: invalid json pointer"},
		},
		{
			name: "invalid context expression",
			value: `            context: $foo
            selector: /pets
            type: jsonpointer`,
			expected: []string{"selector.context is invalid"},
		},
		{
			name: "criterion only type",
			value: `            context: $response.body
            selector: /pets
            type: simple`,
			expected: []string{"type must be one of [`jsonpath, xpath, jsonpointer`]"},
		},
		{
			name: "mismatched version",
			value: `            context: $response.body
            selector: /pets
            type:
              type: jsonpointer
              version: rfc9535`,
			expected: []string{"version must be one of [`rfc6901`]"},
		},
		{
			name: "missing required fields",
			value: `            context: $response.body
            selector: /pets`,
			expected: []string{"`selector.type` is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			messages := validationMessages(t, sprintfDoc(selectorTestDoc, tt.value))
			assertMessagesContain(t, tt.expected, messages)
		})
	}
}
