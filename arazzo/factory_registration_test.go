package arazzo_test

import (
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/values"
	"github.com/stretchr/testify/assert"
)

func TestFactoryRegistration_Arazzo11Types_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  reflect.Type
	}{
		{name: "Selector", typ: reflect.TypeOf(arazzo.Selector{})},
		{name: "OutputValue", typ: reflect.TypeOf(arazzo.OutputValue{})},
		{name: "EitherValue[Selector, Expression]", typ: reflect.TypeOf(values.EitherValue[arazzo.Selector, core.Selector, expression.Expression, string]{})},
		{name: "StepAction", typ: reflect.TypeOf(arazzo.StepAction(""))},
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
