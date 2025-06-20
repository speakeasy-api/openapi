package marshaller

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

type Unmarshallable interface {
	Unmarshal(ctx context.Context, value *yaml.Node) error
}

func Unmarshal(ctx context.Context, node *yaml.Node, out any) error {
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) != 1 {
			return fmt.Errorf("expected 1 node, got %d", len(node.Content))
		}

		return Unmarshal(ctx, node.Content[0], out)
	}

	v := reflect.ValueOf(out)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	for v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}

	return unmarshal(ctx, node, v)
}

func UnmarshalModel(ctx context.Context, node *yaml.Node, structPtr any) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("expected a mapping node, got %v", node.Kind)
	}

	out := reflect.ValueOf(structPtr)

	if out.Kind() == reflect.Ptr {
		out = out.Elem()
	}

	if out.Kind() != reflect.Struct {
		return fmt.Errorf("expected a struct, got %s", out.Kind())
	}

	var unmarshallable CoreModeler

	// Check if struct implements CoreModeler
	if implementsInterface[CoreModeler](out) {
		var ok bool
		unmarshallable, ok = out.Addr().Interface().(CoreModeler)
		if !ok {
			return fmt.Errorf("expected CoreModeler, got %s", out.Type())
		}
	} else {
		return fmt.Errorf("expected struct to implement CoreModeler, got %s", out.Type())
	}

	unmarshallable.SetRootNode(node)

	type Field struct {
		Name     string
		Field    reflect.Value
		Required bool
	}

	// get fields by tag first
	fields := map[string]Field{}
	var extensionsField *reflect.Value
	requiredFields := map[string]Field{} // Track required fields separately

	for i := 0; i < out.NumField(); i++ {
		field := out.Type().Field(i)
		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("key")
		if tag == "" || tag == "extensions" {
			if tag == "extensions" {
				extField := out.Field(i)
				extensionsField = &extField
			}

			continue
		}

		requiredTag := field.Tag.Get("required")
		required := requiredTag == "true"

		if requiredTag == "" {
			nodeAccessor, ok := out.Field(i).Interface().(NodeAccessor)
			if ok {
				fieldType := nodeAccessor.GetValueType()

				if fieldType.Kind() != reflect.Ptr {
					required = fieldType.Kind() != reflect.Map && fieldType.Kind() != reflect.Slice && fieldType.Kind() != reflect.Array
				}
			}
		}

		fieldInfo := Field{
			Name:     field.Name,
			Field:    out.Field(i),
			Required: required,
		}

		fields[tag] = fieldInfo

		// Track required fields for validation
		if required {
			requiredFields[tag] = fieldInfo
		}
	}

	// Process YAML nodes and validate required fields in one pass
	valid := true
	foundRequiredFields := map[string]bool{}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		key := keyNode.Value

		field, ok := fields[key]
		if !ok {
			if !strings.HasPrefix(key, "x-") {
				continue
			}

			if extensionsField != nil {
				if err := UnmarshalExtension(keyNode, valueNode, *extensionsField); err != nil {
					return err
				}
			}
		} else if implementsInterface[NodeMutator](field.Field) {
			if err := unmarshalNode(ctx, keyNode, valueNode, field.Name, field.Field); err != nil {
				return err
			}

			// Mark required field as found
			if field.Required {
				foundRequiredFields[key] = true
			}
		} else {
			return fmt.Errorf("expected field to be marshaller.Node, got %s", field.Field.Type())
		}
	}

	// Check for missing required fields
	for tag := range requiredFields {
		if !foundRequiredFields[tag] {
			unmarshallable.AddValidationError(validation.NewNodeError(fmt.Sprintf("field %s is missing", tag), node))
			valid = false
		}
	}

	unmarshallable.SetValid(valid)

	return nil
}

func UnmarshalKeyValuePair(ctx context.Context, keyNode, valueNode *yaml.Node, outValue any) error {
	out := reflect.ValueOf(outValue)

	if implementsInterface[NodeMutator](out) {
		return unmarshalNode(ctx, keyNode, valueNode, "value", out)
	} else {
		return Unmarshal(ctx, valueNode, outValue)
	}
}

