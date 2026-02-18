package marshaller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v4"
	"golang.org/x/sync/errgroup"
)

// Pre-computed reflection types for performance (reusing from populator.go where possible)
var (
	nodeMutatorType    = reflect.TypeOf((*NodeMutator)(nil)).Elem()
	unmarshallableType = reflect.TypeOf((*Unmarshallable)(nil)).Elem()
	// sequencedMapType and coreModelerType are already defined in populator.go
)

// Unmarshallable is an interface that can be implemented by types that can be unmarshalled from a YAML document.
// These types should handle the node being an alias node and resolve it to the actual value (retaining the original node where needed).
type Unmarshallable interface {
	Unmarshal(ctx context.Context, parentName string, node *yaml.Node) ([]error, error)
}

// Unmarshal will unmarshal the provided document into the specified model.
func Unmarshal[T any](ctx context.Context, in io.Reader, out CoreAccessor[T]) ([]error, error) {
	if out == nil || reflect.ValueOf(out).IsNil() {
		return nil, errors.New("out parameter cannot be nil")
	}

	data, err := io.ReadAll(in)
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

	// Check if the core implements CoreModeler interface
	if coreModeler, ok := any(core).(CoreModeler); ok {
		coreModeler.SetConfig(yml.GetConfigFromDoc(data, &root))
	}

	return UnmarshalNode(ctx, "", &root, out)
}

// UnmarshalNode will unmarshal the provided node into the provided model.
// This method is useful for unmarshaling partial documents, for a full document use Unmarshal as it will retain the full document structure.
func UnmarshalNode[T any](ctx context.Context, parentName string, node *yaml.Node, out CoreAccessor[T]) ([]error, error) {
	if out == nil || reflect.ValueOf(out).IsNil() {
		return nil, errors.New("out parameter cannot be nil")
	}

	core := out.GetCore()

	validationErrs, err := UnmarshalCore(ctx, parentName, node, core)
	if err != nil {
		return nil, err
	}

	if err := PopulateWithContext(*core, out, nil); err != nil {
		return nil, err
	}

	return validationErrs, nil
}

func UnmarshalCore(ctx context.Context, parentName string, node *yaml.Node, out any) ([]error, error) {
	// Store the DocumentNode if this is a top-level document and the output implements CoreModeler
	var documentNode *yaml.Node
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) != 1 {
			return nil, fmt.Errorf("expected 1 node, got `%d` at line `%d`, column `%d`", len(node.Content), node.Line, node.Column)
		}

		// Save the document node for potential use by CoreModeler implementations
		documentNode = node
		node = node.Content[0]
	}

	// Set DocumentNode on CoreModeler implementations after unwrapping
	if documentNode != nil {
		v := reflect.ValueOf(out)
		if v.Kind() == reflect.Ptr && !v.IsNil() {
			v = v.Elem()
		}
		if implementsInterface(v, coreModelerType) {
			if coreModeler, ok := v.Addr().Interface().(CoreModeler); ok {
				coreModeler.SetDocumentNode(documentNode)
			}
		}
	}

	v := reflect.ValueOf(out)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	for v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}

	return unmarshal(ctx, parentName, node, v)
}

func UnmarshalModel(ctx context.Context, node *yaml.Node, structPtr any) ([]error, error) {
	return unmarshalModel(ctx, "", node, structPtr)
}

func UnmarshalKeyValuePair(ctx context.Context, parentName string, keyNode, valueNode *yaml.Node, outValue any) ([]error, error) {
	out := reflect.ValueOf(outValue)

	if implementsInterface(out, nodeMutatorType) {
		return unmarshalNode(ctx, parentName, keyNode, valueNode, "value", out)
	} else {
		return UnmarshalCore(ctx, parentName, valueNode, outValue)
	}
}

// DecodeNode attempts to decode a YAML node into the provided output value.
// It differentiates between type mismatch errors (returned as validation errors)
// and YAML syntax errors (returned as standard errors).
//
// Returns:
//   - []error: validation errors for type mismatches
//   - error: syntax errors or other decode failures
func DecodeNode(ctx context.Context, parentName string, node *yaml.Node, out any) ([]error, error) {
	return decodeNode(ctx, parentName, node, out)
}

