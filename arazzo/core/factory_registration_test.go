package core

import (
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	values "github.com/speakeasy-api/openapi/values/core"
	"github.com/stretchr/testify/assert"
)

func TestFactoryRegistration_Arazzo11Types_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  reflect.Type
	}{
		{name: "Selector", typ: reflect.TypeOf(Selector{})},
		{name: "EitherValue[Selector, string]", typ: reflect.TypeOf(values.EitherValue[Selector, string]{})},
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
