package asyncapi_test

import (
	"testing"
	"testing/fstest"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// traitsResolve unmarshals a document, ignoring its local diagnostics, and resolves its references.
func traitsResolve(t *testing.T, doc string, opts asyncapi.ResolveAllOptions) (*asyncapi.AsyncAPI, []error) {
	t.Helper()

	a, _ := unmarshal(t, doc)
	errs, err := a.ResolveAllReferences(t.Context(), opts)
	require.NoError(t, err, "resolution should not fail")
	return a, errs
}

func TestApplyTraits_UnresolvedTrait_Success(t *testing.T) {
	t.Parallel()

	doc := minimalDoc(`channels:
  c:
    address: c
operations:
  o:
    action: send
    channel:
      $ref: '#/channels/c'
    summary: own
    traits:
      - $ref: '#/components/operationTraits/missing'
      - description: inline
components:
  messages:
    m:
      name: m
      traits:
        - $ref: '#/components/messageTraits/missing'
`)
	a, errs := traitsResolve(t, doc, asyncapi.ResolveAllOptions{})
	assert.Equal(t, []string{
		"[15:15] error validation-invalid-reference operation trait reference `#/components/operationTraits/missing` target not found: not found -- struct is nil at /components/operationTraits/missing",
		"[22:17] error validation-invalid-reference message trait reference `#/components/messageTraits/missing` target not found: not found -- struct is nil at /components/messageTraits/missing",
	}, errorStrings(errs), "only the unresolved traits should be reported")

	operation := a.GetOperations().GetOrZero("o").GetObject().GetEffective()
	require.NotNil(t, operation, "traits should be applied without the unresolved trait")
	assert.Equal(t, "own", operation.GetSummary(), "operation should keep its own summary")
	assert.Equal(t, "inline", operation.GetDescription(), "resolved traits should still apply")
	source, ok := a.GetOperations().GetOrZero("o").GetObject().GetTraitSource("description")
	assert.True(t, ok, "description should have a source")
	assert.Equal(t, 1, source, "description should come from the second trait")

	message := a.GetComponents().Messages.GetOrZero("m").GetObject()
	require.NotNil(t, message.GetEffective(), "traits should be applied without the unresolved trait")
	assert.Equal(t, "m", message.GetEffective().GetName(), "message should keep its own name")
	source, ok = message.GetTraitSource("name")
	assert.True(t, ok, "name should have a source")
	assert.Equal(t, -1, source, "name should come from the message")
}

func TestApplyTraits_AuthoredDiagnostic_Success(t *testing.T) {
	t.Parallel()

	doc := minimalDoc(`components:
  messages:
    m:
      headers:
        type: string
      traits:
        - contentType: application/json
`)
	a, validationErrs := unmarshal(t, doc)
	assert.Equal(t, []string{"error:" + validation.RuleValidationTypeMismatch}, diagnostics(validationErrs), "headers that aren't an object should be reported by Validate")

	errs, err := a.ResolveAllReferences(t.Context(), asyncapi.ResolveAllOptions{})
	require.NoError(t, err, "resolution should not fail")
	assert.Empty(t, errorStrings(errs), "a diagnostic already reported for the authored message should not be repeated")

	effective := a.GetComponents().Messages.GetOrZero("m").GetObject().GetEffective()
	require.NotNil(t, effective, "traits should be applied")
	assert.Equal(t, "application/json", effective.GetContentType(), "trait should supply the content type")
}

func TestApplyTraits_MergedMappingDiagnostic_Success(t *testing.T) {
	t.Parallel()

	doc := minimalDoc(`components:
  messages:
    m:
      headers:
        type: string
      traits:
        - headers:
            description: from trait
`)
	a, validationErrs := unmarshal(t, doc)
	assert.Equal(t, []string{"error:" + validation.RuleValidationTypeMismatch}, diagnostics(validationErrs), "headers that aren't an object should be reported by Validate")

	errs, err := a.ResolveAllReferences(t.Context(), asyncapi.ResolveAllOptions{})
	require.NoError(t, err, "resolution should not fail")
	assert.Empty(t, errorStrings(errs), "a diagnostic on headers merged from the message and a trait should not be repeated")

	effective := a.GetComponents().Messages.GetOrZero("m").GetObject().GetEffective()
	require.NotNil(t, effective, "traits should be applied")
	headers := effective.GetHeaders().GetSchema().GetSchema()
	require.NotNil(t, headers, "headers should be a schema")
	assert.Equal(t, []string{"string"}, headers.GetType(), "message should supply the headers type")
	assert.Equal(t, "from trait", *headers.Description, "trait should supply the headers description")
}

func TestApplyTraits_ExternalOperation_Success(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{"ops.yaml": {Data: []byte(`o:
  action: send
  channel:
    $ref: '#/c'
  traits:
    - $ref: '#/t'
    - summary: inline
c:
  address: c
t:
  description: from trait
`)}}
	doc := minimalDoc("components:\n  operations:\n    o:\n      $ref: './ops.yaml#/o'\n")
	a, errs := traitsResolve(t, doc, asyncapi.ResolveAllOptions{AsyncAPILocation: "api.yaml", VirtualFS: files})
	assert.Empty(t, errs, "resolution should report nothing")

	operation := a.GetComponents().Operations.GetOrZero("o").GetObject()
	require.NotNil(t, operation, "operation should resolve to the other document")
	effective := operation.GetEffective()
	require.NotNil(t, effective, "traits should be applied")
	assert.Equal(t, "inline", effective.GetSummary(), "inline trait should supply the summary")
	assert.Equal(t, "from trait", effective.GetDescription(), "referenced trait should supply the description")
	assert.Equal(t, "c", effective.Channel.GetChannel().GetAddress(), "effective references should resolve against the other document")

	source, ok := operation.GetTraitSource("description")
	assert.True(t, ok, "description should have a source")
	assert.Equal(t, 0, source, "description should come from the first trait")
}

func TestApplyTraits_ExternalTraitDiagnostic_Success(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{"traits.yaml": {Data: []byte("t:\n  headers:\n    default: 5\n")}}
	doc := minimalDoc(`components:
  messages:
    m:
      headers:
        type: object
      traits:
        - $ref: './traits.yaml#/t'
`)
	a, errs := traitsResolve(t, doc, asyncapi.ResolveAllOptions{AsyncAPILocation: "api.yaml", VirtualFS: files})
	assert.Equal(t, []string{"[3:14] error validation-type-mismatch schema.default must conform to the schema type [`object`] (document: traits.yaml)"}, errorStrings(errs), "the inherited diagnostic should be reported")
	require.Len(t, errs, 1, "one diagnostic should be reported")

	var vErr *validation.Error
	require.ErrorAs(t, errs[0], &vErr, "diagnostic should be a validation error")
	assert.Equal(t, "traits.yaml", vErr.DocumentLocation, "diagnostic should name the trait's document")

	message := a.GetComponents().Messages.GetOrZero("m").GetObject()
	effective := message.GetEffective()
	require.NotNil(t, effective, "traits should be applied")
	assert.Nil(t, a.GetResolvedSchema(effective.Headers), "an effective view merged from several documents should not be resolved")
}

func TestApplyTraits_MessageRefresh_Success(t *testing.T) {
	t.Parallel()

	doc := minimalDoc(`components:
  messages:
    m:
      name: m
      traits:
        - contentType: application/json
`)
	a, errs := traitsResolve(t, doc, asyncapi.ResolveAllOptions{})
	require.Empty(t, errs, "resolution should report nothing")

	message := a.GetComponents().Messages.GetOrZero("m").GetObject()
	previous := message.GetEffective()
	require.NotNil(t, previous, "traits should be applied")

	errs, err := a.ResolveAllReferences(t.Context(), asyncapi.ResolveAllOptions{})
	require.NoError(t, err, "resolution should not fail")
	assert.Empty(t, errs, "resolution should report nothing")
	assert.NotSame(t, previous, message.GetEffective(), "each resolution should build a new effective view")
	assert.Equal(t, "application/json", message.GetEffective().GetContentType(), "trait should still apply")
}
