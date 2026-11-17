package rules_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo/linter/rules"
	"github.com/stretchr/testify/assert"
)

// stepsDoc builds a document with a single workflow made of the given steps, workflow actions and components.
func stepsDoc(steps, workflowActions, components string) string {
	return `arazzo: 1.1.0
info:
  title: Steps
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: main
` + workflowActions + `    steps:
` + steps + components
}

func TestUnreachableStepRule_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		steps           string
		workflowActions string
		components      string
	}{
		{
			name: "steps run in order",
			steps: `      - stepId: first
        operationId: a
      - stepId: second
        operationId: b
`,
		},
		{
			name: "goto another workflow falls through when conditional",
			steps: `      - stepId: first
        operationId: a
        onSuccess:
          - name: other
            type: goto
            workflowId: other
            criteria:
              - condition: $statusCode == 202
      - stepId: second
        operationId: b
`,
		},
		{
			name: "conditional end falls through",
			steps: `      - stepId: first
        operationId: a
        onSuccess:
          - name: stop
            type: end
            criteria:
              - condition: $statusCode == 204
      - stepId: second
        operationId: b
`,
		},
		{
			name: "goto reaches a skipped step",
			steps: `      - stepId: first
        operationId: a
        onSuccess:
          - name: skip
            type: goto
            stepId: third
      - stepId: second
        operationId: b
        onSuccess:
          - name: done
            type: end
      - stepId: third
        operationId: c
        onFailure:
          - name: recover
            type: goto
            stepId: second
`,
		},
		{
			name: "referenced component action",
			steps: `      - stepId: first
        operationId: a
        onSuccess:
          - reference: $components.successActions.toThird
      - stepId: second
        operationId: b
        onSuccess:
          - name: done
            type: end
      - stepId: third
        operationId: c
        onFailure:
          - reference: $components.failureActions.toSecond
      - stepId: fourth
        operationId: d
`,
			components: `components:
  successActions:
    toThird:
      name: toThird
      type: goto
      stepId: third
  failureActions:
    toSecond:
      name: toSecond
      type: goto
      stepId: second
`,
		},
		{
			name: "step action overrides the workflow action with the same name",
			workflowActions: `    successActions:
      - name: done
        type: end
`,
			steps: `      - stepId: first
        operationId: a
        onSuccess:
          - name: done
            type: goto
            stepId: second
      - stepId: second
        operationId: b
`,
		},
		{
			name: "workflow failure action goes to a step",
			workflowActions: `    successActions:
      - name: done
        type: end
    failureActions:
      - reference: $components.failureActions.toSecond
`,
			steps: `      - stepId: first
        operationId: a
      - stepId: second
        operationId: b
`,
			components: `components:
  failureActions:
    toSecond:
      name: toSecond
      type: goto
      stepId: second
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, runRule(t, &rules.UnreachableStepRule{}, stepsDoc(tt.steps, tt.workflowActions, tt.components)), "all steps should be reachable")
		})
	}
}

func TestUnreachableStepRule_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		steps           string
		workflowActions string
		expected        []string
	}{
		{
			name: "unconditional end",
			steps: `      - stepId: first
        operationId: a
        onSuccess:
          - name: stop
            type: end
          - name: never
            type: goto
            stepId: third
      - stepId: second
        operationId: b
      - stepId: third
        operationId: c
`,
			expected: []string{
				"warning: step `second` in workflow `main` can never run",
				"warning: step `third` in workflow `main` can never run",
			},
		},
		{
			name: "workflow end action applies to every step",
			workflowActions: `    successActions:
      - name: done
        type: end
`,
			steps: `      - stepId: first
        operationId: a
      - stepId: second
        operationId: b
`,
			expected: []string{"warning: step `second` in workflow `main` can never run"},
		},
		{
			name: "goto past a step",
			steps: `      - stepId: first
        operationId: a
        onSuccess:
          - name: skip
            type: goto
            stepId: third
        onFailure:
          - name: giveUp
            type: end
          - name: never
            type: goto
            stepId: second
      - stepId: second
        operationId: b
      - stepId: third
        operationId: c
        onSuccess:
          - name: toUnknown
            type: goto
            stepId: missing
`,
			expected: []string{"warning: step `second` in workflow `main` can never run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, runRule(t, &rules.UnreachableStepRule{}, stepsDoc(tt.steps, tt.workflowActions, "")), "findings should match")
		})
	}
}

func TestUnreachableStepRule_NoSteps_Success(t *testing.T) {
	t.Parallel()

	assert.Empty(t, runRule(t, &rules.UnreachableStepRule{}, stepsDoc("        []\n", "", "")), "a workflow without steps has nothing to report")
}
