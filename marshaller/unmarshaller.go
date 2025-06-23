package marshaller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"

	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

// Unmarshallable is an interface that can be implemented by types that can be unmarshalled from a YAML document.
// These types should handle the node being an alias node and resolve it to the actual value (retaining the original node where needed).
type Unmarshallable interface {
	Unmarshal(ctx context.Context, node *yaml.Node) ([]error, error)
}

func Unmarshal[T any](ctx context.Context, doc io.Reader, out CoreAccessor[T]) ([]error, error) {
	data, err := io.ReadAll(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to read document: %w", err)
	}

	if len(data) == 0 {
		return nil, errors.New("empty document")
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	core := out.GetCore()
	validationErrs, err := UnmarshalCore(ctx, &root, core)
	if err != nil {
		return nil, err
	}

	// Check if the core implements CoreModeler interface
	if coreModeler, ok := any(core).(CoreModeler); ok {
		coreModeler.SetConfig(yml.GetConfigFromDoc(data, &root))
	}

	if err := Populate(*core, out); err != nil {
		return nil, err
	}

	return validationErrs, nil
}

func UnmarshalCore(ctx context.Context, node *yaml.Node, out any) ([]error, error) {
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) != 1 {
			return nil, fmt.Errorf("expected 1 node, got %d at line %d, column %d", len(node.Content), node.Line, node.Column)
		}

		return UnmarshalCore(ctx, node.Content[0], out)
	}

	v := reflect.ValueOf(out)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	for v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}

	return unmarshal(ctx, node, v)
}

func UnmarshalModel(ctx context.Context, node *yaml.Node, structPtr any) ([]error, error) {
	return unmarshalModel(ctx, node, structPtr)
}

func UnmarshalKeyValuePair(ctx context.Context, keyNode, valueNode *yaml.Node, outValue any) ([]error, error) {
	out := reflect.ValueOf(outValue)

	if implementsInterface[NodeMutator](out) {
		return unmarshalNode(ctx, keyNode, valueNode, "value", out)
	} else {
		return UnmarshalCore(ctx, valueNode, outValue)
	}
}

// DecodeNode attempts to decode a YAML node into the provided output value.
// It differentiates between type mismatch errors (returned as validation errors)
// and YAML syntax errors (returned as standard errors).
//
// Returns:
//   - []error: validation errors for type mismatches
//   - error: syntax errors or other decode failures
func DecodeNode(ctx context.Context, node *yaml.Node, out any) ([]error, error) {
	return decodeNode(ctx, node, out)
}

