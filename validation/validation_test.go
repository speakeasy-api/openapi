package validation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Test the Error type methods
func TestError_Error_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name: "error with valid node",
			err: &Error{
				UnderlyingError: errors.New("test error"),
				Severity:        SeverityError,
				Rule:            RuleValidationTypeMismatch,
				Node: &yaml.Node{
					Line:   10,
					Column: 5,
				},
			},
			expected: "[10:5] error validation-type-mismatch test error",
		},
		{
			name: "error with nil node",
			err: &Error{
				UnderlyingError: errors.New("test error"),
				Severity:        SeverityWarning,
				Rule:            RuleValidationInvalidFormat,
				Node:            nil,
			},
			expected: "[-1:-1] warning validation-invalid-format test error",
		},
		{
			name: "error with zero line/column",
			err: &Error{
				UnderlyingError: errors.New("test error"),
				Severity:        SeverityError,
				Rule:            RuleValidationRequiredField,
				Node: &yaml.Node{
					Line:   0,
					Column: 0,
				},
			},
			expected: "[0:0] error validation-required-field test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestError_Unwrap_Success(t *testing.T) {
	t.Parallel()

	underlyingErr := errors.New("underlying error")
	err := &Error{
		UnderlyingError: underlyingErr,
		Node:            &yaml.Node{Line: 1, Column: 1},
	}

	result := err.Unwrap()
	assert.Equal(t, underlyingErr, result)
}

func TestError_GetLineNumber_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *Error
		expected int
	}{
		{
			name: "valid node with line number",
			err: &Error{
				Node: &yaml.Node{Line: 42, Column: 10},
			},
			expected: 42,
		},
		{
			name: "nil node returns -1",
			err: &Error{
				Node: nil,
			},
			expected: -1,
		},
		{
			name: "zero line number",
			err: &Error{
				Node: &yaml.Node{Line: 0, Column: 5},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.err.GetLineNumber()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestError_GetColumnNumber_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *Error
		expected int
	}{
		{
			name: "valid node with column number",
			err: &Error{
				Node: &yaml.Node{Line: 10, Column: 25},
			},
			expected: 25,
		},
		{
			name: "nil node returns -1",
			err: &Error{
				Node: nil,
			},
			expected: -1,
		},
		{
			name: "zero column number",
			err: &Error{
				Node: &yaml.Node{Line: 5, Column: 0},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.err.GetColumnNumber()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test NewValidationError function
func TestNewValidationError_Success(t *testing.T) {
	t.Parallel()

	underlyingErr := errors.New("test error")
	node := &yaml.Node{Line: 5, Column: 10}

	result := NewValidationError(SeverityError, RuleValidationTypeMismatch, underlyingErr, node)

	var validationErr *Error
	require.ErrorAs(t, result, &validationErr, "should return *Error type")
	assert.Equal(t, underlyingErr, validationErr.UnderlyingError)
	assert.Equal(t, node, validationErr.Node)
	assert.Equal(t, SeverityError, validationErr.Severity)
	assert.Equal(t, RuleValidationTypeMismatch, validationErr.Rule)
}

// Mock types for testing the error creation functions
type mockCoreModeler struct {
	rootNode *yaml.Node
}

func (m *mockCoreModeler) GetRootNode() *yaml.Node {
	return m.rootNode
}

type mockValueNodeGetter struct {
	valueNode *yaml.Node
}

func (m *mockValueNodeGetter) GetValueNodeOrRoot(root *yaml.Node) *yaml.Node {
	if m.valueNode != nil {
		return m.valueNode
	}
	return root
}

type mockSliceNodeGetter struct {
	valueNode *yaml.Node
}

func (m *mockSliceNodeGetter) GetSliceValueNodeOrRoot(index int, root *yaml.Node) *yaml.Node {
	if m.valueNode != nil {
		return m.valueNode
	}
	return root
}

type mockMapKeyNodeGetter struct {
	keyNode *yaml.Node
}

func (m *mockMapKeyNodeGetter) GetMapKeyNodeOrRoot(key string, root *yaml.Node) *yaml.Node {
	if m.keyNode != nil {
		return m.keyNode
	}
	return root
}

type mockMapValueNodeGetter struct {
	valueNode *yaml.Node
}

func (m *mockMapValueNodeGetter) GetMapValueNodeOrRoot(key string, root *yaml.Node) *yaml.Node {
	if m.valueNode != nil {
		return m.valueNode
	}
	return root
}

// Test NewValueError function
func TestNewValueError_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		core         CoreModeler
		nodeGetter   ValueNodeGetter
		expectedNode *yaml.Node
	}{
		{
			name: "with valid root node and value node",
			core: &mockCoreModeler{
				rootNode: &yaml.Node{Line: 1, Column: 1},
			},
			nodeGetter: &mockValueNodeGetter{
				valueNode: &yaml.Node{Line: 5, Column: 10},
			},
			expectedNode: &yaml.Node{Line: 5, Column: 10},
		},
		{
			name: "with nil root node",
			core: &mockCoreModeler{
				rootNode: nil,
			},
			nodeGetter: &mockValueNodeGetter{
				valueNode: &yaml.Node{Line: 5, Column: 10},
			},
			expectedNode: nil,
		},
		{
			name: "with root node but no specific value node",
			core: &mockCoreModeler{
				rootNode: &yaml.Node{Line: 1, Column: 1},
			},
			nodeGetter: &mockValueNodeGetter{
				valueNode: nil, // Will return root node
			},
			expectedNode: &yaml.Node{Line: 1, Column: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			underlyingErr := errors.New("test error")
			result := NewValueError(SeverityError, RuleValidationTypeMismatch, underlyingErr, tt.core, tt.nodeGetter)

			var validationErr *Error
			require.ErrorAs(t, result, &validationErr, "should return *Error type")
			assert.Equal(t, underlyingErr, validationErr.UnderlyingError)
			assert.Equal(t, tt.expectedNode, validationErr.Node)
			assert.Equal(t, SeverityError, validationErr.Severity)
			assert.Equal(t, RuleValidationTypeMismatch, validationErr.Rule)
		})
	}
}

// Test NewSliceError function
func TestNewSliceError_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		core         CoreModeler
		nodeGetter   SliceNodeGetter
		index        int
		expectedNode *yaml.Node
	}{
		{
			name: "with valid root node and slice node",
			core: &mockCoreModeler{
				rootNode: &yaml.Node{Line: 1, Column: 1},
			},
			nodeGetter: &mockSliceNodeGetter{
				valueNode: &yaml.Node{Line: 3, Column: 5},
			},
			index:        2,
			expectedNode: &yaml.Node{Line: 3, Column: 5},
		},
		{
			name: "with nil root node",
			core: &mockCoreModeler{
				rootNode: nil,
			},
			nodeGetter: &mockSliceNodeGetter{
				valueNode: &yaml.Node{Line: 3, Column: 5},
			},
			index:        0,
			expectedNode: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			underlyingErr := errors.New("slice error")
			result := NewSliceError(SeverityError, RuleValidationTypeMismatch, underlyingErr, tt.core, tt.nodeGetter, tt.index)

			var validationErr *Error
			require.ErrorAs(t, result, &validationErr, "should return *Error type")
			assert.Equal(t, underlyingErr, validationErr.UnderlyingError)
			assert.Equal(t, tt.expectedNode, validationErr.Node)
			assert.Equal(t, SeverityError, validationErr.Severity)
			assert.Equal(t, RuleValidationTypeMismatch, validationErr.Rule)
		})
	}
}

