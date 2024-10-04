package marshaller

import (
	"fmt"
	"reflect"
	"unsafe"
)

type ModelFromCore interface {
	FromCore(c any) error
}

func PopulateModel(source any, target any) error {
	s := reflect.ValueOf(source)
	t := reflect.ValueOf(target)

	if s.Kind() == reflect.Ptr {
		if s.IsNil() {
			return nil
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
		return fmt.Errorf("expected struct, got %s", s.Kind())
	}

	for i := 0; i < s.NumField(); i++ {
		field := s.Type().Field(i)
		if !field.IsExported() {
			continue
		}

		tField := t.FieldByName(field.Name)
		if !tField.IsValid() {
			continue
		}

		fieldVal := s.Field(i)
		if fieldVal.Kind() == reflect.Ptr {
			if fieldVal.IsNil() {
				continue
			}
		} else if fieldVal.CanAddr() {
			fieldVal = fieldVal.Addr()
		}

		fieldInt := fieldVal.Interface()

		if field.Name == "Extensions" {
			sem, ok := fieldInt.(ExtensionCoreMap)
			if !ok {
				return fmt.Errorf("expected ExtensionCoreMap, got %v", fieldVal.Type())
			}

			if tField.Kind() == reflect.Ptr {
				tField.Set(reflect.New(tField.Type().Elem()))
			}

			tem, ok := tField.Interface().(ExtensionMap)
			if !ok {
				return fmt.Errorf("expected ExtensionMap, got %v", tField.Type())
			}
			tem.Init()

			for key, value := range sem.All() {
				tem.Set(key, value.Value)
			}

			continue
		}

		var nodeValue any

		if field.Tag.Get("populatorValue") == "true" {
			nodeValue = fieldInt
		} else {
			nodeAccessor, ok := fieldInt.(NodeAccessor)
			if !ok {
				return fmt.Errorf("expected NodeAccessor, got %v", fieldVal.Type())
			}

			nodeValue = nodeAccessor.GetValue()
		}

		if err := populateValue(tField, nodeValue); err != nil {
			return err
		}
	}

	return nil
}

func populateValue(target reflect.Value, nodeValue any) error {
	value := reflect.ValueOf(nodeValue)

	if value.Kind() == reflect.Ptr && value.IsNil() && target.Kind() == reflect.Ptr {
		target.Set(reflect.Zero(target.Type()))
		return nil
	}

	if target.Kind() == reflect.Ptr {
		target.Set(reflect.New(target.Type().Elem()))
	} else {
		target = target.Addr()
	}

	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if target.Type().Implements(reflect.TypeOf((*ModelFromCore)(nil)).Elem()) {
		return target.Interface().(ModelFromCore).FromCore(value.Interface())
	}

	// TODO we are trusting core is a core model we may want to add some sort of marker interface to ensure this is the case
	if target.Elem().Kind() == reflect.Struct {
		cf, ok := target.Elem().Type().FieldByName("core")
		if ok {
			if cf.Type != value.Type() {
				return fmt.Errorf("populateValue expected core field to be of type %s, got %s", cf.Type, value.Type())
			}

			if err := PopulateModel(value.Interface(), target.Interface()); err != nil {
				return err
			}

			tf := target.Elem().FieldByIndex(cf.Index)
			reflect.NewAt(tf.Type(), unsafe.Pointer(tf.UnsafeAddr())).Elem().Set(value)
			return nil
		}
	}

	if target.Type().Implements(reflect.TypeOf((*SequencedMap)(nil)).Elem()) {
		return populateSequencedMap(value, target)
	}

	target = target.Elem()

	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		if value.IsNil() {
			return nil
		}

		target.Set(reflect.MakeSlice(target.Type(), value.Len(), value.Len()))

		for i := 0; i < value.Len(); i++ {
			if err := populateValue(target.Index(i), value.Index(i).Interface()); err != nil {
				return err
			}
		}
	default:
		if value.Type().AssignableTo(target.Type()) {
			target.Set(value)
		} else if value.CanConvert(target.Type()) {
			target.Set(value.Convert(target.Type()))
		} else {
			return fmt.Errorf("cannot convert %v to %v", value.Type(), target.Type())
		}
	}

	return nil
}

func populateSequencedMap(source reflect.Value, target reflect.Value) error {
	sm, ok := source.Addr().Interface().(SequencedMap)
	if !ok {
		return fmt.Errorf("expected source to be SequencedMap, got %s", source.Type())
	}

	tm, ok := target.Interface().(SequencedMap)
	if !ok {
		return fmt.Errorf("expected target to be SequencedMap, got %s", target.Type())
	}

	tm.Init()

	for key, value := range sm.AllUntyped() {
		targetValue := reflect.New(tm.GetValueType()).Elem()
		if err := populateValue(targetValue, value); err != nil {
			return err
		}
		if err := tm.SetUntyped(key, targetValue.Interface()); err != nil {
			return err
		}
	}

	return nil
}