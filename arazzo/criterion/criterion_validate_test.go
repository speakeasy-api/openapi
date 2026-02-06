package criterion_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCriterionExpressionType_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cet  *criterion.CriterionExpressionType
	}{
		{
			name: "valid jsonpath with correct version",
			cet: &criterion.CriterionExpressionType{
				Type:    criterion.CriterionTypeJsonPath,
				Version: criterion.CriterionTypeVersionDraftGoessnerDispatchJsonPath00,
			},
		},
		{
			name: "valid xpath with version 3.0",
			cet: &criterion.CriterionExpressionType{
				Type:    criterion.CriterionTypeXPath,
				Version: criterion.CriterionTypeVersionXPath30,
			},
		},
		{
			name: "valid xpath with version 2.0",
			cet: &criterion.CriterionExpressionType{
				Type:    criterion.CriterionTypeXPath,
				Version: criterion.CriterionTypeVersionXPath20,
			},
		},
		{
			name: "valid xpath with version 1.0",
			cet: &criterion.CriterionExpressionType{
				Type:    criterion.CriterionTypeXPath,
				Version: criterion.CriterionTypeVersionXPath10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			errs := tt.cet.Validate()
			assert.Empty(t, errs, "validation should succeed")
			assert.True(t, tt.cet.Valid, "criterion expression type should be valid")
		})
	}
}

func TestCriterionExpressionType_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		cet           *criterion.CriterionExpressionType
		expectedError string
	}{
		{
			name: "invalid jsonpath version",
			cet: &criterion.CriterionExpressionType{
				Type:    criterion.CriterionTypeJsonPath,
				Version: "invalid-version",
			},
			expectedError: "version must be one of [`draft-goessner-dispatch-jsonpath-00`]",
		},
		{
			name: "invalid xpath version",
			cet: &criterion.CriterionExpressionType{
				Type:    criterion.CriterionTypeXPath,
				Version: "invalid-version",
			},
			expectedError: "version must be one of [`xpath-30, xpath-20, xpath-10`]",
		},
		{
			name: "invalid type",
			cet: &criterion.CriterionExpressionType{
				Type:    "invalid-type",
				Version: criterion.CriterionTypeVersionNone,
			},
			expectedError: "type must be one of [`jsonpath, xpath`]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			errs := tt.cet.Validate()
			require.NotEmpty(t, errs, "validation should fail")
			assert.Contains(t, errs[0].Error(), tt.expectedError, "error message should match")
			assert.False(t, tt.cet.Valid, "criterion expression type should not be valid")
		})
	}
}

func TestCriterionExpressionType_IsTypeProvided(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cet      *criterion.CriterionExpressionType
		expected bool
	}{
		{
			name:     "nil criterion expression type",
			cet:      nil,
			expected: false,
		},
		{
			name: "empty type",
			cet: &criterion.CriterionExpressionType{
				Type: "",
			},
			expected: false,
		},
		{
			name: "type provided",
			cet: &criterion.CriterionExpressionType{
				Type: criterion.CriterionTypeJsonPath,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.cet.IsTypeProvided()
			assert.Equal(t, tt.expected, result, "IsTypeProvided should return expected value")
		})
	}
}

func TestCriterionTypeUnion_GetCore(t *testing.T) {
	t.Parallel()

	ctu := &criterion.CriterionTypeUnion{
		Type: pointer.From(criterion.CriterionTypeSimple),
	}

	core := ctu.GetCore()
	assert.NotNil(t, core, "GetCore should return non-nil value")
}

func TestCriterionTypeUnion_IsTypeProvided(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ctu      *criterion.CriterionTypeUnion
		expected bool
	}{
		{
			name:     "nil criterion type union",
			ctu:      nil,
			expected: false,
		},
		{
			name:     "empty criterion type union",
			ctu:      &criterion.CriterionTypeUnion{},
			expected: false,
		},
		{
			name: "type provided as string",
			ctu: &criterion.CriterionTypeUnion{
				Type: pointer.From(criterion.CriterionTypeSimple),
			},
			expected: true,
		},
		{
			name: "type provided as expression type",
			ctu: &criterion.CriterionTypeUnion{
				ExpressionType: &criterion.CriterionExpressionType{
					Type: criterion.CriterionTypeJsonPath,
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.ctu.IsTypeProvided()
			assert.Equal(t, tt.expected, result, "IsTypeProvided should return expected value")
		})
	}
}

func TestCriterionTypeUnion_GetType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ctu      criterion.CriterionTypeUnion
		expected criterion.CriterionType
	}{
		{
			name:     "empty criterion type union returns simple",
			ctu:      criterion.CriterionTypeUnion{},
			expected: criterion.CriterionTypeSimple,
		},
		{
			name: "type provided as string",
			ctu: criterion.CriterionTypeUnion{
				Type: pointer.From(criterion.CriterionTypeRegex),
			},
			expected: criterion.CriterionTypeRegex,
		},
		{
			name: "type provided as expression type",
			ctu: criterion.CriterionTypeUnion{
				ExpressionType: &criterion.CriterionExpressionType{
					Type: criterion.CriterionTypeJsonPath,
				},
			},
			expected: criterion.CriterionTypeJsonPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.ctu.GetType()
			assert.Equal(t, tt.expected, result, "GetType should return expected value")
		})
	}
}