// Test NewMapKeyError function
func TestNewMapKeyError_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		core         CoreModeler
		nodeGetter   MapKeyNodeGetter
		key          string
		expectedNode *yaml.Node
	}{
		{
			name: "with valid root node and key node",
			core: &mockCoreModeler{
				rootNode: &yaml.Node{Line: 1, Column: 1},
			},
			nodeGetter: &mockMapKeyNodeGetter{
				keyNode: &yaml.Node{Line: 7, Column: 3},
			},
			key:          "testKey",
			expectedNode: &yaml.Node{Line: 7, Column: 3},
		},
		{
			name: "with nil root node",
			core: &mockCoreModeler{
				rootNode: nil,
			},
			nodeGetter: &mockMapKeyNodeGetter{
				keyNode: &yaml.Node{Line: 7, Column: 3},
			},
			key:          "testKey",
			expectedNode: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			underlyingErr := errors.New("map key error")
			result := NewMapKeyError(SeverityError, RuleValidationTypeMismatch, underlyingErr, tt.core, tt.nodeGetter, tt.key)

			var validationErr *Error
			require.ErrorAs(t, result, &validationErr, "should return *Error type")
			assert.Equal(t, underlyingErr, validationErr.UnderlyingError)
			assert.Equal(t, tt.expectedNode, validationErr.Node)
			assert.Equal(t, SeverityError, validationErr.Severity)
			assert.Equal(t, RuleValidationTypeMismatch, validationErr.Rule)
		})
	}
}

