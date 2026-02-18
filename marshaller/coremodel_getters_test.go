package marshaller_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestCoreModel_GetRootNodeLine_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		model    marshaller.CoreModel
		expected int
	}{
		{
			name:     "nil root node returns -1",
			model:    marshaller.CoreModel{},
			expected: -1,
		},
		{
			name:     "returns line number",
			model:    marshaller.CoreModel{RootNode: &yaml.Node{Line: 42}},
			expected: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.model.GetRootNodeLine())
		})
	}
}

func TestCoreModel_GetValid_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		model    marshaller.CoreModel
		expected bool
	}{
		{
			name:     "default is false",
			model:    marshaller.CoreModel{},
			expected: false,
		},
		{
			name:     "returns true when set",
			model:    marshaller.CoreModel{Valid: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.model.GetValid())
		})
	}
}

func TestCoreModel_GetValidYaml_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		model    marshaller.CoreModel
		expected bool
	}{
		{
			name:     "default is false",
			model:    marshaller.CoreModel{},
			expected: false,
		},
		{
			name:     "returns true when set",
			model:    marshaller.CoreModel{ValidYaml: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.model.GetValidYaml())
		})
	}
}

func TestCoreModel_GetUnknownProperties_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		model    *marshaller.CoreModel
		expected []string
	}{
		{
			name:     "nil unknown properties returns empty slice",
			model:    &marshaller.CoreModel{},
			expected: []string{},
		},
		{
			name:     "returns unknown properties",
			model:    &marshaller.CoreModel{UnknownProperties: []string{"prop1", "prop2"}},
			expected: []string{"prop1", "prop2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.model.GetUnknownProperties())
		})
	}
}
