package marshaller

import (
	"reflect"

	"gopkg.in/yaml.v3"
)

// NodeCollector provides utilities for collecting yaml.Node pointers from core models.
// This is useful for features that need to map nodes to contexts (like operation tracking).

// CollectLeafNodes extracts all KeyNode and ValueNode pointers from marshaller.Node fields
// within a core model. It only returns nodes for "leaf" fields - those whose values are
// primitive types or slices/maps of primitives, not nested core models (which get visited
// separately by the walk).
//
// The returned nodes can be used for features like node-to-operation mapping where you
// need to track all yaml.Nodes within a model's scope.
func CollectLeafNodes(core any) []*yaml.Node {
	if core == nil {
		return nil
	}

	var nodes []*yaml.Node
	collectLeafNodesRecursive(reflect.ValueOf(core), &nodes, make(map[uintptr]bool))
	return nodes
}

// collectLeafNodesRecursive traverses the struct using reflection to find marshaller.Node fields
func collectLeafNodesRecursive(v reflect.Value, nodes *[]*yaml.Node, visited map[uintptr]bool) {
	// Handle pointers and interfaces
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	// Only process structs
	if v.Kind() != reflect.Struct {
		return
	}

	// Check for cycles (using pointer address of the struct)
	if v.CanAddr() {
		ptr := v.Addr().Pointer()
		if visited[ptr] {
			return
		}
		visited[ptr] = true
	}

	t := v.Type()

	// Iterate through all fields
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !fieldType.IsExported() {
			continue
		}

		// Check if it's a marshaller.Node type by looking for KeyNode/ValueNode fields
		if isNodeType(fieldType.Type) {
			collectFromNodeField(field, nodes)
			continue
		}

		// Recurse into embedded structs (like CoreModel)
		if fieldType.Anonymous {
			collectLeafNodesRecursive(field, nodes, visited)
		}
	}
}

// isNodeType checks if a type is marshaller.Node[T] by looking for characteristic fields
func isNodeType(t reflect.Type) bool {
	// Handle pointers
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return false
	}

	// Check for the characteristic fields of marshaller.Node
	hasKeyNode := false
	hasValueNode := false
	hasPresent := false

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		switch field.Name {
		case "KeyNode":
			if field.Type == reflect.TypeOf((*yaml.Node)(nil)) {
				hasKeyNode = true
			}
		case "ValueNode":
			if field.Type == reflect.TypeOf((*yaml.Node)(nil)) {
				hasValueNode = true
			}
		case "Present":
			if field.Type.Kind() == reflect.Bool {
				hasPresent = true
			}
		}
	}

	return hasKeyNode && hasValueNode && hasPresent
}

// collectFromNodeField extracts nodes from a marshaller.Node field
func collectFromNodeField(field reflect.Value, nodes *[]*yaml.Node) {
	// Handle pointers
	for field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return
		}
		field = field.Elem()
	}

	if field.Kind() != reflect.Struct {
		return
	}

	// Get KeyNode and ValueNode fields
	keyNodeField := field.FieldByName("KeyNode")
	valueNodeField := field.FieldByName("ValueNode")
	presentField := field.FieldByName("Present")
	valueField := field.FieldByName("Value")

	// Only collect if present
	if presentField.IsValid() && !presentField.Bool() {
		return
	}

	// Add KeyNode if not nil
	if keyNodeField.IsValid() && !keyNodeField.IsNil() {
		if node, ok := keyNodeField.Interface().(*yaml.Node); ok && node != nil {
			*nodes = append(*nodes, node)
		}
	}

	// Add ValueNode if not nil
	if valueNodeField.IsValid() && !valueNodeField.IsNil() {
		if node, ok := valueNodeField.Interface().(*yaml.Node); ok && node != nil {
			*nodes = append(*nodes, node)

			// If the Value is a primitive type (or slice/map of primitives),
			// also collect child nodes from the ValueNode
			if valueField.IsValid() && isLeafValueType(valueField.Type()) {
				collectYAMLNodeChildren(node, nodes)
			}
		}
	}
}

// isLeafValueType returns true if the type represents a leaf value (primitive or container of primitives)
// rather than a core model that will be walked separately
func isLeafValueType(t reflect.Type) bool {
	// Handle pointers
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true

	case reflect.Slice:
		elemType := t.Elem()
		// Slices of primitives are leaf types
		// Slices of core models are not (they get walked)
		return isLeafValueType(elemType)

	case reflect.Map:
		// Maps with primitive keys and values are leaf types
		return isLeafValueType(t.Key()) && isLeafValueType(t.Elem())

	case reflect.Struct:
		// Check if it's a CoreModeler (has GetRootNode method)
		// If so, it's not a leaf - it will be walked separately
		if hasCoreModelerMethod(t) {
			return false
		}
		// Check if it's a marshaller.Node type
		if isNodeType(t) {
			// Get the inner value type and check that
			valueField, found := t.FieldByName("Value")
			if found {
				return isLeafValueType(valueField.Type)
			}
		}
		// Other structs might be leaf types (like custom value types)
		return true

	case reflect.Interface:
		// Can't determine at compile time - assume not leaf
		return false

	default:
		return false
	}
}

// hasCoreModelerMethod checks if a type implements GetRootNode() *yaml.Node
func hasCoreModelerMethod(t reflect.Type) bool {
	// Check both value and pointer receiver
	_, hasMethod := t.MethodByName("GetRootNode")
	if hasMethod {
		return true
	}
	if t.Kind() != reflect.Ptr {
		ptrType := reflect.PointerTo(t)
		_, hasMethod = ptrType.MethodByName("GetRootNode")
	}
	return hasMethod
}

// collectYAMLNodeChildren adds all direct children of a YAML node to the nodes slice
// This is used for simple values like slices of strings where the individual items
// aren't core models but we still want to track their nodes
func collectYAMLNodeChildren(node *yaml.Node, nodes *[]*yaml.Node) {
	if node == nil || node.Content == nil {
		return
	}

	for _, child := range node.Content {
		if child != nil {
			*nodes = append(*nodes, child)
		}
	}
}