// Test NewMapValueError function
func TestNewMapValueError_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		core         CoreModeler
		nodeGetter   MapValueNodeGetter
		key          string
		expectedNode *yaml.Node
	}{
		{
			name: "with valid root node and value node",
			core: &mockCoreModeler{
				rootNode: &yaml.Node{Line: 1, Column: 1},
			},
			nodeGetter: &mockMapValueNodeGetter{
				valueNode: &yaml.Node{Line: 8, Column: 12},
			},
			key:          "testKey",
			expectedNode: &yaml.Node{Line: 8, Column: 12},
		},
		{
			name: "with nil root node",
			core: &mockCoreModeler{
				rootNode: nil,
			},
			nodeGetter: &mockMapValueNodeGetter{
				valueNode: &yaml.Node{Line: 8, Column: 12},
			},
			key:          "testKey",
			expectedNode: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			underlyingErr := errors.New("map value error")
			result := NewMapValueError(SeverityError, RuleValidationTypeMismatch, underlyingErr, tt.core, tt.nodeGetter, tt.key)

			var validationErr *Error
			require.ErrorAs(t, result, &validationErr, "should return *Error type")
			assert.Equal(t, underlyingErr, validationErr.UnderlyingError)
			assert.Equal(t, tt.expectedNode, validationErr.Node)
		})
	}
}

// Test TypeMismatchError
func TestTypeMismatchError_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		msg      string
		args     []any
		expected string
	}{
		{
			name:     "simple message without args",
			msg:      "type mismatch",
			args:     nil,
			expected: "type mismatch",
		},
		{
			name:     "message with formatting args",
			msg:      "expected %s, got %s",
			args:     []any{"string", "int"},
			expected: "expected string, got int",
		},
		{
			name:     "message with multiple args",
			msg:      "field %s at line %d has wrong type",
			args:     []any{"name", 42},
			expected: "field name at line 42 has wrong type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := NewTypeMismatchError("", tt.msg, tt.args...)
			assert.Equal(t, tt.expected, err.Error())
			assert.Equal(t, tt.expected, err.Msg)
		})
	}
}

// Test Severity.String() method
func TestSeverity_String_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		severity Severity
		expected string
	}{
		{
			name:     "error severity",
			severity: SeverityError,
			expected: "error",
		},
		{
			name:     "warning severity",
			severity: SeverityWarning,
			expected: "warning",
		},
		{
			name:     "hint severity",
			severity: SeverityHint,
			expected: "hint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.severity.String()
			assert.Equal(t, tt.expected, result, "severity string should match")
		})
	}
}

// Test Severity.Rank() method
func TestSeverity_Rank_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		severity Severity
		expected int
	}{
		{
			name:     "error severity has rank 2",
			severity: SeverityError,
			expected: 2,
		},
		{
			name:     "warning severity has rank 1",
			severity: SeverityWarning,
			expected: 1,
		},
		{
			name:     "hint severity has rank 0",
			severity: SeverityHint,
			expected: 0,
		},
		{
			name:     "unknown severity treated as error",
			severity: Severity("unknown"),
			expected: 2,
		},
		{
			name:     "empty severity treated as error",
			severity: Severity(""),
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.severity.Rank()
			assert.Equal(t, tt.expected, result, "severity rank should match")
		})
	}
}

// Test Severity.Rank() ordering for comparison
func TestSeverity_Rank_Ordering(t *testing.T) {
	t.Parallel()

	// Verify that error > warning > hint in terms of rank (worse severity = higher rank)
	assert.Greater(t, SeverityError.Rank(), SeverityWarning.Rank(), "error should have higher rank than warning")
	assert.Greater(t, SeverityWarning.Rank(), SeverityHint.Rank(), "warning should have higher rank than hint")
	assert.Greater(t, SeverityError.Rank(), SeverityHint.Rank(), "error should have higher rank than hint")
}

// Test Error.GetSeverity() method
func TestError_GetSeverity_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *Error
		expected Severity
	}{
		{
			name: "error severity",
			err: &Error{
				UnderlyingError: errors.New("test error"),
				Severity:        SeverityError,
			},
			expected: SeverityError,
		},
		{
			name: "warning severity",
			err: &Error{
				UnderlyingError: errors.New("test warning"),
				Severity:        SeverityWarning,
			},
			expected: SeverityWarning,
		},
		{
			name: "hint severity",
			err: &Error{
				UnderlyingError: errors.New("test hint"),
				Severity:        SeverityHint,
			},
			expected: SeverityHint,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.err.GetSeverity()
			assert.Equal(t, tt.expected, result, "severity should match")
		})
	}
}

// Test TypeMismatchError with ParentName
func TestTypeMismatchError_WithParentName_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		parentName string
		msg        string
		args       []any
		expected   string
	}{
		{
			name:       "with parent name",
			parentName: "Response",
			msg:        "type mismatch",
			args:       nil,
			expected:   "Response type mismatch",
		},
		{
			name:       "with parent name and formatting",
			parentName: "Schema",
			msg:        "expected %s, got %s",
			args:       []any{"string", "int"},
			expected:   "Schema expected string, got int",
		},
		{
			name:       "empty parent name",
			parentName: "",
			msg:        "standalone error",
			args:       nil,
			expected:   "standalone error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := NewTypeMismatchError(tt.parentName, tt.msg, tt.args...)
			assert.Equal(t, tt.expected, err.Error(), "error message should match")
		})
	}
}
