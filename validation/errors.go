package validation

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

// Error represents a validation error and the line and column where it occurred
// TODO allow getting the JSON path for line/column for validation errors
type Error struct {
	UnderlyingError error
	Node            *yaml.Node
}

var _ error = (*Error)(nil)

func (e Error) Error() string {
	return fmt.Sprintf("[%d:%d] %s", e.GetLineNumber(), e.GetColumnNumber(), e.UnderlyingError.Error())
}

func (e Error) Unwrap() error {
	return e.UnderlyingError
}

func (e Error) GetLineNumber() int {
	if e.Node == nil {
		return -1
	}
	return e.Node.Line
}

func (e Error) GetColumnNumber() int {
	if e.Node == nil {
		return -1
	}
	return e.Node.Column
}

type valueNodeGetter interface {
	GetValueNodeOrRoot(root *yaml.Node) *yaml.Node
}

type sliceNodeGetter interface {
	GetSliceValueNodeOrRoot(index int, root *yaml.Node) *yaml.Node
}

type mapKeyNodeGetter interface {
	GetMapKeyNodeOrRoot(key string, root *yaml.Node) *yaml.Node
}

type mapValueNodeGetter interface {
	GetMapValueNodeOrRoot(key string, root *yaml.Node) *yaml.Node
}

func NewValidationError(err error, node *yaml.Node) error {
	return &Error{
		UnderlyingError: err,
		Node:            node,
	}
}

type CoreModeler interface {
	GetRootNode() *yaml.Node
}

func NewValueError(err error, core CoreModeler, node valueNodeGetter) error {
	rootNode := core.GetRootNode()

	if rootNode == nil {
		// Fallback if RootNode is not available
		return &Error{
			UnderlyingError: err,
			// Default to line 0, column 0 if we can't get location info
		}
	}
	valueNode := node.GetValueNodeOrRoot(rootNode)

	return &Error{
		UnderlyingError: err,
		Node:            valueNode,
	}
}

func NewSliceError(err error, core CoreModeler, node sliceNodeGetter, index int) error {
	rootNode := core.GetRootNode()

	if rootNode == nil {
		// Fallback if RootNode is not available
		return &Error{
			UnderlyingError: err,
			// Default to line 0, column 0 if we can't get location info
		}
	}
	valueNode := node.GetSliceValueNodeOrRoot(index, rootNode)

	return &Error{
		UnderlyingError: err,
		Node:            valueNode,
	}
}

func NewMapKeyError(err error, core CoreModeler, node mapKeyNodeGetter, key string) error {
	rootNode := core.GetRootNode()

	if rootNode == nil {
		// Fallback if RootNode is not available
		return &Error{
			UnderlyingError: err,
			// Default to line 0, column 0 if we can't get location info
		}
	}
	valueNode := node.GetMapKeyNodeOrRoot(key, rootNode)

	return &Error{
		UnderlyingError: err,
		Node:            valueNode,
	}
}

func NewMapValueError(err error, core CoreModeler, node mapValueNodeGetter, key string) error {
	rootNode := core.GetRootNode()

	if rootNode == nil {
		// Fallback if RootNode is not available
		return &Error{
			UnderlyingError: err,
			// Default to line 0, column 0 if we can't get location info
		}
	}
	valueNode := node.GetMapValueNodeOrRoot(key, rootNode)

	return &Error{
		UnderlyingError: err,
		Node:            valueNode,
	}
}

type TypeMismatchError struct {
	Msg string
}

var _ error = (*TypeMismatchError)(nil)

func NewTypeMismatchError(msg string, args ...any) *TypeMismatchError {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	return &TypeMismatchError{
		Msg: msg,
	}
}

func (e TypeMismatchError) Error() string {
	return e.Msg
}

type MissingFieldError struct {
	Msg string
}

var _ error = (*MissingFieldError)(nil)

func NewMissingFieldError(msg string, args ...any) *MissingFieldError {
	return &MissingFieldError{
		Msg: fmt.Sprintf(msg, args...),
	}
}

func (e MissingFieldError) Error() string {
	return e.Msg
}

type MissingValueError struct {
	Msg string
}

var _ error = (*MissingValueError)(nil)

func NewMissingValueError(msg string, args ...any) *MissingValueError {
	return &MissingValueError{
		Msg: fmt.Sprintf(msg, args...),
	}
}

func (e MissingValueError) Error() string {
	return e.Msg
}

type ValueValidationError struct {
	Msg string
}

var _ error = (*ValueValidationError)(nil)

func NewValueValidationError(msg string, args ...any) *ValueValidationError {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	return &ValueValidationError{
		Msg: msg,
	}
}

func (e ValueValidationError) Error() string {
	return e.Msg
}
