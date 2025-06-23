// Package jsonpointer provides JSONPointer an implementation of RFC6901 https://datatracker.ietf.org/doc/html/rfc6901
package jsonpointer

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/speakeasy-api/openapi/errors"
)

const (
	// ErrNotFound is returned when the target is not found.
	ErrNotFound = errors.Error("not found")
	// ErrInvalidPath is returned when the path is invalid.
	ErrInvalidPath = errors.Error("invalid path")
	// ErrValidation is returned when the jsonpointer is invalid.
	ErrValidation = errors.Error("validation error")
	// ErrSkipInterface is returned when this implementation of the interface is not applicable to the current type.
	ErrSkipInterface = errors.Error("skip interface")
)

const (
	DefaultStructTag = "key"
)

type option func(o *options)

type options struct {
	StructTags []string
}

// WithStructTags will set the type of struct tags to use when navigating structs.
func WithStructTags(structTags ...string) option {
	return func(o *options) {
		o.StructTags = structTags
	}
}

func getOptions(opts []option) *options {
	o := &options{
		StructTags: []string{DefaultStructTag},
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// JSONPointer represents a JSON Pointer value as defined by RFC6901 https://datatracker.ietf.org/doc/html/rfc6901
type JSONPointer string

// Validate will validate the JSONPointer is valid as per RFC6901.
func (j JSONPointer) Validate() error {
	_, err := j.getNavigationStack()
	if err != nil {
		return ErrValidation.Wrap(err)
	}
	return nil
}

// GetTarget will evaluate the JSONPointer against the source and return the target.
// WithStructTags can be used to set the type of struct tags to use when navigating structs.
// If the struct implements any of the Navigable interfaces it will be used to navigate the source.
func GetTarget(source any, pointer JSONPointer, opts ...option) (any, error) {
	o := getOptions(opts)

	stack, err := pointer.getNavigationStack()
	if err != nil {
		return nil, ErrValidation.Wrap(err)
	}

	target, _, err := getCurrentStackTarget(source, stack, "/", o)
	if err != nil {
		return nil, err
	}

	return target, nil
}

// PartsToJSONPointer will convert the exploded parts of a JSONPointer to a JSONPointer.
func PartsToJSONPointer(parts []string) JSONPointer {
	var sb strings.Builder
	for _, part := range parts {
		sb.WriteByte('/')
		sb.WriteString(escape(part))
	}
	return JSONPointer(sb.String())
}

func getCurrentStackTarget(source any, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	if len(stack) == 0 {
		return source, stack, nil
	}

	currentPart := stack[0]
	stack = stack[1:]

	currentPath = buildPath(currentPath, currentPart)

	return getTarget(source, currentPart, stack, currentPath, o)
}

func getTarget(source any, currentPart navigationPart, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	sourceType := reflect.TypeOf(source)
	sourceElemType := sourceType

	if sourceType.Kind() == reflect.Ptr {
		sourceElemType = sourceType.Elem()
	}

	switch sourceElemType.Kind() {
	case reflect.Map:
		return getMapTarget(reflect.ValueOf(source), currentPart, stack, currentPath, o)
	case reflect.Slice, reflect.Array:
		return getSliceTarget(reflect.ValueOf(source), currentPart, stack, currentPath, o)
	case reflect.Struct:
		return getStructTarget(reflect.ValueOf(source), currentPart, stack, currentPath, o)
	default:
		return nil, nil, ErrInvalidPath.Wrap(fmt.Errorf("expected map, slice, or struct, got %s at %s", sourceElemType.Kind(), currentPath))
	}
}

func getMapTarget(sourceVal reflect.Value, currentPart navigationPart, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	sourceValElem := reflect.Indirect(sourceVal)

	if currentPart.Type != partTypeKey {
		return nil, nil, ErrInvalidPath.Wrap(fmt.Errorf("expected key, got %s at %s", currentPart.Type, currentPath))
	}
	if sourceValElem.Type().Key().Kind() != reflect.String {
		return nil, nil, ErrInvalidPath.Wrap(fmt.Errorf("expected map key to be string, got %s at %s", sourceValElem.Type().Key().Kind(), currentPath))
	}
	if sourceValElem.IsNil() {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("map is nil at %s", currentPath))
	}

	key := currentPart.unescapeValue()

	target := sourceValElem.MapIndex(reflect.ValueOf(key))
	if !target.IsValid() || target.IsZero() {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("key %s not found in map at %s", key, currentPath))
	}

	return getCurrentStackTarget(target.Interface(), stack, currentPath, o)
}

func getSliceTarget(sourceVal reflect.Value, currentPart navigationPart, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	sourceValElem := reflect.Indirect(sourceVal)

	if currentPart.Type != partTypeIndex {
		return nil, nil, ErrInvalidPath.Wrap(fmt.Errorf("expected index, got %s at %s", currentPart.Type, currentPath))
	}

	if sourceValElem.Kind() == reflect.Slice && sourceValElem.IsNil() {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("slice is nil at %s", currentPath))
	}

	index := currentPart.getIndex()

	if index < 0 || index >= sourceValElem.Len() {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("index %d out of range for slice/array of length %d at %s", index, sourceValElem.Len(), currentPath))
	}

	return getCurrentStackTarget(sourceValElem.Index(index).Interface(), stack, currentPath, o)
}

// KeyNavigable is an interface that can be implemented by a struct to allow navigation by key, bypassing navigating by struct tags.
type KeyNavigable interface {
	NavigateWithKey(key string) (any, error)
}

// IndexNavigable is an interface that can be implemented by a struct to allow navigation by index if the struct wraps some slice like type.
type IndexNavigable interface {
	NavigateWithIndex(index int) (any, error)
}

