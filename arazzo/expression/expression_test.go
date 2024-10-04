package expression

import (
	_ "embed"
	"errors"
	"testing"

	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpression_Validate_Success(t *testing.T) {
	type args struct {
		e                    Expression
		validateAsExpression bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "$url",
			args: args{
				e:                    Expression("$url"),
				validateAsExpression: true,
			},
		},
		{
			name: "{$url}",
			args: args{
				e:                    Expression("{$url}"),
				validateAsExpression: true,
			},
		},
		{
			name: "$method",
			args: args{
				e:                    Expression("$method"),
				validateAsExpression: true,
			},
		},
		{
			name: "$statusCode",
			args: args{
				e:                    Expression("$statusCode"),
				validateAsExpression: true,
			},
		},
		{
			name: "request body without json pointer",
			args: args{
				e:                    Expression("$request.body"),
				validateAsExpression: true,
			},
		},
		{
			name: "request body enclosed in {} without json pointer",
			args: args{
				e:                    Expression("{$request.body}"),
				validateAsExpression: true,
			},
		},
		{
			name: "request body with json pointer",
			args: args{
				e:                    Expression("$request.body#/some/path"),
				validateAsExpression: true,
			},
		},
		{
			name: "request body enclosed in {} with json pointer",
			args: args{
				e:                    Expression("{$request.body}#/some/path"),
				validateAsExpression: true,
			},
		},
		{
			name: "request header",
			args: args{
				e:                    Expression("$request.header.some-header"),
				validateAsExpression: true,
			},
		},
		{
			name: "request query",
			args: args{
				e:                    Expression("$request.query.someQueryParam"),
				validateAsExpression: true,
			},
		},
		{
			name: "request path",
			args: args{
				e:                    Expression("$request.path.somePathParam"),
				validateAsExpression: true,
			},
		},
		{
			name: "response body",
			args: args{
				e:                    Expression("$response.body"),
				validateAsExpression: true,
			},
		},
		{
			"response body with json pointer",
			args{
				e:                    Expression("$response.body#/some/path"),
				validateAsExpression: true,
			},
		},
		{
			name: "response header",
			args: args{
				e:                    Expression("$response.header.some-header"),
				validateAsExpression: true,
			},
		},
		{
			name: "response header enclose in {}",
			args: args{
				e:                    Expression("{$response.header.some-header}"),
				validateAsExpression: true,
			},
		},
		{
			name: "inputs",
			args: args{
				e:                    Expression("$inputs.someInput"),
				validateAsExpression: true,
			},
		},
		{
			name: "outputs",
			args: args{
				e:                    Expression("$outputs.someOutput"),
				validateAsExpression: true,
			},
		},
		{
			name: "steps",
			args: args{
				e:                    Expression("$steps.someStep"),
				validateAsExpression: true,
			},
		},
		{
			name: "workflows",
			args: args{
				e:                    Expression("$workflows.someWorkflow"),
				validateAsExpression: true,
			},
		},
		{
			name: "source descriptions",
			args: args{
				e:                    Expression("$sourceDescriptions.someSourceDescription"),
				validateAsExpression: true,
			},
		},
		{
			name: "source descriptions sub path",
			args: args{
				e:                    Expression("$sourceDescriptions.someSourceDescription.url"),
				validateAsExpression: true,
			},
		},
		{
			name: "source descriptions sub path with json pointer",
			args: args{
				e:                    Expression("{$sourceDescriptions.petStoreDescription.url}#/paths/~1pet~1findByStatus/get"),
				validateAsExpression: true,
			},
		},
		{
			name: "components",
			args: args{
				e:                    Expression("$components.parameters.someParameter"),
				validateAsExpression: true,
			},
		},
		{
			name: "non expression validates okay if validateAsExpression is false",
			args: args{
				e:                    Expression("some-expression"),
				validateAsExpression: false,
			},
		},
		{
			name: "complex expression with jsonpath",
			args: args{
				e:                    Expression("$sourceDescriptions.museum-api.url#/paths/~1special-events~1{eventId}/get"),
				validateAsExpression: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.e.Validate(tt.args.validateAsExpression)
			require.NoError(t, err)
		})
	}
}

