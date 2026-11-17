package rules_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/linter/rules"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const unusedComponentsDoc = `arazzo: 1.1.0
info:
  title: Components
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: main
    inputs:
      $ref: "#/components/inputs/pet~1input"
    steps:
      - stepId: first
        operationId: getPet
        parameters:
          - reference: $components.parameters.dotted.name
        onSuccess:
          - reference: $components.successActions.done
components:
  inputs:
    pet/input:
      type: object
      properties:
        owner:
          $ref: "#/components/inputs/owner"
    owner:
      type: string
    unusedInput:
      type: object
      properties:
        self:
          $ref: "#/components/inputs/unusedInput"
  parameters:
    dotted.name:
      name: limit
      in: query
      value: 1
    unusedParam:
      name: offset
      in: query
      value: $components.parameters.unusedParam.value
  successActions:
    done:
      name: done
      type: end
    unusedSuccess:
      name: unusedSuccess
      type: end
  failureActions:
    unusedFailure:
      name: unusedFailure
      type: end
`

func TestUnusedComponentRule_Error(t *testing.T) {
	t.Parallel()

	expected := []string{
		"warning: component `unusedInput` in `components.inputs` is never referenced",
		"warning: component `unusedParam` in `components.parameters` is never referenced",
		"warning: component `unusedSuccess` in `components.successActions` is never referenced",
		"warning: component `unusedFailure` in `components.failureActions` is never referenced",
	}
	assert.Equal(t, expected, runRule(t, &rules.UnusedComponentRule{}, unusedComponentsDoc), "unused components should be reported")
}

func TestUnusedComponentRule_DottedNames_Success(t *testing.T) {
	t.Parallel()

	doc := minimalDoc + `components:
  parameters:
    page:
      name: page
      in: query
      value: $components.parameters.page.size.value
    page.size:
      name: size
      in: query
      value: 10
`
	expected := []string{"warning: component `page` in `components.parameters` is never referenced"}
	assert.Equal(t, expected, runRule(t, &rules.UnusedComponentRule{}, doc), "a component referencing a dotted name it prefixes should count as a use of that component")
}

func TestUnusedComponentRule_DescriptionMention_Error(t *testing.T) {
	t.Parallel()

	doc := minimalDoc + `    description: Could use $components.parameters.limit one day.
components:
  parameters:
    limit:
      name: limit
      in: query
      value: 10
`
	expected := []string{"warning: component `limit` in `components.parameters` is never referenced"}
	assert.Equal(t, expected, runRule(t, &rules.UnusedComponentRule{}, doc), "mentions in descriptions should not count as references")
}

func TestUnusedComponentRule_NoComponents_Success(t *testing.T) {
	t.Parallel()

	assert.Empty(t, runRule(t, &rules.UnusedComponentRule{}, minimalDoc), "documents without components have nothing to report")
}

func TestUnusedComponentRule_Fix_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	a := parseDoc(t, unusedComponentsDoc)
	errs := (&rules.UnusedComponentRule{}).Run(ctx, linter.NewDocumentInfo(a, ""), &linter.RuleConfig{})
	require.Len(t, errs, 4, "should report the unused components")
	assert.True(t, (&rules.UnusedComponentRule{}).FixAvailable(), "rule should advertise its fix")

	for _, err := range errs {
		var vErr *validation.Error
		require.ErrorAs(t, err, &vErr, "should be a validation error")
		require.NotNil(t, vErr.Fix, "should have a fix")
		assert.False(t, vErr.Fix.Interactive(), "fix should not need input")
		assert.Nil(t, vErr.Fix.Prompts(), "fix should have no prompts")
		require.NoError(t, vErr.Fix.SetInput(nil), "set input should be a no-op")
		assert.Contains(t, vErr.Fix.Description(), "Remove unused component", "description should match")
		before, after := vErr.Fix.(validation.ChangeDescriber).DescribeChange()
		assert.Contains(t, before, "components.", "before should name the component")
		assert.Empty(t, after, "after should be empty")
		require.NoError(t, vErr.Fix.Apply(a), "fix should apply")
	}

	var buf bytes.Buffer
	require.NoError(t, arazzo.Marshal(ctx, a, &buf), "marshal should succeed")
	assert.NotContains(t, buf.String(), "unused", "unused components should be removed")
	assert.NotContains(t, buf.String(), "failureActions", "empty sections should be removed")
	assert.Contains(t, buf.String(), "dotted.name", "used components should be kept")
}

func TestUnusedComponentRule_FixRemovesComponents_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	a := parseDoc(t, minimalDoc+`components:
  inputs:
    unused:
      type: string
`)
	errs := (&rules.UnusedComponentRule{}).Run(ctx, linter.NewDocumentInfo(a, ""), &linter.RuleConfig{})
	require.Len(t, errs, 1, "should report the unused input")

	var vErr *validation.Error
	require.ErrorAs(t, errs[0], &vErr, "should be a validation error")
	require.NoError(t, vErr.Fix.Apply(a), "fix should apply")
	assert.Nil(t, a.Components, "empty components should be removed")
}

func TestUnusedComponentRule_Fix_Error(t *testing.T) {
	t.Parallel()

	a := parseDoc(t, unusedComponentsDoc)
	errs := (&rules.UnusedComponentRule{}).Run(t.Context(), linter.NewDocumentInfo(a, ""), &linter.RuleConfig{})
	require.NotEmpty(t, errs, "should report unused components")

	var vErr *validation.Error
	require.ErrorAs(t, errs[0], &vErr, "should be a validation error")

	require.Error(t, vErr.Fix.Apply("not a document"), "fix should reject other documents")
	require.Error(t, vErr.Fix.Apply(&arazzo.Arazzo{}), "fix should fail without components")
	require.Error(t, vErr.Fix.Apply(&arazzo.Arazzo{Components: &arazzo.Components{}}), "fix should fail when the component is missing")
}
