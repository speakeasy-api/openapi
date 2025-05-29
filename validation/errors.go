package validation

import (
	"fmt"
	"reflect"

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

func NewValueError(msg string, core any, node valueNodeGetter) error {
	// Use reflection to get the RootNode field from the core model
	v := reflect.ValueOf(core)
	rootNodeField := v.FieldByName("RootNode")

	if !rootNodeField.IsValid() || rootNodeField.IsNil() {
		// Fallback if RootNode is not available
		return &Error{
			Message: msg,
			// Default to line 0, column 0 if we can't get location info
		}
	}

	rootNode := rootNodeField.Interface().(*yaml.Node)
	valueNode := node.GetValueNodeOrRoot(rootNode)

	return &Error{
		Message: msg,
		Line:    valueNode.Line,
		Column:  valueNode.Column,
	}
}

func NewSliceError(msg string, core any, node sliceNodeGetter, index int) error {
	// Use reflection to get the RootNode field from the core model
	v := reflect.ValueOf(core)
	rootNodeField := v.FieldByName("RootNode")

	if !rootNodeField.IsValid() || rootNodeField.IsNil() {
		// Fallback if RootNode is not available
		return &Error{
			Message: msg,
			// Default to line 0, column 0 if we can't get location info
		}
	}

	rootNode := rootNodeField.Interface().(*yaml.Node)
	valueNode := node.GetSliceValueNodeOrRoot(index, rootNode)

	return &Error{
		Message: msg,
		Line:    valueNode.Line,
		Column:  valueNode.Column,
	}
}

func NewMapKeyError(msg string, core any, node mapKeyNodeGetter, key string) error {
	// Use reflection to get the RootNode field from the core model
	v := reflect.ValueOf(core)
	rootNodeField := v.FieldByName("RootNode")

	if !rootNodeField.IsValid() || rootNodeField.IsNil() {
		// Fallback if RootNode is not available
		return &Error{
			Message: msg,
			// Default to line 0, column 0 if we can't get location info
		}
	}

	rootNode := rootNodeField.Interface().(*yaml.Node)
	valueNode := node.GetMapKeyNodeOrRoot(key, rootNode)

	return &Error{
		Message: msg,
		Line:    valueNode.Line,
		Column:  valueNode.Column,
	}
}

func NewMapValueError(msg string, core any, node mapValueNodeGetter, key string) error {
	// Use reflection to get the RootNode field from the core model
	v := reflect.ValueOf(core)
	rootNodeField := v.FieldByName("RootNode")

	if !rootNodeField.IsValid() || rootNodeField.IsNil() {
		// Fallback if RootNode is not available
		return &Error{
			Message: msg,
			// Default to line 0, column 0 if we can't get location info
		}
	}

	rootNode := rootNodeField.Interface().(*yaml.Node)
	valueNode := node.GetMapValueNodeOrRoot(key, rootNode)

	return &Error{
		Message: msg,
		Line:    valueNode.Line,
		Column:  valueNode.Column,
	}
}