func TestExpression_Validate_Failure(t *testing.T) {
	type args struct {
		e Expression
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "invalid expression",
			args: args{
				e: Expression("some-expression"),
			},
			wantErr: errors.New("expression is not valid, must begin with $: some-expression"),
		},
		{
			name: "empty expression",
			args: args{
				e: Expression("$"),
			},
			wantErr: errors.New("expression is not valid, must begin with one of [url, method, statusCode, request, response, inputs, outputs, steps, workflows, sourceDescriptions, components]: $"),
		},
		{
			name: "expression not recognized",
			args: args{
				e: Expression("$some.expression"),
			},
			wantErr: errors.New("expression is not valid, must begin with one of [url, method, statusCode, request, response, inputs, outputs, steps, workflows, sourceDescriptions, components]: $some.expression"),
		},
		{
			name: "missing header token",
			args: args{
				e: Expression("$request.header"),
			},
			wantErr: errors.New("expression is not valid, expected token after $request.header: $request.header"),
		},
		{
			name: "invalid header token",
			args: args{
				e: Expression("$request.header.some@header"),
			},
			wantErr: errors.New("header reference must be a valid token [^[!#$%&'*+\\-.^_`|~\\dA-Za-z]+$]: $request.header.some@header"),
		},
		{
			name: "invalid name",
			args: args{
				e: Expression("$workflows.somé-name"),
			},
			wantErr: errors.New("name reference must be a valid name [^[\x01-\x7f]+$]: $workflows.somé-name"),
		},
		{
			name: "invalid body reference",
			args: args{
				e: Expression("$request.body.something"),
			},
			wantErr: errors.New("expression is not valid, only json pointers are allowed after $request.body: $request.body.something"),
		},
		{
			name: "invalid body json pointer",
			args: args{
				e: Expression("$request.body#some/path"),
			},
			wantErr: errors.New("validation error -- jsonpointer must start with /: some/path"),
		},
		{
			name: "additional characters after simple expression",
			args: args{
				e: Expression("$url.something"),
			},
			wantErr: errors.New("expression is not valid, extra characters after $url: $url.something"),
		},
		{
			name: "invalid source expression",
			args: args{
				e: Expression("$response"),
			},
			wantErr: errors.New("expression is not valid, expected one of [header, query, path, body] after $response: $response"),
		},
		{
			name: "invalid source expression with unknown reference type",
			args: args{
				e: Expression("$request.something"),
			},
			wantErr: errors.New("expression is not valid, expected one of [header, query, path, body] after $request: $request.something"),
		},
		{
			name: "invalid query expression missing name",
			args: args{
				e: Expression("$request.query"),
			},
			wantErr: errors.New("expression is not valid, expected name after $request.query: $request.query"),
		},
		{
			name: "invalid query expression with invalid name",
			args: args{
				e: Expression("$request.query.somé-name"),
			},
			wantErr: errors.New("query reference must be a valid name [^[\x01-\x7f]+$]: $request.query.somé-name"),
		},
		{
			name: "invalid path expression missing name",
			args: args{
				e: Expression("$request.path"),
			},
			wantErr: errors.New("expression is not valid, expected name after $request.path: $request.path"),
		},
		{
			name: "invalid path expression with invalid name",
			args: args{
				e: Expression("$request.path.somé-name"),
			},
			wantErr: errors.New("path reference must be a valid name [^[\x01-\x7f]+$]: $request.path.somé-name"),
		},
		{
			name: "invalid input expression missing name",
			args: args{
				e: Expression("$inputs"),
			},
			wantErr: errors.New("expression is not valid, expected name after $inputs: $inputs"),
		},
		{
			name: "invalid json pointer expression in context",
			args: args{
				e: Expression("$outputs.someOutput#/some/path"),
			},
			wantErr: errors.New("expression is not valid, json pointers are not allowed in current context: $outputs.someOutput#/some/path"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.e.Validate(true)
			assert.EqualError(t, err, tt.wantErr.Error())
		})
	}
}

func TestExpression_IsExpression(t *testing.T) {
	type args struct {
		e Expression
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "simple expression",
			args: args{
				e: Expression("$url"),
			},
			want: true,
		},
		{
			name: "expression with json pointer",
			args: args{
				e: Expression("$request.body#/some/path"),
			},
			want: true,
		},
		{
			name: "expression with json pointer enclosed in {}",
			args: args{
				e: Expression("{$request.body}#/some/path"),
			},
			want: true,
		},
		{
			name: "multiple expressions in string",
			args: args{
				e: Expression(`{$inputs.pet_id}#/some/json/pointer {$inputs.coupon_code}{$inputs.quantity}`),
			},
			want: false,
		},
		{
			name: "not a valid expression",
			args: args{
				e: Expression("Bearer {$inputs.token}"),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.args.e.IsExpression()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExpression_GetType(t *testing.T) {
	e := Expression("$request.body#/some/path")
	assert.Equal(t, ExpressionTypeRequest, e.GetType())
}

func TestExpression_GetJSONPointer(t *testing.T) {
	e := Expression("$request.body#/some/path")
	assert.Equal(t, jsonpointer.JSONPointer("/some/path"), e.GetJSONPointer())

	e = Expression("$request.body")
	assert.Empty(t, e.GetJSONPointer())
}
