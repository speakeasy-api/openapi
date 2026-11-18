package asyncapi_test

import (
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
	"github.com/stretchr/testify/assert"
)

func TestFactoryRegistration_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  reflect.Type
	}{
		{name: "Schema", typ: reflect.TypeOf(asyncapi.Schema{})},
		{name: "JSONSchema", typ: reflect.TypeOf(asyncapi.JSONSchema{})},
		{name: "ExternalDocumentation", typ: reflect.TypeOf(asyncapi.ExternalDocumentation{})},
		{name: "MultiFormatSchema", typ: reflect.TypeOf(asyncapi.MultiFormatSchema{})},
		{name: "SchemaDefinition", typ: reflect.TypeOf(asyncapi.SchemaDefinition{})},
		{name: "SchemaUnion", typ: reflect.TypeOf(asyncapi.SchemaUnion{})},
		{name: "Items", typ: reflect.TypeOf(asyncapi.Items{})},
		{name: "Dependency", typ: reflect.TypeOf(asyncapi.Dependency{})},
		{name: "Type", typ: reflect.TypeOf(asyncapi.Type{})},
		{name: "schema or bool either value", typ: reflect.TypeOf(values.EitherValue[asyncapi.Schema, core.Schema, bool, bool]{})},
		{name: "slice of schemas", typ: reflect.TypeOf([]*asyncapi.JSONSchema{})},
		{name: "map of dependencies", typ: reflect.TypeOf(sequencedmap.Map[string, *asyncapi.Dependency]{})},
		{name: "map of schemas", typ: reflect.TypeOf(sequencedmap.Map[string, *asyncapi.JSONSchema]{})},
		{name: "map of schema unions", typ: reflect.TypeOf(sequencedmap.Map[string, *asyncapi.SchemaUnion]{})},
		{name: "core Schema", typ: reflect.TypeOf(core.Schema{})},
		{name: "core MultiFormatSchema", typ: reflect.TypeOf(core.MultiFormatSchema{})},
		{name: "core SchemaDefinition", typ: reflect.TypeOf(core.SchemaDefinition{})},
		{name: "core SchemaUnion", typ: reflect.TypeOf(core.SchemaUnion{})},
		{name: "core Items", typ: reflect.TypeOf(core.Items{})},
		{name: "core Dependency", typ: reflect.TypeOf(core.Dependency{})},
		{name: "core map of schema unions", typ: reflect.TypeOf(sequencedmap.Map[string, *core.SchemaUnion]{})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.True(t, marshaller.IsRegistered(tt.typ), "type should be registered")
			instance := marshaller.CreateInstance(tt.typ)
			assert.Equal(t, reflect.PointerTo(tt.typ), instance.Type(), "factory should create a pointer to the type")
		})
	}
}
