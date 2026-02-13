package openapi

import (
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
