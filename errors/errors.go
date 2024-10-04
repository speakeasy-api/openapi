package errors

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// ErrSeperator is used to seperate the message from the cause in the error message
const ErrSeperator = " -- "

// Error provides a string based error type allowing the definition of const errors in packages
type Error string

func (s Error) Error() string {
	return string(s)
}

// Is checks if targer error is equivelant to Error
func (s Error) Is(target error) bool {
	return s.Error() == target.Error() || strings.HasPrefix(target.Error(), s.Error()+ErrSeperator)
}

// As will set target errors value to equal Error if they are equivelant
func (s Error) As(target interface{}) bool {
	v := reflect.ValueOf(target).Elem()
	if v.Type().Name() == "Error" && v.CanSet() {
		v.SetString(string(s))
		return true
	}
	return false
}

// Wrap will add the provided error as a cause for this Error and return the wrapped error
func (s Error) Wrap(err error) error {
	return wrappedError{cause: err, msg: string(s)}
}

type wrappedError struct {
	cause error
	msg   string
}

func (w wrappedError) Error() string {
	if w.cause != nil {
		return fmt.Sprintf("%s%s%v", w.msg, ErrSeperator, w.cause)
	}
	return w.msg
}

func (w wrappedError) Is(target error) bool {
	return Error(w.msg).Is(target)
}

func (w wrappedError) As(target interface{}) bool {
	return Error(w.msg).As(target)
}

func (w wrappedError) Unwrap() error {
	return w.cause
}

// The below are just wrappers as we are stealing the namespace of the errors package

// Is checks if err is equivelant to target
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// As will set target errors value to equal Error if they are equivelant
func As(err error, target any) bool {
	return errors.As(err, target)
}

// New returns a new error with the specified message.
func New(message string) error {
	return errors.New(message)
}

// Join returns an error that wraps the given errors.
func Join(errs ...error) error {
	return errors.Join(errs...)
}

type JoinedErrors interface {
	Unwrap() []error
}

func UnwrapErrors(err error) []error {
	if err == nil {
		return nil
	}

	if je, ok := err.(JoinedErrors); ok {
		return je.Unwrap()
	}
	return []error{err}
}
