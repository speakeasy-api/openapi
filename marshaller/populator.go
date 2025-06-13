package marshaller

import (
	"fmt"
	"reflect"
)

type Populator interface {
	Populate(source any) error
}

func Populate(source any, target any) error {
	t := reflect.ValueOf(target)

	if t.Kind() == reflect.Ptr && t.IsNil() {
		t.Set(reflect.New(t.Type().Elem()))
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	s := reflect.ValueOf(source)
	if s.Type().Implements(reflect.TypeOf((*NodeAccessor)(nil)).Elem()) {
		source = source.(NodeAccessor).GetValue()
	}

	return populateValue(source, t)
}

func populateModel(source any, target any) error {
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

			tem.SetCore(fieldInt)

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

		if err := populateValue(nodeValue, tField); err != nil {
			return err
		}
	}

	return nil
}

func populateValue(source any, target reflect.Value) error {
	value := reflect.ValueOf(source)

	if value.Kind() == reflect.Ptr && value.IsNil() && target.Kind() == reflect.Ptr {
		target.Set(reflect.Zero(target.Type()))
		return nil
	}

	if target.Kind() == reflect.Ptr {
		target.Set(reflect.New(target.Type().Elem()))
	} else {
		target = target.Addr()
	}

	if target.Type().Implements(reflect.TypeOf((*Populator)(nil)).Elem()) {
		return target.Interface().(Populator).Populate(value.Interface())
	}

	// Check if target implements CoreSetter interface
	if coreSetter, ok := target.Interface().(CoreSetter); ok {
		if err := populateModel(value.Interface(), target.Interface()); err != nil {
			return err
		}

		coreSetter.SetCoreValue(value.Interface())
		return nil
	}

	target = target.Elem()

	valueDerefed := value
	if value.Kind() == reflect.Ptr {
		valueDerefed = value.Elem()
	}

	switch valueDerefed.Kind() {
	case reflect.Slice, reflect.Array:
		if valueDerefed.IsNil() {
			return nil
		}

		target.Set(reflect.MakeSlice(target.Type(), valueDerefed.Len(), valueDerefed.Len()))

		for i := 0; i < valueDerefed.Len(); i++ {
			if err := populateValue(valueDerefed.Index(i).Interface(), target.Index(i)); err != nil {
				return err
			}
		}
	default:
		if valueDerefed.Type().AssignableTo(target.Type()) {
			target.Set(valueDerefed)
		} else if valueDerefed.CanConvert(target.Type()) {
			target.Set(valueDerefed.Convert(target.Type()))
		} else {
			return fmt.Errorf("cannot convert %v to %v", valueDerefed.Type(), target.Type())
		}
	}

	return nil
}
