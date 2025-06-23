package marshaller

import (
	"context"
	"fmt"
	"iter"
	"reflect"
	"slices"

	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

// sequencedMapInterface defines the interface that sequenced maps must implement
type sequencedMapInterface interface {
	Init()
	SetUntyped(key, value any) error
	AllUntyped() iter.Seq2[any, any]
	GetKeyType() reflect.Type
	GetValueType() reflect.Type
	Len() int
	GetAny(key any) (any, bool)
	SetAny(key, value any)
	DeleteAny(key any)
	KeysAny() iter.Seq[any]
}

// MapGetter interface for syncing operations
type MapGetter interface {
	AllUntyped() iter.Seq2[any, any]
}

// unmarshalSequencedMap unmarshals a YAML node into a sequenced map
func unmarshalSequencedMap(ctx context.Context, node *yaml.Node, target sequencedMapInterface) ([]error, error) {
	resolvedNode := yml.ResolveAlias(node)
	if resolvedNode == nil {
		return nil, fmt.Errorf("node is nil")
	}

	// Check if the node is actually a mapping node
	if resolvedNode.Kind != yaml.MappingNode {
		validationErr := validation.NewTypeMismatchError("expected mapping node for sequenced map, got %v", resolvedNode.Kind)
		return []error{validationErr}, nil
	}

	target.Init()

	g, ctx := errgroup.WithContext(ctx)

	numJobs := len(resolvedNode.Content) / 2
	jobsValidationErrs := make([][]error, numJobs)

	type keyPair struct {
		key   string
		value any
	}

	valuesToSet := make([]keyPair, numJobs)

	for i := 0; i < len(resolvedNode.Content); i += 2 {
		i := i
		g.Go(func() error {
			keyNode := resolvedNode.Content[i]
			valueNode := resolvedNode.Content[i+1]

			// Resolve alias for key node to handle alias keys like *keyAlias :
			resolvedKeyNode := yml.ResolveAlias(keyNode)
			if resolvedKeyNode == nil {
				return fmt.Errorf("failed to resolve key node alias")
			}
			key := resolvedKeyNode.Value

			// Get the value type from the target map
			valueType := target.GetValueType()

			// Create a new instance of the value type
			var concreteValue any
			if valueType.Kind() == reflect.Ptr {
				concreteValue = CreateInstance(valueType.Elem()).Interface()
			} else {
				concreteValue = CreateInstance(valueType).Interface()
			}

			// Unmarshal into the concrete value
			validationErrs, err := UnmarshalKeyValuePair(ctx, keyNode, valueNode, concreteValue)
			if err != nil {
				return err
			}
			jobsValidationErrs[i/2] = append(jobsValidationErrs[i/2], validationErrs...)

			// Extract the value and set it in the map
			if valueType.Kind() != reflect.Ptr {
				// Dereference if the target type is not a pointer
				concreteValue = reflect.ValueOf(concreteValue).Elem().Interface()
			}

			valuesToSet[i/2] = keyPair{
				key:   key,
				value: concreteValue,
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	for _, keyPair := range valuesToSet {
		if err := target.SetUntyped(keyPair.key, keyPair.value); err != nil {
			return nil, err
		}
	}

	var allValidationErrs []error

	for _, jobValidationErrs := range jobsValidationErrs {
		allValidationErrs = append(allValidationErrs, jobValidationErrs...)
	}

	return allValidationErrs, nil
}

// populateSequencedMap populates a target sequenced map from a source sequenced map
func populateSequencedMap(source any, target sequencedMapInterface) error {
	if source == nil {
		return nil
	}

	sourceValue := reflect.ValueOf(source)

	var sm sequencedMapInterface
	var ok bool

	// Handle both pointer and non-pointer cases
	if sourceValue.Kind() == reflect.Ptr {
		// Source is already a pointer
		sm, ok = source.(sequencedMapInterface)
	} else if sourceValue.CanAddr() {
		// Source is addressable, get a pointer to it
		sm, ok = sourceValue.Addr().Interface().(sequencedMapInterface)
	} else {
		// Source is neither a pointer nor addressable
		return fmt.Errorf("expected source to be addressable or a pointer to SequencedMap, got %s", sourceValue.Type())
	}

	if !ok {
		return fmt.Errorf("expected source to be SequencedMap, got %s", sourceValue.Type())
	}

	target.Init()

	for key, value := range sm.AllUntyped() {
		// Get the target value type
		valueType := target.GetValueType()
		valueKind := valueType.Kind()

		// Create a new instance of the target value type
		var targetValue any
		if valueKind == reflect.Ptr {
			targetValue = CreateInstance(valueType.Elem()).Interface()
		} else {
			targetValue = CreateInstance(valueType).Interface()
		}

		if err := Populate(value, targetValue); err != nil {
			return err
		}

		// Extract the value if needed
		if valueKind != reflect.Ptr {
			targetValue = reflect.ValueOf(targetValue).Elem().Interface()
		}

		if err := target.SetUntyped(key, targetValue); err != nil {
			// If direct key setting fails, try to convert the key type using the same
			// mechanism as field-level conversion in populateValue
			keyValue := reflect.ValueOf(key)
			targetKeyType := target.GetKeyType()

			if keyValue.CanConvert(targetKeyType) {
				convertedKey := keyValue.Convert(targetKeyType).Interface()
				if err := target.SetUntyped(convertedKey, targetValue); err != nil {
					return err
				}
			} else {
				return err // Return original error if conversion fails
			}
		}
	}

	return nil
}

// syncSequencedMapChanges syncs changes from a source map to a target map using a sync function
func syncSequencedMapChanges(ctx context.Context, target sequencedMapInterface, model any, valueNode *yaml.Node, syncFunc func(context.Context, any, any, *yaml.Node, bool) (*yaml.Node, error)) (*yaml.Node, error) {
	target.Init()

	mg, ok := model.(MapGetter)
	if !ok {
		return nil, fmt.Errorf("SyncSequencedMapChanges expected model to be a MapGetter, got %s", reflect.TypeOf(model))
	}

	remainingKeys := []string{}

	for k, v := range mg.AllUntyped() {
		keyStr := fmt.Sprintf("%v", k) // TODO this might not work with non string keys

		// Try to convert the key type if needed (similar to populateSequencedMap)
		var targetKey any = k
		keyValue := reflect.ValueOf(k)
		targetKeyType := target.GetKeyType()

		if keyValue.Type() != targetKeyType && keyValue.CanConvert(targetKeyType) {
			targetKey = keyValue.Convert(targetKeyType).Interface()
		}

		lv, _ := target.GetAny(targetKey)

		kn, vn, _ := yml.GetMapElementNodes(ctx, valueNode, keyStr)

		// Recreate the original behavior: lv, _ := m.Get(key); syncFunc(ctx, v, &lv, vn, false); m.Set(key, lv)
		// The original lv was always the concrete type V (or zero value), and &lv was pointer to that type

		valueType := target.GetValueType()

		// Create a concrete typed variable (equivalent to original lv)
		var concreteValue reflect.Value
		if lv != nil {
			// Use the existing value
			concreteValue = reflect.ValueOf(lv)
		} else {
			// Create zero value of the correct type (matching original m.Get behavior when key not found)
			concreteValue = reflect.Zero(valueType)
		}

		// Create an addressable variable to pass to syncFunc (equivalent to &lv)
		addressableVar := reflect.New(valueType)
		addressableVar.Elem().Set(concreteValue)

		vn, err := syncFunc(ctx, v, addressableVar.Interface(), vn, false)
		if err != nil {
			return nil, err
		}

		// Get the updated value and set it back using the converted key (equivalent to m.Set(key, lv))
		updatedValue := addressableVar.Elem().Interface()
		target.SetAny(targetKey, updatedValue)

		valueNode = yml.CreateOrUpdateMapNodeElement(ctx, keyStr, yml.CreateOrUpdateKeyNode(ctx, keyStr, kn), vn, valueNode)
		remainingKeys = append(remainingKeys, keyStr)
	}

	keysToDelete := []any{}

	for k := range target.KeysAny() {
		key := fmt.Sprintf("%v", k) // TODO this might not work with non string keys

		if !slices.Contains(remainingKeys, key) {
			keysToDelete = append(keysToDelete, k)
		}
	}

	for _, key := range keysToDelete {
		target.DeleteAny(key)
		valueNode = yml.DeleteMapNodeElement(ctx, fmt.Sprintf("%v", key), valueNode)
	}

	return valueNode, nil
}
