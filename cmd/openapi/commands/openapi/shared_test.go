package openapi

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{name: "absolute path is not stdin", path: "/tmp/spec.yaml", expected: false},
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

func TestNewOpenAPIProcessor_StdinDetection(t *testing.T) {
	t.Parallel()

	t.Run("dash input sets ReadFromStdin", func(t *testing.T) {
		t.Parallel()
		p, err := NewOpenAPIProcessor("-", "", false)
		require.NoError(t, err)
		assert.True(t, p.ReadFromStdin)
		assert.True(t, p.WriteToStdout, "stdin with no output should write to stdout")
	})

	t.Run("dash input with output file", func(t *testing.T) {
		t.Parallel()
		p, err := NewOpenAPIProcessor("-", "out.yaml", false)
		require.NoError(t, err)
		assert.True(t, p.ReadFromStdin)
		assert.False(t, p.WriteToStdout)
		assert.Equal(t, "out.yaml", p.OutputFile)
	})

	t.Run("file input not stdin", func(t *testing.T) {
		t.Parallel()
		p, err := NewOpenAPIProcessor("spec.yaml", "", false)
		require.NoError(t, err)
		assert.False(t, p.ReadFromStdin)
		assert.True(t, p.WriteToStdout, "no output file should write to stdout")
	})

	t.Run("file input with output file", func(t *testing.T) {
		t.Parallel()
		p, err := NewOpenAPIProcessor("spec.yaml", "out.yaml", false)
		require.NoError(t, err)
		assert.False(t, p.ReadFromStdin)
		assert.False(t, p.WriteToStdout)
		assert.Equal(t, "out.yaml", p.OutputFile)
	})
}

func TestNewOpenAPIProcessor_WriteInPlace(t *testing.T) {
	t.Parallel()

	t.Run("write in place with file input", func(t *testing.T) {
		t.Parallel()
		p, err := NewOpenAPIProcessor("spec.yaml", "", true)
		require.NoError(t, err)
		assert.Equal(t, "spec.yaml", p.OutputFile)
		assert.False(t, p.WriteToStdout)
	})

	t.Run("write in place with stdin is error", func(t *testing.T) {
		t.Parallel()
		_, err := NewOpenAPIProcessor("-", "", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot use --write flag when reading from stdin")
	})

	t.Run("write in place with output file is error", func(t *testing.T) {
		t.Parallel()
		_, err := NewOpenAPIProcessor("spec.yaml", "out.yaml", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot specify output file when using --write flag")
	})
}

func TestInputFileFromArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{name: "no args returns stdin indicator", args: []string{}, expected: "-"},
		{name: "nil args returns stdin indicator", args: nil, expected: "-"},
		{name: "single file arg", args: []string{"spec.yaml"}, expected: "spec.yaml"},
		{name: "explicit dash", args: []string{"-"}, expected: "-"},
		{name: "multiple args returns first", args: []string{"spec.yaml", "out.yaml"}, expected: "spec.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, inputFileFromArgs(tt.args))
		})
	}
}

func TestOutputFileFromArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{name: "no args returns empty", args: []string{}, expected: ""},
		{name: "one arg returns empty", args: []string{"input.yaml"}, expected: ""},
		{name: "two args returns second", args: []string{"input.yaml", "output.yaml"}, expected: "output.yaml"},
		{name: "three args returns second", args: []string{"a", "b", "c"}, expected: "b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, outputFileFromArgs(tt.args))
		})
	}
}

func TestOpenAPIProcessor_IOAccessors(t *testing.T) {
	t.Parallel()

	t.Run("stdin falls back to os.Stdin when nil", func(t *testing.T) {
		t.Parallel()
		p := &OpenAPIProcessor{}
		assert.Equal(t, os.Stdin, p.stdin(), "nil Stdin should fall back to os.Stdin")
	})

	t.Run("stdout falls back to os.Stdout when nil", func(t *testing.T) {
		t.Parallel()
		p := &OpenAPIProcessor{}
		assert.Equal(t, os.Stdout, p.stdout(), "nil Stdout should fall back to os.Stdout")
	})

	t.Run("stderr falls back to os.Stderr when nil", func(t *testing.T) {
		t.Parallel()
		p := &OpenAPIProcessor{}
		assert.Equal(t, os.Stderr, p.stderr(), "nil Stderr should fall back to os.Stderr")
	})

	t.Run("stdin uses override when set", func(t *testing.T) {
		t.Parallel()
		custom := &bytes.Buffer{}
		p := &OpenAPIProcessor{Stdin: custom}
		assert.Equal(t, custom, p.stdin(), "should use custom Stdin")
	})

	t.Run("stdout uses override when set", func(t *testing.T) {
		t.Parallel()
		custom := &bytes.Buffer{}
		p := &OpenAPIProcessor{Stdout: custom}
		assert.Equal(t, custom, p.stdout(), "should use custom Stdout")
	})

	t.Run("stderr uses override when set", func(t *testing.T) {
		t.Parallel()
		custom := &bytes.Buffer{}
		p := &OpenAPIProcessor{Stderr: custom}
		assert.Equal(t, custom, p.stderr(), "should use custom Stderr")
	})
}

func TestStdinOrFileArgs(t *testing.T) {
	t.Parallel()

	validator := stdinOrFileArgs(1, 2)

	t.Run("zero args returns error or nil depending on stdin", func(t *testing.T) {
		t.Parallel()
		err := validator(nil, []string{})
		// When stdin is not piped (typical in tests), this should error.
		// When stdin IS piped (e.g. CI), this returns nil. Both are valid.
		if err != nil {
			assert.Contains(t, err.Error(), "pipe data to stdin")
		}
	})

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
}
