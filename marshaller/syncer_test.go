package marshaller_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func Test_SyncValue_String_Success(t *testing.T) {
	tests := []struct {
		name           string
		source         any
		targetFactory  func() any
		expectedValue  string
		validateTarget func(t *testing.T, target any, expected string)
	}{
		{
			name:   "string to string",
			source: "some-value",
			targetFactory: func() any {
				target := ""
				return &target
			},
			expectedValue: "some-value",
			validateTarget: func(t *testing.T, target any, expected string) {
				assert.Equal(t, expected, *target.(*string))
			},
		},
		{
			name:   "string pointer set to string pointer",
			source: pointer.From("some-value"),
			targetFactory: func() any {
				target := pointer.From("")
				return &target
			},
			expectedValue: "some-value",
			validateTarget: func(t *testing.T, target any, expected string) {
				assert.Equal(t, expected, **target.(**string))
			},
		},
		{
			name:   "string pointer to nil string pointer",
			source: pointer.From("some-value"),
			targetFactory: func() any {
				var target *string
				return &target
			},
			expectedValue: "some-value",
			validateTarget: func(t *testing.T, target any, expected string) {
				assert.Equal(t, expected, **target.(**string))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := tt.targetFactory()
			outNode, err := marshaller.SyncValue(context.Background(), tt.source, target, nil, false)
			require.NoError(t, err)
			assert.Equal(t, testutils.CreateStringYamlNode(tt.expectedValue, 0, 0), outNode)
			tt.validateTarget(t, target, tt.expectedValue)
		})
	}
}

type TestStructSyncer[T any] struct {
	Val *T
}

type TestStructSyncerCore[T any] struct {
	marshaller.CoreModel
	Val *T
}

func (t *TestStructSyncerCore[T]) SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error) {
	mv := reflect.ValueOf(model)
	if mv.Kind() == reflect.Ptr {
		mv = mv.Elem()
	}
	if mv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", mv.Kind())
	}

	var err error
	rootNode, err := marshaller.SyncValue(ctx, mv.FieldByName("Val").Interface(), &t.Val, valueNode, false)
	t.SetRootNode(rootNode)
	return rootNode, err
}

func Test_SyncValue_StructPtrCustomSyncer_Success(t *testing.T) {
	var target *TestStructSyncerCore[int]

	source := &TestStructSyncer[int]{Val: pointer.From(1)}

	outNode, err := marshaller.SyncValue(context.Background(), source, &target, nil, false)
	require.NoError(t, err)
	node := testutils.CreateIntYamlNode(1, 0, 0)
	assert.Equal(t, node, outNode)
	assert.Equal(t, node, target.GetRootNode())
	assert.Equal(t, 1, *target.Val)
}

func Test_SyncValue_StructCustomSyncer_Success(t *testing.T) {
	var target TestStructSyncerCore[int]

	source := TestStructSyncer[int]{Val: pointer.From(1)}

	outNode, err := marshaller.SyncValue(context.Background(), source, &target, nil, false)
	require.NoError(t, err)
	node := testutils.CreateIntYamlNode(1, 0, 0)
	assert.Equal(t, node, outNode)
	assert.Equal(t, node, target.GetRootNode())
}

type TestStruct struct {
	marshaller.Model[TestStructCore]
	Int     int
	Str     string
	StrPtr  *string
	BoolPtr *bool
}

type TestStructCore struct {
	marshaller.CoreModel
	Int     marshaller.Node[int]     `key:"int"`
	Str     marshaller.Node[string]  `key:"str"`
	StrPtr  marshaller.Node[*string] `key:"strPtr"`
	BoolPtr marshaller.Node[*bool]   `key:"boolPtr"`
}

