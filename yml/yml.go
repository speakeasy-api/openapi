package yml

import (
	"context"
	"reflect"
	"strconv"

	"gopkg.in/yaml.v3"
)

func CreateOrUpdateKeyNode(ctx context.Context, key string, keyNode *yaml.Node) *yaml.Node {
	if keyNode != nil {
		resolvedKeyNode := ResolveAlias(keyNode)
		if resolvedKeyNode == nil {
			resolvedKeyNode = keyNode
		}

		resolvedKeyNode.Value = key
		return keyNode
	}

	cfg := GetConfigFromContext(ctx)

	return &yaml.Node{
		Value: key,
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Style: cfg.KeyStringStyle,
	}
}

func CreateOrUpdateScalarNode(ctx context.Context, value any, valueNode *yaml.Node) *yaml.Node {
	var convNode yaml.Node
	if err := convNode.Encode(value); err != nil {
		return nil
	}

	resolvedValueNode := ResolveAlias(valueNode)

	if resolvedValueNode != nil {
		resolvedValueNode.Value = convNode.Value
		resolvedValueNode.Tag = convNode.Tag // Also update the tag to match the new value type
		return valueNode
	}

	cfg := GetConfigFromContext(ctx)

	if convNode.Kind == yaml.ScalarNode && convNode.Tag == "!!str" {
		convNode.Style = cfg.ValueStringStyle
	}

	return &convNode
}

func CreateOrUpdateMapNodeElement(ctx context.Context, key string, keyNode, valueNode, mapNode *yaml.Node) *yaml.Node {
	resolvedMapNode := ResolveAlias(mapNode)

	if resolvedMapNode != nil {
		for i := 0; i < len(resolvedMapNode.Content); i += 2 {
			keyNode := resolvedMapNode.Content[i]
			// Check direct match first
			if keyNode.Value == key {
				resolvedMapNode.Content[i+1] = valueNode
				return mapNode
			}
			// Check alias resolution match for alias keys like *keyAlias
			if resolvedKeyNode := ResolveAlias(keyNode); resolvedKeyNode != nil && resolvedKeyNode.Value == key {
				resolvedMapNode.Content[i+1] = valueNode
				return mapNode
			}
		}

		resolvedMapNode.Content = append(resolvedMapNode.Content, CreateOrUpdateKeyNode(ctx, key, keyNode))
		resolvedMapNode.Content = append(resolvedMapNode.Content, valueNode)

		return mapNode
	}

	return CreateMapNode(ctx, []*yaml.Node{
		CreateOrUpdateKeyNode(ctx, key, keyNode),
		valueNode,
	})
}

func CreateStringNode(value string) *yaml.Node {
	return &yaml.Node{
		Value: value,
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
	}
}

func CreateIntNode(value int64) *yaml.Node {
	return &yaml.Node{
		Value: strconv.FormatInt(value, 10),
		Kind:  yaml.ScalarNode,
		Tag:   "!!int",
	}
}

func CreateFloatNode(value float64) *yaml.Node {
	return &yaml.Node{
		Value: strconv.FormatFloat(value, 'f', -1, 64),
		Kind:  yaml.ScalarNode,
		Tag:   "!!float",
	}
}

func CreateBoolNode(value bool) *yaml.Node {
	return &yaml.Node{
		Value: strconv.FormatBool(value),
		Kind:  yaml.ScalarNode,
		Tag:   "!!bool",
	}
}

func CreateMapNode(ctx context.Context, content []*yaml.Node) *yaml.Node {
	return &yaml.Node{
		Content: content,
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
	}
}

func DeleteMapNodeElement(ctx context.Context, key string, mapNode *yaml.Node) *yaml.Node {
	if mapNode == nil {
		return nil
	}

	resolvedMapNode := ResolveAlias(mapNode)
	if resolvedMapNode == nil {
		return nil
	}

	for i := 0; i < len(resolvedMapNode.Content); i += 2 {
		if resolvedMapNode.Content[i].Value == key {
			mapNode.Content = append(resolvedMapNode.Content[:i], resolvedMapNode.Content[i+2:]...) //nolint:gocritic
			return mapNode
		}
	}

	return mapNode
}

