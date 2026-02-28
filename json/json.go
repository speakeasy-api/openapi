package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v4"
)

// YAMLToJSON converts a YAML node to JSON using a custom JSON writer that preserves formatting.
// This approach is particularly useful when the YAML nodes already represent JSON input (since
// the yaml decoder can parse JSON), and we want to preserve the original JSON formatting.
func YAMLToJSON(node *yaml.Node, indentation int, buffer io.Writer) error {
	return YAMLToJSONWithConfig(node, " ", indentation, true, buffer)
}

// YAMLToJSONWithConfig converts YAML to JSON with full control over formatting.
// When the input nodes already have JSON-style formatting (from JSON input), this preserves it
// by analyzing Line and Column metadata to recreate the original formatting.
func YAMLToJSONWithConfig(node *yaml.Node, indent string, indentCount int, trailingNewline bool, buffer io.Writer) error {
	if node == nil {
		return nil
	}

	// Build the indent string
	indentStr := strings.Repeat(indent, indentCount)

	// Create a JSON writer context
	ctx := &jsonWriteContext{
		indent:       indentStr,
		buffer:       &bytes.Buffer{},
		currentCol:   0,
		forceCompact: len(indentStr) == 0, // Force compact mode when no indentation
	}

	// Write the JSON
	if err := writeJSONNode(ctx, node, 0); err != nil {
		return err
	}

	// Get the output
	output := ctx.buffer.Bytes()

	// Add or remove trailing newline as requested
	if trailingNewline && (len(output) == 0 || output[len(output)-1] != '\n') {
		output = append(output, '\n')
	} else if !trailingNewline && len(output) > 0 && output[len(output)-1] == '\n' {
		output = output[:len(output)-1]
	}

	_, err := buffer.Write(output)
	return err
}

type jsonWriteContext struct {
	indent       string
	buffer       *bytes.Buffer
	currentCol   int
	forceCompact bool // When true, always output compact format
}

// isSingleLineFlowNode checks if a flow-style node is on a single line
func isSingleLineFlowNode(node *yaml.Node) bool {
	if node.Style != yaml.FlowStyle {
		return false
	}

	if len(node.Content) == 0 {
		return true
	}

	// Check if all children are on the same line as the parent
	nodeLine := node.Line
	for _, child := range node.Content {
		if child.Line != nodeLine {
			return false
		}
	}

	return true
}

// hasSpaceAfterColon checks if there's a space after the colon in a mapping node
// Returns true unless we can definitively detect NO space (compact JSON)
func hasSpaceAfterColon(node *yaml.Node) bool {
	if node.Kind != yaml.MappingNode || len(node.Content) < 2 {
		return true // Default to having space
	}

	key := node.Content[0]
	value := node.Content[1]

	if key.Line != value.Line {
		return true // Multi-line, doesn't matter
	}

	// Based on inspection: YAML flow uses Style=Default, JSON uses Style=DoubleQuoted
	// Calculate the width of the key in the source
	var keyWidth int
	if key.Style == yaml.DoubleQuotedStyle || key.Style == yaml.SingleQuotedStyle {
		// Quoted keys (JSON): need to account for quotes
		// Column points to opening quote, width includes both quotes
		keyWidth = len(strconv.Quote(key.Value))
	} else {
		// Unquoted keys (YAML flow-style)
		keyWidth = len(key.Value)
	}

	// Expected column for value with NO space after colon:
	// key.Column + keyWidth + 1 (for colon)
	expectedNoSpaceCol := key.Column + keyWidth + 1

	// If value starts AFTER expectedNoSpaceCol, there's a space
	// If value starts AT expectedNoSpaceCol, there's NO space
	return value.Column > expectedNoSpaceCol
}

// hasSpaceAfterComma checks if there's a space after commas in a sequence node
// Returns true unless we can definitively detect NO space (compact JSON)
func hasSpaceAfterComma(node *yaml.Node) bool {
	if node.Kind != yaml.SequenceNode || len(node.Content) < 2 {
		return true // Default to having space
	}

	first := node.Content[0]
	second := node.Content[1]

	if first.Line != second.Line {
		return true // Multi-line, doesn't matter
	}

	// Calculate width of first element in source
	var firstWidth int
	if first.Kind == yaml.ScalarNode {
		if first.Style == yaml.DoubleQuotedStyle || first.Style == yaml.SingleQuotedStyle {
			// Quoted strings (JSON): account for quotes
			firstWidth = len(strconv.Quote(first.Value))
		} else {
			// Unquoted values (YAML flow-style or numbers)
			firstWidth = len(first.Value)
		}
	} else {
		// For nested structures, default to having space
		return true
	}

	// Expected column for second element with NO space after comma:
	// first.Column + firstWidth + 1 (for comma)
	expectedNoSpaceCol := first.Column + firstWidth + 1

	// If second starts AFTER expectedNoSpaceCol, there's a space
	return second.Column > expectedNoSpaceCol
}

func (ctx *jsonWriteContext) write(s string) {
	ctx.buffer.WriteString(s)
	// Track column (simplified - doesn't handle newlines in s)
	ctx.currentCol += len(s)
}