func Test_SyncChanges_Struct_Success(t *testing.T) {
	source := TestStruct{
		Int:     1,
		Str:     "some-string",
		StrPtr:  pointer.From("some-string-ptr"),
		BoolPtr: pointer.From(true),
	}

	outNode, err := marshaller.SyncValue(context.Background(), &source, source.GetCore(), nil, false)
	require.NoError(t, err)

	node := testutils.CreateMapYamlNode([]*yaml.Node{
		testutils.CreateStringYamlNode("int", 0, 0),
		testutils.CreateIntYamlNode(1, 0, 0),
		testutils.CreateStringYamlNode("str", 0, 0),
		testutils.CreateStringYamlNode("some-string", 0, 0),
		testutils.CreateStringYamlNode("strPtr", 0, 0),
		testutils.CreateStringYamlNode("some-string-ptr", 0, 0),
		testutils.CreateStringYamlNode("boolPtr", 0, 0),
		testutils.CreateBoolYamlNode(true, 0, 0),
	}, 0, 0)

	assert.Equal(t, node, outNode)
	assert.Equal(t, node, source.GetCore().GetRootNode())
	assert.Equal(t, 1, source.GetCore().Int.Value)
	assert.Equal(t, "some-string", source.GetCore().Str.Value)
	assert.Equal(t, "some-string-ptr", *source.GetCore().StrPtr.Value)
	assert.Equal(t, true, *source.GetCore().BoolPtr.Value)
}

func Test_SyncChanges_StructWithOptionalsUnset_Success(t *testing.T) {
	source := TestStruct{
		Int: 1,
		Str: "some-string",
	}

	outNode, err := marshaller.SyncValue(context.Background(), &source, source.GetCore(), nil, false)
	require.NoError(t, err)

	node := testutils.CreateMapYamlNode([]*yaml.Node{
		testutils.CreateStringYamlNode("int", 0, 0),
		testutils.CreateIntYamlNode(1, 0, 0),
		testutils.CreateStringYamlNode("str", 0, 0),
		testutils.CreateStringYamlNode("some-string", 0, 0),
	}, 0, 0)

	assert.Equal(t, node, outNode)
	assert.Equal(t, node, source.GetCore().GetRootNode())
	assert.Equal(t, 1, source.GetCore().Int.Value)
	assert.Equal(t, "some-string", source.GetCore().Str.Value)
	assert.Nil(t, source.GetCore().StrPtr.Value)
	assert.Nil(t, source.GetCore().BoolPtr.Value)
}

func Test_SyncChanges_StructPtr_Success(t *testing.T) {
	source := &TestStruct{
		Int:     1,
		Str:     "some-string",
		StrPtr:  pointer.From("some-string-ptr"),
		BoolPtr: pointer.From(true),
	}

	outNode, err := marshaller.SyncValue(context.Background(), &source, source.GetCore(), nil, false)
	require.NoError(t, err)

	node := testutils.CreateMapYamlNode([]*yaml.Node{
		testutils.CreateStringYamlNode("int", 0, 0),
		testutils.CreateIntYamlNode(1, 0, 0),
		testutils.CreateStringYamlNode("str", 0, 0),
		testutils.CreateStringYamlNode("some-string", 0, 0),
		testutils.CreateStringYamlNode("strPtr", 0, 0),
		testutils.CreateStringYamlNode("some-string-ptr", 0, 0),
		testutils.CreateStringYamlNode("boolPtr", 0, 0),
		testutils.CreateBoolYamlNode(true, 0, 0),
	}, 0, 0)

	assert.Equal(t, node, outNode)
	assert.Equal(t, node, source.GetCore().GetRootNode())
	assert.Equal(t, 1, source.GetCore().Int.Value)
	assert.Equal(t, "some-string", source.GetCore().Str.Value)
	assert.Equal(t, "some-string-ptr", *source.GetCore().StrPtr.Value)
	assert.Equal(t, true, *source.GetCore().BoolPtr.Value)
}

type TestStructNested struct {
	marshaller.Model[TestStructNestedCore]
	TestStruct TestStruct
}

type TestStructNestedCore struct {
	marshaller.CoreModel
	TestStruct marshaller.Node[TestStructCore] `key:"testStruct"`
}

