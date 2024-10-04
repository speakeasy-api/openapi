package marshaller

import (
	"context"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

type Syncer interface {
	SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error)
}

type SyncerWithSyncFunc interface {
	SyncChangesWithSyncFunc(ctx context.Context, model any, valueNode *yaml.Node, syncFunc func(context.Context, any, any, *yaml.Node) (*yaml.Node, error)) (*yaml.Node, error)
}

func SyncValue(ctx context.Context, source any, target any, valueNode *yaml.Node) (node *yaml.Node, err error) {
	s := reflect.ValueOf(source)
	st := reflect.TypeOf(source)
	t := reflect.ValueOf(target)
	tt := reflect.TypeOf(target)

	if s.Kind() == reflect.Ptr {
		if s.IsNil() {
			t.Elem().Set(reflect.Zero(t.Type().Elem()))
			return nil, nil
		}

		s, st = fullyDereference(s, st)
	}

	t, tt = dereferenceToLastPtr(t, tt)

	if tt.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("SyncValue expected pointer, got %s", tt)
	}

	if tt.Elem().Kind() != st.Kind() {
		return nil, fmt.Errorf("SyncValue expected target to be %s, got %s", st.Kind(), tt.Elem().Kind())
	}

	switch {
	case s.Kind() == reflect.Struct && t.Type() == reflect.TypeOf((*yaml.Node)(nil)):
		t.Elem().Set(s)
		return t.Interface().(*yaml.Node), nil
	case s.Kind() == reflect.Struct:
		syncer, ok := t.Interface().(Syncer)
		if ok {
			sv := s.Interface()
			if s.CanAddr() {
				sv = s.Addr().Interface()
			}

			return syncer.SyncChanges(ctx, sv, valueNode)
		}

		syncerWithSyncFunc, ok := t.Interface().(SyncerWithSyncFunc)
		if ok {
			sv := s.Interface()
			if s.CanAddr() {
				sv = s.Addr().Interface()
			}

			return syncerWithSyncFunc.SyncChangesWithSyncFunc(ctx, sv, valueNode, SyncValue)
		}

		return syncChanges(ctx, s.Interface(), t.Interface(), valueNode)
	case s.Kind() == reflect.Map:
		// TODO call sync changes on each value
		panic("not implemented")
	case s.Kind() == reflect.Slice, s.Kind() == reflect.Array:
		return syncArraySlice(ctx, s.Interface(), t.Interface(), valueNode)
	default:
		if st != tt.Elem() {
			// Cast the value to the target type
			s = s.Convert(tt.Elem())
		}
		if !t.Elem().IsValid() {
			t.Set(reflect.New(tt.Elem()))
		}
		t.Elem().Set(s)
		out := yml.CreateOrUpdateScalarNode(ctx, s.Interface(), valueNode)
		return out, nil
	}
}

func syncChanges(ctx context.Context, source any, target any, valueNode *yaml.Node) (*yaml.Node, error) {
	s := reflect.ValueOf(source)
	t := reflect.ValueOf(target)

	if s.Kind() == reflect.Ptr {
		if s.IsNil() {
			panic("not implemented")
		}
		s = s.Elem()
	}
	if t.Kind() == reflect.Ptr && t.IsNil() {
		t.Set(reflect.New(t.Type().Elem()))
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if s.Kind() != reflect.Struct {
		return nil, fmt.Errorf("syncChanges expected struct, got %s", s.Type())
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Type().Field(i)
		if !field.IsExported() {
			continue
		}
		sourceVal := s.FieldByName(field.Name)

		key := field.Tag.Get("key")
		if key == "" {
			continue
		}

		target := t.Field(i)
		if target.Kind() != reflect.Ptr {
			target = target.Addr()
		}

		// If both are nil, we don't need to sync
		if target.IsNil() && sourceVal.IsNil() {
			continue
		}

		if key == "extensions" {
			var err error
			valueNode, err = syncExtensions(ctx, sourceVal.Interface(), target, valueNode)
			if err != nil {
				return nil, err
			}
			continue
		}

		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}

		targetInt := target.Interface()
		sourceInt := sourceVal.Interface()

		nodeMutator, ok := targetInt.(NodeMutator)
		if !ok {
			return nil, fmt.Errorf("syncChanges expected NodeMutator, got %s", target.Type())
		}

		keyNode, valNode, err := nodeMutator.SyncValue(ctx, key, sourceInt)
		if err != nil {
			return nil, err
		}

		if valNode != nil {
			valueNode = yml.CreateOrUpdateMapNodeElement(ctx, key, keyNode, valNode, valueNode)
		} else {
			valueNode = yml.DeleteMapNodeElement(ctx, key, valueNode)
		}
	}

	rn, ok := t.Type().FieldByName("RootNode")
	if !ok {
		return nil, fmt.Errorf("SyncChanges expected a RootNode field on the target %s", t.Type())
	}

	t.FieldByIndex(rn.Index).Set(reflect.ValueOf(valueNode))

	return valueNode, nil
}

