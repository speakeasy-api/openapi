package yml_test

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestContextKey_String_Success(t *testing.T) {
	t.Parallel()
	// This tests the unexported contextKey type indirectly through the config functions
	ctx := t.Context()
	config := &yml.Config{
		Indentation:      4,
		KeyStringStyle:   yaml.DoubleQuotedStyle,
		ValueStringStyle: yaml.SingleQuotedStyle,
		OutputFormat:     yml.OutputFormatJSON,
		OriginalFormat:   yml.OutputFormatYAML,
	}

	ctxWithConfig := yml.ContextWithConfig(ctx, config)
	retrievedConfig := yml.GetConfigFromContext(ctxWithConfig)

	assert.Equal(t, config, retrievedConfig)
}

func TestContextWithConfig_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		config *yml.Config
	}{
		{
			name: "with valid config",
			config: &yml.Config{
				Indentation:      4,
				KeyStringStyle:   yaml.DoubleQuotedStyle,
				ValueStringStyle: yaml.SingleQuotedStyle,
				OutputFormat:     yml.OutputFormatJSON,
				OriginalFormat:   yml.OutputFormatYAML,
			},
		},
		{
			name:   "with nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			result := yml.ContextWithConfig(ctx, tt.config)

			assert.NotNil(t, result)

			// Verify the config can be retrieved
			retrievedConfig := yml.GetConfigFromContext(result)
			if tt.config == nil {
				// Should return default config
				assert.NotNil(t, retrievedConfig)
				assert.Equal(t, 2, retrievedConfig.Indentation)
				assert.Equal(t, yml.OutputFormatYAML, retrievedConfig.OutputFormat)
			} else {
				assert.Equal(t, tt.config, retrievedConfig)
			}
		})
	}
}

func TestGetConfigFromContext_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		setupContext   func() context.Context
		expectedConfig *yml.Config
	}{
		{
			name: "context with config",
			setupContext: func() context.Context {
				config := &yml.Config{
					Indentation:      4,
					KeyStringStyle:   yaml.DoubleQuotedStyle,
					ValueStringStyle: yaml.SingleQuotedStyle,
					OutputFormat:     yml.OutputFormatJSON,
					OriginalFormat:   yml.OutputFormatYAML,
				}
				return yml.ContextWithConfig(t.Context(), config)
			},
			expectedConfig: &yml.Config{
				Indentation:      4,
				KeyStringStyle:   yaml.DoubleQuotedStyle,
				ValueStringStyle: yaml.SingleQuotedStyle,
				OutputFormat:     yml.OutputFormatJSON,
				OriginalFormat:   yml.OutputFormatYAML,
			},
		},
		{
			name:         "context without config",
			setupContext: t.Context,
			expectedConfig: &yml.Config{
				Indentation:      2,
				KeyStringStyle:   0,
				ValueStringStyle: 0,
				OutputFormat:     yml.OutputFormatYAML,
				OriginalFormat:   "",
			},
		},
		{
			name: "context with invalid config type",
			setupContext: func() context.Context {
				// This simulates a corrupted context (though it's hard to create in practice)
				type contextKey string
				return context.WithValue(t.Context(), contextKey("yml-context-key-config"), "invalid")
			},
			expectedConfig: &yml.Config{
				Indentation:      2,
				KeyStringStyle:   0,
				ValueStringStyle: 0,
				OutputFormat:     yml.OutputFormatYAML,
				OriginalFormat:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := tt.setupContext()
			result := yml.GetConfigFromContext(ctx)

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedConfig.Indentation, result.Indentation)
			assert.Equal(t, tt.expectedConfig.KeyStringStyle, result.KeyStringStyle)
			assert.Equal(t, tt.expectedConfig.ValueStringStyle, result.ValueStringStyle)
			assert.Equal(t, tt.expectedConfig.OutputFormat, result.OutputFormat)
		})
	}
}

func TestGetConfigFromDoc_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		data                []byte
		doc                 *yaml.Node
		expectedFormat      yml.OutputFormat
		expectedIndent      int
		expectedIndentStyle yml.IndentationStyle
		expectedKeyStyle    yaml.Style
		expectedValueStyle  yaml.Style
	}{
		{
			name: "YAML document with quoted strings",
			data: []byte(`
  key1: "value1"
  key2: 'value2'
  nested:
    subkey: "subvalue"
`),
			doc: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Tag:  "!!map",
						Content: []*yaml.Node{
							{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
							{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key2", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
							{Value: "value2", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.SingleQuotedStyle},
						},
					},
				},
			},
			expectedFormat:      yml.OutputFormatYAML,
			expectedIndent:      2,
			expectedIndentStyle: yml.IndentationStyleSpace,
			expectedKeyStyle:    0,
			expectedValueStyle:  yaml.DoubleQuotedStyle,
		},
		{
			name: "JSON document",
			data: []byte(`{
  "key1": "value1",
  "key2": "value2"
}`),
			doc: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Tag:  "!!map",
						Content: []*yaml.Node{
							{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str"},
						},
					},
				},
			},
			expectedFormat:      yml.OutputFormatJSON,
			expectedIndent:      2,
			expectedIndentStyle: yml.IndentationStyleSpace,
			expectedKeyStyle:    0,
			expectedValueStyle:  0,
		},
		{
			name: "YAML with 4-space indentation",
			data: []byte(`
    key1: value1
    key2: value2
`),
			doc: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Tag:  "!!map",
						Content: []*yaml.Node{
							{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str"},
						},
					},
				},
			},
			expectedFormat:      yml.OutputFormatYAML,
			expectedIndent:      4,
			expectedIndentStyle: yml.IndentationStyleSpace,
			expectedKeyStyle:    0,
			expectedValueStyle:  0,
		},
		{
			name: "YAML with single tab indentation",
			data: []byte("key1: value1\nnested:\n\tkey2: value2"),
			doc: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Tag:  "!!map",
						Content: []*yaml.Node{
							{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str"},
						},
					},
				},
			},
			expectedFormat:      yml.OutputFormatYAML,
			expectedIndent:      1,
			expectedIndentStyle: yml.IndentationStyleTab,
			expectedKeyStyle:    0,
			expectedValueStyle:  0,
		},
		{
			name: "YAML with double tab indentation",
			data: []byte("key1: value1\nnested:\n\t\tkey2: value2"),
			doc: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Tag:  "!!map",
						Content: []*yaml.Node{
							{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str"},
						},
					},
				},
			},
			expectedFormat:      yml.OutputFormatYAML,
			expectedIndent:      2,
			expectedIndentStyle: yml.IndentationStyleTab,
			expectedKeyStyle:    0,
			expectedValueStyle:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := yml.GetConfigFromDoc(tt.data, tt.doc)

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedFormat, result.OutputFormat)
			assert.Equal(t, tt.expectedFormat, result.OriginalFormat)
			assert.Equal(t, tt.expectedIndent, result.Indentation)
			assert.Equal(t, tt.expectedIndentStyle, result.IndentationStyle)
			assert.Equal(t, tt.expectedKeyStyle, result.KeyStringStyle)
			assert.Equal(t, tt.expectedValueStyle, result.ValueStringStyle)
		})
	}
}

