package marshaller

import (
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"gopkg.in/yaml.v3"
)

// Pre-computed reflection types for performance
var (
	nodeAccessorType   = reflect.TypeOf((*NodeAccessor)(nil)).Elem()
	populatorType      = reflect.TypeOf((*Populator)(nil)).Elem()
	sequencedMapType   = reflect.TypeOf((*interfaces.SequencedMapInterface)(nil)).Elem()
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

		if fieldVal.Kind() == reflect.Ptr {
			if fieldVal.IsNil() {
				continue
			}
		} else if fieldVal.CanAddr() {
			fieldVal = fieldVal.Addr()
		}

		fieldInt := fieldVal.Interface()

		if field.Anonymous {
			if targetSeqMap := getSequencedMapInterface(tField); targetSeqMap != nil {
				sourceForPopulation := getSourceForPopulation(s.Field(i), fieldInt)
				if err := populateSequencedMap(sourceForPopulation, targetSeqMap); err != nil {
					return err
				}
			}
			continue
		}

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

	// Handle nil source early - when source is nil, reflect.ValueOf returns a zero Value
	if !value.IsValid() {
		// Set target to zero value and return
		if target.Kind() == reflect.Ptr {
			target.Set(reflect.Zero(target.Type()))
		} else {
			target.Set(reflect.Zero(target.Type()))
		}
		return nil
	}

	valueType := value.Type()
	valueKind := value.Kind()

	// Skip NodeAccessor check if already extracted in Populate()
	if valueType.Implements(nodeAccessorType) {
		source = source.(NodeAccessor).GetValue()
		value = reflect.ValueOf(source)

		// Check again after extracting from NodeAccessor
		if !value.IsValid() {
			// Set target to zero value and return
			if target.Kind() == reflect.Ptr {
				target.Set(reflect.Zero(target.Type()))
			} else {
				target.Set(reflect.Zero(target.Type()))
			}
			return nil
		}

		valueKind = value.Kind()
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
		return populateSequencedMap(value.Interface(), target.Interface().(interfaces.SequencedMapInterface))
	}

	// Check if target implements CoreSetter interface
	if coreSetter, ok := target.Interface().(CoreSetter); ok {
		if err := populateModel(value.Interface(), target.Interface()); err != nil {
			return err
		}

		coreSetter.SetCoreAny(value.Interface())
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

// getSequencedMapInterface checks if the field implements SequencedMapInterface and returns it
// Handles both pointer and value embeds, initializing if necessary
func getSequencedMapInterface(tField reflect.Value) interfaces.SequencedMapInterface {
	// Check if the TARGET field implements SequencedMapInterface (either directly or via pointer)
	implementsSeqMap := implementsInterface(tField, sequencedMapType)

	if !implementsSeqMap && tField.CanAddr() {
		// For value embeds, check if a pointer to the target field implements the interface
		ptrType := tField.Addr().Type()
		seqMapInterfaceType := reflect.TypeOf((*interfaces.SequencedMapInterface)(nil)).Elem()
		implementsSeqMap = ptrType.Implements(seqMapInterfaceType)
	}

	if !implementsSeqMap {
		return nil
	}

	// Handle embedded sequenced maps directly
	var targetSeqMap interfaces.SequencedMapInterface
	var ok bool

	// For value embeds, initialize the target field if it's not initialized
	if tField.Kind() != reflect.Ptr {
		// This is a value embed - check if it needs initialization
		if tField.CanAddr() {
			if seqMapInterface, ok := tField.Addr().Interface().(interfaces.SequencedMapInterface); ok {
				if !seqMapInterface.IsInitialized() {
					// Initialize the value embed by creating a new instance and copying it
					newInstance := CreateInstance(tField.Type())
					tField.Set(newInstance.Elem())
				}
			}
			targetSeqMap, ok = tField.Addr().Interface().(interfaces.SequencedMapInterface)
		}
	} else {
		// Pointer embed
		if tField.IsNil() {
			tField.Set(CreateInstance(tField.Type().Elem()))
		}
		targetSeqMap, ok = tField.Interface().(interfaces.SequencedMapInterface)
	}

	if ok {
		return targetSeqMap
	}
	return nil
}

// getSourceForPopulation prepares the source field for population
// Handles addressability issues for value embeds
func getSourceForPopulation(originalFieldVal reflect.Value, fieldInt any) any {
	if originalFieldVal.CanAddr() {
		return originalFieldVal.Addr().Interface()
	} else if originalFieldVal.Kind() == reflect.Ptr {
		return originalFieldVal.Interface()
	} else {
		// Create an addressable copy for value embeds so we can use the interface
		ptrType := reflect.PointerTo(originalFieldVal.Type())
		if ptrType.Implements(sequencedMapType) {
			addressableCopy := reflect.New(originalFieldVal.Type())
			addressableCopy.Elem().Set(originalFieldVal)
			return addressableCopy.Interface()
		} else {
			return fieldInt
		}
	}
}

func isEmbeddedSequencedMapType(t reflect.Type) bool {
	// Check both value type and pointer type
	implementsSequencedMap := t.Implements(sequencedMapType) || reflect.PointerTo(t).Implements(sequencedMapType)
	implementsCoreModeler := t.Implements(coreModelerType) || reflect.PointerTo(t).Implements(coreModelerType)

	return implementsSequencedMap && implementsCoreModeler
}
