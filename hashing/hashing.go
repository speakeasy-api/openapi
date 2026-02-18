package hashing

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"go.yaml.in/yaml/v4"
)

func Hash(v any) string {
	hasher := fnv.New64a()
	hashableStr := toHashableString(v)
	_, _ = hasher.Write([]byte(hashableStr))
	return formatHash(hasher.Sum64())
}

// formatHash converts a uint64 hash to a zero-padded 16-character hex string
// without the allocation overhead of fmt.Sprintf.
func formatHash(h uint64) string {
	const hexDigits = "0123456789abcdef"
	var buf [16]byte
	for i := 15; i >= 0; i-- {
		buf[i] = hexDigits[h&0xf]
		h >>= 4
	}
	return string(buf[:])
}

type model interface {
	GetCoreAny() any
	SetCoreAny(core any)
}

func toHashableString(v any) string {
	if v == nil {
		return ""
	}

	var builder strings.Builder

	typ := reflect.TypeOf(v)
	if typ == nil {
		return ""
	}
	switch typ.Kind() {
	case reflect.Slice, reflect.Array:
		sliceVal := reflect.ValueOf(v)

		if typ.Kind() == reflect.Slice && sliceVal.IsNil() {
			return ""
		}

		for i := 0; i < sliceVal.Len(); i++ {
			builder.WriteString(toHashableString(sliceVal.Index(i).Interface()))
		}
	case reflect.Map:
		mapVal := reflect.ValueOf(v)

		if mapVal.IsNil() {
			return ""
		}

		mapKeys := mapVal.MapKeys()
		// Sort keys for deterministic output
		slices.SortFunc(mapKeys, func(a, b reflect.Value) int {
			return strings.Compare(toHashableString(a.Interface()), toHashableString(b.Interface()))
		})

		for _, key := range mapKeys {
			builder.WriteString(toHashableString(key.Interface()))
			builder.WriteString(toHashableString(mapVal.MapIndex(key).Interface()))
		}
	case reflect.Struct:
		// Check if this is a yaml.Node
		if node, ok := v.(yaml.Node); ok {
			builder.WriteString(yamlNodeToHashableString(&node))
		} else {
			builder.WriteString(structToHashableString(v))
		}
	case reflect.Ptr, reflect.Interface:
		val := reflect.ValueOf(v)
		if val.IsNil() {
			return ""
		}

		// Check if this is a sequenced map interface (for pointer types)
		if interfaces.ImplementsInterface[interfaces.SequencedMapInterface](typ) && !interfaces.ImplementsInterface[model](typ) {
			builder.WriteString(sequencedMapToHashableString(v))
		} else {
			builder.WriteString(toHashableString(val.Elem().Interface()))
		}
	default:
		switch v := v.(type) {
		case string:
			builder.WriteString(v)
		case int:
			builder.WriteString(strconv.Itoa(v))
		case int64:
			builder.WriteString(strconv.FormatInt(v, 10))
		case float64:
			builder.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
		case bool:
			builder.WriteString(strconv.FormatBool(v))
		case uint64:
			builder.WriteString(strconv.FormatUint(v, 10))
		case *yaml.Node:
			builder.WriteString(yamlNodeToHashableString(v))
		default:
			builder.WriteString(fmt.Sprintf("%v", v))
		}
	}

	return builder.String()
}

type eitherValue interface {
	IsLeft() bool
	IsRight() bool
}

func structToHashableString(v any) string {
	var builder strings.Builder

	structVal := reflect.ValueOf(v)
	structType := structVal.Type()

	for i := 0; i < structVal.NumField(); i++ {
		fieldType := structType.Field(i)
		fieldVal := structVal.Field(i)

		if fieldType.Anonymous {
			switch {
			case interfaces.ImplementsInterface[interfaces.SequencedMapInterface](reflect.PointerTo(fieldVal.Type())):
				// For value embeds, we need to get the address to access the interface methods
				if fieldVal.CanAddr() {
					builder.WriteString(sequencedMapToHashableString(fieldVal.Addr().Interface()))
				} else {
					builder.WriteString(sequencedMapToHashableString(fieldVal.Interface()))
				}
			case interfaces.ImplementsInterface[eitherValue](reflect.PointerTo(fieldVal.Type())):
				builder.WriteString(structToHashableString(fieldVal.Interface()))
			}
			continue
		}

		if !fieldType.IsExported() {
			continue
		}

		// Ignore extensions field as they are generally metadata and don't impact the equivalence of what we want to match
		if fieldType.Name == "Extensions" {
			continue
		}

		val := toHashableString(fieldVal.Interface())
		if val == "" {
			continue
		}

		builder.WriteString(fieldType.Name)
		builder.WriteString(val)
	}

	return builder.String()
}

// yamlNodeToHashableString recursively processes a YAML node and its children,
// including only semantic content (Tag, Value, Kind) and excluding positional
// metadata (Line, Column, Style, etc.)
func yamlNodeToHashableString(node *yaml.Node) string {
	if node == nil {
		return ""
	}

	var builder strings.Builder

	// Include semantic fields only
	builder.WriteString("Kind")
	builder.WriteString(strconv.Itoa(int(node.Kind)))
	if node.Tag != "" {
		builder.WriteString("Tag" + node.Tag)
	}
	if node.Value != "" {
		builder.WriteString("Value" + node.Value)
	}

	// Recursively process children in Content array
	for _, child := range node.Content {
		builder.WriteString(yamlNodeToHashableString(child))
	}

	return builder.String()
}

func sequencedMapToHashableString(v any) string {
	var builder strings.Builder

	seqMap, ok := v.(interfaces.SequencedMapInterface)
	if !ok {
		return ""
	}

	keys := slices.Collect(seqMap.KeysAny())
	slices.SortFunc(keys, func(a, b any) int {
		return strings.Compare(toHashableString(a), toHashableString(b))
	})

	for _, key := range keys {
		val, ok := seqMap.GetAny(key)
		if !ok {
			continue
		}
		builder.WriteString(toHashableString(key))
		builder.WriteString(toHashableString(val))
	}

	return builder.String()
}
