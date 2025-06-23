package marshaller

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

// Pre-computed reflection types for performance
var (
	nodeAccessorType   = reflect.TypeOf((*NodeAccessor)(nil)).Elem()
	populatorType      = reflect.TypeOf((*Populator)(nil)).Elem()
	sequencedMapType   = reflect.TypeOf((*sequencedMapInterface)(nil)).Elem()
	coreModelerType    = reflect.TypeOf((*CoreModeler)(nil)).Elem()
	yamlNodePtrType    = reflect.TypeOf((*yaml.Node)(nil))
	yamlNodeType       = reflect.TypeOf(yaml.Node{})
	yamlNodePtrPtrType = reflect.TypeOf((**yaml.Node)(nil))
	populatorValueTag  = "populatorValue"
	populatorValueTrue = "true"
)

type Populator interface {
	Populate(source any) error
}

func Populate(source any, target any) error {
	t := reflect.ValueOf(target)

	if t.Kind() == reflect.Ptr && t.IsNil() {
		t.Set(CreateInstance(t.Type().Elem()))
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	s := reflect.ValueOf(source)
	if s.Type().Implements(nodeAccessorType) {
		source = source.(NodeAccessor).GetValue()
	}

	// Special case for yaml.Node conversion (similar to unmarshaller.go:216-223)
	switch {
	case t.Type() == yamlNodePtrType:
		if node, ok := source.(yaml.Node); ok {
			t.Set(reflect.ValueOf(&node))
			return nil
		}
	case t.Type() == yamlNodeType:
		if node, ok := source.(*yaml.Node); ok {
			t.Set(reflect.ValueOf(*node))
			return nil
		}
	case t.Type() == yamlNodePtrPtrType:
		if node, ok := source.(*yaml.Node); ok {
			t.Set(reflect.ValueOf(&node))
			return nil
		}
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
		t.Set(CreateInstance(t.Type().Elem()))
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if s.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %s", s.Kind())
	}

	sType := s.Type()
	numFields := s.NumField()

	for i := 0; i < numFields; i++ {
		field := sType.Field(i)
		if !field.IsExported() {
			continue
		}

		useFieldValue := field.Tag.Get(populatorValueTag) == populatorValueTrue
		tField := t.FieldByIndex(field.Index)
		if !tField.IsValid() {
			continue
		}

		fieldVal := s.Field(i)

		if field.Anonymous {
			if implementsInterface[sequencedMapInterface](fieldVal) {
				useFieldValue = true
			} else {
				continue
			}
		}

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
				return fmt.Errorf("expected ExtensionCoreMap, got %v (interface type: %v, field name: %s.%s)",
					fieldVal.Type(), reflect.TypeOf(fieldInt), sType.Name(), field.Name)
			}

			if tField.Kind() == reflect.Ptr {
				tField.Set(CreateInstance(tField.Type().Elem()))
			}

			tem, ok := tField.Interface().(ExtensionMap)
			if !ok {
				return fmt.Errorf("expected ExtensionMap, got %v (interface type: %v, target field: %s)",
					tField.Type(), reflect.TypeOf(tField.Interface()), field.Name)
			}
			tem.Init()

			for key, value := range sem.All() {
				tem.Set(key, value.Value)
			}

			tem.SetCore(fieldInt)

			continue
		}

		var nodeValue any

		if useFieldValue {
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
	valueType := value.Type()
	valueKind := value.Kind()

	// Skip NodeAccessor check if already extracted in Populate()
	if valueType.Implements(nodeAccessorType) {
		source = source.(NodeAccessor).GetValue()
		value = reflect.ValueOf(source)
	}

	if valueKind == reflect.Ptr && value.IsNil() && target.Kind() == reflect.Ptr {
		target.Set(reflect.Zero(target.Type()))
		return nil
	}

	if target.Kind() == reflect.Ptr {
		target.Set(CreateInstance(target.Type().Elem()))
	} else {
		target = target.Addr()
	}

	targetType := target.Type()
	if targetType.Implements(populatorType) {
		return target.Interface().(Populator).Populate(value.Interface())
	}

	// Check if target is a sequenced map and handle it specially
	if targetType.Implements(sequencedMapType) && !isEmbeddedSequencedMapType(value.Type()) {
		return populateSequencedMap(value.Interface(), target.Interface().(sequencedMapInterface))
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
	if valueKind == reflect.Ptr {
		valueDerefed = value.Elem()
	}

	switch valueDerefed.Kind() {
	case reflect.Slice, reflect.Array:
		if valueDerefed.IsNil() {
			return nil
		}

		target.Set(reflect.MakeSlice(target.Type(), valueDerefed.Len(), valueDerefed.Len()))

		for i := 0; i < valueDerefed.Len(); i++ {
			elementValue := valueDerefed.Index(i).Interface()

			// Extract value from NodeAccessor if needed for array elements
			if valueDerefed.Index(i).Type().Implements(nodeAccessorType) {
				if nodeAccessor, ok := elementValue.(NodeAccessor); ok {
					elementValue = nodeAccessor.GetValue()
				}
			}

			if err := populateValue(elementValue, target.Index(i)); err != nil {
				return err
			}
		}
	default:
		if !valueDerefed.IsValid() {
			// Handle zero/invalid values
			target.Set(reflect.Zero(target.Type()))
			return nil
		}
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

func isEmbeddedSequencedMapType(t reflect.Type) bool {
	// Check both value type and pointer type
	implementsSequencedMap := t.Implements(sequencedMapType) || reflect.PtrTo(t).Implements(sequencedMapType)
	implementsCoreModeler := t.Implements(coreModelerType) || reflect.PtrTo(t).Implements(coreModelerType)

	return implementsSequencedMap && implementsCoreModeler
}
