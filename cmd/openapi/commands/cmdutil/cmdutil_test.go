package cmdutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsStdin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "dash is stdin", path: "-", expected: true},
		{name: "empty is not stdin", path: "", expected: false},
		{name: "file path is not stdin", path: "spec.yaml", expected: false},
		{name: "dash prefix is not stdin", path: "-file", expected: false},
		{name: "double dash is not stdin", path: "--", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsStdin(tt.path))
		})
	}
}

func TestInputFileFromArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{name: "no args returns stdin indicator", args: []string{}, expected: StdinIndicator},
		{name: "nil args returns stdin indicator", args: nil, expected: StdinIndicator},
		{name: "single file arg", args: []string{"spec.yaml"}, expected: "spec.yaml"},
		{name: "explicit dash", args: []string{"-"}, expected: "-"},
		{name: "multiple args returns first", args: []string{"spec.yaml", "out.yaml"}, expected: "spec.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, InputFileFromArgs(tt.args))
		})
	}
}

func TestArgAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		index      int
		defaultVal string
		expected   string
	}{
		{name: "returns value at index", args: []string{"a", "b", "c"}, index: 1, defaultVal: "", expected: "b"},
		{name: "returns first element", args: []string{"a"}, index: 0, defaultVal: "x", expected: "a"},
		{name: "returns default when out of range", args: []string{"a"}, index: 1, defaultVal: "default", expected: "default"},
		{name: "returns default for empty args", args: []string{}, index: 0, defaultVal: "default", expected: "default"},
		{name: "returns default for nil args", args: nil, index: 0, defaultVal: "default", expected: "default"},
		{name: "returns empty default", args: []string{}, index: 0, defaultVal: "", expected: ""},
		{name: "returns default for negative index", args: []string{"a", "b"}, index: -1, defaultVal: "default", expected: "default"},
		{name: "returns default for large negative index", args: []string{"a"}, index: -100, defaultVal: "fallback", expected: "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, ArgAt(tt.args, tt.index, tt.defaultVal))
		})
	}
}

func TestStdinOrFileArgs(t *testing.T) {
	t.Parallel()

	validator := StdinOrFileArgs(1, 2)

	t.Run("accepts one arg", func(t *testing.T) {
		t.Parallel()
		err := validator(nil, []string{"spec.yaml"})
		assert.NoError(t, err)
	})

	t.Run("accepts two args", func(t *testing.T) {
		t.Parallel()
		err := validator(nil, []string{"spec.yaml", "out.yaml"})
		assert.NoError(t, err)
	})

	t.Run("rejects three args with max 2", func(t *testing.T) {
		t.Parallel()
		err := validator(nil, []string{"a", "b", "c"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "accepts at most 2 arg(s)")
	})

	t.Run("unbounded max accepts many args", func(t *testing.T) {
		t.Parallel()
		unbounded := StdinOrFileArgs(1, -1)
		err := unbounded(nil, []string{"a", "b", "c", "d", "e"})
		assert.NoError(t, err, "negative maxArgs should allow unlimited args")
	})

	t.Run("zero minArgs accepts any arg count", func(t *testing.T) {
		t.Parallel()
		noMin := StdinOrFileArgs(0, 2)
		err := noMin(nil, []string{})
		assert.NoError(t, err, "zero minArgs should accept empty args")
	})

	t.Run("error message includes min arg count", func(t *testing.T) {
		t.Parallel()
		min3 := StdinOrFileArgs(3, 5)
		err := min3(nil, []string{"a", "b"})
		if err != nil {
			assert.Contains(t, err.Error(), "requires at least 3 arg(s)")
		}
	})
}
