package marshaller

import (
	"context"
	"fmt"
	"iter"
	"reflect"
	"strings"

	"github.com/speakeasy-api/openapi/errors"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

type Unmarshallable interface {
	Unmarshal(ctx context.Context, value *yaml.Node) error
}

type SequencedMap interface {
	Init()
	SetUntyped(key, value any) error
	AllUntyped() iter.Seq2[any, any]
	GetValueType() reflect.Type
}

var _ SequencedMap = (*sequencedmap.Map[any, any])(nil)

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

	return unmarshal(ctx, node, v)
}

func UnmarshalStruct(ctx context.Context, node *yaml.Node, structPtr any) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("UnmarshalStruct expected a mapping node, got %v", node.Kind)
	}

	out := reflect.ValueOf(structPtr)

	if out.Kind() == reflect.Ptr {
		out = out.Elem()
	}

	// TODO we need to actually check its a struct and its not nil

	type Field struct {
		Name     string
		Field    reflect.Value
		Required bool
	}

	// get fields by tag first
	fields := sequencedmap.New[string, Field]()
	var extensionsField *reflect.Value

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

		fields.Set(tag, Field{
			Name:     field.Name,
			Field:    out.Field(i),
			Required: required,
		})
	}

	foundFields := sequencedmap.New[string, bool]()

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		key := keyNode.Value

		field, ok := fields.Get(key)
		if !ok {
			if !strings.HasPrefix(key, "x-") {
				continue
			}

			if extensionsField != nil {
				if err := unmarshalExtension(keyNode, valueNode, *extensionsField); err != nil {
					return err
				}
			}
		} else {
			if err := unmarshalNode(ctx, keyNode, valueNode, field.Name, field.Field); err != nil {
				return err
			}

			foundFields.Set(key, true)
		}
	}

	for key, field := range fields.All() {
		if !field.Required {
			continue
		}

		if _, ok := foundFields.Get(key); !ok {
			validation.AddValidationError(ctx, &validation.Error{
				Message: fmt.Sprintf("field %s is missing", key),
				Line:    node.Line,
				Column:  node.Column,
			})
		}
	}

	return nil
}

func unmarshal(ctx context.Context, node *yaml.Node, out reflect.Value) error {
	if out.Type() == reflect.TypeOf((*yaml.Node)(nil)) {
		out.Set(reflect.ValueOf(node))
		return nil
	}

	if isUnmarshallable(out) {
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
	_, ok := out.Interface().(SequencedMap)
	if ok {
		return unmarshalSequencedMap(ctx, node, out)
	}

	if out.Kind() == reflect.Ptr {
		out.Set(reflect.New(out.Type().Elem()))
		out = out.Elem()
	}

	switch {
	case out.Kind() == reflect.Struct:
		return UnmarshalStruct(ctx, node, out.Addr().Interface())
	case out.Kind() == reflect.Map:
		return fmt.Errorf("currently unsupported out kind: %v", out.Kind())
	default:
		return fmt.Errorf("expected struct or map, got %s", out.Kind())
	}
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
	if !out.CanSet() {
		return fmt.Errorf("field %s is not settable", fieldName)
	}

	ref := out

	if out.Kind() != reflect.Ptr {
		ref = out.Addr()
	} else if out.IsNil() {
		out.Set(reflect.New(out.Type().Elem()))
		ref = out.Elem().Addr()
	}

	unmarshallable, ok := ref.Interface().(NodeMutator)
	if !ok {
		return errors.New("expected NodeMutator")
	}

	if err := unmarshallable.Unmarshal(ctx, keyNode, valueNode); err != nil {
		return err
	}

	unmarshallable.SetPresent(true)

	if out.Kind() == reflect.Ptr {
		out.Set(reflect.ValueOf(unmarshallable))
	} else {
		out.Set(reflect.ValueOf(unmarshallable).Elem())
	}

	return nil
}

func unmarshalSequencedMap(ctx context.Context, node *yaml.Node, out reflect.Value) error {
	if out.Kind() == reflect.Ptr && out.IsNil() {
		out.Set(reflect.New(out.Type().Elem()))
	}

	sm, ok := out.Interface().(SequencedMap)
	if !ok {
		return fmt.Errorf("expected SequencedMap, got %s", out.Type())
	}

	sm.Init()

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		key := keyNode.Value

		valueOut := reflect.New(sm.GetValueType()).Elem()

		if err := unmarshal(ctx, valueNode, valueOut); err != nil {
			return err
		}

		if err := sm.SetUntyped(key, valueOut.Interface()); err != nil {
			return err
		}
	}

	return nil
}

func isUnmarshallable(out reflect.Value) bool {
	if out.Kind() != reflect.Ptr {
		out = out.Addr()
	}

	return out.Type().Implements(reflect.TypeOf((*Unmarshallable)(nil)).Elem())
}
