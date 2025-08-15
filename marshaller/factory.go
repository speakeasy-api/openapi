package marshaller

import (
	"flag"
	"log"
	"reflect"
	"sync"

	"github.com/speakeasy-api/openapi/sequencedmap"
	"gopkg.in/yaml.v3"
)

// TypeFactory represents a function that creates a new instance of a specific type
type TypeFactory func() interface{}

// CachedFieldInfo contains the cached information about a struct field
type CachedFieldInfo struct {
	Name         string
	Index        int
	Required     bool
	Tag          string
	IsExported   bool
	IsExtensions bool
}

// CachedFieldMaps contains the complete cached field processing result
type CachedFieldMaps struct {
	Fields         map[string]CachedFieldInfo
	ExtensionIndex int             // Index of extensions field, -1 if none
	HasExtensions  bool            // Whether there's an extensions field
	FieldIndexes   map[string]int  // tag -> field index mapping
	RequiredFields map[string]bool // tag -> required status for quick lookup
}

// Global factory registry using sync.Map for better performance
var typeFactories sync.Map

// Global field cache registry - built at type registration time
var fieldCache sync.Map

// RegisterType registers a factory function for a specific type
// This should be called in init() functions of packages that define models
func RegisterType[T any](factory func() *T) {
	var zero T
	typ := reflect.TypeOf(zero)

	// Handle pointer types - we want the element type for lookup
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	typeFactories.Store(typ, TypeFactory(func() interface{} {
		return factory()
	}))

	// Build field cache at registration time for struct types
	if typ.Kind() == reflect.Struct {
		buildFieldCacheForType(typ)
	}
}

// CreateInstance creates a new instance using registered factory or falls back to reflection
func CreateInstance(typ reflect.Type) reflect.Value {
	elemType := typ
	if typ.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	if factory, ok := typeFactories.Load(elemType); ok {
		return reflect.ValueOf(factory.(TypeFactory)())
	}

	if isTesting() {
		log.Printf("ERROR: unregistered type: %s", elemType.String())
	}

	// Fallback to reflection for unregistered types
	return reflect.New(typ)
}

// IsRegistered checks if a type has a registered factory
func IsRegistered(typ reflect.Type) bool {
	elemType := typ
	if typ.Kind() == reflect.Ptr {
		elemType = typ.Elem()
	}

	_, exists := typeFactories.Load(elemType)
	return exists
}

func isTesting() bool {
	return flag.Lookup("test.v") != nil
}

// init registers basic Go types that might be used in marshalling
func init() {
	// Register all primitive Go types
	RegisterType(func() *string { return new(string) })
	RegisterType(func() *bool { return new(bool) })
	RegisterType(func() *int { return new(int) })
	RegisterType(func() *int8 { return new(int8) })
	RegisterType(func() *int16 { return new(int16) })
	RegisterType(func() *int32 { return new(int32) })
	RegisterType(func() *int64 { return new(int64) })
	RegisterType(func() *uint { return new(uint) })
	RegisterType(func() *uint8 { return new(uint8) })
	RegisterType(func() *uint16 { return new(uint16) })
	RegisterType(func() *uint32 { return new(uint32) })
	RegisterType(func() *uint64 { return new(uint64) })
	RegisterType(func() *float32 { return new(float32) })
	RegisterType(func() *float64 { return new(float64) })
	RegisterType(func() *complex64 { return new(complex64) })
	RegisterType(func() *complex128 { return new(complex128) })
	RegisterType(func() *byte { return new(byte) })
	RegisterType(func() *rune { return new(rune) })

	// Register Node wrapped primitive types
	RegisterType(func() *Node[string] { return &Node[string]{} })
	RegisterType(func() *Node[bool] { return &Node[bool]{} })
	RegisterType(func() *Node[int] { return &Node[int]{} })

	// Register slices of primitive types
	RegisterType(func() *[]string { return &[]string{} })
	RegisterType(func() *[]bool { return &[]bool{} })
	RegisterType(func() *[]int { return &[]int{} })
	RegisterType(func() *[]int8 { return &[]int8{} })
	RegisterType(func() *[]int16 { return &[]int16{} })
	RegisterType(func() *[]int32 { return &[]int32{} })
	RegisterType(func() *[]int64 { return &[]int64{} })
	RegisterType(func() *[]uint { return &[]uint{} })
	RegisterType(func() *[]uint8 { return &[]uint8{} })
	RegisterType(func() *[]uint16 { return &[]uint16{} })
	RegisterType(func() *[]uint32 { return &[]uint32{} })
	RegisterType(func() *[]uint64 { return &[]uint64{} })
	RegisterType(func() *[]float32 { return &[]float32{} })
	RegisterType(func() *[]float64 { return &[]float64{} })
	RegisterType(func() *[]complex64 { return &[]complex64{} })
	RegisterType(func() *[]complex128 { return &[]complex128{} })
	RegisterType(func() *[]byte { return &[]byte{} })
	RegisterType(func() *[]rune { return &[]rune{} })

	// YAML types
	RegisterType(func() *yaml.Node { return &yaml.Node{} })
	RegisterType(func() *Node[*yaml.Node] { return &Node[*yaml.Node]{} })
	RegisterType(func() *sequencedmap.Map[string, Node[*yaml.Node]] {
		return &sequencedmap.Map[string, Node[*yaml.Node]]{}
	})

	// Register common sequencedmap.Map types
	RegisterType(func() *sequencedmap.Map[string, Node[string]] {
		return &sequencedmap.Map[string, Node[string]]{}
	})
	RegisterType(func() *sequencedmap.Map[string, string] {
		return &sequencedmap.Map[string, string]{}
	})
}

