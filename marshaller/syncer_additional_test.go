package marshaller_test

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func Test_SyncValue_ArraySlice_Success(t *testing.T) {
	tests := []struct {
		name         string
		source       []int
		targetSetup  func() *[]int
		expectedSync []int
	}{
		{
			name:   "sync equal length slices",
			source: []int{1, 2, 3},
			targetSetup: func() *[]int {
				target := []int{0, 0, 0}
				return &target
			},
			expectedSync: []int{1, 2, 3},
		},
		{
			name:   "sync to longer target slice",
			source: []int{1, 2},
			targetSetup: func() *[]int {
				target := []int{0, 0, 0, 0}
				return &target
			},
			expectedSync: []int{1, 2},
		},
		{
			name:   "sync to shorter target slice",
			source: []int{1, 2, 3, 4},
			targetSetup: func() *[]int {
				target := []int{0, 0}
				return &target
			},
			expectedSync: []int{1, 2, 3, 4},
		},
		{
			name:   "sync to nil target slice",
			source: []int{1, 2, 3},
			targetSetup: func() *[]int {
				var target []int
				return &target
			},
			expectedSync: []int{1, 2, 3},
		},
		{
			name:   "sync nil source to target",
			source: nil,
			targetSetup: func() *[]int {
				target := []int{1, 2, 3}
				return &target
			},
			expectedSync: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := tt.targetSetup()
			
			outNode, err := marshaller.SyncValue(context.Background(), tt.source, target, nil, false)
			require.NoError(t, err)

			if tt.expectedSync == nil {
				assert.Nil(t, outNode)
				assert.Nil(t, *target)
			} else {
				assert.Equal(t, tt.expectedSync, *target)
				assert.NotNil(t, outNode)
			}
		})
	}
}

func Test_SyncValue_ArraySlice_Error(t *testing.T) {
	tests := []struct {
		name   string
		source any
		target any
	}{
		{
			name:   "non-slice source",
			source: "not-a-slice",
			target: &[]int{},
		},
		{
			name:   "non-slice target",
			source: []int{1, 2, 3},
			target: &map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := marshaller.SyncValue(context.Background(), tt.source, tt.target, nil, false)
			require.Error(t, err)
		})
	}
}

func Test_SyncValue_WithExistingValueNode_Success(t *testing.T) {
	ctx := context.Background()
	
	// Create an existing array node with some content
	existingNode := &yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			testutils.CreateIntYamlNode(10, 1, 1),
			testutils.CreateIntYamlNode(20, 2, 1),
		},
	}

	source := []int{1, 2, 3}
	var target []int

	outNode, err := marshaller.SyncValue(ctx, source, &target, existingNode, false)
	require.NoError(t, err)

	assert.Equal(t, []int{1, 2, 3}, target)
	assert.Equal(t, yaml.SequenceNode, outNode.Kind)
	assert.Equal(t, 3, len(outNode.Content))
}

type TestStructForArraySync struct {
	marshaller.Model[TestStructForArraySyncCore]
	Items []string
}

type TestStructForArraySyncCore struct {
	marshaller.CoreModel
	Items marshaller.Node[[]string] `key:"items"`
}

func Test_SyncValue_StructWithArrayField_Success(t *testing.T) {
	ctx := context.Background()

	source := &TestStructForArraySync{
		Items: []string{"a", "b", "c"},
	}

	outNode, err := marshaller.SyncValue(ctx, source, source.GetCore(), nil, false)
	require.NoError(t, err)

	assert.Equal(t, []string{"a", "b", "c"}, source.GetCore().Items.Value)
	assert.NotNil(t, outNode)
	assert.Equal(t, yaml.MappingNode, outNode.Kind)
}