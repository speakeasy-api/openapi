package actions_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/internal/actions"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const actionsDoc = `arazzo: 1.1.0
info:
  title: Actions
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: main
    successActions:
      - name: done
        type: end
      - name: shared
        type: end
    failureActions:
      - name: stop
        type: end
      - reference: $components.failureActions.retry
    steps:
      - stepId: first
        operationId: a
        onSuccess:
          - name: done
            type: goto
            stepId: first
          - reference: $components.successActions.missing
        onFailure:
          - name: stop
            type: goto
            stepId: first
components:
  failureActions:
    retry:
      name: retry
      type: retry
`

func TestActions_Success(t *testing.T) {
	t.Parallel()

	a, _, err := arazzo.Unmarshal(t.Context(), bytes.NewBufferString(actionsDoc), arazzo.WithSkipValidation())
	require.NoError(t, err, "unmarshal should succeed")
	w := a.Workflows[0]
	s := w.Steps[0]

	success := []string{}
	for _, action := range actions.Success(a, w, s) {
		success = append(success, action.Name+":"+string(action.Type))
	}
	assert.Equal(t, []string{"done:goto", "shared:end"}, success, "step actions should override workflow actions by name")

	failure := []string{}
	for _, action := range actions.Failure(a, w, s) {
		failure = append(failure, action.Name+":"+string(action.Type))
	}
	assert.Equal(t, []string{"stop:goto", "retry:retry"}, failure, "references should be resolved")
}

func TestActions_Unresolvable_Success(t *testing.T) {
	t.Parallel()

	reference := expression.Expression("$components.successActions.missing")
	a := &arazzo.Arazzo{}
	w := &arazzo.Workflow{}
	s := &arazzo.Step{
		OnSuccess: []*arazzo.ReusableSuccessAction{nil, {Reference: &reference}},
		OnFailure: []*arazzo.ReusableFailureAction{nil, {Reference: &reference}},
	}

	assert.Empty(t, actions.Success(a, w, s), "references without components should be left out")
	assert.Empty(t, actions.Failure(a, w, s), "references without components should be left out")
	assert.Empty(t, actions.Success(a, w, nil), "nil steps have no actions")
	assert.Empty(t, actions.Failure(nil, w, s), "nil documents have no actions")
}
