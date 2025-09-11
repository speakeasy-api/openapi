package marshaller

import (
	"context"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

type Syncer interface {
	SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error)
}

// SyncValue syncs changes from source to target and returns the updated YAML node.
// For proper YAML node styling (quoted strings, etc.), ensure the context contains
// the config via yml.ContextWithConfig(ctx, config) before calling this function.
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

	// Handle NodeMutator types (like marshaller.Node[T]) by using their SyncValue method
	if t.CanInterface() {
		if nodeMutator, ok := t.Interface().(NodeMutator); ok {
			// Use the NodeMutator's SyncValue method which handles addressability correctly
			_, valueNode, err := nodeMutator.SyncValue(ctx, "", source)
			if err != nil {
				return nil, err
			}
			return valueNode, nil
		}
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

			// If this is an embedded sequenced map, skip the SyncerWithSyncFunc method and use syncChanges instead
			if isEmbeddedSequencedMap(t) {
				return syncChanges(ctx, s.Interface(), t.Interface(), valueNode)
			}

			if implementsInterface(t, sequencedMapType) {
				return syncSequencedMapChanges(ctx, t.Interface().(interfaces.SequencedMapInterface), s.Interface(), valueNode, SyncValue)
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
			t.Set(CreateInstance(tUnderlyingType))
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
		t.Set(CreateInstance(t.Elem().Type()))
	}

	// Handle NodeAccessor types (like marshaller.Node[T]) by extracting their values from target
	if t.CanInterface() {
		if nodeAccessor, ok := t.Interface().(NodeAccessor); ok {
			t = reflect.ValueOf(nodeAccessor.GetValue())
		}
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

		// Handle embedded fields (anonymous fields)
		if field.Anonymous {
			targetField := t.Field(i)
			sourceField := sUnderlying.Field(i)

			if seqMapInterface := initializeAndGetSequencedMapInterface(targetField); seqMapInterface != nil {
				sourceInterface := getSourceInterface(sourceField)
				newValueNode, err := syncSequencedMapChanges(ctx, seqMapInterface, sourceInterface, valueNode, SyncValue)
				if err != nil {
					return nil, err
				}
				valueNode = newValueNode
			}
			continue
		}

		sourceVal := sUnderlying.FieldByName(field.Name)

		key := field.Tag.Get("key")
		if key == "" {
			continue
		}

		fieldTarget := t.Field(i)
		if fieldTarget.Kind() != reflect.Ptr {
			if fieldTarget.CanAddr() {
				fieldTarget = fieldTarget.Addr()
			} else {
				continue
			}
		}

		// If both are nil, we don't need to sync
		if fieldTarget.IsNil() && sourceVal.IsNil() {
			continue
		}

		if key == "extensions" {
			var err error
			valueNode, err = syncExtensions(ctx, sourceVal.Interface(), fieldTarget, valueNode)
			if err != nil {
				return nil, err
			}
			continue
		}

		if fieldTarget.IsNil() {
			fieldTarget.Set(CreateInstance(fieldTarget.Type().Elem()))
		}

		targetInt := fieldTarget.Interface()
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
			return nil, fmt.Errorf("syncChanges expected NodeMutator, got %s", fieldTarget.Type())
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

	// Ensure we have a valid YAML node even for empty structs
	if valueNode == nil {
		// Create an empty mapping node for empty structs
		valueNode = &yaml.Node{
			Kind:  yaml.MappingNode,
			Tag:   "!!map",
			Style: yaml.FlowStyle,
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
		coreSetter.SetCoreAny(t.Interface())
	}

	// Set validity on the core model
	if coreModel, ok := t.Addr().Interface().(CoreModeler); ok {
		coreModel.SetValid(valid, true)
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

	// When arrays are reordered at the high level (e.g., moving workflows around),
	// we need to match source elements with their corresponding target core models
	// by identity (RootNode) rather than by array position to preserve elements.
	reorderedTargets, reorderedNodes := reorderArrayElements(sourceVal, targetVal, valueNode)

	elements := make([]*yaml.Node, sourceVal.Len())

	for i := 0; i < sourceVal.Len(); i++ {
		var sourceValAtIdx any
		if sourceVal.Index(i).CanAddr() {
			sourceValAtIdx = sourceVal.Index(i).Addr().Interface()
		} else {
			sourceValAtIdx = sourceVal.Index(i).Interface()
		}

		var currentElementNode *yaml.Node
		if i < len(reorderedNodes) {
			currentElementNode = reorderedNodes[i]
		}

		var err error
		currentElementNode, err = SyncValue(ctx, sourceValAtIdx, reorderedTargets[i], currentElementNode, false)
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

// reorderArrayElements reorders target array elements and YAML nodes to match source order
// by matching high-level models with their corresponding core models via RootNode identity.
//
// This function solves the problem where array reordering at the high level (e.g., moving
// workflows around) would cause field ordering issues because the sync process was matching
// elements by array position rather than by identity.
//
// The function handles three scenarios:
// 1. Reordering: Source elements are matched with target elements by RootNode identity
// 2. Additions: New source elements (no matching target) get new target slots
// 3. Deletions: Handled automatically as the result arrays are sized to match source length
//
// Returns reordered target elements and YAML nodes that correspond to the source order.
func reorderArrayElements(sourceVal, targetVal reflect.Value, valueNode *yaml.Node) ([]any, []*yaml.Node) {
	sourceLen := sourceVal.Len()
	reorderedTargets := make([]any, sourceLen)
	reorderedNodes := make([]*yaml.Node, sourceLen)

	resolvedValueNode := yml.ResolveAlias(valueNode)

	// Extract original YAML nodes for potential reuse
	var originalNodes []*yaml.Node
	if resolvedValueNode != nil && resolvedValueNode.Content != nil {
		originalNodes = resolvedValueNode.Content
	}

	for i := 0; i < sourceLen; i++ {
		sourceElement := sourceVal.Index(i)

		// Try to get the unique identity (RootNode) of this source element
		var sourceRootNode *yaml.Node
		if sourceElement.CanInterface() {
			if rootNodeAccessor, ok := sourceElement.Interface().(RootNodeAccessor); ok {
				sourceRootNode = rootNodeAccessor.GetRootNode()
			}
		}

		if sourceRootNode == nil {
			// No identity available - this could be a new element or one without RootNode support.
			// Fall back to index-based matching for backward compatibility.
			if i < targetVal.Len() {
				reorderedTargets[i] = targetVal.Index(i).Addr().Interface()
			} else {
				// This is a new element beyond the target array length - create a new target slot
				reorderedTargets[i] = CreateInstance(targetVal.Type().Elem()).Interface()
			}
			if i < len(originalNodes) {
				reorderedNodes[i] = originalNodes[i]
			}
			// Note: reorderedNodes[i] will be nil for new elements, which is correct
			continue
		}

		// Search for a target element with matching RootNode identity
		found := false
		for j := 0; j < targetVal.Len(); j++ {
			targetElement := targetVal.Index(j)

			// Skip nil elements in the target array
			if targetElement.Kind() == reflect.Ptr && targetElement.IsNil() {
				continue
			}

			type valueNodeAccessor interface {
				GetValueNode() *yaml.Node
			}

			// Get the value node from the target element for comparison
			var targetRootNode *yaml.Node
			if targetElement.CanInterface() {
				if rna, ok := targetElement.Interface().(RootNodeAccessor); ok {
					targetRootNode = rna.GetRootNode()
				} else if vna, ok := targetElement.Interface().(valueNodeAccessor); ok {
					targetRootNode = vna.GetValueNode()
				} else if targetElement.CanAddr() {
					if rna, ok := targetElement.Addr().Interface().(RootNodeAccessor); ok {
						targetRootNode = rna.GetRootNode()
					} else if vna, ok := targetElement.Addr().Interface().(valueNodeAccessor); ok {
						targetRootNode = vna.GetValueNode()
					}
				}
			}

			// Only match if both RootNodes are non-nil and equal
			if targetRootNode != nil && targetRootNode == sourceRootNode {
				// Found the matching target element - reuse it to preserve its core state
				reorderedTargets[i] = targetElement.Addr().Interface()
				if j < len(originalNodes) {
					reorderedNodes[i] = originalNodes[j]
				}
				found = true
				break
			}
		}

		if !found {
			// No matching target found - this is a new element that was added to the source array.
			// Create a new target element to sync with.
			newTarget := CreateInstance(targetVal.Type().Elem()).Interface()

			if i < len(originalNodes) {
				reorderedNodes[i] = originalNodes[i]
			}

			reorderedTargets[i] = newTarget
		}
	}

	return reorderedTargets, reorderedNodes
}

// will dereference the last ptr in the type while initializing any higher level pointers
func dereferenceAndInitializeIfNeededToLastPtr(val reflect.Value, source reflect.Value) reflect.Value {
	if val.Kind() == reflect.Ptr && val.IsNil() {
		if (source.Kind() == reflect.Ptr && !source.IsNil()) || (source.Kind() != reflect.Ptr && source.IsValid()) {
			val.Set(CreateInstance(val.Type().Elem()))
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

// initializeAndGetSequencedMapInterface handles initialization and returns SequencedMapInterface for embedded fields
func initializeAndGetSequencedMapInterface(targetField reflect.Value) interfaces.SequencedMapInterface {
	// Handle both pointer and value embeds
	if targetField.Kind() == reflect.Ptr {
		// Pointer embed - initialize if nil
		if targetField.IsNil() {
			targetField.Set(CreateInstance(targetField.Type().Elem()))
		}
	} else {
		// Value embed - check if it needs initialization using IsInitialized method
		if targetField.CanAddr() {
			if seqMapInterface, ok := targetField.Addr().Interface().(interfaces.SequencedMapInterface); ok {
				if !seqMapInterface.IsInitialized() {
					// Initialize the value embed by creating a new instance and copying it
					newInstance := CreateInstance(targetField.Type())
					targetField.Set(newInstance.Elem())
				}
			}
		}
	}

	// Check if it implements SequencedMapInterface for syncing
	if targetField.CanInterface() {
		var seqMapInterface interfaces.SequencedMapInterface
		var ok bool

		// Try direct interface check first (for pointer embeds)
		seqMapInterface, ok = targetField.Interface().(interfaces.SequencedMapInterface)

		// If that fails and the field is addressable, try getting a pointer to it (for value embeds)
		if !ok && targetField.CanAddr() {
			seqMapInterface, ok = targetField.Addr().Interface().(interfaces.SequencedMapInterface)
		}

		if ok {
			return seqMapInterface
		}
	}
	return nil
}

// getSourceInterface prepares the source field interface for syncing
func getSourceInterface(sourceField reflect.Value) any {
	if sourceField.CanInterface() {
		// For pointer embeds, use the field directly (it's already a pointer)
		if sourceField.Kind() == reflect.Ptr {
			return sourceField.Interface()
		}
		// For value embeds, we need to pass a pointer to the source field so it implements MapGetter
		if sourceField.CanAddr() {
			return sourceField.Addr().Interface()
		} else {
			return sourceField.Interface()
		}
	}
	return nil
}

func dereferenceType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Ptr {
		return dereferenceType(typ.Elem())
	}

	return typ
}
