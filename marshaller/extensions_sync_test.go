package marshaller_test

import (
	"context"
	"iter"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Test the else branch in syncExtensions where existing extension keys are updated
func Test_SyncExtensions_ExistingKey_Success(t *testing.T) {
	ctx := context.Background()

	// Create source struct with extensions field that implements ExtensionSourceIterator
	sourceExtensions := map[string]*yaml.Node{
		"x-existing": {Kind: yaml.ScalarNode, Value: "new-value"},
		"x-new":      {Kind: yaml.ScalarNode, Value: "new-extension"},
	}

	source := &StructWithExtensions{
		Extensions: &MockExtensionSourceIterator{
			extensions: sourceExtensions,
		},
	}

	// Create target struct with extensions field that implements ExtensionCoreMap
	target := &StructWithExtensionsTarget{
		Extensions: &MockExtensionCoreMapForSync{},
	}
	
	// Initialize target extensions
	target.Extensions.Init()

	// Pre-populate target with existing extension to trigger the else branch
	existingNode := marshaller.Node[*yaml.Node]{
		Key:       "x-existing",
		KeyNode:   &yaml.Node{Kind: yaml.ScalarNode, Value: "x-existing"},
		Value:     &yaml.Node{Kind: yaml.ScalarNode, Value: "old-value"},
		ValueNode: &yaml.Node{Kind: yaml.ScalarNode, Value: "old-value"},
	}
	target.Extensions.Set("x-existing", existingNode)

	// Create initial value node
	valueNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{},
	}

	// Test SyncValue - this should trigger syncExtensions internally
	resultNode, err := marshaller.SyncValue(ctx, source, target, valueNode, false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Verify that the existing key was updated (else branch was executed)
	updatedNode, exists := target.Extensions.Get("x-existing")
	assert.True(t, exists)
	assert.Equal(t, "new-value", updatedNode.Value.Value)

	// Verify that the new key was added (if branch was executed)
	newNode, exists := target.Extensions.Get("x-new")
	assert.True(t, exists)
	assert.Equal(t, "new-extension", newNode.Value.Value)
}

// Test the cleanup branch in syncExtensions where extensions are removed
func Test_SyncExtensions_CleanupRemovedExtensions_Success(t *testing.T) {
	ctx := context.Background()

	// Create source with only some extensions (missing "x-removed")
	sourceExtensions := map[string]*yaml.Node{
		"x-keep": {Kind: yaml.ScalarNode, Value: "keep-value"},
		"x-new":  {Kind: yaml.ScalarNode, Value: "new-value"},
	}

	source := &StructWithExtensions{
		Extensions: &MockExtensionSourceIterator{
			extensions: sourceExtensions,
		},
	}

	// Create target with more extensions than source (has "x-removed" that should be deleted)
	target := &StructWithExtensionsTarget{
		Extensions: &MockExtensionCoreMapForSync{},
	}
	target.Extensions.Init()

	// Pre-populate target with extensions, including one that will be removed
	keepNode := marshaller.Node[*yaml.Node]{
		Key:       "x-keep",
		KeyNode:   &yaml.Node{Kind: yaml.ScalarNode, Value: "x-keep"},
		Value:     &yaml.Node{Kind: yaml.ScalarNode, Value: "old-keep-value"},
		ValueNode: &yaml.Node{Kind: yaml.ScalarNode, Value: "old-keep-value"},
	}
	target.Extensions.Set("x-keep", keepNode)

	// This extension should be removed because it's not in the source
	removeNode := marshaller.Node[*yaml.Node]{
		Key:       "x-removed",
		KeyNode:   &yaml.Node{Kind: yaml.ScalarNode, Value: "x-removed"},
		Value:     &yaml.Node{Kind: yaml.ScalarNode, Value: "remove-value"},
		ValueNode: &yaml.Node{Kind: yaml.ScalarNode, Value: "remove-value"},
	}
	target.Extensions.Set("x-removed", removeNode)

	// Create initial value node
	valueNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{},
	}

	// Test SyncValue - this should trigger syncExtensions with cleanup
	resultNode, err := marshaller.SyncValue(ctx, source, target, valueNode, false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Verify that the kept extension was updated
	keptNode, exists := target.Extensions.Get("x-keep")
	assert.True(t, exists)
	assert.Equal(t, "keep-value", keptNode.Value.Value)

	// Verify that the new extension was added
	newNode, exists := target.Extensions.Get("x-new")
	assert.True(t, exists)
	assert.Equal(t, "new-value", newNode.Value.Value)

	// Verify that the removed extension was deleted (this tests the cleanup branch)
	_, exists = target.Extensions.Get("x-removed")
	assert.False(t, exists, "x-removed extension should have been deleted")

	// Verify target only has the expected extensions
	extensionCount := 0
	for range target.Extensions.All() {
		extensionCount++
	}
	assert.Equal(t, 2, extensionCount, "Target should only have 2 extensions after cleanup")
}

// Test edge case: all extensions are removed from source
func Test_SyncExtensions_RemoveAllExtensions_Success(t *testing.T) {
	ctx := context.Background()

	// Create source with no extensions
	source := &StructWithExtensions{
		Extensions: &MockExtensionSourceIterator{
			extensions: map[string]*yaml.Node{}, // Empty - all should be removed
		},
	}

	// Create target with multiple extensions that should all be removed
	target := &StructWithExtensionsTarget{
		Extensions: &MockExtensionCoreMapForSync{},
	}
	target.Extensions.Init()

	// Pre-populate target with several extensions
	ext1 := marshaller.Node[*yaml.Node]{
		Key:       "x-ext1",
		KeyNode:   &yaml.Node{Kind: yaml.ScalarNode, Value: "x-ext1"},
		Value:     &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"},
		ValueNode: &yaml.Node{Kind: yaml.ScalarNode, Value: "value1"},
	}
	target.Extensions.Set("x-ext1", ext1)

	ext2 := marshaller.Node[*yaml.Node]{
		Key:       "x-ext2",
		KeyNode:   &yaml.Node{Kind: yaml.ScalarNode, Value: "x-ext2"},
		Value:     &yaml.Node{Kind: yaml.ScalarNode, Value: "value2"},
		ValueNode: &yaml.Node{Kind: yaml.ScalarNode, Value: "value2"},
	}
	target.Extensions.Set("x-ext2", ext2)

	valueNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{},
	}

	// Test SyncValue - this should remove all existing extensions
	resultNode, err := marshaller.SyncValue(ctx, source, target, valueNode, false)
	require.NoError(t, err)
	require.NotNil(t, resultNode)

	// Verify all extensions were removed
	_, exists1 := target.Extensions.Get("x-ext1")
	assert.False(t, exists1, "x-ext1 should have been deleted")

	_, exists2 := target.Extensions.Get("x-ext2")
	assert.False(t, exists2, "x-ext2 should have been deleted")

	// Verify target has no extensions
	extensionCount := 0
	for range target.Extensions.All() {
		extensionCount++
	}
	assert.Equal(t, 0, extensionCount, "Target should have no extensions after cleanup")
}

