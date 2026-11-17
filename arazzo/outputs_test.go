package arazzo_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputValue_FromExpression_Success(t *testing.T) {
	t.Parallel()

	o := arazzo.NewOutputValueFromExpression("$statusCode")
	assert.True(t, o.IsExpression(), "should be an expression")
	assert.False(t, o.IsSelector(), "should not be a selector")
	require.NotNil(t, o.GetExpression(), "expression should be set")
	assert.Equal(t, expression.Expression("$statusCode"), *o.GetExpression(), "expression should match")
	assert.Nil(t, o.GetSelector(), "selector should not be set")
}

func TestOutputValue_FromSelector_Success(t *testing.T) {
	t.Parallel()

	s := &arazzo.Selector{Context: "$response.body", Selector: "/id"}
	o := arazzo.NewOutputValueFromSelector(s)
	assert.True(t, o.IsSelector(), "should be a selector")
	assert.False(t, o.IsExpression(), "should not be an expression")
	assert.Equal(t, s, o.GetSelector(), "selector should match")
	assert.Nil(t, o.GetExpression(), "expression should not be set")
}

func TestOutputValue_Nil_Success(t *testing.T) {
	t.Parallel()

	var o *arazzo.OutputValue
	assert.False(t, o.IsExpression(), "nil output value should not be an expression")
	assert.False(t, o.IsSelector(), "nil output value should not be a selector")
	assert.Nil(t, o.GetExpression(), "nil output value should have no expression")
	assert.Nil(t, o.GetSelector(), "nil output value should have no selector")
}

const outputsTestDoc = `arazzo: 1.1.0
info:
  title: Outputs
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: main
    steps:
      - stepId: first
        operationId: getPet
        outputs:
          pet: $response.body#/pet
`

func TestValidateOutputs_InvalidName_Error(t *testing.T) {
	t.Parallel()

	doc := `arazzo: 1.1.0
info:
  title: Outputs
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: main
    steps:
      - stepId: first
        operationId: getPet
        outputs:
          bad name: $response.body#/pet
`

	messages := validationMessages(t, doc)
	assertMessagesContain(t, []string{"step.outputs name must be a valid name"}, messages)
}

func TestValidateOutputs_EmptyOutputValue_Error(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewBufferString(outputsTestDoc))
	require.NoError(t, err, "unmarshal should succeed")
	require.Empty(t, validationErrs, "document should be valid")

	doc.Workflows[0].Steps[0].Outputs.Set("pet", &arazzo.OutputValue{})

	errs := doc.Validate(ctx)
	require.Len(t, errs, 1, "should report the empty output value")
	assert.Contains(t, errs[0].Error(), "step.outputs value must be a runtime expression or a selector object", "error should match")
}
