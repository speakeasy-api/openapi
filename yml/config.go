package yml

import (
	"bytes"
	"context"

	"go.yaml.in/yaml/v4"
)

type contextKey string

func (c contextKey) String() string {
	return "yml-context-key-" + string(c)
}

const configContextKey = contextKey("config")

type OutputFormat string

const (
	OutputFormatJSON OutputFormat = "json"
	OutputFormatYAML OutputFormat = "yaml"
)

type Config struct {
	KeyStringStyle   yaml.Style   // The default string style to use when creating new keys
	ValueStringStyle yaml.Style   // The default string style to use when creating new nodes
	Indentation      int          // The indentation level of the document
	OutputFormat     OutputFormat // The output format to use when marshalling
	OriginalFormat   OutputFormat // The original input format, helps detect when we are changing formats
}

var defaultConfig = &Config{
	Indentation:      2,
	KeyStringStyle:   0,
	ValueStringStyle: 0,
	OutputFormat:     OutputFormatYAML,
}

func ContextWithConfig(ctx context.Context, config *Config) context.Context {
	if config == nil {
		return ctx
	}

	return context.WithValue(ctx, configContextKey, config)
}

func GetConfigFromContext(ctx context.Context) *Config {
	val := ctx.Value(configContextKey)
	if val == nil {
		def := *defaultConfig
		return &def
	}

	cfg, ok := val.(*Config)
	if !ok {
		def := *defaultConfig
		return &def
	}

	return cfg
}

func GetConfigFromDoc(data []byte, doc *yaml.Node) *Config {
	cfg := *defaultConfig

	cfg.OutputFormat, cfg.Indentation = inspectData(data)
	cfg.OriginalFormat = cfg.OutputFormat

	// Only extract string styles from the document if it's YAML
	// For JSON input, keep the default YAML styles
	if cfg.OriginalFormat == OutputFormatYAML {
		getGlobalStringStyle(doc, &cfg)
	}

	return &cfg
}

func inspectData(data []byte) (OutputFormat, int) {
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))

	foundIndentation := false
	foundDocFormat := false

	indentation := 2
	docFormat := OutputFormatYAML

	for i, line := range lines {
		trimLine := bytes.TrimSpace(line)

		if len(trimLine) == 0 {
			continue
		}

		switch trimLine[0] {
		case '#':
			continue
		case '{':
			docFormat = OutputFormatJSON
			foundDocFormat = true
		default:
			if len(line) != len(trimLine) && !foundIndentation {
				indentation = len(line) - len(trimLine)
				foundIndentation = true
			}
		}

		// If we have found everything we need or have iterated too long we can stop
		if foundIndentation && (foundDocFormat || i > 5) {
			break
		}
	}
	return docFormat, indentation
}

func getGlobalStringStyle(doc *yaml.Node, cfg *Config) {
	foundMapKeyStyle := false
	foundStringValueStyle := false

	var navigate func(node *yaml.Node)
	navigate = func(node *yaml.Node) {
		switch node.Kind {
		case yaml.DocumentNode:
			navigate(node.Content[0])
		case yaml.SequenceNode:
			for _, n := range node.Content {
				navigate(n)

				if foundMapKeyStyle && foundStringValueStyle {
					return
				}
			}
		case yaml.MappingNode:
			for i, n := range node.Content {
				if i%2 == 0 {
					if n.Kind == yaml.ScalarNode && n.Tag == "!!str" {
						cfg.KeyStringStyle = n.Style
						foundMapKeyStyle = true
					}
				} else {
					navigate(n)
					if foundMapKeyStyle && foundStringValueStyle {
						return
					}
				}
			}
		case yaml.ScalarNode:
			if node.Tag == "!!str" {
				cfg.ValueStringStyle = node.Style
				foundStringValueStyle = true
			}
		case yaml.AliasNode:
			navigate(node.Alias)
		default:
			panic("unknown node kind: " + NodeKindToString(node.Kind))
		}
	}

	navigate(doc)
}
