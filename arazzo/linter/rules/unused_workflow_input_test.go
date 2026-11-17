package rules_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo/linter/rules"
	"github.com/stretchr/testify/assert"
)

// inputsDoc builds a document with a workflow using the given inputs and step parameter value.
func inputsDoc(inputs, value, components string) string {
	return `arazzo: 1.1.0
info:
  title: Inputs
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: main
    inputs:
` + inputs + `
    steps:
      - stepId: first
        operationId: getPet
        parameters:
          - name: petId
            in: query
            value: ` + value + `
` + components
}

func TestUnusedWorkflowInputRule_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		inputs     string
		value      string
		components string
	}{
		{
			name: "inputs read with $inputs",
			inputs: `      type: object
      properties:
        petId:
          type: integer
        user.name:
          type: string`,
			value: `"{$inputs.petId}-{$inputs.user.name.first}"`,
		},
		{
			name: "whole of $inputs used",
			inputs: `      type: object
      properties:
        petId:
          type: integer`,
			value: "$inputs",
		},
		{
			name:   "unresolvable components reference",
			inputs: `      $ref: "#/components/inputs/missing"`,
			value:  "1",
		},
		{
			name:   "reference outside components",
			inputs: `      $ref: "https://example.com/schemas/inputs.json"`,
			value:  "1",
		},
		{
			name:   "components input that is itself a reference",
			inputs: `      $ref: "#/components/inputs/alias"`,
			value:  "1",
			components: `components:
  inputs:
    alias:
      $ref: "#/components/inputs/other"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, runRule(t, &rules.UnusedWorkflowInputRule{}, inputsDoc(tt.inputs, tt.value, tt.components)), "nothing should be reported")
		})
	}
}

func TestUnusedWorkflowInputRule_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		inputs     string
		components string
		expected   []string
	}{
		{
			name: "inline inputs",
			inputs: `      type: object
      properties:
        petId:
          type: integer
        unused:
          type: string`,
			expected: []string{"warning: input `unused` of workflow `main` is never used"},
		},
		{
			name:   "inputs from components",
			inputs: `      $ref: "#/components/inputs/shared"`,
			components: `components:
  inputs:
    shared:
      type: object
      properties:
        petId:
          type: integer
        unused:
          type: string
`,
			expected: []string{"warning: input `unused` of workflow `main` is never used"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, runRule(t, &rules.UnusedWorkflowInputRule{}, inputsDoc(tt.inputs, "$inputs.petId", tt.components)), "findings should match")
		})
	}
}

func TestUnusedWorkflowInputRule_NoInputs_Success(t *testing.T) {
	t.Parallel()

	assert.Empty(t, runRule(t, &rules.UnusedWorkflowInputRule{}, minimalDoc), "workflows without inputs have nothing to report")
}
