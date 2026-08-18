// Package sortfmt provides deterministic JSON formatting for OpenAPI documents.
//
// The formatter orders mapping keys using a fixed priority list, alphabetizes
// maps under paths, properties, and schemas, and sorts selected arrays by a
// stable display key. It intentionally changes the order of parameters and
// composition arrays, so callers should opt in only when sorted output is more
// important than preserving authored array order. Selected arrays whose sort
// keys have different JSON types return an error, matching Python's ordering
// behavior. Format also rejects lone UTF-16 surrogate escapes rather than
// silently replacing them with a different character.
package sortfmt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

var alphabetizedMapKeys = map[string]struct{}{
	"paths":      {},
	"properties": {},
	"schemas":    {},
}

var sortedListKeys = map[string]struct{}{
	"required":   {},
	"parameters": {},
	"oneOf":      {},
	"anyOf":      {},
	"allOf":      {},
}

var keyPriority = func() map[string]int {
	keys := []string{
		"openapi",
		"info",
		"servers",
		"paths",
		"components",
		"title",
		"url",
		"name",
		"in",
		"operationId",
		"description",
		"parameters",
		"type",
		"properties",
		"required",
		"examples",
	}

	result := make(map[string]int, len(keys))
	for i, key := range keys {
		result[key] = i
	}
	return result
}()

// Format reads a JSON document, applies the sorted formatting style, and
// writes two-space-indented JSON with a trailing newline.
func Format(r io.Reader, w io.Writer) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read document: %w", err)
	}

	document, err := parseJSON(data)
	if err != nil {
		return err
	}

	if err := Sort(document); err != nil {
		return fmt.Errorf("sort document: %w", err)
	}

	if err := Write(document, w); err != nil {
		return fmt.Errorf("write document: %w", err)
	}

	return nil
}

func parseJSON(data []byte) (*yaml.Node, error) {
	if !utf8.Valid(data) {
		return nil, errors.New("parse JSON document: input is not valid UTF-8")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("parse JSON document: %w", err)
	}

	if err := decoder.Decode(&value); err != io.EOF {
		if err == nil {
			return nil, errors.New("parse JSON document: multiple JSON values")
		}
		return nil, fmt.Errorf("parse JSON document: %w", err)
	}

	if offset, ok := findLoneSurrogateEscape(data); ok {
		return nil, fmt.Errorf("parse JSON document: lone surrogate escape at byte %d is not supported", offset)
	}

	content, err := jsonValueToNode(value)
	if err != nil {
		return nil, fmt.Errorf("parse JSON document: %w", err)
	}

	return &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{content},
	}, nil
}

func findLoneSurrogateEscape(data []byte) (int, bool) {
	inString := false
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '"':
			inString = !inString
		case '\\':
			if !inString || i+1 >= len(data) {
				continue
			}
			if data[i+1] != 'u' {
				i++
				continue
			}

			codePoint, ok := parseHexCodeUnit(data, i+2)
			if !ok {
				continue
			}
			switch {
			case codePoint >= 0xd800 && codePoint <= 0xdbff:
				nextEscape := i + 6
				if nextEscape+5 >= len(data) || data[nextEscape] != '\\' || data[nextEscape+1] != 'u' {
					return i, true
				}
				lowSurrogate, validLow := parseHexCodeUnit(data, nextEscape+2)
				if !validLow || lowSurrogate < 0xdc00 || lowSurrogate > 0xdfff {
					return i, true
				}
				i = nextEscape + 5
			case codePoint >= 0xdc00 && codePoint <= 0xdfff:
				return i, true
			default:
				i += 5
			}
		}
	}
	return 0, false
}

func parseHexCodeUnit(data []byte, start int) (uint16, bool) {
	if start+4 > len(data) {
		return 0, false
	}

	var result uint16
	for _, digit := range data[start : start+4] {
		result <<= 4
		switch {
		case digit >= '0' && digit <= '9':
			result |= uint16(digit - '0')
		case digit >= 'a' && digit <= 'f':
			result |= uint16(digit-'a') + 10
		case digit >= 'A' && digit <= 'F':
			result |= uint16(digit-'A') + 10
		default:
			return 0, false
		}
	}
	return result, true
}

func jsonValueToNode(value any) (*yaml.Node, error) {
	switch typedValue := value.(type) {
	case map[string]any:
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for key, childValue := range typedValue {
			child, err := jsonValueToNode(childValue)
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
				child,
			)
		}
		return node, nil
	case []any:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, childValue := range typedValue {
			child, err := jsonValueToNode(childValue)
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, child)
		}
		return node, nil
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: typedValue}, nil
	case json.Number:
		tag := "!!int"
		if strings.ContainsAny(string(typedValue), ".eE") {
			tag = "!!float"
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: string(typedValue)}, nil
	case bool:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(typedValue)}, nil
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}, nil
	default:
		return nil, fmt.Errorf("unsupported JSON value type %T", value)
	}
}

