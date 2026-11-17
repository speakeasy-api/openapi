package arazzoexplore

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/yml"
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
						Outputs: sequencedmap.New(
							sequencedmap.NewElem[string, *arazzo.OutputValue]("empty", nil),
							sequencedmap.NewElem("zero", &arazzo.OutputValue{}),
						),
					},
				},
				SuccessActions: []*arazzo.ReusableSuccessAction{nil},
				FailureActions: []*arazzo.ReusableFailureAction{nil},
			},
		},
	}

	expected := []Item{
		{Kind: KindWorkflow, Label: "partial", Details: []string{"Steps: 1"}},
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
				"  - zero",
			},
		},
	}

	assert.Equal(t, expected, Collect(a), "nil entries should be skipped and unusual values described")
}

func TestCollect_MultilineValues_Success(t *testing.T) {
	t.Parallel()

	a := &arazzo.Arazzo{
		Workflows: arazzo.Workflows{
			{
				WorkflowID:  "multiline",
				Summary:     pointer.From("Adopt\na pet"),
				Description: pointer.From("First line.\r\nSecond line.\n"),
				Steps: arazzo.Steps{
					{
						StepID:      "step",
						OperationID: pointer.From(expression.Expression("op")),
						Parameters: []*arazzo.ReusableParameter{
							{Object: &arazzo.Parameter{Name: "note", In: pointer.From(arazzo.InQuery), Value: yml.CreateStringNode("one\ntwo")}},
						},
					},
				},
			},
		},
	}

	items := Collect(a)
	assert.Equal(t, "Adopt a pet", items[0].Summary, "the row summary should be on one line")
	assert.Equal(t, []string{"Summary: Adopt", "  a pet", "Description: First line.", "  Second line.", "Steps: 1"}, items[0].Details, "multiline fields should continue on indented lines")
	assert.Equal(t, []string{"Operation: op", "Parameters:", "  - note (query) = one", "    two"}, items[1].Details, "multiline entries should continue on indented lines")
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