// Note: Testing the error path within the else branch (lines 98-100) is complex
// because it requires the Node.SyncValue() method to return an error, but Node is
// a concrete struct. The main else branch (lines 95-101) is successfully tested above.

// Structs for testing with proper field tags to trigger syncExtensions
type StructWithExtensions struct {
	marshaller.CoreModel
	Extensions *MockExtensionSourceIterator `key:"extensions"`
}

type StructWithExtensionsTarget struct {
	marshaller.CoreModel
	Extensions *MockExtensionCoreMapForSync `key:"extensions"`
}

// Mock ExtensionSourceIterator for testing
type MockExtensionSourceIterator struct {
	extensions map[string]*yaml.Node
}

func (m *MockExtensionSourceIterator) All() iter.Seq2[string, *yaml.Node] {
	return func(yield func(string, *yaml.Node) bool) {
		for k, v := range m.extensions {
			if !yield(k, v) {
				return
			}
		}
	}
}

// Mock ExtensionCoreMap for testing syncExtensions
type MockExtensionCoreMapForSync struct {
	items map[string]marshaller.Node[*yaml.Node]
}

func (m *MockExtensionCoreMapForSync) Init() {
	if m.items == nil {
		m.items = make(map[string]marshaller.Node[*yaml.Node])
	}
}

func (m *MockExtensionCoreMapForSync) Get(key string) (marshaller.Node[*yaml.Node], bool) {
	node, exists := m.items[key]
	return node, exists
}

func (m *MockExtensionCoreMapForSync) Set(key string, value marshaller.Node[*yaml.Node]) {
	if m.items == nil {
		m.Init()
	}
	m.items[key] = value
}

func (m *MockExtensionCoreMapForSync) Delete(key string) {
	if m.items != nil {
		delete(m.items, key)
	}
}

func (m *MockExtensionCoreMapForSync) All() iter.Seq2[string, marshaller.Node[*yaml.Node]] {
	return func(yield func(string, marshaller.Node[*yaml.Node]) bool) {
		if m.items != nil {
			for k, v := range m.items {
				if !yield(k, v) {
					return
				}
			}
		}
	}
}