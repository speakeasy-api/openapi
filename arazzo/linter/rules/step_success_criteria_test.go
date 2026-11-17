package rules_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo/linter/rules"
	"github.com/stretchr/testify/assert"
)

func TestStepSuccessCriteriaRule_Success(t *testing.T) {
	t.Parallel()

	doc := stepsDoc(`      - stepId: checked
        operationId: a
        successCriteria:
          - condition: $statusCode == 200
      - stepId: workflowStep
        workflowId: other
`, "", "")
	assert.Empty(t, runRule(t, &rules.StepSuccessCriteriaRule{}, doc), "steps with criteria and workflow steps should not be reported")
}

func TestStepSuccessCriteriaRule_Error(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"warning: step `first` in workflow `main` has no successCriteria"}, runRule(t, &rules.StepSuccessCriteriaRule{}, minimalDoc), "steps without criteria should be reported")
}