func unmarshal(ctx context.Context, node *yaml.Node, out reflect.Value) ([]error, error) {
	resolvedNode := yml.ResolveAlias(node)

	switch {
	case out.Type() == reflect.TypeOf((*yaml.Node)(nil)):
		out.Set(reflect.ValueOf(node))
		return nil, nil
	case out.Type() == reflect.TypeOf(yaml.Node{}):
		out.Set(reflect.ValueOf(*node))
		return nil, nil
	}

	if implementsInterface[NodeMutator](out) {
		if out.Kind() != reflect.Ptr {
			out = out.Addr()
		}

		if out.IsNil() {
			out.Set(CreateInstance(out.Type().Elem()))
		}

		nodeMutator, ok := out.Interface().(NodeMutator)
		if !ok {
			return nil, fmt.Errorf("expected NodeMutator, got %s at line %d, column %d", out.Type(), resolvedNode.Line, resolvedNode.Column)
		}

		return nodeMutator.Unmarshal(ctx, nil, node)
	}

	if isEmbeddedSequencedMap(out) {
		return unmarshalMapping(ctx, node, out)
	}

	if implementsInterface[Unmarshallable](out) {
		if out.Kind() != reflect.Ptr {
			out = out.Addr()
		}

		if out.IsNil() {
			out.Set(CreateInstance(out.Type().Elem()))
		}

		unmarshallable, ok := out.Interface().(Unmarshallable)
		if !ok {
			return nil, fmt.Errorf("expected Unmarshallable, got %s at line %d, column %d", out.Type(), resolvedNode.Line, resolvedNode.Column)
		}

		return unmarshallable.Unmarshal(ctx, node)
	}

	if implementsInterface[sequencedMapInterface](out) {
		if out.Kind() != reflect.Ptr {
			out = out.Addr()
		}

		if out.IsNil() {
			out.Set(CreateInstance(out.Type().Elem()))
		}

		seqMapInterface, ok := out.Interface().(sequencedMapInterface)
		if !ok {
			return nil, fmt.Errorf("expected sequencedMapInterface, got %s at line %d, column %d", out.Type(), resolvedNode.Line, resolvedNode.Column)
		}

		return unmarshalSequencedMap(ctx, node, seqMapInterface)
	}

	// Type-guided unmarshalling: check target type first, then validate node compatibility
	switch {
	case isStructType(out):
		// Target expects a struct/object
		if validationErrs, err := validateNodeKind(resolvedNode, yaml.MappingNode, "struct"); err != nil || validationErrs != nil {
			return validationErrs, err
		}
		return unmarshalMapping(ctx, node, out)

	case isSliceType(out):
		// Target expects a slice/array
		if validationErrs, err := validateNodeKind(resolvedNode, yaml.SequenceNode, "slice"); err != nil || validationErrs != nil {
			return validationErrs, err
		}
		return unmarshalSequence(ctx, node, out)

	case isMapType(out):
		// Target expects a map
		if validationErrs, err := validateNodeKind(resolvedNode, yaml.MappingNode, "map"); err != nil || validationErrs != nil {
			return validationErrs, err
		}
		return unmarshalMapping(ctx, node, out)

	default:
		// Target expects a scalar value (string, int, bool, etc.)
		if validationErrs, err := validateNodeKind(resolvedNode, yaml.ScalarNode, out.Type().String()); err != nil || validationErrs != nil {
			return validationErrs, err
		}
		return decodeNode(ctx, resolvedNode, out.Addr().Interface())
	}
}

func unmarshalMapping(ctx context.Context, node *yaml.Node, out reflect.Value) ([]error, error) {
	if out.Kind() == reflect.Ptr {
		out.Set(CreateInstance(out.Type().Elem()))
		out = out.Elem()
	}

	resolvedNode := yml.ResolveAlias(node)

	switch {
	case out.Kind() == reflect.Struct:
		if implementsInterface[CoreModeler](out) {
			return unmarshalModel(ctx, node, out.Addr().Interface())
		} else {
			return unmarshalStruct(ctx, node, out.Addr().Interface())
		}
	case out.Kind() == reflect.Map:
		return nil, fmt.Errorf("currently unsupported out kind: %v (type: %s) at line %d, column %d", out.Kind(), out.Type(), resolvedNode.Line, resolvedNode.Column)
	default:
		return nil, fmt.Errorf("expected struct or map, got %s (type: %s) at line %d, column %d", out.Kind(), out.Type(), resolvedNode.Line, resolvedNode.Column)
	}
}

