package marshaller

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/internal/testutils"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSyncValue_String(t *testing.T) {
	target := ""
	outNode, err := SyncValue(context.Background(), "some-value", &target, nil)
	require.NoError(t, err)
	assert.Equal(t, testutils.CreateStringYamlNode("some-value", 0, 0), outNode)
	assert.Equal(t, "some-value", target)
}

func TestSyncValue_StringPtrSet(t *testing.T) {
	target := pointer.From("")
	outNode, err := SyncValue(context.Background(), pointer.From("some-value"), &target, nil)
	require.NoError(t, err)
	assert.Equal(t, testutils.CreateStringYamlNode("some-value", 0, 0), outNode)
	assert.Equal(t, "some-value", *target)
}

func TestSyncValue_StringPtrNil(t *testing.T) {
	var target *string
	outNode, err := SyncValue(context.Background(), pointer.From("some-value"), &target, nil)
	require.NoError(t, err)
	assert.Equal(t, testutils.CreateStringYamlNode("some-value", 0, 0), outNode)
	assert.Equal(t, "some-value", *target)
}

type TestStructSyncer[T any] struct {
	Val *T
}

type TestStructSyncerCore[T any] struct {
	Val *T

	RootNode *yaml.Node
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
	t.RootNode, err = SyncValue(ctx, mv.FieldByName("Val").Interface(), &t.Val, valueNode)
	return t.RootNode, err
}

func TestSyncValue_StructPtr_CustomSyncer(t *testing.T) {
	var target *TestStructSyncerCore[int]

	source := &TestStructSyncer[int]{Val: pointer.From(1)}

	outNode, err := SyncValue(context.Background(), source, &target, nil)
	require.NoError(t, err)
	node := testutils.CreateIntYamlNode(1, 0, 0)
	assert.Equal(t, node, outNode)
	assert.Equal(t, node, target.RootNode)
	assert.Equal(t, 1, *target.Val)
}

func TestSyncValue_Struct_CustomSyncer(t *testing.T) {
	var target TestStructSyncerCore[int]

	source := TestStructSyncer[int]{Val: pointer.From(1)}

	outNode, err := SyncValue(context.Background(), source, &target, nil)
	require.NoError(t, err)
	node := testutils.CreateIntYamlNode(1, 0, 0)
	assert.Equal(t, node, outNode)
	assert.Equal(t, node, target.RootNode)
}

type TestStruct struct {
	Int     int
	Str     string
	StrPtr  *string
	BoolPtr *bool
}

type TestStructCore struct {
	Int     Node[int]     `key:"int"`
	Str     Node[string]  `key:"str"`
	StrPtr  Node[*string] `key:"strPtr"`
	BoolPtr Node[*bool]   `key:"boolPtr"`

	RootNode *yaml.Node
}

func TestSyncChanges_Struct(t *testing.T) {
	var target TestStructCore

	source := TestStruct{
		Int:     1,
		Str:     "some-string",
		StrPtr:  pointer.From("some-string-ptr"),
		BoolPtr: pointer.From(true),
	}

	outNode, err := SyncValue(context.Background(), source, &target, nil)
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
	assert.Equal(t, node, target.RootNode)
	assert.Equal(t, 1, target.Int.Value)
	assert.Equal(t, "some-string", target.Str.Value)
	assert.Equal(t, "some-string-ptr", *target.StrPtr.Value)
	assert.Equal(t, true, *target.BoolPtr.Value)
}

func TestSyncChanges_StructPtr(t *testing.T) {
	var target *TestStructCore

	source := &TestStruct{
		Int:     1,
		Str:     "some-string",
		StrPtr:  pointer.From("some-string-ptr"),
		BoolPtr: pointer.From(true),
	}

	outNode, err := SyncValue(context.Background(), source, &target, nil)
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
	assert.Equal(t, node, target.RootNode)
	assert.Equal(t, 1, target.Int.Value)
	assert.Equal(t, "some-string", target.Str.Value)
	assert.Equal(t, "some-string-ptr", *target.StrPtr.Value)
	assert.Equal(t, true, *target.BoolPtr.Value)
}