func (ctx *jsonWriteContext) writeByte(b byte) {
	ctx.buffer.WriteByte(b)
	if b == '\n' {
		ctx.currentCol = 0
	} else {
		ctx.currentCol++
	}
}

func writeJSONNode(ctx *jsonWriteContext, node *yaml.Node, depth int) error {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.DocumentNode:
		// Unwrap document node
		if len(node.Content) > 0 {
			return writeJSONNode(ctx, node.Content[0], depth)
		}
		return nil

	case yaml.MappingNode:
		return writeJSONObject(ctx, node, depth)

	case yaml.SequenceNode:
		return writeJSONArray(ctx, node, depth)

	case yaml.ScalarNode:
		return writeJSONScalar(ctx, node)

	case yaml.AliasNode:
		// Resolve alias using yml package helper
		resolved := yml.ResolveAlias(node)
		if resolved != nil {
			return writeJSONNode(ctx, resolved, depth)
		}
		return nil

	default:
		return fmt.Errorf("unknown node kind: %v", node.Kind)
	}
}

func writeJSONObject(ctx *jsonWriteContext, node *yaml.Node, depth int) error {
	if len(node.Content) == 0 {
		ctx.write("{}")
		return nil
	}

	// Resolve merge keys first
	mergedContent := resolveMergeKeys(node.Content)

	// Check if this is a single-line flow node (preserve spacing)
	isSingleLine := isSingleLineFlowNode(node)
	preserveSpacing := isSingleLine && node.Style == yaml.FlowStyle

	// Determine if we should format as multi-line
	isMultiLine := !preserveSpacing &&
		((node.Style != yaml.FlowStyle && !ctx.forceCompact) ||
			shouldBeMultiLine(ctx, node, mergedContent))

	ctx.writeByte('{')

	if isMultiLine {
		ctx.writeByte('\n')
	}

	firstItem := true
	// Process key-value pairs (using merged content)
	for i := 0; i < len(mergedContent); i += 2 {
		if i+1 >= len(mergedContent) {
			break
		}

		keyNode := mergedContent[i]
		valueNode := mergedContent[i+1]

		// Add comma before this item if not the first
		if !firstItem {
			ctx.writeByte(',')
			if isMultiLine {
				ctx.writeByte('\n')
			} else if !ctx.forceCompact {
				// For single-line flow nodes, preserve original spacing
				if preserveSpacing && !hasSpaceAfterColon(node) {
					// No space in original
				} else {
					// Add space (default or original had space)
					ctx.writeByte(' ')
				}
			}
		}
		firstItem = false

		// Add indentation for multi-line
		if isMultiLine {
			ctx.write(strings.Repeat(ctx.indent, depth+1))
		}

		// Write key - always as a quoted string (JSON requirement)
		ctx.write(quoteJSONString(keyNode.Value))

		// Add space after colon based on context
		switch {
		case ctx.forceCompact:
			ctx.write(":")
		case preserveSpacing:
			// Preserve original spacing for single-line flow nodes
			if hasSpaceAfterColon(node) {
				ctx.write(": ")
			} else {
				ctx.write(":")
			}
		default:
			// Default: add space
			ctx.write(": ")
		}

		// Write value
		if err := writeJSONNode(ctx, valueNode, depth+1); err != nil {
			return err
		}
	}

	if isMultiLine {
		ctx.writeByte('\n')
		ctx.write(strings.Repeat(ctx.indent, depth))
	}
	ctx.writeByte('}')

	return nil
}

func writeJSONArray(ctx *jsonWriteContext, node *yaml.Node, depth int) error {
	if len(node.Content) == 0 {
		ctx.write("[]")
		return nil
	}

	// Check if this is a single-line flow node (preserve spacing)
	isSingleLine := isSingleLineFlowNode(node)
	preserveSpacing := isSingleLine && node.Style == yaml.FlowStyle

	// Determine if we should format as multi-line
	isMultiLine := !preserveSpacing &&
		((node.Style != yaml.FlowStyle && !ctx.forceCompact) ||
			shouldBeMultiLine(ctx, node, node.Content))

	ctx.writeByte('[')

	if isMultiLine {
		ctx.writeByte('\n')
	}

	for i, child := range node.Content {
		// Add indentation for multi-line
		if isMultiLine {
			ctx.write(strings.Repeat(ctx.indent, depth+1))
		}

		if err := writeJSONNode(ctx, child, depth+1); err != nil {
			return err
		}

		// Add comma if not the last item
		if i+1 < len(node.Content) {
			ctx.writeByte(',')
			if isMultiLine {
				ctx.writeByte('\n')
			} else if !ctx.forceCompact {
				// For single-line flow nodes, preserve original spacing
				if preserveSpacing && !hasSpaceAfterComma(node) {
					// No space in original
				} else {
					// Add space (default or original had space)
					ctx.writeByte(' ')
				}
			}
		} else if isMultiLine {
			ctx.writeByte('\n')
		}
	}

	if isMultiLine {
		ctx.write(strings.Repeat(ctx.indent, depth))
	}
	ctx.writeByte(']')

	return nil
}