func CreateOrUpdateSliceNode(ctx context.Context, elements []*yaml.Node, valueNode *yaml.Node) *yaml.Node {
	resolvedValueNode := ResolveAlias(valueNode)
	if resolvedValueNode != nil {
		resolvedValueNode.Content = elements
		return valueNode
	}

	return &yaml.Node{
		Content: elements,
		Kind:    yaml.SequenceNode,
		Tag:     "!!seq",
	}
}

func GetMapElementNodes(ctx context.Context, mapNode *yaml.Node, key string) (*yaml.Node, *yaml.Node, bool) {
	resolvedMapNode := ResolveAlias(mapNode)
	if resolvedMapNode == nil {
		return nil, nil, false
	}

	if resolvedMapNode.Kind != yaml.MappingNode {
		return nil, nil, false
	}

	for i := 0; i < len(resolvedMapNode.Content); i += 2 {
		keyNode := resolvedMapNode.Content[i]
		// Check direct match first
		if keyNode.Value == key {
			return keyNode, resolvedMapNode.Content[i+1], true
		}
		// Check alias resolution match for alias keys like *keyAlias
		if resolvedKeyNode := ResolveAlias(keyNode); resolvedKeyNode != nil && resolvedKeyNode.Value == key {
			return keyNode, resolvedMapNode.Content[i+1], true
		}
	}

	return nil, nil, false
}

func ResolveAlias(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.AliasNode:
		return ResolveAlias(node.Alias)
	default:
		return node
	}
}

// IsMergeKey returns true if the given node is a YAML merge key (<<).
func IsMergeKey(node *yaml.Node) bool {
	return node != nil && node.Kind == yaml.ScalarNode && node.Tag == "!!merge" && node.Value == "<<"
}

// ResolveMergeKeys processes a mapping node's content and expands any YAML merge keys (<<).
// Explicit keys in the mapping take precedence over merged keys (per the YAML merge key spec).
// Merged mappings are recursively flattened so nested merge chains are fully expanded.
// Returns new content with merge keys expanded, or the original content if no merge keys are present.
func ResolveMergeKeys(content []*yaml.Node) []*yaml.Node {
	return resolveMergeKeys(content, nil)
}

// resolveKeyValue returns the effective key string for a node, resolving aliases.
func resolveKeyValue(node *yaml.Node) string {
	resolved := ResolveAlias(node)
	if resolved == nil {
		return node.Value
	}
	return resolved.Value
}

func resolveMergeKeys(content []*yaml.Node, seen map[*yaml.Node]bool) []*yaml.Node {
	// Trim trailing orphan key (odd-length content) so all loops can assume pairs
	if len(content)%2 == 1 {
		content = content[:len(content)-1]
	}
	if len(content) < 2 {
		return content
	}

	// Single pass: detect merge keys and collect explicit keys simultaneously
	hasMergeKey := false
	numMergePairs := 0
	explicitKeys := make(map[string]struct{})

	for i := 0; i < len(content); i += 2 {
		if IsMergeKey(content[i]) {
			hasMergeKey = true
			numMergePairs++
		} else {
			explicitKeys[resolveKeyValue(content[i])] = struct{}{}
		}
	}
	if !hasMergeKey {
		return content
	}

	// Build result: start with merged content, then explicit content
	// (explicit keys override merged ones)
	var mergedContent []*yaml.Node
	seenMerged := make(map[string]struct{})

	for i := 0; i < len(content); i += 2 {
		if !IsMergeKey(content[i]) {
			continue
		}

		resolved := ResolveAlias(content[i+1])
		if resolved == nil {
			continue
		}

		collectMergedPairs(resolved, explicitKeys, seenMerged, &mergedContent, seen)
	}

	// Build final result: merged content first, then explicit keys
	explicitLen := len(content) - 2*numMergePairs
	result := make([]*yaml.Node, 0, len(mergedContent)+explicitLen)
	result = append(result, mergedContent...)

	for i := 0; i < len(content); i += 2 {
		if IsMergeKey(content[i]) {
			continue
		}
		result = append(result, content[i], content[i+1])
	}

	return result
}

