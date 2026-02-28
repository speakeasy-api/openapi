package yml_test

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
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

func TestGetConfigFromDoc_TrailingWhitespace_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		data           []byte
		expectedIndent int
	}{
		{
			name: "YAML with trailing whitespace on lines",
			data: []byte(`key: value    
nested:    
  child: value    `),
			expectedIndent: 2,
		},
		{
			name:           "YAML with trailing tabs",
			data:           []byte("key: value\t\t\nnested:\t\t\n  child: value\t\t"),
			expectedIndent: 2,
		},
		{
			name:           "line with only leading brace and trailing whitespace",
			data:           []byte("{    "),
			expectedIndent: 2, // default when no indentation found
		},
		{
			name: "mixed leading and trailing whitespace",
			data: []byte(`  key: value   
  nested:   
    child: value   `),
			expectedIndent: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a minimal document structure
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
			assert.Equal(t, tt.expectedIndent, result.Indentation, "should correctly calculate indentation ignoring trailing whitespace")
		})
	}
}

func TestGetConfigFromDoc_WithMixedStringStyles_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		data               []byte
		doc                *yaml.Node
		expectedKeyStyle   yaml.Style
		expectedValueStyle yaml.Style
	}{
		{
			name: "mostly double quoted values",
			data: []byte(`
  key1: "value1"
  key2: "value2"
  key3: 'value3'
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
							{Value: "value2", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key3", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
							{Value: "value3", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.SingleQuotedStyle},
						},
					},
				},
			},
			expectedKeyStyle:   0,
			expectedValueStyle: yaml.DoubleQuotedStyle,
		},
		{
			name: "mostly single quoted values",
			data: []byte(`
  key1: 'value1'
  key2: 'value2'
  key3: "value3"
`),
			doc: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Tag:  "!!map",
						Content: []*yaml.Node{
							{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
							{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.SingleQuotedStyle},
							{Value: "key2", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
							{Value: "value2", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.SingleQuotedStyle},
							{Value: "key3", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
							{Value: "value3", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
						},
					},
				},
			},
			expectedKeyStyle:   0,
			expectedValueStyle: yaml.SingleQuotedStyle,
		},
		{
			name: "numeric strings excluded from value style detection",
			data: []byte(`
  key1: "123"
  key2: "456"
  key3: "actual string"
  key4: "another string"
  key5: "third string"
`),
			doc: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Tag:  "!!map",
						Content: []*yaml.Node{
							{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
							{Value: "123", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key2", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
							{Value: "456", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key3", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
							{Value: "actual string", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key4", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
							{Value: "another string", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key5", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
							{Value: "third string", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
						},
					},
				},
			},
			expectedKeyStyle:   0,
			expectedValueStyle: yaml.DoubleQuotedStyle,
		},
		{
			name: "mixed key styles chooses most common",
			data: []byte(`
  "key1": value1
  "key2": value2
  key3: value3
`),
			doc: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Tag:  "!!map",
						Content: []*yaml.Node{
							{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "value1", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
							{Value: "key2", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "value2", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
							{Value: "key3", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
							{Value: "value3", Kind: yaml.ScalarNode, Tag: "!!str", Style: 0},
						},
					},
				},
			},
			expectedKeyStyle:   yaml.DoubleQuotedStyle,
			expectedValueStyle: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := yml.GetConfigFromDoc(tt.data, tt.doc)

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedKeyStyle, result.KeyStringStyle, "should select most common key string style")
			assert.Equal(t, tt.expectedValueStyle, result.ValueStringStyle, "should select most common value string style, excluding numbers")
		})
	}
}

func TestGetConfigFromDoc_WithNumericStrings_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		data               []byte
		doc                *yaml.Node
		expectedValueStyle yaml.Style
		description        string
	}{
		{
			name: "integers as strings",
			data: []byte(`
  key1: "123"
  key2: "456"
  key3: "789"
  key4: "real string"
`),
			doc: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Tag:  "!!map",
						Content: []*yaml.Node{
							{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "123", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key2", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "456", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key3", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "789", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key4", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "real string", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
						},
					},
				},
			},
			expectedValueStyle: yaml.DoubleQuotedStyle,
			description:        "should ignore numeric strings and use style from actual strings",
		},
		{
			name: "floats as strings",
			data: []byte(`
  key1: "3.14"
  key2: "2.71"
  key3: "text value"
  key4: "another text"
  key5: "third text"
`),
			doc: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Tag:  "!!map",
						Content: []*yaml.Node{
							{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "3.14", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key2", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "2.71", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key3", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "text value", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key4", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "another text", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key5", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "third text", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
						},
					},
				},
			},
			expectedValueStyle: yaml.DoubleQuotedStyle,
			description:        "should ignore float strings and use style from actual strings",
		},
		{
			name: "scientific notation as strings",
			data: []byte(`
  key1: "1e10"
  key2: "2.5e-3"
  key3: "regular text"
  key4: "more text"
  key5: "even more text"
`),
			doc: &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Tag:  "!!map",
						Content: []*yaml.Node{
							{Value: "key1", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "1e10", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key2", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "2.5e-3", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key3", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "regular text", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key4", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "more text", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
							{Value: "key5", Kind: yaml.ScalarNode, Tag: "!!str"},
							{Value: "even more text", Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
						},
					},
				},
			},
			expectedValueStyle: yaml.DoubleQuotedStyle,
			description:        "should ignore scientific notation strings and use style from actual strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := yml.GetConfigFromDoc(tt.data, tt.doc)

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedValueStyle, result.ValueStringStyle, tt.description)
		})
	}
}

func TestIndentationStyle_ToIndent_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		style    yml.IndentationStyle
		expected string
	}{
		{
			name:     "space style returns space character",
			style:    yml.IndentationStyleSpace,
			expected: " ",
		},
		{
			name:     "tab style returns tab character",
			style:    yml.IndentationStyleTab,
			expected: "\t",
		},
		{
			name:     "unknown style returns empty string",
			style:    yml.IndentationStyle("unknown"),
			expected: "",
		},
		{
			name:     "empty style returns empty string",
			style:    yml.IndentationStyle(""),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.style.ToIndent()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDefaultConfig_Success(t *testing.T) {
	t.Parallel()

	config := yml.GetDefaultConfig()

	require.NotNil(t, config, "GetDefaultConfig should not return nil")
	assert.Equal(t, 2, config.Indentation, "default indentation should be 2")
	assert.Equal(t, yml.IndentationStyleSpace, config.IndentationStyle, "default indentation style should be space")
	assert.Equal(t, yml.OutputFormatYAML, config.OutputFormat, "default output format should be YAML")
	assert.Equal(t, yaml.Style(0), config.KeyStringStyle, "default key string style should be 0")
	assert.Equal(t, yaml.Style(0), config.ValueStringStyle, "default value string style should be 0")
}
