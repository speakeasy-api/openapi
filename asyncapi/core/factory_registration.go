package core

import (
	schemacore "github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	values "github.com/speakeasy-api/openapi/values/core"
)

// init registers all AsyncAPI core types with the marshaller factory system
func init() {
	marshaller.RegisterType(func() *AsyncAPI { return &AsyncAPI{} })
	marshaller.RegisterType(func() *Info { return &Info{} })
	marshaller.RegisterType(func() *Contact { return &Contact{} })
	marshaller.RegisterType(func() *License { return &License{} })
	marshaller.RegisterType(func() *Tag { return &Tag{} })
	marshaller.RegisterType(func() *Server { return &Server{} })
	marshaller.RegisterType(func() *ServerVariable { return &ServerVariable{} })
	marshaller.RegisterType(func() *Channel { return &Channel{} })
	marshaller.RegisterType(func() *Parameter { return &Parameter{} })
	marshaller.RegisterType(func() *Operation { return &Operation{} })
	marshaller.RegisterType(func() *OperationTrait { return &OperationTrait{} })
	marshaller.RegisterType(func() *OperationReply { return &OperationReply{} })
	marshaller.RegisterType(func() *OperationReplyAddress { return &OperationReplyAddress{} })
	marshaller.RegisterType(func() *Message { return &Message{} })
	marshaller.RegisterType(func() *MessageTrait { return &MessageTrait{} })
	marshaller.RegisterType(func() *MessageExample { return &MessageExample{} })
	marshaller.RegisterType(func() *CorrelationID { return &CorrelationID{} })
	marshaller.RegisterType(func() *SecurityScheme { return &SecurityScheme{} })
	marshaller.RegisterType(func() *OAuthFlows { return &OAuthFlows{} })
	marshaller.RegisterType(func() *OAuthFlow { return &OAuthFlow{} })
	marshaller.RegisterType(func() *Bindings { return &Bindings{} })
	marshaller.RegisterType(func() *Components { return &Components{} })
	marshaller.RegisterType(func() *ReferenceObject { return &ReferenceObject{} })

	registerReference[*Server]()
	registerReference[*ServerVariable]()
	registerReference[*Channel]()
	registerReference[*Parameter]()
	registerReference[*Operation]()
	registerReference[*OperationTrait]()
	registerReference[*OperationReply]()
	registerReference[*OperationReplyAddress]()
	registerReference[*Message]()
	registerReference[*MessageTrait]()
	registerReference[*CorrelationID]()
	registerReference[*SecurityScheme]()
	registerReference[*Bindings]()

	marshaller.RegisterType(func() *[]*Tag { return &[]*Tag{} })
	marshaller.RegisterType(func() *[]*ReferenceObject { return &[]*ReferenceObject{} })
	marshaller.RegisterType(func() *[]*MessageExample { return &[]*MessageExample{} })

	marshaller.RegisterType(func() *sequencedmap.Map[string, *schemacore.SchemaUnion] {
		return &sequencedmap.Map[string, *schemacore.SchemaUnion]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *schemacore.ExternalDocumentation] {
		return &sequencedmap.Map[string, *schemacore.ExternalDocumentation]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Tag] {
		return &sequencedmap.Map[string, *Tag]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[string]] {
		return &sequencedmap.Map[string, marshaller.Node[string]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[values.Value]] {
		return &sequencedmap.Map[string, marshaller.Node[values.Value]]{}
	})
}

// registerReference registers a Reference, a slice of References and a map of References for the provided type.
func registerReference[T marshaller.CoreModeler]() {
	marshaller.RegisterType(func() *Reference[T] { return &Reference[T]{} })
	marshaller.RegisterType(func() *[]*Reference[T] { return &[]*Reference[T]{} })
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Reference[T]] {
		return &sequencedmap.Map[string, *Reference[T]]{}
	})
}