func unmarshal(ctx context.Context, node *yaml.Node, out reflect.Value) error {
	switch {
	case out.Type() == reflect.TypeOf((*yaml.Node)(nil)):
		out.Set(reflect.ValueOf(node))
		return nil
	case out.Type() == reflect.TypeOf(yaml.Node{}):
		out.Set(reflect.ValueOf(*node))
		return nil
	}

	if implementsInterface[Unmarshallable](out) {
		if out.Kind() != reflect.Ptr {
			out = out.Addr()
		}

		if out.IsNil() {
			out.Set(reflect.New(out.Type().Elem()))
		}

		unmarshallable, ok := out.Interface().(Unmarshallable)
		if !ok {
			return fmt.Errorf("expected Unmarshallable, got %s", out.Type())
		}

		return unmarshallable.Unmarshal(ctx, node)
	}

	switch node.Kind {
	case yaml.MappingNode:
		return unmarshalMapping(ctx, node, out)
	case yaml.ScalarNode:
		return node.Decode(out.Addr().Interface())
	case yaml.SequenceNode:
		return unmarshalSequence(ctx, node, out)
	case yaml.AliasNode:
		return fmt.Errorf("currently unsupported node kind: %v", node.Kind)
	default:
		return fmt.Errorf("invalid node kind: %v", node.Kind)
	}
}

func unmarshalMapping(ctx context.Context, node *yaml.Node, out reflect.Value) error {
	if out.Kind() == reflect.Ptr {
		out.Set(reflect.New(out.Type().Elem()))
		out = out.Elem()
	}

	switch {
	case out.Kind() == reflect.Struct:
		if implementsInterface[CoreModeler](out) {
			return UnmarshalModel(ctx, node, out.Addr().Interface())
		} else {
			return unmarshalStruct(ctx, node, out.Addr().Interface())
		}
	case out.Kind() == reflect.Map:
		return fmt.Errorf("currently unsupported out kind: %v", out.Kind())
	default:
		return fmt.Errorf("expected struct or map, got %s", out.Kind())
	}
}

func unmarshalStruct(_ context.Context, node *yaml.Node, structPtr any) error {
	// TODO do we need a custom implementation for this? This implementation will treat any child of a normal struct as also a normal struct unless it implements a custom unmarshaller
	return node.Decode(structPtr)
}

func unmarshalSequence(ctx context.Context, node *yaml.Node, out reflect.Value) error {
	if out.Kind() != reflect.Slice {
		return fmt.Errorf("expected slice, got %s", out.Kind())
	}

	out.Set(reflect.MakeSlice(out.Type(), len(node.Content), len(node.Content)))

	for i := 0; i < len(node.Content); i++ {
		valueNode := node.Content[i]

		if err := unmarshal(ctx, valueNode, out.Index(i)); err != nil {
			return err
		}
	}

	return nil
}

func unmarshalNode(ctx context.Context, keyNode, valueNode *yaml.Node, fieldName string, out reflect.Value) error {
	ref := out

	if out.Kind() != reflect.Ptr {
		if out.CanSet() {
			ref = out.Addr()
		} else {
			// For non-settable values (like local variables), we need to work with what we have
			// This typically happens when out is already a pointer to the value we want to modify
			ref = out
		}
	} else if out.IsNil() {
		if out.CanSet() {
			out.Set(reflect.New(out.Type().Elem()))
			ref = out.Elem().Addr()
		} else {
			return fmt.Errorf("field %s is a nil pointer and cannot be set", fieldName)
		}
	}

	unmarshallable, ok := ref.Interface().(NodeMutator)
	if !ok {
		return fmt.Errorf("expected field to be marshaller.Node, got %s", ref.Type())
	}

	if err := unmarshallable.Unmarshal(ctx, keyNode, valueNode); err != nil {
		return err
	}

	// Fix: Only set the value if the original field can be set
	if out.CanSet() {
		if out.Kind() == reflect.Ptr {
			out.Set(reflect.ValueOf(unmarshallable))
		} else {
			// Get the value from the unmarshallable and set it directly
			unmarshallableValue := reflect.ValueOf(unmarshallable)
			if unmarshallableValue.Kind() == reflect.Ptr {
				unmarshallableValue = unmarshallableValue.Elem()
			}
			out.Set(unmarshallableValue)
		}
	}

	return nil
}

func implementsInterface[T any](out reflect.Value) bool {
	// Store original value to check directly
	original := out

	// Unwrap interface if needed
	for out.Kind() == reflect.Interface && !out.IsNil() {
		out = out.Elem()
	}

	// Get addressable value if needed
	if out.Kind() != reflect.Ptr {
		if !out.CanAddr() {
			// Try checking the original value directly
			return original.Type().Implements(reflect.TypeOf((*T)(nil)).Elem())
		}
		out = out.Addr()
	}

	return out.Type().Implements(reflect.TypeOf((*T)(nil)).Elem())
}
