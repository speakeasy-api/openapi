package rules_test

import (
	"testing"
	"testing/fstest"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/linter/rules"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const untypedSources = `  - name: typed
    url: typed.yaml
    type: openapi
  - name: untyped
    url: untyped.yaml
`

// sourceTypeFix runs the rule and returns the fix for the untyped source.
func sourceTypeFix(t *testing.T, a *arazzo.Arazzo, sources *arazzo.SourceDocuments) validation.Fix {
	t.Helper()

	ctx := rules.ContextWithSourceDocuments(t.Context(), sources)
	errs := (&rules.SourceDescriptionTypeRule{}).Run(ctx, linter.NewDocumentInfo(a, ""), &linter.RuleConfig{})
	require.Equal(t, []string{"warning: sourceDescription `untyped` has no type"}, findings(errs), "untyped source should be reported")

	var vErr *validation.Error
	require.ErrorAs(t, errs[0], &vErr, "should be a validation error")
	require.NotNil(t, vErr.Fix, "should have a fix")
	return vErr.Fix
}

func TestSourceDescriptionTypeRule_LoadedSourceFix_Success(t *testing.T) {
	t.Parallel()

	a := parseDoc(t, sourcesDoc("1.1.0", untypedSources))
	sources, loadErrs := arazzo.LoadSourceDescriptions(t.Context(), a, arazzo.LoadSourcesOptions{
		VirtualFS: fstest.MapFS{
			"typed.yaml":   {Data: []byte("openapi: 3.1.0\ninfo:\n  title: Typed\n  version: 1.0.0\n")},
			"untyped.yaml": {Data: []byte("arazzo: 1.1.0\n")},
		},
	})
	require.Empty(t, loadErrs, "sources should load")

	fix := sourceTypeFix(t, a, sources)
	assert.True(t, (&rules.SourceDescriptionTypeRule{}).FixAvailable(), "rule should advertise its fix")
	assert.False(t, fix.Interactive(), "fix should not need input when the source was loaded")
	assert.Nil(t, fix.Prompts(), "fix should have no prompts")
	assert.Equal(t, "Set the type of sourceDescription `untyped` to `arazzo`", fix.Description(), "description should match")
	_, after := fix.(validation.ChangeDescriber).DescribeChange()
	assert.Equal(t, "type: arazzo", after, "change should match")

	require.NoError(t, fix.Apply(a), "fix should apply")
	assert.Equal(t, arazzo.SourceDescriptionType(arazzo.SourceDescriptionTypeArazzo), a.SourceDescriptions[1].Type, "type should be set")
}

func TestSourceDescriptionTypeRule_InteractiveFix_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		arazzoVersion   string
		expectedChoices []string
	}{
		{name: "Arazzo 1.0 doesn't offer asyncapi", arazzoVersion: "1.0.1", expectedChoices: []string{"openapi", "arazzo"}},
		{name: "Arazzo 1.1 offers asyncapi", arazzoVersion: "1.1.0", expectedChoices: []string{"openapi", "arazzo", "asyncapi"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := parseDoc(t, sourcesDoc(tt.arazzoVersion, untypedSources))
			fix := sourceTypeFix(t, a, nil)
			assert.True(t, fix.Interactive(), "fix should need input when the source wasn't loaded")
			assert.Equal(t, "Set the type of sourceDescription `untyped`", fix.Description(), "description should match")
			require.Len(t, fix.Prompts(), 1, "fix should have one prompt")
			assert.Equal(t, tt.expectedChoices, fix.Prompts()[0].Choices, "choices should match")

			require.Error(t, fix.SetInput(nil), "fix should need one response")
			require.Error(t, fix.SetInput([]string{"OpenAPI"}), "fix should reject types that aren't offered")
			require.NoError(t, fix.SetInput([]string{"openapi"}), "set input should succeed")
			require.NoError(t, fix.Apply(a), "fix should apply")
			assert.Equal(t, arazzo.SourceDescriptionType(arazzo.SourceDescriptionTypeOpenAPI), a.SourceDescriptions[1].Type, "type should be set")
		})
	}
}

func TestSourceDescriptionTypeRule_Fix_Error(t *testing.T) {
	t.Parallel()

	fix := sourceTypeFix(t, parseDoc(t, sourcesDoc("1.1.0", untypedSources)), nil)

	require.Error(t, fix.Apply("not a document"), "fix should reject other documents")
	require.Error(t, fix.Apply(&arazzo.Arazzo{}), "fix should fail when the source is missing")
	require.Error(t, fix.Apply(&arazzo.Arazzo{SourceDescriptions: arazzo.SourceDescriptions{nil, nil}}), "fix should fail when the source is nil")
	require.Error(t, fix.Apply(&arazzo.Arazzo{SourceDescriptions: arazzo.SourceDescriptions{nil, {Name: "renamed"}}}), "fix should fail when the source changed")
}