// Sort applies the sorted formatting order to a YAML node in place. The node
// must represent data parsed from JSON. Duplicate object keys retain their last
// value, matching encoding/json and Python's json module.
func Sort(node *yaml.Node) error {
	return sortNode(node, "")
}

func sortNode(node *yaml.Node, parentKey string) error {
	if node == nil {
		return errors.New("cannot sort a nil node")
	}

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) != 1 {
			return errors.New("document must contain exactly one value")
		}
		return sortNode(node.Content[0], parentKey)
	case yaml.MappingNode:
		return sortMapping(node, parentKey)
	case yaml.SequenceNode:
		return sortSequence(node, parentKey)
	case yaml.ScalarNode:
		return nil
	default:
		return fmt.Errorf("unsupported YAML node kind %d", node.Kind)
	}
}

type mappingPair struct {
	key   *yaml.Node
	value *yaml.Node
}

func sortMapping(node *yaml.Node, parentKey string) error {
	if len(node.Content)%2 != 0 {
		return errors.New("mapping has an unmatched key")
	}

	byKey := make(map[string]mappingPair, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]
		if keyNode == nil || valueNode == nil {
			return errors.New("mapping contains a nil key or value")
		}
		if keyNode.Kind != yaml.ScalarNode || keyNode.Tag != "!!str" {
			return errors.New("JSON object key must be a string")
		}
		byKey[keyNode.Value] = mappingPair{key: keyNode, value: valueNode}
	}

	pairs := make([]mappingPair, 0, len(byKey))
	for _, pair := range byKey {
		pairs = append(pairs, pair)
	}

	_, alphabetize := alphabetizedMapKeys[parentKey]
	sort.Slice(pairs, func(i, j int) bool {
		left := pairs[i].key.Value
		right := pairs[j].key.Value
		if alphabetize {
			return left < right
		}

		leftPriority, leftPrioritized := keyPriority[left]
		if !leftPrioritized {
			leftPriority = len(keyPriority)
		}
		rightPriority, rightPrioritized := keyPriority[right]
		if !rightPrioritized {
			rightPriority = len(keyPriority)
		}

		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		return left < right
	})

	sortedContent := make([]*yaml.Node, 0, len(pairs)*2)
	for _, pair := range pairs {
		if err := sortNode(pair.value, pair.key.Value); err != nil {
			return err
		}
		sortedContent = append(sortedContent, pair.key, pair.value)
	}
	node.Content = sortedContent

	return nil
}

func sortSequence(node *yaml.Node, parentKey string) error {
	for i, child := range node.Content {
		if child == nil {
			return fmt.Errorf("%s item %d is nil", parentKey, i)
		}
		if err := sortNode(child, parentKey); err != nil {
			return err
		}
	}

	if _, ok := sortedListKeys[parentKey]; !ok {
		return nil
	}
	if len(node.Content) < 2 {
		return nil
	}

	keys := make([]pythonSortKey, len(node.Content))
	for i, child := range node.Content {
		key, err := listSortKey(child)
		if err != nil {
			return fmt.Errorf("create sort key for %s item %d: %w", parentKey, i, err)
		}
		keys[i] = key
	}

	type sequenceItem struct {
		node *yaml.Node
		key  pythonSortKey
	}
	items := make([]sequenceItem, len(node.Content))
	for i, child := range node.Content {
		items[i] = sequenceItem{node: child, key: keys[i]}
	}

	var comparisonErr error
	sort.SliceStable(items, func(i, j int) bool {
		if comparisonErr != nil {
			return false
		}
		less, err := items[i].key.less(items[j].key)
		if err != nil {
			comparisonErr = err
			return false
		}
		return less
	})
	if comparisonErr != nil {
		return fmt.Errorf("sort %s items: %w", parentKey, comparisonErr)
	}
	for i, item := range items {
		node.Content[i] = item.node
	}

	return nil
}

type pythonSortKeyKind int

const (
	pythonSortKeyString pythonSortKeyKind = iota
	pythonSortKeyStringList
)

type pythonSortKey struct {
	kind        pythonSortKeyKind
	stringValue string
	stringList  []string
}

func newPythonStringSortKey(value string) pythonSortKey {
	return pythonSortKey{kind: pythonSortKeyString, stringValue: value}
}

func (k pythonSortKey) less(other pythonSortKey) (bool, error) {
	if k.kind != other.kind {
		return false, errors.New("cannot compare selected-list sort keys of different JSON types")
	}

	switch k.kind {
	case pythonSortKeyString:
		return k.stringValue < other.stringValue, nil
	case pythonSortKeyStringList:
		for i := 0; i < min(len(k.stringList), len(other.stringList)); i++ {
			if k.stringList[i] != other.stringList[i] {
				return k.stringList[i] < other.stringList[i], nil
			}
		}
		return len(k.stringList) < len(other.stringList), nil
	default:
		return false, fmt.Errorf("unsupported selected-list sort key kind %d", k.kind)
	}
}

