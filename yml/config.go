package yml

import (
	"bytes"
	"context"
	"strconv"

	"gopkg.in/yaml.v3"
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

type IndentationStyle string

const (
	IndentationStyleSpace IndentationStyle = "space"
	IndentationStyleTab   IndentationStyle = "tab"
)

func (i IndentationStyle) ToIndent() string {
	switch i {
	case IndentationStyleSpace:
		return " "
	case IndentationStyleTab:
		return "\t"
	default:
		return ""
	}
}

type Config struct {
	KeyStringStyle   yaml.Style       // The default string style to use when creating new keys
	ValueStringStyle yaml.Style       // The default string style to use when creating new nodes
	Indentation      int              // The indentation level of the document
	IndentationStyle IndentationStyle // The indentation style of the document valid for JSON only
	OutputFormat     OutputFormat     // The output format to use when marshalling
	OriginalFormat   OutputFormat     // The original input format, helps detect when we are changing formats
	TrailingNewline  bool             // Whether the original document had a trailing newline
}

var defaultConfig = &Config{
	Indentation:      2,
	IndentationStyle: IndentationStyleSpace,
	KeyStringStyle:   0,
	ValueStringStyle: 0,
	OutputFormat:     OutputFormatYAML,
}

func GetDefaultConfig() *Config {
	return defaultConfig
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

	cfg.OutputFormat, cfg.Indentation, cfg.IndentationStyle = inspectData(data)
	cfg.OriginalFormat = cfg.OutputFormat

	// Check if the original data had a trailing newline
	cfg.TrailingNewline = len(data) > 0 && data[len(data)-1] == '\n'

	// Only extract string styles from the document if it's YAML
	// For JSON input, keep the default YAML styles
	if cfg.OriginalFormat == OutputFormatYAML {
		getGlobalStringStyle(doc, &cfg)
	}

	return &cfg
}

func inspectData(data []byte) (OutputFormat, int, IndentationStyle) {
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))

	foundIndentation := false
	foundDocFormat := false

	indentation := 2
	indentationStyle := IndentationStyleSpace
	docFormat := OutputFormatYAML

	// Track the minimum leading whitespace to establish baseline
	minLeadingWhitespace := -1

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
			// Calculate leading whitespace by counting from the start
			currentLeading := 0
			for currentLeading < len(line) && (line[currentLeading] == ' ' || line[currentLeading] == '\t') {
				currentLeading++
			}

			// Track minimum leading whitespace (baseline indentation)
			if minLeadingWhitespace == -1 || currentLeading < minLeadingWhitespace {
				minLeadingWhitespace = currentLeading
			}

			// Look for indentation relative to the baseline
			if currentLeading > minLeadingWhitespace && !foundIndentation {
				// Extract the indentation (difference from baseline)
				leadingWhitespace := line[minLeadingWhitespace:currentLeading]

				if len(leadingWhitespace) > 0 {
					// Check the first character to determine tab vs space
					if leadingWhitespace[0] == '\t' {
						indentationStyle = IndentationStyleTab
						// Count consecutive tabs
						indentation = 0
						for _, ch := range leadingWhitespace {
							if ch == '\t' {
								indentation++
							} else {
								break
							}
						}
					} else if leadingWhitespace[0] == ' ' {
						indentationStyle = IndentationStyleSpace
						// Count consecutive spaces
						indentation = 0
						for _, ch := range leadingWhitespace {
							if ch == ' ' {
								indentation++
							} else {
								break
							}
						}
					}
					foundIndentation = true
				}
			}
		}

		// If we have found everything we need or have iterated too long we can stop
		if foundIndentation && (foundDocFormat || i > 10) {
			break
		}
	}
	return docFormat, indentation, indentationStyle
}

func getGlobalStringStyle(doc *yaml.Node, cfg *Config) {
	const minSamples = 3

	keyStyles := make([]yaml.Style, 0, minSamples)
	valueStyles := make([]yaml.Style, 0, minSamples)

	var navigate func(node *yaml.Node)
	navigate = func(node *yaml.Node) {
		if len(keyStyles) >= minSamples && len(valueStyles) >= minSamples {
			return
		}

		switch node.Kind {
		case yaml.DocumentNode:
			navigate(node.Content[0])
		case yaml.SequenceNode:
			for _, n := range node.Content {
				navigate(n)
			}
		case yaml.MappingNode:
			for i, n := range node.Content {
				if i%2 == 0 {
					if n.Kind == yaml.ScalarNode && n.Tag == "!!str" && len(keyStyles) < minSamples {
						keyStyles = append(keyStyles, n.Style)
					}
				} else {
					navigate(n)
				}
			}
		case yaml.ScalarNode:
			if node.Tag == "!!str" && len(valueStyles) < minSamples {
				// Exclude quoted numbers - they need quotes but don't represent typical string style
				if !looksLikeNumber(node.Value) {
					valueStyles = append(valueStyles, node.Style)
				}
			}
		case yaml.AliasNode:
			navigate(node.Alias)
		default:
			panic("unknown node kind: " + NodeKindToString(node.Kind))
		}
	}

	navigate(doc)

	// Choose the most common style for keys
	if len(keyStyles) > 0 {
		cfg.KeyStringStyle = mostCommonStyle(keyStyles)
	}

	// Choose the most common style for values
	if len(valueStyles) > 0 {
		cfg.ValueStringStyle = mostCommonStyle(valueStyles)
	}
}

// looksLikeNumber returns true if the string value looks like a number
func looksLikeNumber(s string) bool {
	if s == "" {
		return false
	}

	// Try parsing as float (covers int, float, scientific notation)
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// mostCommonStyle returns the most frequently occurring style from the provided styles
func mostCommonStyle(styles []yaml.Style) yaml.Style {
	if len(styles) == 0 {
		return 0
	}

	counts := make(map[yaml.Style]int)
	for _, style := range styles {
		counts[style]++
	}

	// Find the style with the highest count
	var maxCount int
	var mostCommon yaml.Style
	for style, count := range counts {
		if count > maxCount {
			maxCount = count
			mostCommon = style
		}
	}

	return mostCommon
}
