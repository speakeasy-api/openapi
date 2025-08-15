package marshaller

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"slices"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

// MapGetter interface for syncing operations
type MapGetter interface {
	AllUntyped() iter.Seq2[any, any]
}

// unmarshalSequencedMap unmarshals a YAML node into a sequenced map
func unmarshalSequencedMap(ctx context.Context, parentName string, node *yaml.Node, target interfaces.SequencedMapInterface) ([]error, error) {
	resolvedNode := yml.ResolveAlias(node)
	if resolvedNode == nil {
		return nil, errors.New("node is nil")
	}

	// Check if the node is actually a mapping node
	if resolvedNode.Kind != yaml.MappingNode {
		validationErr := validation.NewTypeMismatchError("%sexpected mapping node for sequenced map, got %v", getOptionalParentName(parentName), resolvedNode.Kind)
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
		g.Go(func() error {
			keyNode := resolvedNode.Content[i]
			valueNode := resolvedNode.Content[i+1]

			// Resolve alias for key node to handle alias keys like *keyAlias :
			resolvedKeyNode := yml.ResolveAlias(keyNode)
			if resolvedKeyNode == nil {
				return errors.New("failed to resolve key node alias")
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
			validationErrs, err := UnmarshalKeyValuePair(ctx, parentName, keyNode, valueNode, concreteValue)
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
func populateSequencedMap(source any, target interfaces.SequencedMapInterface) error {
	if source == nil {
		return nil
	}

	sourceValue := reflect.ValueOf(source)

	var sm interfaces.SequencedMapInterface
	var ok bool

	// Handle both pointer and non-pointer cases
	switch {
	case sourceValue.Kind() == reflect.Ptr:
		// Source is already a pointer
		sm, ok = source.(interfaces.SequencedMapInterface)
	case sourceValue.CanAddr():
		// Source is addressable, get a pointer to it
		sm, ok = sourceValue.Addr().Interface().(interfaces.SequencedMapInterface)
	default:
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
func syncSequencedMapChanges(ctx context.Context, target interfaces.SequencedMapInterface, model any, valueNode *yaml.Node, syncFunc func(context.Context, any, any, *yaml.Node, bool) (*yaml.Node, error)) (*yaml.Node, error) {
	target.Init()

	var mg MapGetter
	var ok bool

	// Try direct interface check first
	mg, ok = model.(MapGetter)

	// If that fails, try getting a pointer to the model (for value embeds)
	if !ok {
		modelValue := reflect.ValueOf(model)
		if modelValue.CanAddr() {
			mg, ok = modelValue.Addr().Interface().(MapGetter)
		}
	}

	if !ok {
		return nil, fmt.Errorf("SyncSequencedMapChanges expected model to be a MapGetter, got %s", reflect.TypeOf(model))
	}

	remainingKeys := []string{}
	hasEntries := false

	for k, v := range mg.AllUntyped() {
		hasEntries = true
		keyStr := fmt.Sprintf("%v", k) // TODO this might not work with non string keys

		// Try to convert the key type if needed (similar to populateSequencedMap)
		targetKey := k
		keyValue := reflect.ValueOf(k)
		targetKeyType := target.GetKeyType()

		if keyValue.Type() != targetKeyType && keyValue.CanConvert(targetKeyType) {
			targetKey = keyValue.Convert(targetKeyType).Interface()
		}

		lv, _ := target.GetAny(targetKey)

		kn, vn, _ := yml.GetMapElementNodes(ctx, valueNode, keyStr)

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

	// If no entries were processed but we have an embedded map, ensure we create an empty mapping node
	if !hasEntries && valueNode == nil {
		valueNode = &yaml.Node{
			Kind: yaml.MappingNode,
		}
	}

	return valueNode, nil
}
