package arazzo

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckWorkflowInputParameters_SkipsUnresolvedParameters_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	doc := `arazzo: 1.1.0
info:
  title: Inputs
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
workflows:
  - workflowId: target
    inputs:
      type: object
      properties:
        petId:
          type: integer
    steps:
      - stepId: first
        operationId: getPet
`
	a, _, err := Unmarshal(ctx, bytes.NewBufferString(doc), WithSkipValidation())
	require.NoError(t, err, "unmarshal should succeed")

	parameters := []*ReusableParameter{
		nil,
		{},
		{Object: &Parameter{Name: ""}},
		{Object: &Parameter{Name: "petId"}},
	}

	result := checkWorkflowInputParameters("step", nil, parameters, a.Workflows.Find("target"))
	assert.Empty(t, result.unknown, "unresolved parameters should be skipped")
	assert.Empty(t, result.missing, "no inputs should be missing")
}

func TestGetWorkflowInputs_AdditionalProperties_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	tests := []struct {
		name          string
		inputs        string
		expectedKnown bool
	}{
		{
			name: "additional properties allowed",
			inputs: `      type: object
      additionalProperties: true
      properties:
        petId:
          type: integer`,
			expectedKnown: false,
		},
		{
			name: "additional properties schema",
			inputs: `      type: object
      additionalProperties:
        type: string`,
			expectedKnown: false,
		},
		{
			name: "additional properties disallowed",
			inputs: `      type: object
      additionalProperties: false
      properties:
        petId:
          type: integer`,
			expectedKnown: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := "arazzo: 1.1.0\ninfo:\n  title: Inputs\n  version: 1.0.0\nworkflows:\n  - workflowId: target\n    inputs:\n" + tt.inputs + "\n    steps: []\n"
			a, _, err := Unmarshal(ctx, bytes.NewBufferString(doc), WithSkipValidation())
			require.NoError(t, err, "unmarshal should succeed")

			assert.Equal(t, tt.expectedKnown, getWorkflowInputs(a.Workflows.Find("target")).known, "known should match")
		})
	}
}
