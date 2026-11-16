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
	t.Parallel()
	type args struct {
		e Expression
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "$url",
			args: args{
				e: Expression("$url"),
			},
		},
		{
			name: "{$url}",
			args: args{
				e: Expression("{$url}"),
			},
		},
		{
			name: "$method",
			args: args{
				e: Expression("$method"),
			},
		},
		{
			name: "$statusCode",
			args: args{
				e: Expression("$statusCode"),
			},
		},
		{
			name: "request body without json pointer",
			args: args{
				e: Expression("$request.body"),
			},
		},
		{
			name: "request body enclosed in {} without json pointer",
			args: args{
				e: Expression("{$request.body}"),
			},
		},
		{
			name: "request body with json pointer",
			args: args{
				e: Expression("$request.body#/some/path"),
			},
		},
		{
			name: "request body enclosed in {} with json pointer",
			args: args{
				e: Expression("{$request.body}#/some/path"),
			},
		},
		{
			name: "request header",
			args: args{
				e: Expression("$request.header.some-header"),
			},
		},
		{
			name: "request query",
			args: args{
				e: Expression("$request.query.someQueryParam"),
			},
		},
		{
			name: "request path",
			args: args{
				e: Expression("$request.path.somePathParam"),
			},
		},
		{
			name: "response body",
			args: args{
				e: Expression("$response.body"),
			},
		},
		{
			"response body with json pointer",
			args{
				e: Expression("$response.body#/some/path"),
			},
		},
		{
			name: "response header",
			args: args{
				e: Expression("$response.header.some-header"),
			},
		},
		{
			name: "response header enclose in {}",
			args: args{
				e: Expression("{$response.header.some-header}"),
			},
		},
		{
			name: "inputs",
			args: args{
				e: Expression("$inputs.someInput"),
			},
		},
		{
			name: "outputs",
			args: args{
				e: Expression("$outputs.someOutput"),
			},
		},
		{
			name: "outputs with json pointer",
			args: args{
				e: Expression("$outputs.someOutput#/some/path"),
			},
		},
		{
			name: "steps",
			args: args{
				e: Expression("$steps.someStep"),
			},
		},
		{
			name: "step outputs with json pointer",
			args: args{
				e: Expression("$steps.someStep.outputs.someOutput#/some/path"),
			},
		},
		{
			name: "workflows",
			args: args{
				e: Expression("$workflows.someWorkflow"),
			},
		},
		{
			name: "workflow outputs with json pointer",
			args: args{
				e: Expression("$workflows.someWorkflow.outputs.someOutput#/some/path"),
			},
		},
		{
			name: "source descriptions",
			args: args{
				e: Expression("$sourceDescriptions.someSourceDescription"),
			},
		},
		{
			name: "source descriptions sub path",
			args: args{
				e: Expression("$sourceDescriptions.someSourceDescription.url"),
			},
		},
		{
			name: "source descriptions sub path with json pointer",
			args: args{
				e: Expression("{$sourceDescriptions.petStoreDescription.url}#/paths/~1pet~1findByStatus/get"),
			},
		},
		{
			name: "components",
			args: args{
				e: Expression("$components.parameters.someParameter"),
			},
		},
		{
			name: "complex expression with jsonpath",
			args: args{
				e: Expression("$sourceDescriptions.museum-api.url#/paths/~1special-events~1{eventId}/get"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.args.e.Validate()
			require.NoError(t, err)
		})
	}
}

func TestExpression_Validate_Failure(t *testing.T) {
	t.Parallel()
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
			name: "invalid json pointer expression in inputs expression",
			args: args{
				e: Expression("$inputs.someInput#/some/path"),
			},
			wantErr: errors.New("expression is not valid, json pointers are not allowed in current context: $inputs.someInput#/some/path"),
		},
		{
			name: "invalid json pointer expression in workflow inputs expression",
			args: args{
				e: Expression("$workflows.someWorkflow.inputs.someInput#/some/path"),
			},
			wantErr: errors.New("expression is not valid, json pointers are not allowed in current context: $workflows.someWorkflow.inputs.someInput#/some/path"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.args.e.Validate()
			require.EqualError(t, err, tt.wantErr.Error())
		})
	}
}

