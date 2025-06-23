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

// Global factory registry using sync.Map for better performance
var typeFactories sync.Map

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
}
