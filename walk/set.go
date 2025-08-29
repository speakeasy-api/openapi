package walk

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/internal/interfaces"
)

type model interface {
	GetCoreAny() any
	SetCoreAny(core any)
}

func SetAtLocation[T any](parent any, l LocationContext[T], value any) error {
	parentVal := reflect.ValueOf(parent)
	if parentVal.Kind() != reflect.Ptr {
		return errors.New("parent value must be a pointer")
	}
	originalPtr := parentVal
	parentVal = parentVal.Elem()
	if parentVal.Kind() == reflect.Interface {
		parentVal = parentVal.Elem()
	}

	switch parentVal.Kind() {
	case reflect.Map:
		return setAtMap(parentVal, l, value)
	case reflect.Slice:
		return setAtSlice(parentVal, l, value)
	case reflect.Struct:
		return setAtStruct(originalPtr, l, value)
	default:
		return fmt.Errorf("expected map, slice, or struct, got %s", parentVal.Kind())
	}
}

func setAtMap[T any](parentVal reflect.Value, l LocationContext[T], value any) error {
	if l.ParentKey == nil {
		return errors.New("parent key is nil")
	}

	parentVal.SetMapIndex(reflect.ValueOf(*l.ParentKey), reflect.ValueOf(value))

	return nil
}

func setAtSlice[T any](parentVal reflect.Value, l LocationContext[T], value any) error {
	if l.ParentIndex == nil {
		return errors.New("parent index is nil")
	}

	parentVal.Index(*l.ParentIndex).Set(reflect.ValueOf(value))

	return nil
}

func setAtStruct[T any](parentVal reflect.Value, l LocationContext[T], value any) error {
	// Ensure we have a model interface
	if !parentVal.CanInterface() {
		return errors.New("parent value cannot be interfaced")
	}

	// Check if this struct implements SequencedMapInterface and we have a ParentKey
	// This means we're setting a key in the sequenced map
	if l.ParentKey != nil {
		if sequencedmap, ok := parentVal.Interface().(interfaces.SequencedMapInterface); ok {
			return setAtSequencedMap(sequencedmap, l, value)
		}
	}

	// Otherwise, check if this is a model interface and try to set a field
	modelInterface, isModel := parentVal.Interface().(model)
	if isModel {
		return setAtField(parentVal, modelInterface, l, value)
	}

	return errors.New("expected model interface or sequenced map interface")
}

func setAtField[T any](parentVal reflect.Value, model model, l LocationContext[T], value any) error {
	// Get the core model
	coreAny := model.GetCoreAny()
	if coreAny == nil {
		return errors.New("core model is nil")
	}

	coreVal := reflect.ValueOf(coreAny)
	if coreVal.Kind() == reflect.Ptr {
		if coreVal.IsNil() {
			return errors.New("core model pointer is nil")
		}
		coreVal = coreVal.Elem()
	}

	if coreVal.Kind() != reflect.Struct {
		return fmt.Errorf("expected core model to be struct, got %s", coreVal.Kind())
	}

	coreType := coreVal.Type()
	if coreType.Kind() == reflect.Ptr {
		coreType = coreType.Elem()
	}

	// Handle case where we have both ParentField and ParentKey/ParentIndex
	// This means we need to find the field first, then recursively call SetAtLocation on that field
	if l.ParentField != "" && (l.ParentKey != nil || l.ParentIndex != nil) {
		// Find the field by ParentField in the core model to get the correct index
		coreFieldIndex := -1
		for i := 0; i < coreType.NumField(); i++ {
			field := coreType.Field(i)
			if !field.IsExported() {
				continue
			}

			keyTag := field.Tag.Get("key")
			if keyTag == l.ParentField {
				coreFieldIndex = i
				break
			}
		}

		if coreFieldIndex == -1 {
			return fmt.Errorf("field %s not found in core model", l.ParentField)
		}

		// Use the same index to get the field from the high-level model
		highLevelVal := parentVal.Elem()
		field := highLevelVal.Field(coreFieldIndex)
		if !field.CanSet() {
			return fmt.Errorf("field %s is not settable", l.ParentField)
		}

		// Create a new LocationContext with just the key/index and recursively call SetAtLocation
		// We need to get a pointer to the actual field, not a copy from Interface()
		if !field.CanAddr() {
			return fmt.Errorf("field %s cannot be addressed", l.ParentField)
		}

		var fieldPtr interface{}
		if field.Kind() == reflect.Ptr {
			// If the field is already a pointer, use it directly
			fieldPtr = field.Interface()
		} else {
			// If the field is not a pointer, get its address
			fieldPtr = field.Addr().Interface()
		}

		newLocationCtx := LocationContext[T]{
			ParentKey:   l.ParentKey,
			ParentIndex: l.ParentIndex,
		}

		return SetAtLocation(fieldPtr, newLocationCtx, value)
	}

	if l.ParentField == "" {
		return errors.New("parent field is unset")
	}

	coreFieldIndex := -1
	for i := 0; i < coreType.NumField(); i++ {
		field := coreType.Field(i)
		if !field.IsExported() {
			continue
		}

		keyTag := field.Tag.Get("key")
		if keyTag == l.ParentField {
			coreFieldIndex = i
			break
		}
	}

	if coreFieldIndex == -1 {
		return fmt.Errorf("field %s not found in core model", l.ParentField)
	}

	field := parentVal.Elem().Field(coreFieldIndex)
	if !field.CanSet() {
		return fmt.Errorf("field %s is not settable", l.ParentField)
	}

	field.Set(reflect.ValueOf(value))

	return nil
}

func setAtSequencedMap[T any](sequencedmap interfaces.SequencedMapInterface, l LocationContext[T], value any) error {
	if l.ParentKey == nil {
		return errors.New("parent key is nil")
	}

	sequencedmap.SetAny(*l.ParentKey, value)

	return nil
}