func TestExpression_IsExpression(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			got := tt.args.e.IsExpression()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExpression_Validate_Arazzo11_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		e    Expression
	}{
		{name: "$self", e: "$self"},
		{name: "$message.payload", e: "$message.payload"},
		{name: "$message.payload with json pointer", e: "$message.payload#/orderId"},
		{name: "$message.header", e: "$message.header.X-Correlation-Id"},
		{name: "$response.body with json pointer", e: "$response.body#/id"},
		{name: "$inputs with json pointer", e: "$inputs.order#/items/0"},
		{name: "$inputs with dotted identifier", e: "$inputs.order.id"},
		{name: "$outputs with json pointer", e: "$outputs.result#/value"},
		{name: "$steps outputs", e: "$steps.get-pet_1.outputs.pet"},
		{name: "$steps outputs with json pointer", e: "$steps.getPet.outputs.pet#/name"},
		{name: "$workflows inputs", e: "$workflows.login.inputs.username"},
		{name: "$workflows outputs with json pointer", e: "$workflows.login.outputs.token#/value"},
		{name: "$sourceDescriptions operation", e: "$sourceDescriptions.petStore.findPetsByStatus"},
		{name: "$sourceDescriptions url with json pointer", e: "{$sourceDescriptions.petStore.url}#/paths/~1pets/get"},
		{name: "$components parameters", e: "$components.parameters.page"},
		{name: "$components successActions", e: "$components.successActions.done"},
		{name: "$components failureActions", e: "$components.failureActions.retry"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, tt.e.Validate(WithArazzo11Grammar()), "expression should be valid in the Arazzo 1.1 grammar")
		})
	}
}

func TestExpression_Validate_Arazzo11_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		e       Expression
		wantErr string
	}{
		{name: "$self with extra parts", e: "$self.foo", wantErr: "expression is not valid, extra characters after $self: $self.foo"},
		{name: "$message dotted payload", e: "$message.payload.orderId", wantErr: "expression is not valid, only json pointers are allowed after $message.payload: $message.payload.orderId"},
		{name: "$steps without outputs", e: "$steps.getPet.pet", wantErr: "expression is not valid, expected $steps.<stepId>.outputs.<name>: $steps.getPet.pet"},
		{name: "$steps with no output name", e: "$steps.getPet.outputs", wantErr: "expression is not valid, expected $steps.<stepId>.outputs.<name>: $steps.getPet.outputs"},
		{name: "$workflows with unknown field", e: "$workflows.login.steps.s1", wantErr: "expression is not valid, expected $workflows.<workflowId>.inputs.<name> or $workflows.<workflowId>.outputs.<name>: $workflows.login.steps.s1"},
		{name: "$sourceDescriptions without reference", e: "$sourceDescriptions.petStore", wantErr: "expression is not valid, expected $sourceDescriptions.<name>.<reference>: $sourceDescriptions.petStore"},
		{name: "$components inputs", e: "$components.inputs.pet", wantErr: "expression is not valid, expected $components.<parameters|successActions|failureActions>.<name>: $components.inputs.pet"},
		{name: "$inputs with invalid characters", e: "$inputs.pet!id", wantErr: "expression is not valid, expected a name matching [^[A-Za-z0-9._\\-]+$] after $inputs: $inputs.pet!id"},
		{name: "json pointer on $statusCode", e: "$statusCode#/foo", wantErr: "expression is not valid, json pointers are not allowed in current context: $statusCode#/foo"},
		{name: "unknown root", e: "$foo.bar", wantErr: "expression is not valid, must begin with one of [url, method, statusCode, request, response, inputs, outputs, steps, workflows, sourceDescriptions, components, message, self]: $foo.bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.e.Validate(WithArazzo11Grammar())
			require.EqualError(t, err, tt.wantErr, "expression should be invalid in the Arazzo 1.1 grammar")
		})
	}
}

func TestExpression_Validate_Arazzo11OnlyRoots_ErrorWithoutOption(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		e    Expression
	}{
		{name: "$self", e: "$self"},
		{name: "$message", e: "$message.payload"},
		{name: "payload source", e: "$response.payload"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tt.e.Validate(), "Arazzo 1.1 only expressions should be invalid without the Arazzo 1.1 grammar")
		})
	}
}

func TestExpression_IsExpression_Arazzo11(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		e            Expression
		wantDefault  bool
		wantArazzo11 bool
	}{
		{name: "$self", e: "$self", wantDefault: false, wantArazzo11: true},
		{name: "$message", e: "$message.payload#/id", wantDefault: false, wantArazzo11: true},
		{name: "$url", e: "$url", wantDefault: true, wantArazzo11: true},
		{name: "literal", e: "not an expression", wantDefault: false, wantArazzo11: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantDefault, tt.e.IsExpression(), "default grammar recognition should match")
			assert.Equal(t, tt.wantArazzo11, tt.e.IsExpression(WithArazzo11Grammar()), "Arazzo 1.1 grammar recognition should match")
		})
	}
}

func TestExpression_GetType(t *testing.T) {
	t.Parallel()
	e := Expression("$request.body#/some/path")
	assert.Equal(t, ExpressionTypeRequest, e.GetType())
}

func TestExpression_GetJSONPointer(t *testing.T) {
	t.Parallel()
	e := Expression("$request.body#/some/path")
	assert.Equal(t, jsonpointer.JSONPointer("/some/path"), e.GetJSONPointer())

	e = Expression("$request.body")
	assert.Empty(t, e.GetJSONPointer())
}