func TestGetConfigFromDoc_WithComplexDocument_Success(t *testing.T) {
	t.Parallel()
	// Test with a more complex document structure
	data := []byte(`
  # Comment
  array:
    - item1
    - item2
  map:
    nested: "value"
`)

	doc := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "array", Kind: yaml.ScalarNode, Tag: "!!str"},
					{
						Kind: yaml.SequenceNode,
						Tag:  "!!seq",
						Content: []*yaml.Node{
							{Value: "item1", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "item2", Kind: yaml.ScalarNode, Tag: "!!str"},
						},
					},
					{Value: "map", Kind: yaml.ScalarNode, Tag: "!!str"},
					{
						Kind: yaml.MappingNode,
						Tag:  "!!map",
						Content: []*yaml.Node{
							{Value: "nested", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "value", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
						},
					},
				},
			},
		},
	}

	result := yml.GetConfigFromDoc(data, doc)

	require.NotNil(t, result)
	assert.Equal(t, yml.OutputFormatYAML, result.OutputFormat)
	assert.Equal(t, yml.OutputFormatYAML, result.OriginalFormat)
	assert.Equal(t, 2, result.Indentation)
	assert.Equal(t, yml.IndentationStyleSpace, result.IndentationStyle)
	assert.Equal(t, yaml.Style(0), result.KeyStringStyle)
	assert.Equal(t, yaml.Style(0), result.ValueStringStyle) // First string found doesn't have style
}

func TestGetConfigFromDoc_WithAliasNodes_Success(t *testing.T) {
	t.Parallel()
	// Test with alias nodes
	data := []byte(`
  key1: &anchor "value1"
  key2: *anchor
`)

	anchorNode := &yaml.Node{
		Value: "value1",
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Style: yaml.DoubleQuotedStyle,
	}

	doc := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
					anchorNode,
					{Value: "key2", Kind: yaml.ScalarNode, Tag: "!!str"},
					{Kind: yaml.AliasNode, Alias: anchorNode},
				},
			},
		},
	}

	result := yml.GetConfigFromDoc(data, doc)

	require.NotNil(t, result)
	assert.Equal(t, yml.OutputFormatYAML, result.OutputFormat)
	assert.Equal(t, yaml.DoubleQuotedStyle, result.ValueStringStyle)
}

func TestGetConfigFromDoc_EdgeCases_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		data                []byte
		expectedFormat      yml.OutputFormat
		expectedIndent      int
		expectedIndentStyle yml.IndentationStyle
	}{
		{
			name:                "empty data",
			data:                []byte(""),
			expectedFormat:      yml.OutputFormatYAML,
			expectedIndent:      2,
			expectedIndentStyle: yml.IndentationStyleSpace,
		},
		{
			name:                "only comments",
			data:                []byte("# Just a comment\n# Another comment"),
			expectedFormat:      yml.OutputFormatYAML,
			expectedIndent:      2,
			expectedIndentStyle: yml.IndentationStyleSpace,
		},
		{
			name:                "only whitespace",
			data:                []byte("   \n  \n   "),
			expectedFormat:      yml.OutputFormatYAML,
			expectedIndent:      2,
			expectedIndentStyle: yml.IndentationStyleSpace,
		},
		{
			name:                "JSON with no indentation",
			data:                []byte(`{"key":"value"}`),
			expectedFormat:      yml.OutputFormatJSON,
			expectedIndent:      2,
			expectedIndentStyle: yml.IndentationStyleSpace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a minimal valid document structure to avoid panics
			doc := &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind:    yaml.MappingNode,
						Tag:     "!!map",
						Content: []*yaml.Node{},
					},
				},
			}

			result := yml.GetConfigFromDoc(tt.data, doc)

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedFormat, result.OutputFormat)
			assert.Equal(t, tt.expectedIndent, result.Indentation)
			assert.Equal(t, tt.expectedIndentStyle, result.IndentationStyle)
		})
	}
}