func Test_SyncChanges_NestedStruct_Success(t *testing.T) {
	source := TestStructNested{
		TestStruct: TestStruct{
			Int:     1,
			Str:     "some-string",
			StrPtr:  pointer.From("some-string-ptr"),
			BoolPtr: pointer.From(true),
		},
	}

	outNode, err := marshaller.SyncValue(context.Background(), &source, source.GetCore(), nil, false)
	require.NoError(t, err)

	nestedNode := testutils.CreateMapYamlNode([]*yaml.Node{
		testutils.CreateStringYamlNode("int", 0, 0),
		testutils.CreateIntYamlNode(1, 0, 0),
		testutils.CreateStringYamlNode("str", 0, 0),
		testutils.CreateStringYamlNode("some-string", 0, 0),
		testutils.CreateStringYamlNode("strPtr", 0, 0),
		testutils.CreateStringYamlNode("some-string-ptr", 0, 0),
		testutils.CreateStringYamlNode("boolPtr", 0, 0),
		testutils.CreateBoolYamlNode(true, 0, 0),
	}, 0, 0)

	node := testutils.CreateMapYamlNode([]*yaml.Node{
		testutils.CreateStringYamlNode("testStruct", 0, 0),
		nestedNode,
	}, 0, 0)

	assert.Equal(t, node, outNode)
	assert.Equal(t, node, source.GetCore().GetRootNode())
	assert.Equal(t, nestedNode, source.TestStruct.GetCore().GetRootNode())
	assert.Equal(t, 1, source.GetCore().TestStruct.Value.Int.Value)
	assert.Equal(t, "some-string", source.GetCore().TestStruct.Value.Str.Value)
	assert.Equal(t, "some-string-ptr", *source.GetCore().TestStruct.Value.StrPtr.Value)
	assert.Equal(t, true, *source.GetCore().TestStruct.Value.BoolPtr.Value)
}

type TestInt int

func Test_SyncValue_TypeDefinition_Success(t *testing.T) {
	var target TestInt
	outNode, err := marshaller.SyncValue(context.Background(), 1, &target, nil, false)
	require.NoError(t, err)
	assert.Equal(t, testutils.CreateIntYamlNode(1, 0, 0), outNode)
	assert.Equal(t, TestInt(1), target)
}

type TestStructWithExtensions struct {
	marshaller.Model[TestStructWithExtensionsCore]
	Extensions *extensions.Extensions
}

type TestStructWithExtensionsCore struct {
	marshaller.CoreModel
	Extensions core.Extensions `key:"extensions"`
}

func Test_SyncValue_TypeWithExtensions_Success(t *testing.T) {
	var source TestStructWithExtensions

	extensionNode := testutils.CreateMapYamlNode(
		[]*yaml.Node{
			testutils.CreateStringYamlNode("name", 0, 0),
			testutils.CreateStringYamlNode("test", 0, 0),
			testutils.CreateStringYamlNode("value", 0, 0),
			testutils.CreateIntYamlNode(1, 0, 0),
		}, 0, 0)

	source.Extensions = extensions.New(extensions.NewElem("x-speakeasy-test", extensionNode))

	outNode, err := marshaller.SyncValue(context.Background(), &source, source.GetCore(), nil, false)
	require.NoError(t, err)

	node := testutils.CreateMapYamlNode(
		[]*yaml.Node{
			testutils.CreateStringYamlNode("x-speakeasy-test", 0, 0),
			extensionNode,
		}, 0, 0)

	assert.Equal(t, node, outNode)
	assert.Equal(t, node, source.GetCore().GetRootNode())
	assert.True(t, source.Extensions.GetCore().Has("x-speakeasy-test"))
}

// Test struct with required and optional fields for validity testing
type TestValidityStruct struct {
	RequiredField *string
	OptionalField *string
	Valid         bool
	core          TestValidityCoreModel
}

type TestValidityCoreModel struct {
	marshaller.CoreModel
	RequiredField marshaller.Node[*string] `key:"required" required:"true"`
	OptionalField marshaller.Node[*string] `key:"optional"`
}

func (t *TestValidityStruct) GetCore() *TestValidityCoreModel {
	return &t.core
}

