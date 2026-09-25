package yml

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// Block indentation is stripped on decode, so leading whitespace means more-indented.
func hasMoreIndentedLine(value string) bool {
	for _, line := range strings.Split(value, "\n") {
		if line == "" {
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			return true
		}
	}

	return false
}

// StabilizeFoldedScalars restyles folded block scalars that contain a more-indented
// line as literal blocks, leaving every other node untouched.
//
// gopkg.in/yaml.v3 injects a line break before a more-indented line in a folded scalar
// on every encode, so a document that is decoded and re-encoded repeatedly accumulates
// blank lines inside such scalars. The break lands inside the scalar, so it becomes part
// of the decoded value rather than cosmetic whitespace. Literal blocks reproduce their
// value verbatim and round trip unchanged.
//
// Only the representation changes; the decoded value is identical. Call this immediately
// before encoding a node tree.
func StabilizeFoldedScalars(node *yaml.Node) {
	if node == nil {
		return
	}

	if node.Kind == yaml.ScalarNode && node.Style&yaml.FoldedStyle != 0 && hasMoreIndentedLine(node.Value) {
		node.Style = node.Style&^yaml.FoldedStyle | yaml.LiteralStyle
	}

	for _, child := range node.Content {
		StabilizeFoldedScalars(child)
	}
}
