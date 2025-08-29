package walk_test

import (
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller/tests"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/walk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetAtLocation_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		parent   any
		location walk.LocationContext[string]
		value    any
		validate func(t *testing.T, parent any)
	}{
		{
			name:   "set value in native Go map with string key",
			parent: map[string]string{"existing": "value"},
			location: walk.LocationContext[string]{
				ParentKey: pointer.From("newKey"),
			},
			value: "newValue",
			validate: func(t *testing.T, parent any) {
				t.Helper()
				m := parent.(map[string]string)
				assert.Equal(t, "newValue", m["newKey"], "new key should be set")
				assert.Equal(t, "value", m["existing"], "existing key should remain")
			},
		},
		{
			name:   "overwrite existing value in native Go map",
			parent: map[string]string{"key": "oldValue"},
			location: walk.LocationContext[string]{
				ParentKey: pointer.From("key"),
			},
			value: "newValue",
			validate: func(t *testing.T, parent any) {
				t.Helper()
				m := parent.(map[string]string)
				assert.Equal(t, "newValue", m["key"], "key should be overwritten")
			},
		},
		{
			name:   "set value in native Go slice",
			parent: []string{"first", "second", "third"},
			location: walk.LocationContext[string]{
				ParentIndex: pointer.From(1),
			},
			value: "modified",
			validate: func(t *testing.T, parent any) {
				t.Helper()
				s := parent.([]string)
				assert.Equal(t, []string{"first", "modified", "third"}, s, "slice should be modified at index")
			},
		},
		{
			name:   "set value in sequencedmap",
			parent: sequencedmap.New(sequencedmap.NewElem("key1", "value1")),
			location: walk.LocationContext[string]{
				ParentKey: pointer.From("key2"),
			},
			value: "value2",
			validate: func(t *testing.T, parent any) {
				t.Helper()
				sm := parent.(*sequencedmap.Map[string, string])
				val, ok := sm.Get("key2")
				assert.True(t, ok, "new key should exist")
				assert.Equal(t, "value2", val, "new key should have correct value")

				val1, ok1 := sm.Get("key1")
				assert.True(t, ok1, "existing key should remain")
				assert.Equal(t, "value1", val1, "existing key should have correct value")
			},
		},
		{
			name:   "overwrite value in sequencedmap",
			parent: sequencedmap.New(sequencedmap.NewElem("key1", "oldValue")),
			location: walk.LocationContext[string]{
				ParentKey: pointer.From("key1"),
			},
			value: "newValue",
			validate: func(t *testing.T, parent any) {
				t.Helper()
				sm := parent.(*sequencedmap.Map[string, string])
				val, ok := sm.Get("key1")
				assert.True(t, ok, "key should exist")
				assert.Equal(t, "newValue", val, "key should be overwritten")
			},
		},
		{
			name: "set field in normal model",
			parent: &tests.TestPrimitiveHighModel{
				StringField: "original",
			},
			location: walk.LocationContext[string]{
				ParentField: "stringField",
			},
			value: "modified",
			validate: func(t *testing.T, parent any) {
				t.Helper()
				model := parent.(*tests.TestPrimitiveHighModel)
				assert.Equal(t, "modified", model.StringField, "field should be modified")
			},
		},
		{
			name: "set pointer field in normal model",
			parent: &tests.TestPrimitiveHighModel{
				StringPtrField: pointer.From("original"),
			},
			location: walk.LocationContext[string]{
				ParentField: "stringPtrField",
			},
			value: pointer.From("modified"),
			validate: func(t *testing.T, parent any) {
				t.Helper()
				model := parent.(*tests.TestPrimitiveHighModel)
				require.NotNil(t, model.StringPtrField, "pointer field should not be nil")
				assert.Equal(t, "modified", *model.StringPtrField, "pointer field should be modified")
			},
		},
		{
			name: "set bool field in normal model",
			parent: &tests.TestPrimitiveHighModel{
				BoolField: false,
			},
			location: walk.LocationContext[string]{
				ParentField: "boolField",
			},
			value: true,
			validate: func(t *testing.T, parent any) {
				t.Helper()
				model := parent.(*tests.TestPrimitiveHighModel)
				assert.True(t, model.BoolField, "bool field should be modified")
			},
		},
		{
			name: "set int field in normal model",
			parent: &tests.TestPrimitiveHighModel{
				IntField: 42,
			},
			location: walk.LocationContext[string]{
				ParentField: "intField",
			},
			value: 100,
			validate: func(t *testing.T, parent any) {
				t.Helper()
				model := parent.(*tests.TestPrimitiveHighModel)
				assert.Equal(t, 100, model.IntField, "int field should be modified")
			},
		},
		{
			name: "set float field in normal model",
			parent: &tests.TestPrimitiveHighModel{
				Float64Field: 3.14,
			},
			location: walk.LocationContext[string]{
				ParentField: "float64Field",
			},
			value: 2.71,
			validate: func(t *testing.T, parent any) {
				t.Helper()
				model := parent.(*tests.TestPrimitiveHighModel)
				assert.InDelta(t, 2.71, model.Float64Field, 0.001, "float field should be modified")
			},
		},
		{
			name: "set value in embedded sequencedmap model",
			parent: &tests.TestEmbeddedMapHighModel{
				Map: *sequencedmap.New(sequencedmap.NewElem("existing", "value")),
			},
			location: walk.LocationContext[string]{
				ParentKey: pointer.From("newKey"),
			},
			value: "newValue",
			validate: func(t *testing.T, parent any) {
				t.Helper()
				model := parent.(*tests.TestEmbeddedMapHighModel)
				val, ok := model.Get("newKey")
				assert.True(t, ok, "new key should exist in embedded map")
				assert.Equal(t, "newValue", val, "new key should have correct value")

				existingVal, existingOk := model.Get("existing")
				assert.True(t, existingOk, "existing key should remain")
				assert.Equal(t, "value", existingVal, "existing key should have correct value")
			},
		},
		{
			name: "set field in embedded sequencedmap model with fields",
			parent: &tests.TestEmbeddedMapWithFieldsHighModel{
				Map:       *sequencedmap.New[string, *tests.TestPrimitiveHighModel](),
				NameField: "original",
			},
			location: walk.LocationContext[string]{
				ParentField: "name",
			},
			value: "modified",
			validate: func(t *testing.T, parent any) {
				t.Helper()
				model := parent.(*tests.TestEmbeddedMapWithFieldsHighModel)
				assert.Equal(t, "modified", model.NameField, "name field should be modified")
			},
		},
		{
			name: "set map entry in embedded sequencedmap model with fields",
			parent: &tests.TestEmbeddedMapWithFieldsHighModel{
				Map:       *sequencedmap.New[string, *tests.TestPrimitiveHighModel](),
				NameField: "test",
			},
			location: walk.LocationContext[string]{
				ParentKey: pointer.From("mapKey"),
			},
			value: &tests.TestPrimitiveHighModel{StringField: "mapValue"},
			validate: func(t *testing.T, parent any) {
				t.Helper()
				model := parent.(*tests.TestEmbeddedMapWithFieldsHighModel)
				val, ok := model.Get("mapKey")
				assert.True(t, ok, "map key should exist")
				require.NotNil(t, val, "map value should not be nil")
				assert.Equal(t, "mapValue", val.StringField, "map value should be correct")
			},
		},
		{
			name: "set value in field map using both ParentField and ParentKey",
			parent: &tests.TestComplexHighModel{
				MapPrimitiveField: sequencedmap.New(sequencedmap.NewElem("existing", "value")),
			},
			location: walk.LocationContext[string]{
				ParentField: "mapField",
				ParentKey:   pointer.From("newKey"),
			},
			value: "newValue",
			validate: func(t *testing.T, parent any) {
				t.Helper()
				model := parent.(*tests.TestComplexHighModel)
				val, ok := model.MapPrimitiveField.Get("newKey")
				assert.True(t, ok, "new key should exist in map field")
				assert.Equal(t, "newValue", val, "new key should be set in map field")

				existingVal, existingOk := model.MapPrimitiveField.Get("existing")
				assert.True(t, existingOk, "existing key should remain in map field")
				assert.Equal(t, "value", existingVal, "existing key should have correct value")
			},
		},
		{
			name: "set value in field slice using both ParentField and ParentIndex",
			parent: &tests.TestComplexHighModel{
				ArrayField: []string{"first", "second", "third"},
			},
			location: walk.LocationContext[string]{
				ParentField: "arrayField",
				ParentIndex: pointer.From(1),
			},
			value: "modified",
			validate: func(t *testing.T, parent any) {
				t.Helper()
				model := parent.(*tests.TestComplexHighModel)
				assert.Equal(t, []string{"first", "modified", "third"}, model.ArrayField, "slice element should be modified at index in field")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parent := tt.parent
			if reflect.TypeOf(parent).Kind() != reflect.Ptr {
				parent = &tt.parent
			}

			err := walk.SetAtLocation(parent, tt.location, tt.value)
			require.NoError(t, err, "SetAtLocation should not return error")
			tt.validate(t, tt.parent)
		})
	}
}
