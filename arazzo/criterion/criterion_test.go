package criterion_test

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/arazzo/expression"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCriterion_Validate_Success(t *testing.T) {
	type args struct {
		c    *criterion.Criterion
		opts []validation.Option
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "successfully validate criterion with empty json object condition",
			args: args{
				c: &criterion.Criterion{
					Context: pointer.From(expression.Expression("$response.body")),
					Type: criterion.CriterionTypeUnion{
						Type: pointer.From(criterion.CriterionTypeSimple),
					},
					Condition: `
[
  {}
]`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.c.Sync(context.Background())
			require.NoError(t, err)
			errs := tt.args.c.Validate(tt.args.opts...)
			assert.Empty(t, errs)
		})
	}
}
