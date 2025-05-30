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
	SyncChangesWithSyncFunc(ctx context.Context, model any, valueNode *yaml.Node, syncFunc func(context.Context, any, any, *yaml.Node, bool) (*yaml.Node, error)) (*yaml.Node, error)
}

func SyncValue(ctx context.Context, source any, target any, valueNode *yaml.Node, skipCustomSyncer bool) (node *yaml.Node, err error) {
	s := reflect.ValueOf(source)
	t := reflect.ValueOf(target)

	if t.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("SyncValue expected target to be a pointer, got %s", t.Kind())
	}

	s = dereferenceToLastPtr(s)
	t = dereferenceAndInitializeIfNeededToLastPtr(t, reflect.ValueOf(source))

	if s.Kind() == reflect.Ptr && s.IsNil() {
		if !t.IsZero() {
			t.Elem().Set(reflect.Zero(t.Type().Elem()))
		}
		return nil, nil
	}

	sUnderlying := getUnderlyingValue(s)
	tUnderlyingType := dereferenceType(t.Type())

	if sUnderlying.Kind() != tUnderlyingType.Kind() {
		return nil, fmt.Errorf("SyncValue expected target to be %s, got %s", sUnderlying.Kind(), tUnderlyingType.Kind())
	}

	switch {
	case sUnderlying.Kind() == reflect.Struct && t.Type() == reflect.TypeOf((*yaml.Node)(nil)):
		t.Set(s)
		return t.Interface().(*yaml.Node), nil
	case sUnderlying.Kind() == reflect.Struct:
		if !skipCustomSyncer {
			syncer, ok := t.Interface().(Syncer)
			if ok {
				sv := s.Interface()

				return syncer.SyncChanges(ctx, sv, valueNode)
			}

			syncerWithSyncFunc, ok := t.Interface().(SyncerWithSyncFunc)
			if ok {
				sv := s.Interface()

				return syncerWithSyncFunc.SyncChangesWithSyncFunc(ctx, sv, valueNode, SyncValue)
			}
		}

		return syncChanges(ctx, s.Interface(), t.Interface(), valueNode)
	case sUnderlying.Kind() == reflect.Map:
		// TODO call sync changes on each value
		panic("not implemented")
	case sUnderlying.Kind() == reflect.Slice, sUnderlying.Kind() == reflect.Array:
		return syncArraySlice(ctx, sUnderlying.Interface(), t.Interface(), valueNode)
	default:
		if sUnderlying.Type() != tUnderlyingType {
			// Cast the value to the target type
			sUnderlying = sUnderlying.Convert(tUnderlyingType)
		}
		if !t.Elem().IsValid() {
			t.Set(reflect.New(tUnderlyingType))
		}
		t.Elem().Set(sUnderlying)
		out := yml.CreateOrUpdateScalarNode(ctx, sUnderlying.Interface(), valueNode)
		return out, nil
	}
}

func syncChanges(ctx context.Context, source any, target any, valueNode *yaml.Node) (*yaml.Node, error) {
	s := reflect.ValueOf(source)
	t := reflect.ValueOf(target)

	if s.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("syncChanges expected source to be a pointer, got %s", s.Kind())
	}

	if t.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("syncChanges expected target to be a pointer, got %s", t.Kind())
	}

	if s.IsNil() {
		panic("not implemented")
	}

	if t.IsNil() {
		t.Set(reflect.New(t.Elem().Type()))
	}

	sUnderlying := getUnderlyingValue(s)
	t = getUnderlyingValue(t)

	if sUnderlying.Kind() != reflect.Struct {
		return nil, fmt.Errorf("syncChanges expected struct, got %s", s.Type())
	}

	valid := true

	for i := 0; i < t.NumField(); i++ {
		field := t.Type().Field(i)
		if !field.IsExported() {
			continue
		}
		sourceVal := sUnderlying.FieldByName(field.Name)

		key := field.Tag.Get("key")
		if key == "" {
			continue
		}

		target := t.Field(i)
		if target.Kind() != reflect.Ptr {
			if target.CanAddr() {
				target = target.Addr()
			} else {
				continue
			}
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
		var sourceInt any
		if !sourceVal.IsValid() {
			continue
		}
		if sourceVal.CanAddr() {
			sourceInt = sourceVal.Addr().Interface()
		} else {
			sourceInt = sourceVal.Interface()
		}

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
			nodeMutator.SetPresent(true)
		} else {
			valueNode = yml.DeleteMapNodeElement(ctx, key, valueNode)
			nodeMutator.SetPresent(false)
		}

		// Check if this field is required for validity
		if valid {
			requiredTag := field.Tag.Get("required")
			required := requiredTag == "true"

			if requiredTag == "" {
				fieldValue := t.Field(i)
				if nodeAccessor, ok := fieldValue.Interface().(NodeAccessor); ok {
					fieldType := nodeAccessor.GetValueType()

					if fieldType.Kind() != reflect.Ptr {
						required = fieldType.Kind() != reflect.Map && fieldType.Kind() != reflect.Slice && fieldType.Kind() != reflect.Array
					}
				}
			}

			if required {
				fieldValue := t.Field(i)
				// Check if the field has a Present boolean field (for Node[T] types)
				if presentField := fieldValue.FieldByName("Present"); presentField.IsValid() && presentField.Kind() == reflect.Bool {
					if !presentField.Bool() {
						valid = false
					}
				} else if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
					// Fallback for non-Node fields
					valid = false
				}
			}
		}
	}

	// Populate the RootNode of the target with the result
	if coreModel, ok := t.Addr().Interface().(CoreModeler); ok {
		coreModel.SetRootNode(valueNode)
	} else {
		return nil, fmt.Errorf("SyncChanges expected target to implement CoreModeler, got %s", t.Type())
	}

	// Update the core of the source with the updated value
	if coreSetter, ok := s.Interface().(CoreSetter); ok {
		coreSetter.SetCoreValue(t.Interface())
	}

	// Set validity on the core model
	if coreModel, ok := t.Addr().Interface().(CoreModeler); ok {
		coreModel.SetValid(valid)
	}

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

		var sourceValAtIdx any
		if sourceVal.Index(i).CanAddr() {
			sourceValAtIdx = sourceVal.Index(i).Addr().Interface()
		} else {
			sourceValAtIdx = sourceVal.Index(i).Interface()
		}

		currentElementNode, err = SyncValue(ctx, sourceValAtIdx, targetVal.Index(i).Addr().Interface(), currentElementNode, false)
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

// will dereference the last ptr in the type while initializing any higher level pointers
func dereferenceAndInitializeIfNeededToLastPtr(val reflect.Value, source reflect.Value) reflect.Value {
	if val.Kind() == reflect.Ptr && val.IsNil() {
		if (source.Kind() == reflect.Ptr && !source.IsNil()) || (source.Kind() != reflect.Ptr && source.IsValid()) {
			val.Set(reflect.New(val.Type().Elem()))
		}
	}
	if val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Ptr {
		sourceVal := source
		if sourceVal.Kind() == reflect.Ptr {
			sourceVal = sourceVal.Elem()
		}

		return dereferenceAndInitializeIfNeededToLastPtr(val.Elem(), sourceVal)
	}

	return val
}

// will dereference the last ptr in the type
func dereferenceToLastPtr(val reflect.Value) reflect.Value {
	if val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Ptr {
		return dereferenceToLastPtr(val.Elem())
	}

	return val
}

func getUnderlyingValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	return v
}

func dereferenceType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Ptr {
		return dereferenceType(typ.Elem())
	}

	return typ
}
