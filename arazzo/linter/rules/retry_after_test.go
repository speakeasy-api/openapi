package rules_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo/linter/rules"
	"github.com/stretchr/testify/assert"
)

func TestRetryAfterRule_Success(t *testing.T) {
	t.Parallel()

	doc := stepsDoc(`      - stepId: first
        operationId: a
        onFailure:
          - name: wait
            type: retry
            retryAfter: 1
          - name: stop
            type: end
`, "", "")
	assert.Empty(t, runRule(t, &rules.RetryAfterRule{}, doc), "retries with retryAfter should not be reported")
}

func TestRetryAfterRule_Error(t *testing.T) {
	t.Parallel()

	doc := stepsDoc(`      - stepId: first
        operationId: a
        onFailure:
          - name: stepRetry
            type: retry
          - reference: $components.failureActions.componentRetry
`, `    failureActions:
      - name: workflowRetry
        type: retry
`, `components:
  failureActions:
    componentRetry:
      name: componentRetry
      type: retry
`)

	expected := []string{
		"hint: retry failureAction `workflowRetry` has no retryAfter, so it retries immediately",
		"hint: retry failureAction `stepRetry` has no retryAfter, so it retries immediately",
		"hint: retry failureAction `componentRetry` has no retryAfter, so it retries immediately",
	}
	assert.Equal(t, expected, runRule(t, &rules.RetryAfterRule{}, doc), "each retry without retryAfter should be reported once")
}
