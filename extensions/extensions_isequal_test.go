package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestExtensions_IsEqual_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ext1     *Extensions
		ext2     *Extensions
		expected bool
	}{
		{
			name:     "both nil extensions should be equal",
			ext1:     nil,
			ext2:     nil,
			expected: true,
		},
		{
			name:     "nil extension and empty extension should be equal",
			ext1:     nil,
			ext2:     New(),
			expected: true,
		},
		{
			name:     "empty extension and nil extension should be equal",
			ext1:     New(),
			ext2:     nil,
			expected: true,
		},
		{
			name:     "both empty extensions should be equal",
			ext1:     New(),
			ext2:     New(),
			expected: true,
		},
		{
			name: "extensions with same key-value pairs should be equal",
			ext1: func() *Extensions {
				ext := New()
				ext.Set("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"})
				ext.Set("x-another", &yaml.Node{Kind: yaml.ScalarNode, Value: "value2"})
				return ext
			}(),
			ext2: func() *Extensions {
				ext := New()
				ext.Set("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"})
				ext.Set("x-another", &yaml.Node{Kind: yaml.ScalarNode, Value: "value2"})
				return ext
			}(),
			expected: true,
		},
		{
			name: "extensions with same key-value pairs in different order should be equal",
			ext1: func() *Extensions {
				ext := New()
				ext.Set("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"})
				ext.Set("x-another", &yaml.Node{Kind: yaml.ScalarNode, Value: "value2"})
				return ext
			}(),
			ext2: func() *Extensions {
				ext := New()
				ext.Set("x-another", &yaml.Node{Kind: yaml.ScalarNode, Value: "value2"})
				ext.Set("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"})
				return ext
			}(),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.ext1.IsEqual(tt.ext2)
			assert.Equal(t, tt.expected, actual, "extensions should match expected equality")
		})
	}
}

func TestExtensions_IsEqual_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ext1     *Extensions
		ext2     *Extensions
		expected bool
	}{
		{
			name: "nil extension vs non-empty extension should not be equal",
			ext1: nil,
			ext2: func() *Extensions {
				ext := New()
				ext.Set("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"})
				return ext
			}(),
			expected: false,
		},
		{
			name: "non-empty extension vs nil extension should not be equal",
			ext1: func() *Extensions {
				ext := New()
				ext.Set("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"})
				return ext
			}(),
			ext2:     nil,
			expected: false,
		},
		{
			name: "empty extension vs non-empty extension should not be equal",
			ext1: New(),
			ext2: func() *Extensions {
				ext := New()
				ext.Set("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"})
				return ext
			}(),
			expected: false,
		},
		{
			name: "extensions with different values should not be equal",
			ext1: func() *Extensions {
				ext := New()
				ext.Set("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"})
				ext.Set("x-another", &yaml.Node{Kind: yaml.ScalarNode, Value: "value2"})
				return ext
			}(),
			ext2: func() *Extensions {
				ext := New()
				ext.Set("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"})
				ext.Set("x-another", &yaml.Node{Kind: yaml.ScalarNode, Value: "different"})
				return ext
			}(),
			expected: false,
		},
		{
			name: "extensions with different keys should not be equal",
			ext1: func() *Extensions {
				ext := New()
				ext.Set("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"})
				ext.Set("x-another", &yaml.Node{Kind: yaml.ScalarNode, Value: "value2"})
				return ext
			}(),
			ext2: func() *Extensions {
				ext := New()
				ext.Set("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"})
				ext.Set("x-different", &yaml.Node{Kind: yaml.ScalarNode, Value: "value2"})
				return ext
			}(),
			expected: false,
		},
		{
			name: "extensions with different lengths should not be equal",
			ext1: func() *Extensions {
				ext := New()
				ext.Set("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"})
				return ext
			}(),
			ext2: func() *Extensions {
				ext := New()
				ext.Set("x-custom", &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"})
				ext.Set("x-another", &yaml.Node{Kind: yaml.ScalarNode, Value: "value2"})
				return ext
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.ext1.IsEqual(tt.ext2)
			assert.Equal(t, tt.expected, actual, "extensions should match expected equality")
		})
	}
}

func TestExtensions_IsEqual_WithComplexValues(t *testing.T) {
	t.Parallel()

	t.Run("extensions with same complex YAML values should be equal", func(t *testing.T) {
		t.Parallel()
		ext1 := New()
		ext1.Set("x-complex", &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "nested"},
				{Kind: yaml.ScalarNode, Value: "value"},
			},
		})

		ext2 := New()
		ext2.Set("x-complex", &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "nested"},
				{Kind: yaml.ScalarNode, Value: "value"},
			},
		})

		actual := ext1.IsEqual(ext2)
		assert.True(t, actual, "extensions with same complex values should be equal")
	})

	t.Run("extensions with different complex YAML values should not be equal", func(t *testing.T) {
		t.Parallel()
		ext1 := New()
		ext1.Set("x-complex", &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "nested"},
				{Kind: yaml.ScalarNode, Value: "value1"},
			},
		})

		ext2 := New()
		ext2.Set("x-complex", &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "nested"},
				{Kind: yaml.ScalarNode, Value: "value2"},
			},
		})

		actual := ext1.IsEqual(ext2)
		assert.False(t, actual, "extensions with different complex values should not be equal")
	})

	t.Run("extensions with array values should be compared correctly", func(t *testing.T) {
		t.Parallel()
		ext1 := New()
		ext1.Set("x-array", &yaml.Node{
			Kind: yaml.SequenceNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "item1"},
				{Kind: yaml.ScalarNode, Value: "item2"},
			},
		})

		ext2 := New()
		ext2.Set("x-array", &yaml.Node{
			Kind: yaml.SequenceNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "item1"},
				{Kind: yaml.ScalarNode, Value: "item2"},
			},
		})

		actual := ext1.IsEqual(ext2)
		assert.True(t, actual, "extensions with same array values should be equal")
	})
}
