package expression

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractExpressions(t *testing.T) {
	t.Parallel()
	type args struct {
		expression string
	}
	tests := []struct {
		name string
		args args
		want []Expression
	}{
		// Valid expressions
		{
			name: "simple expression as string",
			args: args{
				expression: "$url",
			},
			want: []Expression{
				Expression("$url"),
			},
		},
		{
			name: "simple expression enclosed in {}",
			args: args{
				expression: "{$url}",
			},
			want: []Expression{
				Expression("{$url}"),
			},
		},
		{
			name: "request body expression with json pointer",
			args: args{
				expression: "$request.body#/some/path",
			},
			want: []Expression{
				Expression("$request.body#/some/path"),
			},
		},
		{
			name: "expression with json pointer enclosed in {}",
			args: args{
				expression: "{$request.body}#/some/path",
			},
			want: []Expression{
				Expression("{$request.body}#/some/path"),
			},
		},
		{
			name: "multiple expressions in string",
			args: args{
				expression: `{$inputs.pet_id}#/some/json/pointer {$inputs.coupon_code}{$inputs.quantity}`,
			},
			want: []Expression{
				Expression("{$inputs.pet_id}#/some/json/pointer"),
				Expression("{$inputs.coupon_code}"),
				Expression("{$inputs.quantity}"),
			},
		},
		{
			name: "multiple expressions in json",
			args: args{
				expression: `
{
	"petOrder": {
		"petId": "{$inputs.pet_id}",
		"couponCode": "{$inputs.coupon_code}",
		"quantity": "{$inputs.quantity}",
		"status": "placed",
		"complete": false
	}
}`,
			},
			want: []Expression{
				Expression("{$inputs.pet_id}"),
				Expression("{$inputs.coupon_code}"),
				Expression("{$inputs.quantity}"),
			},
		},
		{
			name: "multiple expressions in xml",
			args: args{
				expression: `
<petOrder>
	<petId>{$inputs.pet_id}</petId>
	<couponCode>{$inputs.coupon_code}</couponCode>
	<quantity>{$inputs.quantity}</quantity>
	<status>placed</status>
	<complete>false</complete>
</petOrder>`,
			},
			want: []Expression{
				Expression("{$inputs.pet_id}"),
				Expression("{$inputs.coupon_code}"),
				Expression("{$inputs.quantity}"),
			},
		},
		// Invalid expressions
		{
			name: "simple expression like string",
			args: args{
				expression: "{$some thing similar to an expression}",
			},
			want: []Expression{},
		},
		{
			name: "invalid expression",
			args: args{
				expression: "$$some",
			},
			want: []Expression{},
		},
		{
			name: "expression like string with terminating { char",
			args: args{
				expression: "{$something{similar to an expression}}",
			},
			want: []Expression{},
		},
		{
			name: "mix of valid and invalid expressions",
			args: args{
				expression: `{$inputs.pet_id}#/some/json/pointer {$some thing similar to an expression}`,
			},
			want: []Expression{
				Expression("{$inputs.pet_id}#/some/json/pointer"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ExtractExpressions(tt.args.expression)
			assert.Equal(t, tt.want, got)
		})
	}
}
