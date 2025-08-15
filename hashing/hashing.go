package hashing

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/internal/interfaces"
)

func Hash(v any) string {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(toHashableString(v)))
	return fmt.Sprintf("%016x", hasher.Sum64())
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
		builder.WriteString(structToHashableString(v))
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
