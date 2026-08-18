package sortfmt

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"

	"gopkg.in/yaml.v3"
)

// Write serializes a sorted YAML node using the JSON representation produced
// by Python's json.dump with indent=2, followed by one trailing newline.
func Write(node *yaml.Node, w io.Writer) error {
	writer := &jsonWriter{w: w}
	if err := writer.writeNode(node, 0); err != nil {
		return err
	}
	writer.writeString("\n")
	return writer.err
}

type jsonWriter struct {
	w   io.Writer
	err error
}

func (w *jsonWriter) writeString(value string) {
	if w.err != nil {
		return
	}
	written, err := io.WriteString(w.w, value)
	if err == nil && written != len(value) {
		err = io.ErrShortWrite
	}
	w.err = err
}

func (w *jsonWriter) writeIndent(depth int) {
	w.writeString(strings.Repeat("  ", depth))
}

func (w *jsonWriter) writeNode(node *yaml.Node, depth int) error {
	if node == nil {
		return errors.New("cannot write a nil node")
	}

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) != 1 {
			return errors.New("document must contain exactly one value")
		}
		return w.writeNode(node.Content[0], depth)
	case yaml.MappingNode:
		return w.writeMapping(node, depth)
	case yaml.SequenceNode:
		return w.writeSequence(node, depth)
	case yaml.ScalarNode:
		return w.writeScalar(node)
	default:
		return fmt.Errorf("unsupported YAML node kind %d", node.Kind)
	}
}

func (w *jsonWriter) writeMapping(node *yaml.Node, depth int) error {
	if len(node.Content)%2 != 0 {
		return errors.New("mapping has an unmatched key")
	}
	if len(node.Content) == 0 {
		w.writeString("{}")
		return w.err
	}

	w.writeString("{\n")
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]
		if keyNode == nil || valueNode == nil {
			return errors.New("mapping contains a nil key or value")
		}
		if keyNode.Kind != yaml.ScalarNode || keyNode.Tag != "!!str" {
			return errors.New("JSON object key must be a string")
		}

		w.writeIndent(depth + 1)
		w.writeString(quoteJSONString(keyNode.Value))
		w.writeString(": ")
		if err := w.writeNode(valueNode, depth+1); err != nil {
			return err
		}
		if i+2 < len(node.Content) {
			w.writeString(",")
		}
		w.writeString("\n")
	}
	w.writeIndent(depth)
	w.writeString("}")
	return w.err
}

func (w *jsonWriter) writeSequence(node *yaml.Node, depth int) error {
	if len(node.Content) == 0 {
		w.writeString("[]")
		return w.err
	}

	w.writeString("[\n")
	for i, child := range node.Content {
		w.writeIndent(depth + 1)
		if err := w.writeNode(child, depth+1); err != nil {
			return err
		}
		if i+1 < len(node.Content) {
			w.writeString(",")
		}
		w.writeString("\n")
	}
	w.writeIndent(depth)
	w.writeString("]")
	return w.err
}

func (w *jsonWriter) writeScalar(node *yaml.Node) error {
	switch node.Tag {
	case "!!str":
		w.writeString(quoteJSONString(node.Value))
	case "!!bool":
		value, err := boolValue(node.Value)
		if err != nil {
			return err
		}
		if value {
			w.writeString("true")
		} else {
			w.writeString("false")
		}
	case "!!null":
		w.writeString("null")
	case "!!int":
		value, err := formatInteger(node.Value)
		if err != nil {
			return err
		}
		w.writeString(value)
	case "!!float":
		value, err := formatFloat(node.Value)
		if err != nil {
			return err
		}
		w.writeString(value)
	default:
		return fmt.Errorf("unsupported scalar tag %s", node.Tag)
	}
	return w.err
}

func quoteJSONString(value string) string {
	var result strings.Builder
	result.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"':
			result.WriteString(`\"`)
		case '\\':
			result.WriteString(`\\`)
		case '\b':
			result.WriteString(`\b`)
		case '\t':
			result.WriteString(`\t`)
		case '\n':
			result.WriteString(`\n`)
		case '\f':
			result.WriteString(`\f`)
		case '\r':
			result.WriteString(`\r`)
		default:
			switch {
			case r >= 0x20 && r <= 0x7e:
				result.WriteRune(r)
			case r <= 0xffff:
				fmt.Fprintf(&result, `\u%04x`, r)
			default:
				high, low := utf16.EncodeRune(r)
				fmt.Fprintf(&result, `\u%04x\u%04x`, high, low)
			}
		}
	}
	result.WriteByte('"')
	return result.String()
}

func pythonFloat(value float64) (string, error) {
	if math.IsNaN(value) {
		return "NaN", nil
	}
	if math.IsInf(value, 1) {
		return "Infinity", nil
	}
	if math.IsInf(value, -1) {
		return "-Infinity", nil
	}
	if value == 0 {
		if math.Signbit(value) {
			return "-0.0", nil
		}
		return "0.0", nil
	}

	formatted := strconv.FormatFloat(value, 'g', -1, 64)
	exponentIndex := strings.IndexByte(formatted, 'e')
	if exponentIndex == -1 {
		if !strings.ContainsRune(formatted, '.') {
			formatted += ".0"
		}
		return formatted, nil
	}

	mantissa := formatted[:exponentIndex]
	exponent, err := strconv.Atoi(formatted[exponentIndex+1:])
	if err != nil {
		return "", fmt.Errorf("parse float exponent %q: %w", formatted, err)
	}

	if exponent >= -4 && exponent < 16 {
		return expandFloat(mantissa, exponent), nil
	}

	exponentSign := "+"
	if exponent < 0 {
		exponentSign = "-"
		exponent = -exponent
	}
	return mantissa + "e" + exponentSign + fmt.Sprintf("%02d", exponent), nil
}

func expandFloat(mantissa string, exponent int) string {
	sign := ""
	if strings.HasPrefix(mantissa, "-") {
		sign = "-"
		mantissa = strings.TrimPrefix(mantissa, "-")
	}

	digits := strings.ReplaceAll(mantissa, ".", "")
	decimalPosition := exponent + 1
	switch {
	case decimalPosition <= 0:
		return sign + "0." + strings.Repeat("0", -decimalPosition) + digits
	case decimalPosition >= len(digits):
		return sign + digits + strings.Repeat("0", decimalPosition-len(digits)) + ".0"
	default:
		return sign + digits[:decimalPosition] + "." + digits[decimalPosition:]
	}
}
