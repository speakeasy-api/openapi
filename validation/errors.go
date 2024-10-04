package validation

import "fmt"

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
