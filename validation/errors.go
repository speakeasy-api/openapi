package validation

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Error represents a validation error and the line and column where it occurred
// TODO allow getting the JSON path for line/column for validation errors
type Error struct {
	Line    int
	Column  int
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("[%d:%d] %s", e.Line, e.Column, e.Message)
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

func NewNodeError(msg string, node *yaml.Node) error {
	return &Error{
		Message: msg,
		Line:    node.Line,
		Column:  node.Column,
	}
}

type CoreModeler interface {
	GetRootNode() *yaml.Node
}

func NewValueError(msg string, core CoreModeler, node valueNodeGetter) error {
	rootNode := core.GetRootNode()

	if rootNode == nil {
		// Fallback if RootNode is not available
		return &Error{
			Message: msg,
			// Default to line 0, column 0 if we can't get location info
		}
	}
	valueNode := node.GetValueNodeOrRoot(rootNode)

	return &Error{
		Message: msg,
		Line:    valueNode.Line,
		Column:  valueNode.Column,
	}
}

func NewSliceError(msg string, core CoreModeler, node sliceNodeGetter, index int) error {
	rootNode := core.GetRootNode()

	if rootNode == nil {
		// Fallback if RootNode is not available
		return &Error{
			Message: msg,
			// Default to line 0, column 0 if we can't get location info
		}
	}
	valueNode := node.GetSliceValueNodeOrRoot(index, rootNode)

	return &Error{
		Message: msg,
		Line:    valueNode.Line,
		Column:  valueNode.Column,
	}
}

func NewMapKeyError(msg string, core CoreModeler, node mapKeyNodeGetter, key string) error {
	rootNode := core.GetRootNode()

	if rootNode == nil {
		// Fallback if RootNode is not available
		return &Error{
			Message: msg,
			// Default to line 0, column 0 if we can't get location info
		}
	}
	valueNode := node.GetMapKeyNodeOrRoot(key, rootNode)

	return &Error{
		Message: msg,
		Line:    valueNode.Line,
		Column:  valueNode.Column,
	}
}

func NewMapValueError(msg string, core CoreModeler, node mapValueNodeGetter, key string) error {
	rootNode := core.GetRootNode()

	if rootNode == nil {
		// Fallback if RootNode is not available
		return &Error{
			Message: msg,
			// Default to line 0, column 0 if we can't get location info
		}
	}
	valueNode := node.GetMapValueNodeOrRoot(key, rootNode)

	return &Error{
		Message: msg,
		Line:    valueNode.Line,
		Column:  valueNode.Column,
	}
}
