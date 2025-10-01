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
				Node: &yaml.Node{
					Line:   10,
					Column: 5,
				},
			},
			expected: "[10:5] test error",
		},
		{
			name: "error with nil node",
			err: &Error{
				UnderlyingError: errors.New("test error"),
				Node:            nil,
			},
			expected: "[-1:-1] test error",
		},
		{
			name: "error with zero line/column",
			err: &Error{
				UnderlyingError: errors.New("test error"),
				Node: &yaml.Node{
					Line:   0,
					Column: 0,
				},
			},
			expected: "[0:0] test error",
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

	result := NewValidationError(underlyingErr, node)

	var validationErr *Error
	require.ErrorAs(t, result, &validationErr, "should return *Error type")
	assert.Equal(t, underlyingErr, validationErr.UnderlyingError)
	assert.Equal(t, node, validationErr.Node)
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
		nodeGetter   valueNodeGetter
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
			result := NewValueError(underlyingErr, tt.core, tt.nodeGetter)

			var validationErr *Error
			require.ErrorAs(t, result, &validationErr, "should return *Error type")
			assert.Equal(t, underlyingErr, validationErr.UnderlyingError)
			assert.Equal(t, tt.expectedNode, validationErr.Node)
		})
	}
}

// Test NewSliceError function
func TestNewSliceError_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		core         CoreModeler
		nodeGetter   sliceNodeGetter
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
			result := NewSliceError(underlyingErr, tt.core, tt.nodeGetter, tt.index)

			var validationErr *Error
			require.ErrorAs(t, result, &validationErr, "should return *Error type")
			assert.Equal(t, underlyingErr, validationErr.UnderlyingError)
			assert.Equal(t, tt.expectedNode, validationErr.Node)
		})
	}
}

// Test NewMapKeyError function
func TestNewMapKeyError_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		core         CoreModeler
		nodeGetter   mapKeyNodeGetter
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
			result := NewMapKeyError(underlyingErr, tt.core, tt.nodeGetter, tt.key)

			var validationErr *Error
			require.ErrorAs(t, result, &validationErr, "should return *Error type")
			assert.Equal(t, underlyingErr, validationErr.UnderlyingError)
			assert.Equal(t, tt.expectedNode, validationErr.Node)
		})
	}
}

// Test NewMapValueError function
func TestNewMapValueError_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		core         CoreModeler
		nodeGetter   mapValueNodeGetter
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
			result := NewMapValueError(underlyingErr, tt.core, tt.nodeGetter, tt.key)

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

// Test MissingFieldError
func TestMissingFieldError_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		msg      string
		args     []any
		expected string
	}{
		{
			name:     "simple missing field message",
			msg:      "required field missing",
			args:     nil,
			expected: "required field missing",
		},
		{
			name:     "missing field with field name",
			msg:      "required field '%s' is missing",
			args:     []any{"name"},
			expected: "required field 'name' is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := NewMissingFieldError(tt.msg, tt.args...)
			assert.Equal(t, tt.expected, err.Error())
			assert.Equal(t, tt.expected, err.Msg)
		})
	}
}

// Test MissingValueError
func TestMissingValueError_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		msg      string
		args     []any
		expected string
	}{
		{
			name:     "simple missing value message",
			msg:      "value is required",
			args:     nil,
			expected: "value is required",
		},
		{
			name:     "missing value with context",
			msg:      "value for field '%s' is required",
			args:     []any{"description"},
			expected: "value for field 'description' is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := NewMissingValueError(tt.msg, tt.args...)
			assert.Equal(t, tt.expected, err.Error())
			assert.Equal(t, tt.expected, err.Msg)
		})
	}
}

// Test ValueValidationError
func TestValueValidationError_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		msg      string
		args     []any
		expected string
	}{
		{
			name:     "simple validation error",
			msg:      "invalid value",
			args:     nil,
			expected: "invalid value",
		},
		{
			name:     "validation error with formatting",
			msg:      "value '%s' is not valid for field '%s'",
			args:     []any{"invalid", "status"},
			expected: "value 'invalid' is not valid for field 'status'",
		},
		{
			name:     "validation error with no args but formatting placeholders",
			msg:      "value %s is invalid",
			args:     []any{},
			expected: "value %s is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := NewValueValidationError(tt.msg, tt.args...)
			assert.Equal(t, tt.expected, err.Error())
			assert.Equal(t, tt.expected, err.Msg)
		})
	}
}