// buildFieldCacheForType builds the field cache for a struct type at registration time
func buildFieldCacheForType(structType reflect.Type) {
	if structType.Kind() != reflect.Struct {
		return
	}

	fields := make(map[string]CachedFieldInfo)
	fieldIndexes := make(map[string]int)
	requiredFields := make(map[string]bool)
	extensionIndex := -1
	hasExtensions := false

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Skip anonymous fields (embedded structs/maps are handled separately)
		if field.Anonymous {
			continue
		}

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("key")
		if tag == "" {
			// Handle extensions field specially
			if tag == "extensions" {
				extensionIndex = i
				hasExtensions = true
				fields[tag] = CachedFieldInfo{
					Name:         field.Name,
					Index:        i,
					Required:     false,
					Tag:          tag,
					IsExported:   true,
					IsExtensions: true,
				}
			}
			continue
		}

		// Determine if field is required
		requiredTag := field.Tag.Get("required")
		required := requiredTag == "true"

		// If no explicit required tag, use the same logic as the original unmarshaller
		if requiredTag == "" {
			// Create a zero value of the field to check if it implements NodeAccessor
			fieldVal := reflect.New(field.Type).Elem()
			if nodeAccessor, ok := fieldVal.Interface().(NodeAccessor); ok {
				fieldType := nodeAccessor.GetValueType()
				if fieldType.Kind() != reflect.Ptr {
					required = fieldType.Kind() != reflect.Map && fieldType.Kind() != reflect.Slice && fieldType.Kind() != reflect.Array
				}
			}
		}

		// Store the cached field info
		fields[tag] = CachedFieldInfo{
			Name:         field.Name,
			Index:        i,
			Required:     required,
			Tag:          tag,
			IsExported:   true,
			IsExtensions: tag == "extensions",
		}

		// Track extensions field
		if tag == "extensions" {
			extensionIndex = i
			hasExtensions = true
		} else {
			// Build field index maps at cache time (this is the expensive work we want to avoid)
			fieldIndexes[tag] = i
			if required {
				requiredFields[tag] = true
			}
		}
	}

	// Store complete cached result including pre-built field indexes
	cachedMaps := CachedFieldMaps{
		Fields:         fields,
		ExtensionIndex: extensionIndex,
		HasExtensions:  hasExtensions,
		FieldIndexes:   fieldIndexes,
		RequiredFields: requiredFields,
	}

	fieldCache.Store(structType, cachedMaps)
}

// getFieldMapCached returns the cached field maps for a struct type
// This is much faster than the reflection-heavy loop in unmarshalModel
func getFieldMapCached(structType reflect.Type) CachedFieldMaps {
	if cached, ok := fieldCache.Load(structType); ok {
		return cached.(CachedFieldMaps)
	}

	if isTesting() {
		log.Printf("CACHE MISS: building field cache on-demand for unregistered type: %s", structType.String())
	}

	// Build cache on-demand for unregistered types
	buildFieldCacheForType(structType)

	return getFieldMapCached(structType)
}

// ClearGlobalFieldCache clears the global field cache.
// This is useful for testing or memory management when the cache is no longer needed.
func ClearGlobalFieldCache() {
	fieldCache.Range(func(key, value interface{}) bool {
		fieldCache.Delete(key)
		return true
	})
}

// FieldCacheStats returns basic statistics about the field cache
type FieldCacheStats struct {
	Size int64
}

// GetFieldCacheStats returns statistics about the global field cache
func GetFieldCacheStats() FieldCacheStats {
	var size int64
	fieldCache.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return FieldCacheStats{Size: size}
}
