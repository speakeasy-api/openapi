package testutils

import (
	"iter"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

// TODO use these more in tests
func CreateStringYamlNode(value string, line, column int) *yaml.Node {
	return &yaml.Node{
		Value:  value,
		Kind:   yaml.ScalarNode,
		Tag:    "!!str",
		Line:   line,
		Column: column,
	}
}

func CreateIntYamlNode(value int, line, column int) *yaml.Node {
	return &yaml.Node{
		Value:  strconv.Itoa(value),
		Kind:   yaml.ScalarNode,
		Tag:    "!!int",
		Line:   line,
		Column: column,
	}
}

func CreateBoolYamlNode(value bool, line, column int) *yaml.Node {
	return &yaml.Node{
		Value:  strconv.FormatBool(value),
		Kind:   yaml.ScalarNode,
		Tag:    "!!bool",
		Line:   line,
		Column: column,
	}
}

func CreateMapYamlNode(contents []*yaml.Node, line, column int) *yaml.Node {
	return &yaml.Node{
		Content: contents,
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Line:    line,
		Column:  column,
	}
}

type SequencedMap interface {
	Len() int
	AllUntyped() iter.Seq2[any, any]
	GetUntyped(key any) (any, bool)
}

// isInterfaceNil checks if an interface has a nil underlying value
func isInterfaceNil(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

func AssertEqualSequencedMap(t *testing.T, expected, actual SequencedMap) {
	t.Helper()
	// Check if both are truly nil (interface with nil type and value)
	if expected == nil && actual == nil {
		return
	}

	// Check if either is nil or has a nil underlying value
	expectedIsNil := expected == nil || (expected != nil && isInterfaceNil(expected))
	actualIsNil := actual == nil || (actual != nil && isInterfaceNil(actual))

	if expectedIsNil && actualIsNil {
		return
	}

	if expectedIsNil || actualIsNil {
		assert.Fail(t, "expected and actual must not be nil")
		return
	}

	assert.EqualExportedValues(t, expected, actual)
	assert.Equal(t, expected.Len(), actual.Len())

	alreadySeen := map[any]bool{}

	for k, v := range expected.AllUntyped() {
		actualV, ok := actual.GetUntyped(k)
		assert.True(t, ok)
		assert.EqualExportedValues(t, v, actualV)

		alreadySeen[k] = true
	}

	for k, v := range actual.AllUntyped() {
		if _, ok := alreadySeen[k]; ok {
			continue
		}

		actualV, ok := actual.GetUntyped(k)
		assert.True(t, ok)
		assert.EqualExportedValues(t, v, actualV)
	}
}
