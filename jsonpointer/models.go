package jsonpointer

import (
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
)

type model interface {
	GetCoreAny() any
	SetCoreAny(core any)
}

func navigateModel(sourceVal reflect.Value, currentPart navigationPart, stack []navigationPart, currentPath string, o *options) (any, []navigationPart, error) {
	// Models support both key-based and index-based navigation (treat index as key)

	// Ensure we have a model interface
	if !sourceVal.CanInterface() {
		return nil, nil, fmt.Errorf("source value cannot be interfaced at %s", currentPath)
	}

	modelInterface, ok := sourceVal.Interface().(model)
	if !ok {
		return nil, nil, fmt.Errorf("expected model interface, got %s at %s", sourceVal.Type(), currentPath)
	}

	// Get the core model
	coreAny := modelInterface.GetCoreAny()
	if coreAny == nil {
		return nil, nil, fmt.Errorf("core model is nil at %s", currentPath)
	}

	coreVal := reflect.ValueOf(coreAny)
	if coreVal.Kind() == reflect.Ptr {
		if coreVal.IsNil() {
			return nil, nil, fmt.Errorf("core model pointer is nil at %s", currentPath)
		}
		coreVal = coreVal.Elem()
	}

	if coreVal.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("expected core model to be struct, got %s at %s", coreVal.Kind(), currentPath)
	}

	key := currentPart.unescapeValue()

	sourceType := sourceVal.Type()
	if sourceType.Kind() == reflect.Ptr {
		sourceType = sourceType.Elem()
	}

	for i := 0; i < sourceType.NumField(); i++ {
		field := sourceType.Field(i)
		if field.Anonymous {
			fieldVal := sourceVal
			if fieldVal.Kind() == reflect.Ptr {
				fieldVal = fieldVal.Elem()
			}
			embeddedField := fieldVal.Field(i)

			// Check if the field is an embedded sequenced map
			fieldType := embeddedField.Type()

			// Handle both pointer and value embeds
			var keyNavigable KeyNavigable
			var ok bool

			if fieldType.Kind() == reflect.Ptr {
				// Pointer embed: check if the field itself implements the interface
				if !embeddedField.IsNil() {
					keyNavigable, ok = embeddedField.Interface().(KeyNavigable)
				}
			} else {
				// Value embed: check if the pointer to the field implements the interface
				ptrType := reflect.PointerTo(fieldType)
				if interfaces.ImplementsInterface[interfaces.SequencedMapInterface](ptrType) {
					if embeddedField.CanAddr() {
						keyNavigable, ok = embeddedField.Addr().Interface().(KeyNavigable)
					}
				}
			}

			if ok && keyNavigable != nil {
				if value, err := keyNavigable.NavigateWithKey(key); err == nil {
					return getCurrentStackTarget(value, stack, currentPath, o)
				}
			}
		}
	}

	// Find the corresponding field in the core model by matching the key tag
	coreFieldIndex := -1
	for i := 0; i < coreVal.NumField(); i++ {
		field := coreVal.Type().Field(i)
		if !field.IsExported() {
			continue
		}

		keyTag := field.Tag.Get("key")
		if keyTag == key {
			coreFieldIndex = i
			break
		}
	}

	if coreFieldIndex == -1 {
		// Field not found in core model, try searching the associated YAML node
		// Check if the model implements CoreModeler interface (which has GetRootNode)
		if coreModeler, ok := coreAny.(marshaller.CoreModeler); ok {
			rootNode := coreModeler.GetRootNode()
			if rootNode != nil {
				// Use the existing YAML node navigation logic to search for the key
				result, newStack, err := getYamlNodeTarget(rootNode, currentPart, stack, currentPath, o)
				if err == nil {
					return result, newStack, nil
				}
			}
		}

		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("key %s not found in core model or YAML node at %s", currentPart.Value, currentPath))
	}

	// Find the corresponding field in the high-level model
	// The field should have the same name as the core field (without the marshaller.Node wrapper)
	coreFieldName := coreVal.Type().Field(coreFieldIndex).Name

	sourceType = sourceVal.Type()
	if sourceType.Kind() == reflect.Ptr {
		sourceType = sourceType.Elem()
	}

	highField, found := sourceType.FieldByName(coreFieldName)
	if !found {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("field %s not found in high-level model at %s", coreFieldName, currentPath))
	}

	// Get the field value from the high-level model
	highVal := sourceVal
	if highVal.Kind() == reflect.Ptr {
		highVal = highVal.Elem()
	}

	fieldVal := highVal.FieldByIndex(highField.Index)
	if !fieldVal.IsValid() {
		return nil, nil, ErrNotFound.Wrap(fmt.Errorf("field %s is not valid at %s", coreFieldName, currentPath))
	}

	// If this is the final navigation (no more parts in stack), return the field value directly
	if len(stack) == 0 {
		return fieldVal.Interface(), stack, nil
	}

	// For intermediate navigation, we need to handle value types that implement model interface
	var target any
	if fieldVal.Kind() != reflect.Ptr && fieldVal.CanAddr() {
		// Check if this value type implements the model interface when addressed
		addrVal := fieldVal.Addr()
		if _, ok := addrVal.Interface().(model); ok {
			// If it's a model, take its address for further navigation
			target = addrVal.Interface()
		} else {
			// If it's not a model, use the value as-is
			target = fieldVal.Interface()
		}
	} else {
		// For pointer types or non-addressable values, use as-is
		target = fieldVal.Interface()
	}

	// Continue navigation with the remaining stack
	return getCurrentStackTarget(target, stack, currentPath, o)
}
