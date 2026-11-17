package arazzo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestUpgradeFrom10To11_NilEntries_Success(t *testing.T) {
	t.Parallel()

	doc := &Arazzo{
		Workflows: []*Workflow{
			nil,
			{WorkflowID: "main", Steps: []*Step{nil}},
		},
	}

	assert.NotPanics(t, func() { upgradeFrom10To11(doc) }, "nil workflows and steps should be skipped")
}

func TestValidateSelectorNode_InvalidNode_Error(t *testing.T) {
	t.Parallel()

	errs := validateSelectorNode(t.Context(), &yaml.Node{Kind: yaml.DocumentNode})
	require.Len(t, errs, 1, "should report the unmarshal error")
	assert.Contains(t, errs[0].Error(), "invalid selector object", "error should match")
}
