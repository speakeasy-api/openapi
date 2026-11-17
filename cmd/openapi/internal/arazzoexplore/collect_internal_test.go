package arazzoexplore

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
)

// TestCollect_IncompleteModels_Success covers documents built in code, which can have nil entries and values the
// parser wouldn't produce.
func TestCollect_IncompleteModels_Success(t *testing.T) {
	t.Parallel()

	a := &arazzo.Arazzo{
		Workflows: arazzo.Workflows{
			nil,
			{
				WorkflowID: "partial",
				Steps: arazzo.Steps{
					nil,
					{
						StepID:          "odd",
						Parameters:      []*arazzo.ReusableParameter{nil, {}},
						SuccessCriteria: []*criterion.Criterion{nil},
						OnSuccess: []*arazzo.ReusableSuccessAction{
							nil,
							{Object: &arazzo.SuccessAction{Name: "nowhere", Type: arazzo.SuccessActionTypeGoto}},
							{Object: &arazzo.SuccessAction{Name: "custom", Type: "pause"}},
						},
						OnFailure: []*arazzo.ReusableFailureAction{
							nil,
							{Object: &arazzo.FailureAction{Name: "custom", Type: "pause"}},
						},
						Outputs: sequencedmap.New(sequencedmap.NewElem[string, *arazzo.OutputValue]("empty", nil)),
					},
				},
				SuccessActions: []*arazzo.ReusableSuccessAction{nil},
				FailureActions: []*arazzo.ReusableFailureAction{nil},
			},
		},
	}

	expected := []Item{
		{Kind: KindWorkflow, Label: "partial", Details: []string{"Steps: 2"}},
		{
			Kind:  KindStep,
			Label: "odd",
			Badge: BadgeNone,
			Details: []string{
				"Parameters:",
				"  - (empty)",
				"On success:",
				"  - nowhere: go to ?",
				"  - custom: pause",
				"On failure:",
				"  - custom: pause",
				"Outputs:",
				"  - empty",
			},
		},
	}

	assert.Equal(t, expected, Collect(a), "nil entries should be skipped and unusual values described")
}

func TestBadgeColor_Success(t *testing.T) {
	t.Parallel()

	assert.Equal(t, colorGreen, badgeColor(BadgeOperation), "known badges should have their color")
	assert.Equal(t, colorGray, badgeColor(BadgeNone), "other badges should be gray")
}

func TestValue_Success(t *testing.T) {
	t.Parallel()

	assert.Empty(t, value(nil), "a nil value should be empty")
}