func unmarshalModel(ctx context.Context, node *yaml.Node, structPtr any) ([]error, error) {
	resolvedNode := yml.ResolveAlias(node)

	if resolvedNode.Kind != yaml.MappingNode {
		return []error{
			validation.NewNodeError(validation.NewTypeMismatchError("expected a mapping node, got %s", yml.NodeKindToString(resolvedNode.Kind)), resolvedNode),
		}, nil
	}

	out := reflect.ValueOf(structPtr)

	if out.Kind() == reflect.Ptr {
		out = out.Elem()
	}

	if out.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a struct, got %s (type: %s) at line %d, column %d", out.Kind(), out.Type(), resolvedNode.Line, resolvedNode.Column)
	}

	var unmarshallable CoreModeler

	// Check if struct implements CoreModeler
	if implementsInterface[CoreModeler](out) {
		var ok bool
		unmarshallable, ok = out.Addr().Interface().(CoreModeler)
		if !ok {
			return nil, fmt.Errorf("expected CoreModeler, got %s at line %d, column %d", out.Type(), resolvedNode.Line, resolvedNode.Column)
		}
	} else {
		return nil, fmt.Errorf("expected struct to implement CoreModeler, got %s at line %d, column %d", out.Type(), resolvedNode.Line, resolvedNode.Column)
	}

	unmarshallable.SetRootNode(node)

	type Field struct {
		Name     string
		Field    reflect.Value
		Required bool
	}

	// get fields by tag first
	fields := map[string]Field{}
	var extensionsField *reflect.Value
	requiredFields := map[string]Field{} // Track required fields separately

	var embeddedMap sequencedMapInterface

	for i := 0; i < out.NumField(); i++ {
		field := out.Type().Field(i)

		if field.Anonymous {
			fieldVal := out.Field(i)

			// Check if the field is a embedded sequenced map
			if implementsInterface[sequencedMapInterface](fieldVal) {
				if fieldVal.IsNil() {
					fieldVal.Set(CreateInstance(fieldVal.Type().Elem()))
				}
				embeddedMap = fieldVal.Interface().(sequencedMapInterface)
			}
			continue
		}

		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("key")
		if tag == "" || tag == "extensions" {
			if tag == "extensions" {
				extField := out.Field(i)
				extensionsField = &extField
			}

			continue
		}

		requiredTag := field.Tag.Get("required")
		required := requiredTag == "true"

		if requiredTag == "" {
			nodeAccessor, ok := out.Field(i).Interface().(NodeAccessor)
			if ok {
				fieldType := nodeAccessor.GetValueType()

				if fieldType.Kind() != reflect.Ptr {
					required = fieldType.Kind() != reflect.Map && fieldType.Kind() != reflect.Slice && fieldType.Kind() != reflect.Array
				}
			}
		}

		fieldInfo := Field{
			Name:     field.Name,
			Field:    out.Field(i),
			Required: required,
		}

		fields[tag] = fieldInfo

		// Track required fields for validation
		if required {
			requiredFields[tag] = fieldInfo
		}
	}

	// Process YAML nodes and validate required fields in one pass
	foundRequiredFields := sync.Map{}

	numJobs := len(resolvedNode.Content) / 2

	var mapNode *yaml.Node
	var jobMapContent [][]*yaml.Node

	if embeddedMap != nil {
		copy := *resolvedNode
		mapNode = &copy
		jobMapContent = make([][]*yaml.Node, numJobs)
	}

	jobValidationErrs := make([][]error, numJobs)

	// Mutex to protect concurrent access to extensionsField
	var extensionsMutex sync.Mutex

	// TODO allow concurrency to be configurable
	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < len(resolvedNode.Content); i += 2 {
		i := i
		g.Go(func() error {
			keyNode := resolvedNode.Content[i]
			valueNode := resolvedNode.Content[i+1]

			key := keyNode.Value

			field, ok := fields[key]
			if !ok {
				if strings.HasPrefix(key, "x-") && extensionsField != nil {
					// Lock access to extensionsField to prevent concurrent modification
					extensionsMutex.Lock()
					defer extensionsMutex.Unlock()
					err := UnmarshalExtension(keyNode, valueNode, *extensionsField)
					if err != nil {
						return err
					}
				} else if embeddedMap != nil {
					// Skip alias definitions - these are nodes where:
					// 1. The value node has an anchor (e.g., &keyAlias)
					// 2. The key is not an alias reference (doesn't start with *)
					if valueNode.Anchor != "" && !strings.HasPrefix(key, "*") {
						// This is an alias definition, skip it from embedded map processing
						// but it should still be preserved at the document level
						return nil
					}
					jobMapContent[i/2] = append(jobMapContent[i/2], keyNode, valueNode)
				}
			} else if implementsInterface[NodeMutator](field.Field) {
				fieldValidationErrs, err := unmarshalNode(ctx, keyNode, valueNode, field.Name, field.Field)
				if err != nil {
					return err
				}
				jobValidationErrs[i/2] = append(jobValidationErrs[i/2], fieldValidationErrs...)

				// Mark required field as found
				if field.Required {
					foundRequiredFields.Store(key, true)
				}
			} else {
				return fmt.Errorf("expected field '%s' to be marshaller.Node, got %s at line %d, column %d (key: %s)", field.Name, field.Field.Type(), keyNode.Line, keyNode.Column, key)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var validationErrs []error

	for _, jobValidationErrs := range jobValidationErrs {
		validationErrs = append(validationErrs, jobValidationErrs...)
	}

	var mapContent []*yaml.Node
	for _, jobMapContent := range jobMapContent {
		mapContent = append(mapContent, jobMapContent...)
	}

	// Check for missing required fields
	for tag := range requiredFields {
		if _, ok := foundRequiredFields.Load(tag); !ok {
			validationErrs = append(validationErrs, validation.NewNodeError(validation.NewMissingFieldError("field %s is missing", tag), resolvedNode))
		}
	}

	if embeddedMap != nil {
		mapNode.Content = mapContent
		embeddedMapValidationErrs, err := unmarshalSequencedMap(ctx, mapNode, embeddedMap)
		if err != nil {
			return nil, err
		}
		validationErrs = append(validationErrs, embeddedMapValidationErrs...)
	}

	// Use the errors to determine the validity of the model
	unmarshallable.DetermineValidity(validationErrs)

	return validationErrs, nil
}

func unmarshalStruct(ctx context.Context, node *yaml.Node, structPtr any) ([]error, error) {
	return decodeNode(ctx, node, structPtr)
}

func decodeNode(_ context.Context, node *yaml.Node, out any) ([]error, error) {
	resolvedNode := yml.ResolveAlias(node)
	if resolvedNode == nil {
		return nil, fmt.Errorf("node is nil")
	}

	// Attempt to decode the node
	err := resolvedNode.Decode(out)
	if err == nil {
		return nil, nil // Success
	}

	// Check if this is a type mismatch error
	if isTypeMismatchError(err) {
		// Convert type mismatch to validation error
		validationErr := validation.NewNodeError(validation.NewTypeMismatchError(err.Error()), resolvedNode)
		return []error{validationErr}, nil
	}

	// For all other errors (syntax, etc.), return as standard error
	return nil, err
}

func unmarshalSequence(ctx context.Context, node *yaml.Node, out reflect.Value) ([]error, error) {
	resolvedNode := yml.ResolveAlias(node)

	if out.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected slice, got %s (type: %s) at line %d, column %d", out.Kind(), out.Type(), resolvedNode.Line, resolvedNode.Column)
	}

	out.Set(reflect.MakeSlice(out.Type(), len(resolvedNode.Content), len(resolvedNode.Content)))

	g, ctx := errgroup.WithContext(ctx)

	numJobs := len(resolvedNode.Content)

	jobValidationErrs := make([][]error, numJobs)

	for i := 0; i < numJobs; i++ {
		i := i
		g.Go(func() error {
			valueNode := resolvedNode.Content[i]

			elementValidationErrs, err := unmarshal(ctx, valueNode, out.Index(i))
			if err != nil {
				return err
			}
			jobValidationErrs[i] = append(jobValidationErrs[i], elementValidationErrs...)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var validationErrs []error

	for _, jobValidationErrs := range jobValidationErrs {
		validationErrs = append(validationErrs, jobValidationErrs...)
	}

	return validationErrs, nil
}

func unmarshalNode(ctx context.Context, keyNode, valueNode *yaml.Node, fieldName string, out reflect.Value) ([]error, error) {
	ref := out
	resolvedKeyNode := yml.ResolveAlias(keyNode)

	if out.Kind() != reflect.Ptr {
		if out.CanSet() {
			ref = out.Addr()
		} else {
			// For non-settable values (like local variables), we need to work with what we have
			// This typically happens when out is already a pointer to the value we want to modify
			ref = out
		}
	} else if out.IsNil() {
		if out.CanSet() {
			out.Set(reflect.New(out.Type().Elem()))
			ref = out.Elem().Addr()
		} else {
			return nil, fmt.Errorf("field %s is a nil pointer and cannot be set at line %d, column %d", fieldName, resolvedKeyNode.Line, resolvedKeyNode.Column)
		}
	}

	unmarshallable, ok := ref.Interface().(NodeMutator)
	if !ok {
		return nil, fmt.Errorf("expected field '%s' to be marshaller.Node, got %s at line %d, column %d", fieldName, ref.Type(), resolvedKeyNode.Line, resolvedKeyNode.Column)
	}

	validationErrs, err := unmarshallable.Unmarshal(ctx, keyNode, valueNode)
	if err != nil {
		return nil, err
	}

	// Fix: Only set the value if the original field can be set
	if out.CanSet() {
		if out.Kind() == reflect.Ptr {
			out.Set(reflect.ValueOf(unmarshallable))
		} else {
			// Get the value from the unmarshallable and set it directly
			unmarshallableValue := reflect.ValueOf(unmarshallable)
			if unmarshallableValue.Kind() == reflect.Ptr {
				unmarshallableValue = unmarshallableValue.Elem()
			}
			out.Set(unmarshallableValue)
		}
	}

	return validationErrs, nil
}

func implementsInterface[T any](out reflect.Value) bool {
	// Store original value to check directly
	original := out

	// Unwrap interface if needed
	for out.Kind() == reflect.Interface && !out.IsNil() {
		out = out.Elem()
	}

	// Get addressable value if needed
	if out.Kind() != reflect.Ptr {
		if !out.CanAddr() {
			// Try checking the original value directly
			return original.Type().Implements(reflect.TypeOf((*T)(nil)).Elem())
		}
		out = out.Addr()
	}

	return out.Type().Implements(reflect.TypeOf((*T)(nil)).Elem())
}

func isEmbeddedSequencedMap(out reflect.Value) bool {
	return implementsInterface[sequencedMapInterface](out) && implementsInterface[CoreModeler](out)
}

// isStructType checks if the reflect.Value represents a struct type (direct or pointer to struct)
func isStructType(out reflect.Value) bool {
	return out.Kind() == reflect.Struct || (out.Kind() == reflect.Ptr && out.Type().Elem().Kind() == reflect.Struct)
}

// isSliceType checks if the reflect.Value represents a slice type (direct or pointer to slice)
func isSliceType(out reflect.Value) bool {
	return out.Kind() == reflect.Slice || (out.Kind() == reflect.Ptr && out.Type().Elem().Kind() == reflect.Slice)
}

// isMapType checks if the reflect.Value represents a map type (direct or pointer to map)
func isMapType(out reflect.Value) bool {
	return out.Kind() == reflect.Map || (out.Kind() == reflect.Ptr && out.Type().Elem().Kind() == reflect.Map)
}

// validateNodeKind checks if the node kind matches the expected kind and returns appropriate error
func validateNodeKind(resolvedNode *yaml.Node, expectedKind yaml.Kind, expectedType string) ([]error, error) {
	if resolvedNode.Kind != expectedKind {
		expectedKindStr := yml.NodeKindToString(expectedKind)
		actualKindStr := yml.NodeKindToString(resolvedNode.Kind)

		return []error{
			validation.NewNodeError(validation.NewTypeMismatchError("expected %s for %s, got %s",
				expectedKindStr, expectedType, actualKindStr), resolvedNode),
		}, nil
	}
	return nil, nil
}

// isTypeMismatchError checks if the error is a YAML type mismatch error
// using proper type checking instead of string matching
func isTypeMismatchError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a yaml.TypeError directly
	if _, ok := err.(*yaml.TypeError); ok {
		return true
	}

	// Check using errors.As for wrapped errors
	var yamlTypeErr *yaml.TypeError
	return errors.As(err, &yamlTypeErr)
}
