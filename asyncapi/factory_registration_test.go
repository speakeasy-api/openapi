package asyncapi_test

import (
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
)

func TestFactoryRegistration_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  reflect.Type
	}{
		{name: "AsyncAPI", typ: reflect.TypeOf(asyncapi.AsyncAPI{})},
		{name: "Info", typ: reflect.TypeOf(asyncapi.Info{})},
		{name: "Contact", typ: reflect.TypeOf(asyncapi.Contact{})},
		{name: "License", typ: reflect.TypeOf(asyncapi.License{})},
		{name: "Tag", typ: reflect.TypeOf(asyncapi.Tag{})},
		{name: "core AsyncAPI", typ: reflect.TypeOf(core.AsyncAPI{})},
		{name: "core Info", typ: reflect.TypeOf(core.Info{})},
		{name: "core Contact", typ: reflect.TypeOf(core.Contact{})},
		{name: "core License", typ: reflect.TypeOf(core.License{})},
		{name: "core Tag", typ: reflect.TypeOf(core.Tag{})},
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