// collectMergedPairs collects key-value pairs from a merge target (mapping or sequence of mappings),
// recursively resolving any nested merge keys within the target.
func collectMergedPairs(node *yaml.Node, explicitKeys, seenMerged map[string]struct{}, out *[]*yaml.Node, seen map[*yaml.Node]bool) {
	switch node.Kind {
	case yaml.MappingNode:
		// Cycle guard: prevent infinite loops from circular aliases
		if seen == nil {
			seen = make(map[*yaml.Node]bool)
		}
		if seen[node] {
			return
		}
		seen[node] = true

		// Recursively flatten the merged mapping's own merge keys first
		flatContent := resolveMergeKeys(node.Content, seen)

		for j := 0; j < len(flatContent); j += 2 {
			key := resolveKeyValue(flatContent[j])
			if _, isExplicit := explicitKeys[key]; !isExplicit {
				if _, alreadyMerged := seenMerged[key]; !alreadyMerged {
					*out = append(*out, flatContent[j], flatContent[j+1])
					seenMerged[key] = struct{}{}
				}
			}
		}
	case yaml.SequenceNode:
		// Sequence of mappings merge: <<: [*alias1, *alias2]
		for _, item := range node.Content {
			resolvedItem := ResolveAlias(item)
			if resolvedItem == nil || resolvedItem.Kind != yaml.MappingNode {
				continue
			}
			collectMergedPairs(resolvedItem, explicitKeys, seenMerged, out, seen)
		}
	}
}

// EqualNodes compares two yaml.Node instances for equality.
// It performs a deep comparison of the essential fields.
func EqualNodes(a, b *yaml.Node) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Resolve aliases before comparison
	resolvedA := ResolveAlias(a)
	resolvedB := ResolveAlias(b)

	if resolvedA == nil && resolvedB == nil {
		return true
	}
	if resolvedA == nil || resolvedB == nil {
		return false
	}

	// Compare essential fields
	if resolvedA.Kind != resolvedB.Kind {
		return false
	}
	if resolvedA.Tag != resolvedB.Tag {
		return false
	}
	if resolvedA.Value != resolvedB.Value {
		return false
	}

	// Compare content for complex nodes
	if len(resolvedA.Content) != len(resolvedB.Content) {
		return false
	}
	for i, contentA := range resolvedA.Content {
		if !EqualNodes(contentA, resolvedB.Content[i]) {
			return false
		}
	}

	return true
}

// TypeToYamlTags returns all acceptable YAML tags for a given reflect.Type.
// For numeric types, both the specific tag and !!str are acceptable since
// YAML can decode string representations of numbers.
// For pointer types, !!null is also acceptable.
func TypeToYamlTags(typ reflect.Type) []string {
	if typ == nil {
		return nil
	}

	// Check if this is a pointer type
	isPointer := typ.Kind() == reflect.Ptr
	if isPointer {
		typ = typ.Elem()
	}

	var tags []string
	switch typ.Kind() {
	case reflect.String:
		tags = append(tags, "!!bool")
		fallthrough
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		tags = append(tags, "!!float", "!!str", "!!int")
	case reflect.Bool:
		tags = append(tags, "!!bool", "!!str") // Allow string representations of booleans
	case reflect.Struct, reflect.Map:
		tags = append(tags, "!!map")
	case reflect.Slice, reflect.Array:
		tags = append(tags, "!!seq")
	default:
		return nil
	}

	// For pointer types, also accept !!null
	if isPointer {
		tags = append(tags, "!!null")
	}

	return tags
}

func NodeTagToString(tag string) string {
	switch tag {
	case "!!str":
		return "string"
	case "!!int":
		return "int"
	case "!!float":
		return "float"
	case "!!bool":
		return "bool"
	case "!!map":
		return "object"
	case "!!seq":
		return "sequence"
	case "!!null":
		return "null"
	default:
		return tag
	}
}
