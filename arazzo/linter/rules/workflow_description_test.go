package rules_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo/linter/rules"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowDescriptionRule_Success(t *testing.T) {
	t.Parallel()

	doc := workflowsDoc(`  - workflowId: summarised
    summary: Adopt a pet
    steps:
      - stepId: one
        operationId: x
  - workflowId: described
    description: Adopts a pet.
    steps:
      - stepId: one
        operationId: x
`)
	assert.Empty(t, runRule(t, &rules.WorkflowDescriptionRule{}, doc), "documented workflows should not be reported")
}

func TestWorkflowDescriptionRule_Error(t *testing.T) {
	t.Parallel()

	doc := workflowsDoc(`  - workflowId: empty
    summary: "  "
    description: ""
    steps:
      - stepId: one
        operationId: x
`)
	assert.Equal(t, []string{"hint: workflow `empty` has no summary or description"}, runRule(t, &rules.WorkflowDescriptionRule{}, doc), "undocumented workflows should be reported")
}