// NavigableNoder is an interface that can be implemented by a struct to allow returning an alternative node to evaluate instead of the struct itself.
type NavigableNoder interface {
	GetNavigableNode() (any, error)
}

func getStructTarget(sourceVal reflect.Value, currentPart navigationPart, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	if sourceVal.Type().Implements(reflect.TypeOf((*NavigableNoder)(nil)).Elem()) {
		val, stack, err := getNavigableNoderTarget(sourceVal, currentPart, stack, currentPath, o)
		if err != nil {
			if !errors.Is(err, ErrSkipInterface) {
				return nil, nil, err
			}
		} else {
			return val, stack, nil
		}
	}

	switch currentPart.Type {
	case partTypeKey:
		return getKeyBasedStructTarget(sourceVal, currentPart, stack, currentPath, o)
	case partTypeIndex:
		return getIndexBasedStructTarget(sourceVal, currentPart, stack, currentPath, o)
	default:
		return nil, nil, ErrInvalidPath.Wrap(fmt.Errorf("expected key or index, got %s at %s", currentPart.Type, currentPath))
	}
}

func getKeyBasedStructTarget(sourceVal reflect.Value, currentPart navigationPart, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	if sourceVal.Type().Implements(reflect.TypeOf((*KeyNavigable)(nil)).Elem()) {
		val, stack, err := getNavigableWithKeyTarget(sourceVal, currentPart, stack, currentPath, o)
		if err != nil {
			if !errors.Is(err, ErrSkipInterface) {
				return nil, nil, err
			}
		} else {
			return val, stack, nil
		}
	}

	if sourceVal.Kind() == reflect.Ptr && sourceVal.IsNil() {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("struct is nil at %s", currentPath))
	}

	key := currentPart.unescapeValue()

	sourceValElem := reflect.Indirect(sourceVal)

	for i := 0; i < sourceValElem.NumField(); i++ {
		field := sourceValElem.Type().Field(i)
		if !field.IsExported() {
			continue
		}
		fieldVal := sourceValElem.Field(i)

		tags := []string{}

		for _, tag := range o.StructTags {
			if field.Tag.Get(tag) != "" {
				tags = append(tags, field.Tag.Get(tag))
			}
		}

		fieldKey := field.Name
		if len(tags) > 0 {
			fieldKey = tags[0]
		}

		if fieldKey == key {
			return getCurrentStackTarget(fieldVal.Interface(), stack, currentPath, o)
		}
	}

	return nil, nil, ErrNotFound.Wrap(fmt.Errorf("key %s not found in %v at %s", key, sourceVal.Type(), currentPath))
}

func getIndexBasedStructTarget(sourceVal reflect.Value, currentPart navigationPart, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	if sourceVal.Type().Implements(reflect.TypeOf((*IndexNavigable)(nil)).Elem()) {
		val, stack, err := getNavigableWithIndexTarget(sourceVal, currentPart, stack, currentPath, o)
		if err != nil {
			if errors.Is(err, ErrSkipInterface) {
				return nil, nil, fmt.Errorf("can't navigate by index on %s at %s", sourceVal.Type(), currentPath)
			}
			return nil, nil, err
		} else {
			return val, stack, nil
		}
	} else {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("expected IndexNavigable, got %s at %s", sourceVal.Kind(), currentPath))
	}
}

func getNavigableWithKeyTarget(sourceVal reflect.Value, currentPart navigationPart, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	if sourceVal.Kind() == reflect.Ptr && sourceVal.IsNil() {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("source is nil at %s", currentPath))
	}

	kn, ok := sourceVal.Interface().(KeyNavigable)
	if !ok {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("expected keyNavigable, got %s at %s", sourceVal.Kind(), currentPath))
	}

	key := currentPart.unescapeValue()

	value, err := kn.NavigateWithKey(key)
	if err != nil {
		return nil, nil, ErrNotFound.Wrap(err)
	}

	return getCurrentStackTarget(value, stack, currentPath, o)
}

func getNavigableWithIndexTarget(sourceVal reflect.Value, currentPart navigationPart, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	if sourceVal.Kind() == reflect.Ptr && sourceVal.IsNil() {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("source is nil at %s", currentPath))
	}

	kn, ok := sourceVal.Interface().(IndexNavigable)
	if !ok {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("expected indexNavigable, got %s at %s", sourceVal.Kind(), currentPath))
	}

	index := currentPart.getIndex()

	value, err := kn.NavigateWithIndex(index)
	if err != nil {
		return nil, nil, ErrNotFound.Wrap(err)
	}

	return getCurrentStackTarget(value, stack, currentPath, o)
}

func getNavigableNoderTarget(sourceVal reflect.Value, currentPart navigationPart, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	if sourceVal.Kind() == reflect.Ptr && sourceVal.IsNil() {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("source is nil at %s", currentPath))
	}

	nn, ok := sourceVal.Interface().(NavigableNoder)
	if !ok {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("expected navigableNoder, got %s at %s", sourceVal.Kind(), currentPath))
	}

	value, err := nn.GetNavigableNode()
	if err != nil {
		return nil, nil, err
	}

	return getTarget(value, currentPart, stack, currentPath, o)
}

func buildPath(currentPath string, currentPart navigationPart) string {
	if !strings.HasSuffix(currentPath, "/") {
		currentPath += "/"
	}
	return currentPath + currentPart.Value
}

// EscapeString escapes a string for use as a reference token in a JSON pointer according to RFC6901.
// It replaces "~" with "~0" and "/" with "~1" as required by the specification.
// This function should be used when constructing JSON pointers from string values that may contain
// these special characters.
func EscapeString(s string) string {
	return escape(s)
}

func escape(part string) string {
	return strings.ReplaceAll(strings.ReplaceAll(part, "~", "~0"), "/", "~1")
}