func Test_SyncChanges_ValidityWithRequiredFields_Success(t *testing.T) {
	ctx := context.Background()

	// Test case 1: All required fields present - should be valid
	t.Run("valid when required fields present", func(t *testing.T) {
		mainModel := &TestValidityStruct{
			RequiredField: pointer.From("test value"),
			OptionalField: nil,
		}

		coreModel := &TestValidityCoreModel{}
		coreModel.RequiredField.Present = true
		coreModel.RequiredField.Value = pointer.From("test value")

		valueNode := &yaml.Node{Kind: yaml.MappingNode}

		_, err := marshaller.SyncValue(ctx, mainModel, coreModel, valueNode, false)
		require.NoError(t, err)

		assert.True(t, coreModel.GetValid(), "Expected core model to be valid when required field is present")
	})

	// Test case 2: Required field missing - should be invalid
	t.Run("invalid when required field missing", func(t *testing.T) {
		mainModel := &TestValidityStruct{
			RequiredField: nil, // nil value should result in no sync
			OptionalField: nil,
		}

		coreModel := &TestValidityCoreModel{}
		// RequiredField.Present is false by default
		coreModel.RequiredField.Value = pointer.From("test value")

		valueNode := &yaml.Node{Kind: yaml.MappingNode}

		_, err := marshaller.SyncValue(ctx, mainModel, coreModel, valueNode, false)
		require.NoError(t, err)

		assert.False(t, coreModel.GetValid(), "Expected core model to be invalid when required field is not present")
	})

	// Test case 3: Optional field missing - should still be valid
	t.Run("valid when optional field missing", func(t *testing.T) {
		mainModel := &TestValidityStruct{
			RequiredField: pointer.From("test value"),
			OptionalField: nil,
		}

		coreModel := &TestValidityCoreModel{}
		coreModel.RequiredField.Present = true
		coreModel.RequiredField.Value = pointer.From("test value")
		// OptionalField.Present is false by default

		valueNode := &yaml.Node{Kind: yaml.MappingNode}

		_, err := marshaller.SyncValue(ctx, mainModel, coreModel, valueNode, false)
		require.NoError(t, err)

		assert.True(t, coreModel.GetValid(), "Expected core model to be valid when only optional field is missing")
	})

	// Test case 4: Initially invalid becomes valid after sync
	t.Run("invalid becomes valid after required field added", func(t *testing.T) {
		mainModel := &TestValidityStruct{
			RequiredField: pointer.From("new value"),
			OptionalField: nil,
		}

		coreModel := &TestValidityCoreModel{}
		coreModel.SetValid(false) // Start as invalid
		// Initially no fields are present

		valueNode := &yaml.Node{Kind: yaml.MappingNode}

		_, err := marshaller.SyncValue(ctx, mainModel, coreModel, valueNode, false)
		require.NoError(t, err)

		// After sync, the required field should be present and model should be valid
		assert.True(t, coreModel.GetValid(), "Expected core model to become valid after syncing required field")
		assert.True(t, coreModel.RequiredField.Present, "Expected required field to be marked as present after sync")
	})
}

func Test_SyncChanges_ValidityWithInferredRequiredFields_Success(t *testing.T) {
	ctx := context.Background()

	// Test struct with inferred required field (non-pointer string)
	type TestInferredValidityCoreModel struct {
		marshaller.CoreModel
		InferredRequired marshaller.Node[string]  `key:"inferred"`    // Should be required (non-pointer)
		InferredOptional marshaller.Node[*string] `key:"inferredOpt"` // Should be optional (pointer)
	}

	type TestInferredValidityStruct struct {
		InferredRequired string
		InferredOptional *string
		Valid            bool
	}

	t.Run("inferred required field validation", func(t *testing.T) {
		mainModel := &TestInferredValidityStruct{
			InferredRequired: "test",
			InferredOptional: nil,
		}

		coreModel := &TestInferredValidityCoreModel{}
		// Initially no fields are present

		valueNode := &yaml.Node{Kind: yaml.MappingNode}

		_, err := marshaller.SyncValue(ctx, mainModel, coreModel, valueNode, false)
		require.NoError(t, err)

		// Non-pointer string field should be inferred as required and should be present after sync
		assert.True(t, coreModel.GetValid(), "Expected core model to be valid after syncing non-pointer required field")
		assert.True(t, coreModel.InferredRequired.Present, "Expected non-pointer required field to be present after sync")
	})
}