func TestCriterionTypeUnion_GetVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ctu      criterion.CriterionTypeUnion
		expected criterion.CriterionTypeVersion
	}{
		{
			name:     "empty criterion type union returns none",
			ctu:      criterion.CriterionTypeUnion{},
			expected: criterion.CriterionTypeVersionNone,
		},
		{
			name: "type provided as string returns none",
			ctu: criterion.CriterionTypeUnion{
				Type: pointer.From(criterion.CriterionTypeRegex),
			},
			expected: criterion.CriterionTypeVersionNone,
		},
		{
			name: "type provided as expression type with version",
			ctu: criterion.CriterionTypeUnion{
				ExpressionType: &criterion.CriterionExpressionType{
					Type:    criterion.CriterionTypeJsonPath,
					Version: criterion.CriterionTypeVersionDraftGoessnerDispatchJsonPath00,
				},
			},
			expected: criterion.CriterionTypeVersionDraftGoessnerDispatchJsonPath00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.ctu.GetVersion()
			assert.Equal(t, tt.expected, result, "GetVersion should return expected value")
		})
	}
}

func TestCriterion_GetCondition_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		criterion         *criterion.Criterion
		expectedCondition *criterion.Condition
	}{
		{
			name: "valid simple condition",
			criterion: &criterion.Criterion{
				Condition: "$statusCode == 200",
			},
			expectedCondition: &criterion.Condition{
				Expression: expression.Expression("$statusCode"),
				Operator:   criterion.OperatorEQ,
				Value:      "200",
			},
		},
		{
			name: "raw value returns nil",
			criterion: &criterion.Criterion{
				Condition: "some raw value",
			},
			expectedCondition: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.criterion.Sync(t.Context())
			require.NoError(t, err, "sync should succeed")

			cond, err := tt.criterion.GetCondition()
			require.NoError(t, err, "GetCondition should succeed")

			if tt.expectedCondition == nil {
				assert.Nil(t, cond, "condition should be nil")
			} else {
				require.NotNil(t, cond, "condition should not be nil")
				assert.Equal(t, tt.expectedCondition.Expression, cond.Expression, "expression should match")
				assert.Equal(t, tt.expectedCondition.Operator, cond.Operator, "operator should match")
				assert.Equal(t, tt.expectedCondition.Value, cond.Value, "value should match")
			}
		})
	}
}

func TestCriterion_Validate_WithTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		criterion *criterion.Criterion
		wantError bool
	}{
		{
			name: "valid simple type with context",
			criterion: &criterion.Criterion{
				Context:   pointer.From(expression.Expression("$response.body")),
				Condition: "$statusCode == 200",
				Type: criterion.CriterionTypeUnion{
					Type: pointer.From(criterion.CriterionTypeSimple),
				},
			},
			wantError: false,
		},
		{
			name: "valid simple condition without explicit type",
			criterion: &criterion.Criterion{
				Condition: "$statusCode == 200",
			},
			wantError: false,
		},
		{
			name: "valid regex type",
			criterion: &criterion.Criterion{
				Context:   pointer.From(expression.Expression("$response.body")),
				Condition: "^[a-z]+$",
				Type: criterion.CriterionTypeUnion{
					Type: pointer.From(criterion.CriterionTypeRegex),
				},
			},
			wantError: false,
		},
		{
			name: "invalid regex pattern",
			criterion: &criterion.Criterion{
				Context:   pointer.From(expression.Expression("$response.body")),
				Condition: "[invalid",
				Type: criterion.CriterionTypeUnion{
					Type: pointer.From(criterion.CriterionTypeRegex),
				},
			},
			wantError: true,
		},
		{
			name: "valid jsonpath type",
			criterion: &criterion.Criterion{
				Context:   pointer.From(expression.Expression("$response.body")),
				Condition: "$[?count(@.pets) > 0]",
				Type: criterion.CriterionTypeUnion{
					Type: pointer.From(criterion.CriterionTypeJsonPath),
				},
			},
			wantError: false,
		},
		{
			name: "invalid jsonpath expression",
			criterion: &criterion.Criterion{
				Context:   pointer.From(expression.Expression("$response.body")),
				Condition: "$[invalid jsonpath",
				Type: criterion.CriterionTypeUnion{
					Type: pointer.From(criterion.CriterionTypeJsonPath),
				},
			},
			wantError: true,
		},
		{
			name: "xpath type validation skipped",
			criterion: &criterion.Criterion{
				Context:   pointer.From(expression.Expression("$response.body")),
				Condition: "//book[@category='web']",
				Type: criterion.CriterionTypeUnion{
					Type: pointer.From(criterion.CriterionTypeXPath),
				},
			},
			wantError: false,
		},
		{
			name: "invalid type",
			criterion: &criterion.Criterion{
				Condition: "$statusCode == 200",
				Type: criterion.CriterionTypeUnion{
					Type: pointer.From(criterion.CriterionType("invalid")),
				},
			},
			wantError: true,
		},
		{
			name: "missing context when type is set",
			criterion: &criterion.Criterion{
				Condition: "$statusCode == 200",
				Type: criterion.CriterionTypeUnion{
					Type: pointer.From(criterion.CriterionTypeJsonPath),
				},
			},
			wantError: true,
		},
		{
			name: "invalid context expression",
			criterion: &criterion.Criterion{
				Context:   pointer.From(expression.Expression("invalid_expression")),
				Condition: "$[?count(@.pets) > 0]",
				Type: criterion.CriterionTypeUnion{
					Type: pointer.From(criterion.CriterionTypeJsonPath),
				},
			},
			wantError: true,
		},
		{
			name: "missing condition",
			criterion: &criterion.Criterion{
				Type: criterion.CriterionTypeUnion{
					Type: pointer.From(criterion.CriterionTypeSimple),
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.criterion.Sync(t.Context())
			require.NoError(t, err, "sync should succeed")

			errs := tt.criterion.Validate()
			if tt.wantError {
				assert.NotEmpty(t, errs, "validation should fail")
				assert.False(t, tt.criterion.Valid, "criterion should not be valid")
			} else {
				assert.Empty(t, errs, "validation should succeed")
				assert.True(t, tt.criterion.Valid, "criterion should be valid")
			}
		})
	}
}