func writeJSONScalar(ctx *jsonWriteContext, node *yaml.Node) error {
	switch node.Tag {
	case "!!str":
		// JSON string - must be quoted and escaped
		ctx.write(quoteJSONString(node.Value))
		return nil

	case "!!int":
		// Check for invalid JSON number formats (e.g., leading zeros)
		// In YAML, "009911" might be parsed as int, but JSON doesn't allow leading zeros
		// Treat such values as strings to preserve them correctly
		if hasInvalidJSONNumberFormat(node.Value) {
			ctx.write(quoteJSONString(node.Value))
			return nil
		}
		ctx.write(node.Value)
		return nil

	case "!!float":
		// Check for invalid JSON number formats (e.g., leading zeros)
		// YAML may tag values like "009911" as float, but they're invalid in JSON
		// Treat such values as strings to preserve them correctly
		if hasInvalidJSONNumberFormat(node.Value) {
			ctx.write(quoteJSONString(node.Value))
			return nil
		}
		ctx.write(node.Value)
		return nil

	case "!!bool":
		// Booleans true/True/TRUE to true compatible with JSON
		ctx.write(strings.ToLower(node.Value))
		return nil

	case "!!null":
		// Null
		ctx.write("null")
		return nil

	default:
		// Default to quoted string
		ctx.write(quoteJSONString(node.Value))
		return nil
	}
}

// hasInvalidJSONNumberFormat checks if a numeric string would be invalid in JSON.
// JSON numbers cannot have leading zeros (except for "0" itself or "0.x" floats).
func hasInvalidJSONNumberFormat(value string) bool {
	if len(value) < 2 {
		return false
	}

	// Handle negative numbers
	start := 0
	if value[0] == '-' || value[0] == '+' {
		start = 1
		if len(value) < 2 {
			return false
		}
	}

	// Check for leading zero followed by more digits (invalid in JSON)
	// "0" alone is valid, "0.x" is valid, but "00", "01", "007" etc. are not
	if value[start] == '0' && len(value) > start+1 && value[start+1] >= '0' && value[start+1] <= '9' {
		return true
	}

	return false
}

// resolveMergeKeys processes YAML merge keys (<<) and returns content with merged values
func resolveMergeKeys(content []*yaml.Node) []*yaml.Node {
	if len(content) == 0 {
		return content
	}

	result := make([]*yaml.Node, 0, len(content))
	mergedKeys := make(map[string]bool) // Track which keys have been merged

	// First pass: collect all merge key content
	var mergeContent []*yaml.Node
	for i := 0; i < len(content); i += 2 {
		if i+1 >= len(content) {
			break
		}

		keyNode := content[i]
		valueNode := content[i+1]

		// Check for merge key
		if keyNode.Value == "<<" {
			// Resolve the alias to get the merged content
			resolved := yml.ResolveAlias(valueNode)
			if resolved != nil && resolved.Kind == yaml.MappingNode {
				// Add all key-value pairs from the merged content
				for j := 0; j < len(resolved.Content); j += 2 {
					if j+1 < len(resolved.Content) {
						mergeKey := resolved.Content[j]
						mergeValue := resolved.Content[j+1]
						if !mergedKeys[mergeKey.Value] {
							mergeContent = append(mergeContent, mergeKey, mergeValue)
							mergedKeys[mergeKey.Value] = true
						}
					}
				}
			}
		}
	}

	// Second pass: add merged content first, then original content (original overrides merged)
	result = append(result, mergeContent...)

	// Add non-merge keys
	for i := 0; i < len(content); i += 2 {
		if i+1 >= len(content) {
			break
		}

		keyNode := content[i]
		valueNode := content[i+1]

		// Skip merge keys themselves
		if keyNode.Value == "<<" {
			continue
		}

		// Add this key-value pair (it will override any merged value with same key)
		result = append(result, keyNode, valueNode)
	}

	return result
}

// shouldBeMultiLine determines if a node's children should be formatted on multiple lines
// by checking if the first child is on a different line than the parent, OR if children
// are on different lines from each other
func shouldBeMultiLine(ctx *jsonWriteContext, parent *yaml.Node, children []*yaml.Node) bool {
	// Force compact if requested
	if ctx.forceCompact {
		return false
	}

	if len(children) == 0 {
		return false
	}

	// Check if first child is on different line than parent
	firstChild := children[0]
	if firstChild.Line != parent.Line {
		return true
	}

	// Also check if children are on different lines from each other
	for _, node := range children[1:] {
		if node.Line != firstChild.Line {
			return true
		}
	}

	return false
}

// quoteJSONString properly quotes and escapes a string for JSON output
func quoteJSONString(s string) string {
	// Use json.Encoder with SetEscapeHTML(false) for proper JSON escaping without HTML entity encoding
	// This handles nul characters correctly (\u0000) while keeping & as & instead of \u0026
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(s); err != nil {
		// Fallback to strconv.Quote if encoding fails (shouldn't happen for strings)
		return strconv.Quote(s)
	}

	// Encoder.Encode adds a newline, so we need to trim it
	result := buf.String()
	return strings.TrimSuffix(result, "\n")
}
