package arazzo

import (
	"testing"

	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v3"
)

func TestArazzo_NilComponentActions_Success(t *testing.T) {
	t.Parallel()

	a := &Arazzo{
		Components: &Components{
			SuccessActions: sequencedmap.New(sequencedmap.NewElem[string, *SuccessAction]("nilAction", nil)),
			FailureActions: sequencedmap.New(sequencedmap.NewElem[string, *FailureAction]("nilAction", nil)),
		},
	}

	assert.Empty(t, a.validateComponentActionsInputParameters(), "nil component actions should be skipped")
	assert.Empty(t, a.ValidateSources(t.Context(), &SourceDocuments{}), "nil component actions should be skipped by source validation")
}

func TestVisitExpressionScalars_NilNode_Success(t *testing.T) {
	t.Parallel()

	visited := []*yaml.Node{}
	visitExpressionScalars(nil, expressionScopeWorkflow, map[expressionVisit]bool{}, func(node *yaml.Node) {
		visited = append(visited, node)
	})
	assert.Empty(t, visited, "nothing should be visited")
}
