package asyncapi

import (
	"errors"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func traitsInternalDoc(t *testing.T, doc string) *AsyncAPI {
	t.Helper()

	a, _, err := Unmarshal(t.Context(), strings.NewReader(doc))
	require.NoError(t, err, "unmarshal should succeed")
	return a
}

// traitsInternalCall returns a resolution call for the root of a document.
func traitsInternalCall(a *AsyncAPI) *resolveCall {
	return &resolveCall{doc: a, root: location{doc: a}, state: &resolutionState{}, queued: map[any]bool{}, errs: []error{}}
}

// traitsBreakField replaces a field's value with an empty document node, which can't be unmarshalled.
func traitsBreakField(t *testing.T, root *yaml.Node, field string) {
	t.Helper()

	_, value, found := yml.GetMapElementNodes(t.Context(), root, field)
	require.True(t, found, "field %s should be present", field)
	*value = yaml.Node{Kind: yaml.DocumentNode}
}

func traitsErrorStrings(errs []error) []string {
	out := make([]string, 0, len(errs))
	for _, err := range errs {
		out = append(out, err.Error())
	}
	return out
}

func TestApplyTraits_UnmarshalEffective_Error(t *testing.T) {
	t.Parallel()

	a := traitsInternalDoc(t, `asyncapi: 3.1.0
info:
  title: T
  version: '1'
channels:
  c:
    address: c
operations:
  o:
    action: send
    channel:
      $ref: '#/channels/c'
    summary: own
    traits:
      - description: from trait
components:
  messages:
    m:
      name: own
      traits:
        - contentType: application/json
`)
	operation := a.GetOperations().GetOrZero("o").GetObject()
	message := a.GetComponents().Messages.GetOrZero("m").GetObject()
	traitsBreakField(t, operation.GetCore().GetRootNode(), "summary")
	traitsBreakField(t, message.GetCore().GetRootNode(), "name")

	rc := traitsInternalCall(a)
	rc.applyOperationTraits(t.Context(), operation, rc.root)
	rc.applyMessageTraits(t.Context(), message, rc.root)

	assert.Equal(t, []string{
		"[10:5] error validation-invalid-syntax expected 1 node, got `0` at line `0`, column `0`",
		"[19:7] error validation-invalid-syntax expected 1 node, got `0` at line `0`, column `0`",
	}, traitsErrorStrings(rc.errs), "effective objects that can't be unmarshalled should be reported at the authored object")
	assert.Nil(t, operation.GetEffective(), "no effective operation should be set")
	assert.Nil(t, message.GetEffective(), "no effective message should be set")
}

func TestEffectiveView_Nil_Success(t *testing.T) {
	t.Parallel()

	assert.Nil(t, (*Message)(nil).effectiveView(), "a nil message should have no effective view")
	assert.Nil(t, (*Operation)(nil).effectiveView(), "a nil operation should have no effective view")

	var view *effectiveView[Message]
	assert.Nil(t, view.get(), "a nil view should have no object")
	_, ok := view.source("name")
	assert.False(t, ok, "a nil view should have no sources")
}

func TestMerger_CopyNil_Success(t *testing.T) {
	t.Parallel()

	m := merger{nodeDocs: map[*yaml.Node]string{}}
	assert.Nil(t, m.copy(nil, "other.yaml"), "copying nothing should return nothing")
	assert.Empty(t, m.nodeDocs, "no document should be recorded")
}

func TestInheritedErrors_NotValidationError_Success(t *testing.T) {
	t.Parallel()

	rc := traitsInternalCall(nil)
	rc.inheritedErrors([]error{errors.New("plain")}, diagnosticSet{}, map[*yaml.Node]string{}, "", rc.root)
	assert.Empty(t, rc.errs, "errors without a location should not be attributed to the effective view")
}
