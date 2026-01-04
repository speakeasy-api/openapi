package validation

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityHint    Severity = "hint"
)

func (s Severity) String() string {
	return string(s)
}

// Rank returns a numeric rank for severity comparison.
// Higher rank means worse severity.
// SeverityError = 2, SeverityWarning = 1, SeverityHint = 0.
// Unknown severities are treated as SeverityError.
func (s Severity) Rank() int {
	switch s {
	case SeverityError:
		return 2
	case SeverityWarning:
		return 1
	case SeverityHint:
		return 0
	default:
		return 2 // Treat unknown as error
	}
}

// Error represents a validation error and the line and column where it occurred
// TODO allow getting the JSON path for line/column for validation errors
type Error struct {
	UnderlyingError error
	Node            *yaml.Node
	Severity        Severity
	Rule            string
}

var _ error = (*Error)(nil)

func (e Error) Error() string {
	return fmt.Sprintf("[%d:%d] %s %s %s", e.GetLineNumber(), e.GetColumnNumber(), e.Severity, e.Rule, e.UnderlyingError.Error())
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

func (e Error) GetSeverity() Severity {
	return e.Severity
}

// ValueNodeGetter provides access to value nodes for error reporting.
type ValueNodeGetter interface {
	GetValueNodeOrRoot(root *yaml.Node) *yaml.Node
}

// SliceNodeGetter provides access to slice element nodes for error reporting.
type SliceNodeGetter interface {
	GetSliceValueNodeOrRoot(index int, root *yaml.Node) *yaml.Node
}

// MapKeyNodeGetter provides access to map key nodes for error reporting.
type MapKeyNodeGetter interface {
	GetMapKeyNodeOrRoot(key string, root *yaml.Node) *yaml.Node
}

// MapValueNodeGetter provides access to map value nodes for error reporting.
type MapValueNodeGetter interface {
	GetMapValueNodeOrRoot(key string, root *yaml.Node) *yaml.Node
}

func NewValidationError(severity Severity, rule string, err error, node *yaml.Node) error {
	return &Error{
		UnderlyingError: err,
		Node:            node,
		Severity:        severity,
		Rule:            rule,
	}
}

type CoreModeler interface {
	GetRootNode() *yaml.Node
}

func NewValueError(severity Severity, rule string, err error, core CoreModeler, node ValueNodeGetter) error {
	rootNode := core.GetRootNode()

	if rootNode == nil {
		// Fallback if RootNode is not available
		return &Error{
			UnderlyingError: err,
			// Default to line 0, column 0 if we can't get location info
			Severity: severity,
			Rule:     rule,
		}
	}
	valueNode := node.GetValueNodeOrRoot(rootNode)

	return &Error{
		UnderlyingError: err,
		Node:            valueNode,
		Severity:        severity,
		Rule:            rule,
	}
}

func NewSliceError(severity Severity, rule string, err error, core CoreModeler, node SliceNodeGetter, index int) error {
	rootNode := core.GetRootNode()

	if rootNode == nil {
		// Fallback if RootNode is not available
		return &Error{
			UnderlyingError: err,
			// Default to line 0, column 0 if we can't get location info
			Severity: severity,
			Rule:     rule,
		}
	}
	valueNode := node.GetSliceValueNodeOrRoot(index, rootNode)

	return &Error{
		UnderlyingError: err,
		Node:            valueNode,
		Severity:        severity,
		Rule:            rule,
	}
}

func NewMapKeyError(severity Severity, rule string, err error, core CoreModeler, node MapKeyNodeGetter, key string) error {
	rootNode := core.GetRootNode()

	if rootNode == nil {
		// Fallback if RootNode is not available
		return &Error{
			UnderlyingError: err,
			// Default to line 0, column 0 if we can't get location info
			Severity: severity,
			Rule:     rule,
		}
	}
	valueNode := node.GetMapKeyNodeOrRoot(key, rootNode)

	return &Error{
		UnderlyingError: err,
		Node:            valueNode,
		Severity:        severity,
		Rule:            rule,
	}
}

func NewMapValueError(severity Severity, rule string, err error, core CoreModeler, node MapValueNodeGetter, key string) error {
	rootNode := core.GetRootNode()

	if rootNode == nil {
		// Fallback if RootNode is not available
		return &Error{
			UnderlyingError: err,
			// Default to line 0, column 0 if we can't get location info
			Severity: severity,
			Rule:     rule,
		}
	}
	valueNode := node.GetMapValueNodeOrRoot(key, rootNode)

	return &Error{
		UnderlyingError: err,
		Node:            valueNode,
		Severity:        severity,
		Rule:            rule,
	}
}

type TypeMismatchError struct {
	Msg        string
	ParentName string
}

var _ error = (*TypeMismatchError)(nil)

func NewTypeMismatchError(parentName, msg string, args ...any) *TypeMismatchError {
	if len(args) > 0 {
		msg = fmt.Errorf(msg, args...).Error()
	}

	return &TypeMismatchError{
		Msg:        msg,
		ParentName: parentName,
	}
}

func (e TypeMismatchError) Error() string {
	name := e.ParentName
	if name != "" {
		name += " "
	}

	return fmt.Sprintf("%s%s", name, e.Msg)
}
