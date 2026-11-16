package arazzo_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// expressionReferencesTestDoc builds a document whose `main` workflow has the extra steps appended, alongside an
// `other` workflow that can be referenced.
func expressionReferencesTestDoc(mainInputs, steps, components string) string {
	return `arazzo: 1.1.0
info:
  title: References
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: main
    inputs:
` + mainInputs + `
    steps:
      - stepId: first
        operationId: getPet
        outputs:
          id: $response.body#/id
` + steps + `
  - workflowId: other
    inputs:
      type: object
      required:
        - petId
      properties:
        petId:
          type: integer
    outputs:
      result: $steps.only.outputs.value
    steps:
      - stepId: only
        operationId: getPet
        outputs:
          value: $response.body
` + components
}

const expressionReferencesTestInputs = `      type: object
      properties:
        petId:
          type: integer`

func validateExpressionReferencesDoc(t *testing.T, doc string) []string {
	t.Helper()

	_, validationErrs, err := arazzo.Unmarshal(t.Context(), bytes.NewBufferString(doc))
	require.NoError(t, err, "unmarshal should succeed")
	return severityMessages(validationErrs)
}

func TestArazzo_Validate_ExpressionReferences_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mainInputs string
		steps      string
		components string
	}{
		{
			name:       "references that exist",
			mainInputs: expressionReferencesTestInputs,
			steps: `      - stepId: second
        operationId: getPet
        parameters:
          - name: id
            in: query
            value: $steps.first.outputs.id
          - name: input
            in: query
            value: $inputs.petId
          - name: nested
            in: query
            value: $inputs.petId.nested
          - name: workflowOutput
            in: query
            value: $workflows.other.outputs.result
          - name: workflowInput
            in: query
            value: $workflows.other.inputs.petId`,
		},
		{
			name: "inputs allowing additional properties",
			mainInputs: `      type: object
      additionalProperties: true`,
			steps: `      - stepId: second
        operationId: getPet
        parameters:
          - name: input
            in: query
            value: $inputs.anything`,
		},
		{
			name:       "descriptions and extensions of workflows and steps aren't checked",
			mainInputs: expressionReferencesTestInputs,
			steps: `      - stepId: second
        description: Runs after $steps.missing
        operationId: getPet
        x-note: $inputs.missing`,
		},
		{
			name:       "local workflow step and actions with matching parameters",
			mainInputs: expressionReferencesTestInputs,
			steps: `      - stepId: second
        workflowId: other
        parameters:
          - name: petId
            value: 1
        onSuccess:
          - reference: $components.successActions.retryOther
        onFailure:
          - reference: $components.failureActions.recoverOther`,
			components: `components:
  successActions:
    retryOther:
      name: retryOther
      type: goto
      workflowId: other
      parameters:
        - name: petId
          value: 1
  failureActions:
    recoverOther:
      name: recoverOther
      type: goto
      workflowId: other
      parameters:
        - name: petId
          value: 1
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, validateExpressionReferencesDoc(t, expressionReferencesTestDoc(tt.mainInputs, tt.steps, tt.components)), "document should be valid")
		})
	}
}

func TestArazzo_Validate_ExpressionReferences_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		steps      string
		components string
		expected   []string
	}{
		{
			name: "unknown references",
			steps: `      - stepId: second
        operationId: getPet
        parameters:
          - name: step
            in: query
            value: $steps.missing.outputs.id
          - name: stepOutput
            in: query
            value: $steps.first.outputs.missing
          - name: input
            in: query
            value: $inputs.missing
          - name: workflow
            in: query
            value: $workflows.missing.outputs.result
          - name: workflowOutput
            in: query
            value: $workflows.other.outputs.missing
          - name: workflowInput
            in: query
            value: $workflows.other.inputs.missing`,
			expected: []string{
				"error: expression `$steps.missing.outputs.id` references step `missing` which doesn't exist in workflow `main`",
				"error: expression `$steps.first.outputs.missing` references output `missing` which isn't defined by step `first`",
				"error: expression `$inputs.missing` references input `missing` which isn't defined by workflow `main`",
				"error: expression `$workflows.missing.outputs.result` references workflow `missing` which doesn't exist",
				"error: expression `$workflows.other.outputs.missing` references output `missing` which isn't defined by workflow `other`",
				"error: expression `$workflows.other.inputs.missing` references input `missing` which isn't defined by workflow `other`",
			},
		},
		{
			name: "aliased payloads are only checked once",
			steps: `      - stepId: second
        operationId: createPet
        requestBody:
          payload: &payload
            id: $inputs.missing
      - stepId: third
        operationId: createPet
        requestBody:
          payload: *payload`,
			expected: []string{
				"error: expression `$inputs.missing` references input `missing` which isn't defined by workflow `main`",
			},
		},
		{
			name: "aliased objects reused as payloads are checked as payloads",
			steps: `      - stepId: second
        operationId: createPet
        parameters:
          - &param
            name: id
            in: query
            value: 1
            x-extra: $inputs.missing
        requestBody:
          payload: *param`,
			expected: []string{
				"error: expression `$inputs.missing` references input `missing` which isn't defined by workflow `main`",
			},
		},
		{
			name: "payload fields named like workflow fields are checked",
			steps: `      - stepId: second
        operationId: createPet
        requestBody:
          payload:
            description: $inputs.description
            dependsOn: $inputs.dependsOn
            x-id: $inputs.extension`,
			expected: []string{
				"error: expression `$inputs.description` references input `description` which isn't defined by workflow `main`",
				"error: expression `$inputs.dependsOn` references input `dependsOn` which isn't defined by workflow `main`",
				"error: expression `$inputs.extension` references input `extension` which isn't defined by workflow `main`",
			},
		},
		{
			name: "dotted parameter names must match an input exactly",
			steps: `      - stepId: second
        workflowId: other
        parameters:
          - name: petId.nested
            value: 1`,
			expected: []string{
				"error: step is missing required input `petId` of workflow `other`",
				"error: step.parameters `petId.nested` is not an input of workflow `other`",
			},
		},
		{
			name: "local workflow step and component actions with mismatched parameters",
			steps: `      - stepId: second
        workflowId: other
        parameters:
          - name: unknown
            value: 1`,
			components: `components:
  successActions:
    retryOther:
      name: retryOther
      type: goto
      workflowId: other
  failureActions:
    recoverOther:
      name: recoverOther
      type: goto
      workflowId: other
      parameters:
        - name: unknown
          value: 1
`,
			expected: []string{
				"error: step is missing required input `petId` of workflow `other`",
				"error: step.parameters `unknown` is not an input of workflow `other`",
				"error: successAction is missing required input `petId` of workflow `other`",
				"error: failureAction is missing required input `petId` of workflow `other`",
				"error: failureAction.parameters `unknown` is not an input of workflow `other`",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, validateExpressionReferencesDoc(t, expressionReferencesTestDoc(expressionReferencesTestInputs, tt.steps, tt.components)), "validation errors should match")
		})
	}
}