func unmarshal(ctx context.Context, parentName string, node *yaml.Node, out reflect.Value) ([]error, error) {
	resolvedNode := yml.ResolveAlias(node)
	if resolvedNode == nil {
		return nil, nil
	}

	switch {
	case out.Type() == reflect.TypeOf((*yaml.Node)(nil)):
		out.Set(reflect.ValueOf(node))
		return nil, nil
	case out.Type() == reflect.TypeOf(yaml.Node{}):
		out.Set(reflect.ValueOf(*node))
		return nil, nil
	}

	if implementsInterface(out, nodeMutatorType) {
		if out.Kind() != reflect.Ptr {
			out = out.Addr()
		}

		if out.IsNil() {
			out.Set(CreateInstance(out.Type().Elem()))
		}

		nodeMutator, ok := out.Interface().(NodeMutator)
		if !ok {
			return nil, fmt.Errorf("expected NodeMutator, got `%s` at line `%d`, column `%d`", out.Type(), resolvedNode.Line, resolvedNode.Column)
		}

		return nodeMutator.Unmarshal(ctx, parentName, nil, node)
	}

	if isEmbeddedSequencedMap(out) {
		return unmarshalMapping(ctx, parentName, node, out)
	}

	if implementsInterface(out, unmarshallableType) {
		if out.Kind() != reflect.Ptr {
			out = out.Addr()
		}

		if out.IsNil() {
			out.Set(CreateInstance(out.Type().Elem()))
		}

		unmarshallable, ok := out.Interface().(Unmarshallable)
		if !ok {
			return nil, fmt.Errorf("expected Unmarshallable, got `%s` at line `%d`, column `%d`", out.Type(), resolvedNode.Line, resolvedNode.Column)
		}

		validationErrs, err := unmarshallable.Unmarshal(ctx, parentName, node)
		if err != nil {
			return nil, err
		}

		if implementsInterface(out, coreModelerType) {
			if coreModeler, ok := out.Interface().(CoreModeler); ok {
				coreModeler.SetRootNode(node)
			}
		}

		return validationErrs, nil
	}

	if implementsInterface(out, sequencedMapType) {
		if out.Kind() != reflect.Ptr {
			out = out.Addr()
		}

		if out.IsNil() {
			out.Set(CreateInstance(out.Type().Elem()))
		}

		seqMapInterface, ok := out.Interface().(interfaces.SequencedMapInterface)
		if !ok {
			return nil, fmt.Errorf("expected sequencedMapInterface, got `%s` at line `%d`, column `%d`", out.Type(), resolvedNode.Line, resolvedNode.Column)
		}

		return unmarshalSequencedMap(ctx, parentName, node, seqMapInterface)
	}

	// Type-guided unmarshalling: check target type first, then validate node compatibility
	switch {
	case isStructType(out):
		// Target expects a struct/object
		if err := validateNodeKind(resolvedNode, yaml.MappingNode, parentName, nil, "object"); err != nil {
			return []error{err}, nil //nolint:nilerr
		}
		return unmarshalMapping(ctx, parentName, node, out)

	case isSliceType(out):
		// Target expects a slice/array
		if err := validateNodeKind(resolvedNode, yaml.SequenceNode, parentName, nil, "sequence"); err != nil {
			return []error{err}, nil //nolint:nilerr
		}
		return unmarshalSequence(ctx, parentName, node, out)

	case isMapType(out):
		// Target expects a map
		if err := validateNodeKind(resolvedNode, yaml.MappingNode, parentName, nil, "object"); err != nil {
			return []error{err}, nil //nolint:nilerr
		}
		return unmarshalMapping(ctx, parentName, node, out)

	default:
		// Target expects a scalar value (string, int, bool, etc.)
		outType := out.Type()
		if outType.Kind() == reflect.Ptr {
			outType = outType.Elem()
		}

		if err := validateNodeKind(resolvedNode, yaml.ScalarNode, parentName, out.Type(), outType.String()); err != nil {
			return []error{err}, nil //nolint:nilerr
		}
		return decodeNode(ctx, parentName, resolvedNode, out.Addr().Interface())
	}
}

