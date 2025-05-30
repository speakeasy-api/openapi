package marshaller_test

import (
	"iter"
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Mock NodeAccessor for testing
type TestNodeAccessor struct {
	value any
}

func (t TestNodeAccessor) GetValue() any {
	return t.value
}

func (t TestNodeAccessor) GetValueType() reflect.Type {
	if t.value == nil {
		return nil
	}
	return reflect.TypeOf(t.value)
}

// Simple struct that implements CoreSetter to trigger populateModel
type SimpleCoreSetterTarget struct {
	Value string
	core  any
}

func (s *SimpleCoreSetterTarget) SetCoreValue(core any) {
	s.core = core
}

// Source struct where all fields implement NodeAccessor (as expected by populateModel)
type NodeAccessorSource struct {
	Value TestNodeAccessor
}

func Test_PopulateModel_CoreSetter_With_NodeAccessor(t *testing.T) {
	source := NodeAccessorSource{
		Value: TestNodeAccessor{value: "test-value"},
	}

	target := &SimpleCoreSetterTarget{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	// Verify the field was populated via populateModel using NodeAccessor.GetValue()
	assert.Equal(t, "test-value", target.Value)
	
	// Verify SetCoreValue was called (proves CoreSetter path was taken)
	assert.NotNil(t, target.core)
}

// Test Extensions field handling in populateModel
type SourceWithExtensionsNodeAccessor struct {
	Value      TestNodeAccessor
	Extensions *MockExtensionCoreMap
}

type TargetWithExtensions struct {
	Value      string
	Extensions *MockExtensionMap
	core       any
}

func (t *TargetWithExtensions) SetCoreValue(core any) {
	t.core = core
}

// Mock ExtensionCoreMap interface
type MockExtensionCoreMap struct {
	items map[string]marshaller.Node[*yaml.Node]
}

func (m *MockExtensionCoreMap) Get(key string) (marshaller.Node[*yaml.Node], bool) {
	if m.items == nil {
		return marshaller.Node[*yaml.Node]{}, false
	}
	node, ok := m.items[key]
	return node, ok
}

func (m *MockExtensionCoreMap) Set(key string, value marshaller.Node[*yaml.Node]) {
	if m.items == nil {
		m.items = make(map[string]marshaller.Node[*yaml.Node])
	}
	m.items[key] = value
}

func (m *MockExtensionCoreMap) Delete(key string) {
	if m.items != nil {
		delete(m.items, key)
	}
}

func (m *MockExtensionCoreMap) All() iter.Seq2[string, marshaller.Node[*yaml.Node]] {
	return func(yield func(string, marshaller.Node[*yaml.Node]) bool) {
		if m.items == nil {
			return
		}
		for k, v := range m.items {
			if !yield(k, v) {
				return
			}
		}
	}
}

func (m *MockExtensionCoreMap) Init() {
	if m.items == nil {
		m.items = make(map[string]marshaller.Node[*yaml.Node])
	}
}

// Mock ExtensionMap interface
type MockExtensionMap struct {
	data map[string]*yaml.Node
	core any
}

func (m *MockExtensionMap) Init() {
	if m.data == nil {
		m.data = make(map[string]*yaml.Node)
	}
}

func (m *MockExtensionMap) Set(key string, value *yaml.Node) {
	if m.data == nil {
		m.data = make(map[string]*yaml.Node)
	}
	m.data[key] = value
}

func (m *MockExtensionMap) SetCore(core any) {
	m.core = core
}

func Test_PopulateModel_Extensions_Handling_NodeAccessor(t *testing.T) {
	// Create an extension node with the test value
	extensionNode := marshaller.Node[*yaml.Node]{
		Value: &yaml.Node{Kind: yaml.ScalarNode, Value: "extension-value"},
	}
	
	extensionMap := &MockExtensionCoreMap{}
	extensionMap.Init()
	extensionMap.Set("x-test", extensionNode)
	
	source := SourceWithExtensionsNodeAccessor{
		Value: TestNodeAccessor{value: "test-value"},
		Extensions: extensionMap,
	}

	target := &TargetWithExtensions{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	// Verify basic field
	assert.Equal(t, "test-value", target.Value)
	
	// Verify Extensions field was handled specially
	assert.NotNil(t, target.Extensions)
	assert.NotNil(t, target.Extensions.data)
	// The extension value should be the YAML node
	extensionValue := target.Extensions.data["x-test"]
	assert.NotNil(t, extensionValue)
	assert.Equal(t, "extension-value", extensionValue.Value)
	
	// Verify Extensions.SetCore was called
	assert.NotNil(t, target.Extensions.core)
	
	// Verify target SetCoreValue was called
	assert.NotNil(t, target.core)
}

// Test populatorValue tag handling
type SourceWithTaggedField struct {
	TaggedField   string `populatorValue:"true"`
	UntaggedField TestNodeAccessor
}

type TargetWithTag struct {
	TaggedField   string
	UntaggedField string
	core          any
}

func (t *TargetWithTag) SetCoreValue(core any) {
	t.core = core
}

func Test_PopulateModel_PopulatorValue_Tag_NodeAccessor(t *testing.T) {
	source := SourceWithTaggedField{
		TaggedField:   "tagged-value",
		UntaggedField: TestNodeAccessor{value: "untagged-value"},
	}

	target := &TargetWithTag{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	// Verify tagged field used direct value (not NodeAccessor.GetValue)
	assert.Equal(t, "tagged-value", target.TaggedField)
	
	// Verify untagged field used NodeAccessor.GetValue
	assert.Equal(t, "untagged-value", target.UntaggedField)
	
	// Verify SetCoreValue was called
	assert.NotNil(t, target.core)
}

// Test error case: source is not a struct
func Test_PopulateModel_NonStruct_Error_CoreSetter(t *testing.T) {
	target := &SimpleCoreSetterTarget{}

	err := marshaller.PopulateModel("not-a-struct", target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected struct, got string")
}

// Test missing target field (should be ignored)
type SourceWithExtraNodeAccessor struct {
	Value      TestNodeAccessor
	ExtraField TestNodeAccessor
}

type TargetMissingField struct {
	Value string
	// ExtraField is missing
	core any
}

func (t *TargetMissingField) SetCoreValue(core any) {
	t.core = core
}

func Test_PopulateModel_MissingTargetField_NodeAccessor(t *testing.T) {
	source := SourceWithExtraNodeAccessor{
		Value:      TestNodeAccessor{value: "test-value"},
		ExtraField: TestNodeAccessor{value: "extra-value"},
	}

	target := &TargetMissingField{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	// Should succeed and ignore missing field
	assert.Equal(t, "test-value", target.Value)
	assert.NotNil(t, target.core)
}

// Test with nil pointer source field
type SourceWithNilPointer struct {
	PtrField TestNodeAccessor
	CanAddr  TestNodeAccessor
}

type TargetWithPointer struct {
	PtrField *string
	CanAddr  string
	core     any
}

func (t *TargetWithPointer) SetCoreValue(core any) {
	t.core = core
}

func Test_PopulateModel_PointerHandling_NodeAccessor(t *testing.T) {
	source := SourceWithNilPointer{
		PtrField: TestNodeAccessor{value: "ptr-value"},
		CanAddr:  TestNodeAccessor{value: "can-addr-value"},
	}

	target := &TargetWithPointer{}

	err := marshaller.PopulateModel(source, target)
	require.NoError(t, err)

	// Verify fields were populated
	require.NotNil(t, target.PtrField)
	assert.Equal(t, "ptr-value", *target.PtrField)
	assert.Equal(t, "can-addr-value", target.CanAddr)
	
	// Verify SetCoreValue was called
	assert.NotNil(t, target.core)
}

// Test error: invalid NodeAccessor interface
type SourceWithInvalidAccessor struct {
	Value string // This is not a NodeAccessor
}

type TargetInvalidAccessor struct {
	Value string
	core  any
}

func (t *TargetInvalidAccessor) SetCoreValue(core any) {
	t.core = core
}

func Test_PopulateModel_Invalid_NodeAccessor_Error(t *testing.T) {
	source := SourceWithInvalidAccessor{
		Value: "not-a-node-accessor",
	}

	target := &TargetInvalidAccessor{}

	err := marshaller.PopulateModel(source, target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected NodeAccessor")
}