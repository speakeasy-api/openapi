package arazzo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// payloadReplacementTestDoc is an Arazzo 1.1 document, the first %s verb is replaced by the request body content
// type and the second by the replacements.
const payloadReplacementTestDoc = `arazzo: 1.1.0
info:
  title: Payload replacements
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: main
    inputs:
      type: object
      properties:
        pet:
          type: object
    steps:
      - stepId: first
        operationId: createPet
        requestBody:
          contentType: %s
          payload: {}
          replacements:
%s
`

func TestPayloadReplacement_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		contentType  string
		replacements string
	}{
		{
			name:        "json pointer target by default",
			contentType: "application/json",
			replacements: `            - target: /pet/name
              value: rex`,
		},
		{
			name:        "xpath target for xml content type",
			contentType: "application/xml",
			replacements: `            - target: /pet/name
              value: rex
            - target: //pet[1]/name
              value: fido`,
		},
		{
			name:        "jsonpath target with explicit selector type",
			contentType: "application/json",
			replacements: `            - target: $.pet.name
              targetSelectorType: jsonpath
              value: rex`,
		},
		{
			name:        "non string values and selector values",
			contentType: "application/json",
			replacements: `            - target: /pet/age
              value: 3
            - target: /pet/tags
              value: [a, b]
            - target: /pet/name
              value:
                context: $inputs.pet
                selector: /name
                type: jsonpointer`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			messages := validationMessages(t, fmtPayloadReplacementDoc(tt.contentType, tt.replacements))
			assert.Empty(t, messages, "payload replacements should be valid")
		})
	}
}

func TestPayloadReplacement_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		contentType  string
		replacements string
		expected     []string
	}{
		{
			name:        "invalid json pointer target",
			contentType: "application/json",
			replacements: `            - target: pet/name
              value: rex`,
			expected: []string{"payloadReplacement.target is invalid: invalid json pointer"},
		},
		{
			name:        "invalid jsonpath target",
			contentType: "application/json",
			replacements: `            - target: $[
              targetSelectorType: jsonpath
              value: rex`,
			expected: []string{"payloadReplacement.target is invalid: invalid jsonpath expression"},
		},
		{
			name:        "criterion only target selector type",
			contentType: "application/json",
			replacements: `            - target: /pet/name
              targetSelectorType: regex
              value: rex`,
			expected: []string{"type must be one of [`jsonpath, xpath, jsonpointer`]"},
		},
		{
			name:        "invalid selector value",
			contentType: "application/json",
			replacements: `            - target: /pet/name
              value:
                context: $inputs.pet
                selector: name
                type: jsonpointer`,
			expected: []string{"selector.selector is invalid: invalid json pointer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			messages := validationMessages(t, fmtPayloadReplacementDoc(tt.contentType, tt.replacements))
			assertMessagesContain(t, tt.expected, messages)
		})
	}
}
