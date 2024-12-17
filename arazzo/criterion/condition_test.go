package criterion

import (
	"fmt"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo/expression"
	"github.com/stretchr/testify/assert"
)

func TestNewCondition(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		raw           string
		expected      *Condition
		expectedError error
	}{
		"empty string": {
			raw:      "",
			expected: nil,
		},
		"expression only": {
			raw:           "$statusCode",
			expected:      nil,
			expectedError: fmt.Errorf("condition must at least be in the format [expression] [operator] [value]"),
		},
		"expression and operator only": {
			raw:           "$statusCode ==",
			expected:      nil,
			expectedError: fmt.Errorf("condition must at least be in the format [expression] [operator] [value]"),
		},
		"$statusCode == 200": {
			raw: "$statusCode == 200",
			expected: &Condition{
				Expression: expression.Expression("$statusCode"),
				Operator:   OperatorEQ,
				Value:      "200",
			},
		},
		"$response.body#/test == 'string literal with spaces'": {
			raw: "$response.body#/test == 'string literal with spaces'",
			expected: &Condition{
				Expression: expression.Expression("$response.body#/test"),
				Operator:   OperatorEQ,
				Value:      "'string literal with spaces'",
			},
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			actual, actualError := newCondition(testCase.raw)

			if testCase.expectedError != nil {
				assert.EqualError(t, actualError, testCase.expectedError.Error())
			}

			assert.EqualExportedValues(t, testCase.expected, actual)
		})
	}
}