func unmarshalMapping(ctx context.Context, parentName string, node *yaml.Node, out reflect.Value) ([]error, error) {
	if out.Kind() == reflect.Ptr {
		out.Set(CreateInstance(out.Type().Elem()))
		out = out.Elem()
	}

	resolvedNode := yml.ResolveAlias(node)
	if resolvedNode == nil {
		return nil, nil
	}

	switch {
	case out.Kind() == reflect.Struct:
		if implementsInterface(out, coreModelerType) {
			return unmarshalModel(ctx, parentName, node, out.Addr().Interface())
		} else {
			return unmarshalStruct(ctx, parentName, node, out.Addr().Interface())
		}
	case out.Kind() == reflect.Map:
		return nil, fmt.Errorf("currently unsupported out kind: `%v` (type: `%s`) at line `%d`, column `%d`", out.Kind(), out.Type(), resolvedNode.Line, resolvedNode.Column)
	default:
		return nil, fmt.Errorf("expected struct or map, got `%s` (type: `%s`) at line `%d`, column `%d`", out.Kind(), out.Type(), resolvedNode.Line, resolvedNode.Column)
	}
}

func unmarshalModel(ctx context.Context, parentName string, node *yaml.Node, structPtr any) ([]error, error) {
	resolvedNode := yml.ResolveAlias(node)
	if resolvedNode == nil {
		return nil, nil
	}

	out := reflect.ValueOf(structPtr)

	if out.Kind() == reflect.Ptr {
		out = out.Elem()
	}

	if out.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a struct, got `%s` (type: `%s`) at line `%d`, column `%d`", out.Kind(), out.Type(), resolvedNode.Line, resolvedNode.Column)
	}
	structType := out.Type()

	// Get the "model" tag value from the embedded CoreModel field which should be the first field always
	if structType.NumField() < 1 {
		return nil, fmt.Errorf("expected embedded CoreModel field, got `%s` at line `%d`, column `%d`", out.Type(), resolvedNode.Line, resolvedNode.Column)
	}
	field := structType.Field(0)
	if field.Type != reflect.TypeOf(CoreModel{}) {
		return nil, fmt.Errorf("expected embedded CoreModel field to be of type CoreModel, got `%s` at line `%d`, column `%d`", out.Type(), resolvedNode.Line, resolvedNode.Column)
	}

	modelTag := field.Tag.Get("model")
	if modelTag == "" {
		return nil, fmt.Errorf("expected embedded CoreModel field to have a 'model' tag, got `%s` at line `%d`, column `%d`", out.Type(), resolvedNode.Line, resolvedNode.Column)
	}

	if resolvedNode.Kind != yaml.MappingNode {
		return []error{
			validation.NewValidationError(validation.SeverityError, validation.RuleValidationTypeMismatch, validation.NewTypeMismatchError(parentName, "expected `object`, got `%s`", yml.NodeKindToString(resolvedNode.Kind)), resolvedNode),
		}, nil
	}

	var unmarshallable CoreModeler

	// Check if struct implements CoreModeler
	if implementsInterface(out, coreModelerType) {
		var ok bool
		unmarshallable, ok = out.Addr().Interface().(CoreModeler)
		if !ok {
			return nil, fmt.Errorf("expected CoreModeler, got `%s` at line `%d`, column `%d`", out.Type(), resolvedNode.Line, resolvedNode.Column)
		}
	} else {
		return nil, fmt.Errorf("expected struct to implement CoreModeler, got `%s` at line `%d`, column `%d`", out.Type(), resolvedNode.Line, resolvedNode.Column)
	}

	unmarshallable.SetRootNode(node)

	// Get cached field information, build it if not available
	fieldMap := getFieldMapCached(structType)

	// Handle extensions field using cached index
	var extensionsField *reflect.Value
	if fieldMap.HasExtensions {
		extField := out.Field(fieldMap.ExtensionIndex)
		extensionsField = &extField
	}

	// Handle embedded maps (these need runtime reflection)
	var embeddedMap interfaces.SequencedMapInterface
	for i := 0; i < out.NumField(); i++ {
		field := structType.Field(i)
		if field.Anonymous {
			fieldVal := out.Field(i)
			if seqMap := initializeEmbeddedSequencedMap(fieldVal); seqMap != nil {
				embeddedMap = seqMap
			}
			continue
		}
	}

	// Pre-scan for duplicate keys and determine which indices to skip
	// For duplicate keys, we only process the last occurrence (YAML spec behavior)
	// and report earlier occurrences as validation errors
	type keyInfo struct {
		firstLine int // line of first occurrence
		lastIndex int // index of last occurrence (the one we'll process)
	}
	seenKeys := make(map[string]*keyInfo)
	indicesToSkip := make(map[int]bool) // indices that are duplicates (not the last occurrence)
	var duplicateKeyErrs []error

	for i := 0; i < len(resolvedNode.Content); i += 2 {
		keyNode := resolvedNode.Content[i]
		key := keyNode.Value
		if info, exists := seenKeys[key]; exists {
			// This is a duplicate - mark the previous last occurrence for skipping
			indicesToSkip[info.lastIndex] = true
			// Create validation error for the earlier occurrence
			duplicateKeyErrs = append(duplicateKeyErrs, validation.NewValidationError(
				validation.SeverityWarning,
				validation.RuleValidationDuplicateKey,
				fmt.Errorf("mapping key `%q` at line `%d` is a duplicate; previous definition at line `%d`", key, keyNode.Line, info.firstLine),
				keyNode,
			))
			// Update to track this as the new last occurrence
			info.lastIndex = i / 2
		} else {
			seenKeys[key] = &keyInfo{firstLine: keyNode.Line, lastIndex: i / 2}
		}
	}

	// Process YAML nodes and validate required fields in one pass
	foundRequiredFields := sync.Map{}

	numJobs := len(resolvedNode.Content) / 2

	var mapNode yaml.Node
	var jobMapContent [][]*yaml.Node

	if embeddedMap != nil {
		mapNode = *resolvedNode
		jobMapContent = make([][]*yaml.Node, numJobs)
	}

	jobValidationErrs := make([][]error, numJobs)

	// Track unknown properties (non-extension, non-field, non-embedded map properties)
	var unknownPropertiesMutex sync.Mutex
	unknownProperties := make([]string, 0, numJobs)

	// Mutex to protect concurrent access to extensionsField
	var extensionsMutex sync.Mutex

	// TODO allow concurrency to be configurable
	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < len(resolvedNode.Content); i += 2 {
		g.Go(func() error {
			// Skip duplicate keys (all but the last occurrence)
			if indicesToSkip[i/2] {
				return nil
			}

			keyNode := resolvedNode.Content[i]
			valueNode := resolvedNode.Content[i+1]

			key := keyNode.Value

			// Direct field index lookup (eliminates map[string]Field allocation)
			fieldIndex, ok := fieldMap.FieldIndexes[key]
			if !ok {
				switch {
				case strings.HasPrefix(key, "x-") && extensionsField != nil:
					// Lock access to extensionsField to prevent concurrent modification
					extensionsMutex.Lock()
					defer extensionsMutex.Unlock()
					err := UnmarshalExtension(keyNode, valueNode, *extensionsField)
					if err != nil {
						return err
					}
				case embeddedMap != nil:
					// Skip alias definitions - these are nodes where:
					// 1. The value node has an anchor (e.g., &keyAlias)
					// 2. The key is not an alias reference (doesn't start with *)
					if valueNode.Anchor != "" && !strings.HasPrefix(key, "*") {
						// This is an alias definition, skip it from embedded map processing
						// but it should still be preserved at the document level
						return nil
					}
					jobMapContent[i/2] = append(jobMapContent[i/2], keyNode, valueNode)
				default:
					// This is an unknown property (not a recognized field, not an extension, not in embedded map)
					unknownPropertiesMutex.Lock()
					unknownProperties = append(unknownProperties, key)
					unknownPropertiesMutex.Unlock()
				}
			} else {
				// Get field info from cache and field value directly
				cachedField := fieldMap.Fields[key]
				fieldVal := out.Field(fieldIndex)

				if implementsInterface(fieldVal, nodeMutatorType) {
					fieldValidationErrs, err := unmarshalNode(ctx, modelTag+"."+key, keyNode, valueNode, cachedField.Name, fieldVal)
					if err != nil {
						return err
					}
					jobValidationErrs[i/2] = append(jobValidationErrs[i/2], fieldValidationErrs...)

					// Mark required field as found
					if fieldMap.RequiredFields[key] {
						foundRequiredFields.Store(key, true)
					}
				} else {
					return fmt.Errorf("expected field `%s` to be marshaller.Node, got `%s` at line `%d`, column `%d` (key: `%s`)", cachedField.Name, fieldVal.Type(), keyNode.Line, keyNode.Column, key)
				}
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var validationErrs []error

	// Add duplicate key validation errors first
	validationErrs = append(validationErrs, duplicateKeyErrs...)

	for _, jobValidationErrs := range jobValidationErrs {
		validationErrs = append(validationErrs, jobValidationErrs...)
	}

	var mapContent []*yaml.Node
	for _, jobMapContent := range jobMapContent {
		mapContent = append(mapContent, jobMapContent...)
	}

	// Check for missing required fields using cached required field info
	for tag := range fieldMap.RequiredFields {
		if _, ok := foundRequiredFields.Load(tag); !ok {
			validationErrs = append(validationErrs, validation.NewValidationError(validation.SeverityError, validation.RuleValidationRequiredField, fmt.Errorf("`%s.%s` is required", modelTag, tag), resolvedNode))
		}
	}

	if embeddedMap != nil {
		mapNode.Content = mapContent
		embeddedMapValidationErrs, err := unmarshalSequencedMap(ctx, modelTag, &mapNode, embeddedMap)
		if err != nil {
			return nil, err
		}
		validationErrs = append(validationErrs, embeddedMapValidationErrs...)
	}

	// Store unknown properties in the core model if any were found
	if len(unknownProperties) > 0 {
		unmarshallable.SetUnknownProperties(unknownProperties)
	}

	// Use the errors to determine the validity of the model
	unmarshallable.DetermineValidity(validationErrs)

	return validationErrs, nil
}

func unmarshalStruct(ctx context.Context, parentName string, node *yaml.Node, structPtr any) ([]error, error) {
	return decodeNode(ctx, parentName, node, structPtr)
}

func decodeNode(_ context.Context, parentName string, node *yaml.Node, out any) ([]error, error) {
	resolvedNode := yml.ResolveAlias(node)
	if resolvedNode == nil {
		return nil, errors.New("node is nil")
	}

	// Attempt to decode the node
	err := resolvedNode.Decode(out)
	if err == nil {
		return nil, nil // Success
	}

	// Check if this is a type mismatch error
	if yamlTypeErr := asTypeMismatchError(err); yamlTypeErr != nil {
		// Convert type mismatch to validation error
		errStrings := make([]string, len(yamlTypeErr.Errors))
		for i, e := range yamlTypeErr.Errors {
			errStrings[i] = e.Error()
		}
		validationErr := validation.NewValidationError(validation.SeverityError, validation.RuleValidationTypeMismatch, validation.NewTypeMismatchError(parentName, strings.Join(errStrings, ", ")), resolvedNode)
		return []error{validationErr}, nil
	}

	// For all other errors (syntax, etc.), return as standard error
	return nil, err
}

func unmarshalSequence(ctx context.Context, parentName string, node *yaml.Node, out reflect.Value) ([]error, error) {
	resolvedNode := yml.ResolveAlias(node)
	if resolvedNode == nil {
		return nil, nil
	}

	if out.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected `slice`, got `%s` (type: `%s`) at line `%d`, column `%d`", out.Kind(), out.Type(), resolvedNode.Line, resolvedNode.Column)
	}

	out.Set(reflect.MakeSlice(out.Type(), len(resolvedNode.Content), len(resolvedNode.Content)))

	g, ctx := errgroup.WithContext(ctx)

	numJobs := len(resolvedNode.Content)

	jobValidationErrs := make([][]error, numJobs)

	for i := 0; i < numJobs; i++ {
		g.Go(func() error {
			valueNode := resolvedNode.Content[i]

			elementValidationErrs, err := unmarshal(ctx, fmt.Sprintf("%s.%d", parentName, i), valueNode, out.Index(i))
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

func unmarshalNode(ctx context.Context, parentName string, keyNode, valueNode *yaml.Node, fieldName string, out reflect.Value) ([]error, error) {
	ref := out
	resolvedKeyNode := yml.ResolveAlias(keyNode)
	if resolvedKeyNode == nil {
		return nil, nil
	}

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
			return nil, fmt.Errorf("field `%s` is a nil pointer and cannot be set at line `%d`, column `%d`", fieldName, resolvedKeyNode.Line, resolvedKeyNode.Column)
		}
	}

	unmarshallable, ok := ref.Interface().(NodeMutator)
	if !ok {
		return nil, fmt.Errorf("expected field `%s` to be marshaller.Node, got `%s` at line `%d`, column `%d`", fieldName, ref.Type(), resolvedKeyNode.Line, resolvedKeyNode.Column)
	}

	validationErrs, err := unmarshallable.Unmarshal(ctx, parentName, keyNode, valueNode)
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

func implementsInterface(out reflect.Value, interfaceType reflect.Type) bool {
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
			return original.Type().Implements(interfaceType)
		}
		out = out.Addr()
	}

	return out.Type().Implements(interfaceType)
}

func isEmbeddedSequencedMap(out reflect.Value) bool {
	return implementsInterface(out, sequencedMapType) && implementsInterface(out, coreModelerType)
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
func validateNodeKind(resolvedNode *yaml.Node, expectedKind yaml.Kind, parentName string, reflectType reflect.Type, expectedType string) error {
	if resolvedNode == nil {
		return validation.NewValidationError(validation.SeverityError, validation.RuleValidationTypeMismatch, validation.NewTypeMismatchError(parentName, "expected `%s`, got nil", yml.NodeKindToString(expectedKind)), nil)
	}

	// Check if the node kind matches
	isSameKind := resolvedNode.Kind == expectedKind
	tagMatches := true

	// Get acceptable tags based on whether we have a reflect.Type or just a string
	var acceptableTags []string
	if reflectType != nil {
		acceptableTags = yml.TypeToYamlTags(reflectType)
	} else {
		switch expectedType {
		case "object":
			acceptableTags = []string{"!!map"}
		case "sequence":
			acceptableTags = []string{"!!seq"}
		}
	}

	if len(acceptableTags) > 0 {
		tagMatches = false
		for _, acceptableTag := range acceptableTags {
			if acceptableTag == resolvedNode.Tag {
				tagMatches = true
				break
			}
		}
	}

	if !isSameKind {
		actualKindStr := yml.NodeKindToString(resolvedNode.Kind)

		if actualKindStr == "scalar" {
			// Truncate value to prevent file content disclosure
			// External references that fail to parse may contain sensitive file content
			value := resolvedNode.Value
			maxLen := 7
			if len(value) < maxLen {
				maxLen = len(value) / 2
			}
			if len(value) > maxLen {
				value = value[:maxLen] + "..."
			}
			actualKindStr = fmt.Sprintf("`%s`", value)
		} else {
			actualKindStr = fmt.Sprintf("`%s`", actualKindStr)
		}

		return validation.NewValidationError(validation.SeverityError, validation.RuleValidationTypeMismatch, validation.NewTypeMismatchError(parentName, "expected `%s`, got %s", expectedType, actualKindStr), resolvedNode)
	}

	if !tagMatches {
		return validation.NewValidationError(validation.SeverityError, validation.RuleValidationTypeMismatch, validation.NewTypeMismatchError(parentName, "expected `%s`, got `%s`", expectedType, yml.NodeTagToString(resolvedNode.Tag)), resolvedNode)
	}
	return nil
}

// asTypeMismatchError returns the underlying yaml.LoadErrors if the error is a type mismatch error
func asTypeMismatchError(err error) *yaml.LoadErrors {
	var yamlTypeErr *yaml.LoadErrors
	if errors.As(err, &yamlTypeErr) {
		return yamlTypeErr
	}
	return nil
}

// initializeEmbeddedSequencedMap handles initialization of embedded sequenced maps
func initializeEmbeddedSequencedMap(fieldVal reflect.Value) interfaces.SequencedMapInterface {
	// Check if the field is a embedded sequenced map
	if !implementsInterface(fieldVal, sequencedMapType) {
		return nil
	}

	// Handle both pointer and value embeds
	if fieldVal.Kind() == reflect.Ptr {
		// Pointer embed - check if nil and initialize if needed
		if fieldVal.IsNil() {
			fieldVal.Set(CreateInstance(fieldVal.Type().Elem()))
		}
		return fieldVal.Interface().(interfaces.SequencedMapInterface)
	} else {
		// Value embed - check if initialized and initialize if needed
		if seqMapInterface, ok := fieldVal.Addr().Interface().(interfaces.SequencedMapInterface); ok {
			if !seqMapInterface.IsInitialized() {
				// Initialize the value embed by creating a new instance and copying it
				newInstance := CreateInstance(fieldVal.Type())
				fieldVal.Set(newInstance.Elem())
			}
			return seqMapInterface
		}
	}
	return nil
}
