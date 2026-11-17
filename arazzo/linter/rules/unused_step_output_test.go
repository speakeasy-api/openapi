package rules_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo/linter/rules"
	"github.com/stretchr/testify/assert"
)

func TestUnusedStepOutputRule_Success(t *testing.T) {
	t.Parallel()

	doc := stepsDoc(`      - stepId: first
        operationId: a
        outputs:
          id: $response.body#/id
          owner.name: $response.body#/owner/name
      - stepId: second
        operationId: b
        parameters:
          - name: id
            in: query
            value: $steps.first.outputs.id
          - name: owner
            in: query
            value: "{$steps.first.outputs.owner.name.first}"
        outputs:
          all: $response.body
      - stepId: third
        operationId: c
        requestBody:
          payload: $steps.second.outputs
`, "", "")

	assert.Empty(t, runRule(t, &rules.UnusedStepOutputRule{}, doc), "used outputs should not be reported")
}

func TestUnusedStepOutputRule_Components_Success(t *testing.T) {
	t.Parallel()

	doc := stepsDoc(`      - stepId: first
        operationId: a
        outputs:
          id: $response.body#/id
      - stepId: second
        operationId: b
        parameters:
          - reference: $components.parameters.id
`, "", `components:
  parameters:
    id:
      name: id
      in: query
      value: $steps.first.outputs.id
`)
	assert.Empty(t, runRule(t, &rules.UnusedStepOutputRule{}, doc), "outputs read by reusable components should count as used")
}

func TestUnusedStepOutputRule_DescriptionMention_Error(t *testing.T) {
	t.Parallel()

	doc := stepsDoc(`      - stepId: first
        operationId: a
        description: Returns $steps.first.outputs.unused for later.
        outputs:
          unused: $response.body#/name
`, "", "")
	assert.Equal(t, []string{"hint: output `unused` of step `first` is never used"}, runRule(t, &rules.UnusedStepOutputRule{}, doc), "mentions in descriptions should not count as reads")
}

func TestUnusedStepOutputRule_Error(t *testing.T) {
	t.Parallel()

	doc := stepsDoc(`      - stepId: first
        operationId: a
        outputs:
          id: $response.body#/id
          unused: $response.body#/name
`, "", "") + `    outputs:
      id: $steps.first.outputs.id
`

	assert.Equal(t, []string{"hint: output `unused` of step `first` is never used"}, runRule(t, &rules.UnusedStepOutputRule{}, doc), "unused outputs should be reported")
}