func syncArraySlice(ctx context.Context, source any, target any, valueNode *yaml.Node) (*yaml.Node, error) {
	sourceVal := reflect.ValueOf(source)
	targetVal := reflect.ValueOf(target)

	if sourceVal.IsNil() && targetVal.IsNil() {
		return valueNode, nil
	}

	if sourceVal.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected source to be slice, got %s", sourceVal.Kind())
	}

	if targetVal.Kind() != reflect.Ptr || targetVal.Elem().Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected target to be slice, got %s", targetVal.Kind())
	}

	targetVal = targetVal.Elem()

	if sourceVal.IsNil() {
		targetVal.Set(reflect.Zero(targetVal.Type()))
		return nil, nil
	}

	if targetVal.IsNil() {
		targetVal.Set(reflect.MakeSlice(targetVal.Type(), 0, 0))
	}

	if targetVal.Len() > sourceVal.Len() {
		// Shorten the slice
		tempVal := reflect.MakeSlice(targetVal.Type(), sourceVal.Len(), sourceVal.Len())
		for i := 0; i < sourceVal.Len(); i++ {
			tempVal.Index(i).Set(targetVal.Index(i))
		}

		targetVal.Set(tempVal)
	}

	if targetVal.Len() < sourceVal.Len() {
		// Equalize the slice
		tempVal := reflect.MakeSlice(targetVal.Type(), sourceVal.Len(), sourceVal.Len())

		for i := 0; i < targetVal.Len(); i++ {
			tempVal.Index(i).Set(targetVal.Index(i))
		}
		for i := targetVal.Len(); i < sourceVal.Len(); i++ {
			tempVal.Index(i).Set(reflect.Zero(targetVal.Type().Elem()))
		}

		targetVal.Set(tempVal)
	}

	elements := make([]*yaml.Node, sourceVal.Len())

	for i := 0; i < sourceVal.Len(); i++ {
		var currentElementNode *yaml.Node
		if valueNode != nil && i < len(valueNode.Content) {
			currentElementNode = valueNode.Content[i]
		}

		var err error
		currentElementNode, err = SyncValue(ctx, sourceVal.Index(i).Interface(), targetVal.Index(i).Addr().Interface(), currentElementNode)
		if err != nil {
			return nil, err
		}

		if currentElementNode == nil {
			panic("unexpected nil node")
		}

		elements[i] = currentElementNode
	}

	return yml.CreateOrUpdateSliceNode(ctx, elements, valueNode), nil
}

func fullyDereference(val reflect.Value, typ reflect.Type) (reflect.Value, reflect.Type) {
	if typ.Kind() == reflect.Ptr {
		return fullyDereference(val.Elem(), typ.Elem())
	}

	return val, typ
}

// will dereference the last ptr in the type while initializing any higher level pointers
func dereferenceToLastPtr(val reflect.Value, typ reflect.Type) (reflect.Value, reflect.Type) {
	if typ.Kind() == reflect.Ptr && val.IsNil() {
		val.Set(reflect.New(typ.Elem()))
	}
	if typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Ptr {
		return dereferenceToLastPtr(val.Elem(), typ.Elem())
	}

	return val, typ
}
