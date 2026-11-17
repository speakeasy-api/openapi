package rules_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/linter/rules"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfoDescriptionRule_Success(t *testing.T) {
	t.Parallel()

	doc := `arazzo: 1.1.0
info:
  title: Described
  description: Adopts pets.
  version: 1.0.0
sourceDescriptions: []
workflows: []
`
	assert.Empty(t, runRule(t, &rules.InfoDescriptionRule{}, doc), "a described document should not be reported")
}

func TestInfoDescriptionRule_Error(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"warning: info section is missing a description"}, runRule(t, &rules.InfoDescriptionRule{}, minimalDoc), "missing description should be reported")
}

func TestInfoDescriptionRule_Fix_Success(t *testing.T) {
	t.Parallel()

	a := parseDoc(t, minimalDoc)
	errs := (&rules.InfoDescriptionRule{}).Run(t.Context(), linter.NewDocumentInfo(a, ""), &linter.RuleConfig{})
	require.Len(t, errs, 1, "should report the missing description")

	var vErr *validation.Error
	require.ErrorAs(t, errs[0], &vErr, "should be a validation error")
	fix := vErr.Fix
	require.NotNil(t, fix, "should have a fix")
	assert.True(t, (&rules.InfoDescriptionRule{}).FixAvailable(), "rule should advertise its fix")
	assert.True(t, fix.Interactive(), "fix should need input")
	assert.Equal(t, "Add a description to the info section", fix.Description(), "description should match")
	require.Len(t, fix.Prompts(), 1, "fix should have one prompt")

	require.Error(t, fix.SetInput(nil), "fix should need one response")
	require.NoError(t, fix.SetInput([]string{"Adopts pets."}), "set input should succeed")
	require.Error(t, fix.Apply("not a document"), "fix should reject other documents")
	require.NoError(t, fix.Apply(a), "fix should apply")
	assert.Equal(t, "Adopts pets.", *a.Info.Description, "description should be set")
	assert.Empty(t, (&rules.InfoDescriptionRule{}).Run(t.Context(), linter.NewDocumentInfo(a, ""), &linter.RuleConfig{}), "fixed document should not be reported")
}

func TestInfoDescriptionRule_EmptyDescription_Error(t *testing.T) {
	t.Parallel()

	a := &arazzo.Arazzo{Info: arazzo.Info{Description: pointerTo("")}}
	assert.Len(t, (&rules.InfoDescriptionRule{}).Run(t.Context(), linter.NewDocumentInfo(a, ""), &linter.RuleConfig{}), 1, "an empty description should be reported")
}
