package asyncapi_test

import (
	"reflect"
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
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
		{name: "Server", typ: reflect.TypeOf(asyncapi.Server{})},
		{name: "ServerVariable", typ: reflect.TypeOf(asyncapi.ServerVariable{})},
		{name: "Channel", typ: reflect.TypeOf(asyncapi.Channel{})},
		{name: "Parameter", typ: reflect.TypeOf(asyncapi.Parameter{})},
		{name: "Operation", typ: reflect.TypeOf(asyncapi.Operation{})},
		{name: "OperationTrait", typ: reflect.TypeOf(asyncapi.OperationTrait{})},
		{name: "OperationReply", typ: reflect.TypeOf(asyncapi.OperationReply{})},
		{name: "OperationReplyAddress", typ: reflect.TypeOf(asyncapi.OperationReplyAddress{})},
		{name: "Message", typ: reflect.TypeOf(asyncapi.Message{})},
		{name: "MessageTrait", typ: reflect.TypeOf(asyncapi.MessageTrait{})},
		{name: "MessageExample", typ: reflect.TypeOf(asyncapi.MessageExample{})},
		{name: "CorrelationID", typ: reflect.TypeOf(asyncapi.CorrelationID{})},
		{name: "SecurityScheme", typ: reflect.TypeOf(asyncapi.SecurityScheme{})},
		{name: "OAuthFlows", typ: reflect.TypeOf(asyncapi.OAuthFlows{})},
		{name: "OAuthFlow", typ: reflect.TypeOf(asyncapi.OAuthFlow{})},
		{name: "Bindings", typ: reflect.TypeOf(asyncapi.Bindings{})},
		{name: "Components", typ: reflect.TypeOf(asyncapi.Components{})},
		{name: "ReferenceObject", typ: reflect.TypeOf(asyncapi.ReferenceObject{})},
		{name: "ReferencedServer", typ: reflect.TypeOf(asyncapi.ReferencedServer{})},
		{name: "ReferencedChannel", typ: reflect.TypeOf(asyncapi.ReferencedChannel{})},
		{name: "ReferencedOperation", typ: reflect.TypeOf(asyncapi.ReferencedOperation{})},
		{name: "ReferencedMessage", typ: reflect.TypeOf(asyncapi.ReferencedMessage{})},
		{name: "ReferencedBindings", typ: reflect.TypeOf(asyncapi.ReferencedBindings{})},
		{name: "slice of referenced security schemes", typ: reflect.TypeOf([]*asyncapi.ReferencedSecurityScheme{})},
		{name: "map of referenced channels", typ: reflect.TypeOf(sequencedmap.Map[string, *asyncapi.ReferencedChannel]{})},
		{name: "core AsyncAPI", typ: reflect.TypeOf(core.AsyncAPI{})},
		{name: "core Info", typ: reflect.TypeOf(core.Info{})},
		{name: "core Contact", typ: reflect.TypeOf(core.Contact{})},
		{name: "core License", typ: reflect.TypeOf(core.License{})},
		{name: "core Tag", typ: reflect.TypeOf(core.Tag{})},
		{name: "core Server", typ: reflect.TypeOf(core.Server{})},
		{name: "core Channel", typ: reflect.TypeOf(core.Channel{})},
		{name: "core Operation", typ: reflect.TypeOf(core.Operation{})},
		{name: "core Message", typ: reflect.TypeOf(core.Message{})},
		{name: "core SecurityScheme", typ: reflect.TypeOf(core.SecurityScheme{})},
		{name: "core Bindings", typ: reflect.TypeOf(core.Bindings{})},
		{name: "core Components", typ: reflect.TypeOf(core.Components{})},
		{name: "core ReferenceObject", typ: reflect.TypeOf(core.ReferenceObject{})},
		{name: "core Reference to Server", typ: reflect.TypeOf(core.Reference[*core.Server]{})},
		{name: "core slice of Reference to MessageTrait", typ: reflect.TypeOf([]*core.Reference[*core.MessageTrait]{})},
		{name: "core map of Reference to Message", typ: reflect.TypeOf(sequencedmap.Map[string, *core.Reference[*core.Message]]{})},
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