func listSortKey(node *yaml.Node) (pythonSortKey, error) {
	if node.Kind == yaml.MappingNode {
		for _, key := range []string{"name", "$ref", "type"} {
			if value, ok := mappingValue(node, key); ok {
				switch {
				case value.Kind == yaml.ScalarNode && value.Tag == "!!str":
					return newPythonStringSortKey(value.Value), nil
				case value.Kind == yaml.SequenceNode:
					items := make([]string, len(value.Content))
					for i, item := range value.Content {
						if item.Kind != yaml.ScalarNode || item.Tag != "!!str" {
							return pythonSortKey{}, fmt.Errorf("%s array sort key item %d must be a string", key, i)
						}
						items[i] = item.Value
					}
					return pythonSortKey{kind: pythonSortKeyStringList, stringList: items}, nil
				default:
					return pythonSortKey{}, fmt.Errorf("%s sort key must be a string or string array", key)
				}
			}
		}
		value, err := pythonRepr(node)
		return newPythonStringSortKey(value), err
	}

	if node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
		return newPythonStringSortKey(node.Value), nil
	}

	value, err := pythonRepr(node)
	return newPythonStringSortKey(value), err
}

func mappingValue(node *yaml.Node, key string) (*yaml.Node, bool) {
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1], true
		}
	}
	return nil, false
}

func pythonRepr(node *yaml.Node) (string, error) {
	switch node.Kind {
	case yaml.MappingNode:
		parts := make([]string, 0, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			value, err := pythonRepr(node.Content[i+1])
			if err != nil {
				return "", err
			}
			parts = append(parts, quotePythonString(node.Content[i].Value)+": "+value)
		}
		return "{" + strings.Join(parts, ", ") + "}", nil
	case yaml.SequenceNode:
		parts := make([]string, len(node.Content))
		for i, child := range node.Content {
			value, err := pythonRepr(child)
			if err != nil {
				return "", err
			}
			parts[i] = value
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!str":
			return quotePythonString(node.Value), nil
		case "!!bool":
			value, err := boolValue(node.Value)
			if err != nil {
				return "", err
			}
			if value {
				return "True", nil
			}
			return "False", nil
		case "!!null":
			return "None", nil
		case "!!int":
			return formatInteger(node.Value)
		case "!!float":
			return formatPythonReprFloat(node.Value)
		default:
			return "", fmt.Errorf("unsupported scalar tag %s", node.Tag)
		}
	default:
		return "", fmt.Errorf("unsupported YAML node kind %d", node.Kind)
	}
}

func boolValue(value string) (bool, error) {
	switch value {
	case "true", "True", "TRUE":
		return true, nil
	case "false", "False", "FALSE":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q", value)
	}
}

func quotePythonString(value string) string {
	quote := byte('\'')
	if strings.ContainsRune(value, '\'') && !strings.ContainsRune(value, '"') {
		quote = '"'
	}

	var result strings.Builder
	result.WriteByte(quote)
	for _, r := range value {
		switch r {
		case '\\':
			result.WriteString(`\\`)
		case '\n':
			result.WriteString(`\n`)
		case '\r':
			result.WriteString(`\r`)
		case '\t':
			result.WriteString(`\t`)
		default:
			switch {
			case r == rune(quote):
				result.WriteByte('\\')
				result.WriteRune(r)
			case strconv.IsPrint(r):
				result.WriteRune(r)
			default:
				switch {
				case r <= 0xff:
					fmt.Fprintf(&result, `\x%02x`, r)
				case r <= 0xffff:
					fmt.Fprintf(&result, `\u%04x`, r)
				default:
					fmt.Fprintf(&result, `\U%08x`, r)
				}
			}
		}
	}
	result.WriteByte(quote)
	return result.String()
}

func formatInteger(value string) (string, error) {
	var integer big.Int
	if _, ok := integer.SetString(value, 10); !ok {
		return "", fmt.Errorf("invalid JSON integer %q", value)
	}
	return integer.String(), nil
}

func formatFloat(value string) (string, error) {
	// Preserve integer-shaped numeric tokens when Sort is called on a YAML node.
	if !strings.ContainsAny(value, ".eE") {
		return formatInteger(value)
	}

	number, err := strconv.ParseFloat(value, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return "", fmt.Errorf("invalid JSON number %q: %w", value, err)
	}
	return pythonFloat(number)
}

func formatPythonReprFloat(value string) (string, error) {
	formatted, err := formatFloat(value)
	if err != nil {
		return "", err
	}

	switch formatted {
	case "Infinity":
		return "inf", nil
	case "-Infinity":
		return "-inf", nil
	case "NaN":
		return "nan", nil
	default:
		return formatted, nil
	}
}
