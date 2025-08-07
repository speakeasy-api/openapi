package utils

import (
	"strings"
	"sync"
)

// StringBuilderPool provides a pool of string builders to reduce allocations
// when building strings, especially for repeated operations like reference resolution.
var StringBuilderPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

// BuildAbsoluteReference efficiently builds an absolute reference string by combining
// a base reference with a JSON pointer. For this specific 3-string concatenation pattern,
// Go's optimized string concatenation is faster than string builders.
func BuildAbsoluteReference(baseRef, jsonPtr string) string {
	if jsonPtr == "" {
		return baseRef
	}
	return baseRef + "#" + jsonPtr
}

// BuildString efficiently builds a string from multiple parts using a pooled string builder.
// This is useful for any string concatenation operations that happen frequently.
func BuildString(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}

	builder := StringBuilderPool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		StringBuilderPool.Put(builder)
	}()

	for _, part := range parts {
		builder.WriteString(part)
	}
	return builder.String()
}

// JoinWithSeparator efficiently joins strings with a separator using a pooled string builder.
// This is more efficient than strings.Join for frequently called operations.
func JoinWithSeparator(separator string, parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}

	builder := StringBuilderPool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		StringBuilderPool.Put(builder)
	}()

	builder.WriteString(parts[0])
	for i := 1; i < len(parts); i++ {
		builder.WriteString(separator)
		builder.WriteString(parts[i])
	}
	return builder.String()
}
