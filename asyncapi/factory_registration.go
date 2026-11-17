package asyncapi

import (
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
)

// init registers all AsyncAPI types with the marshaller factory system
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

	registerReference[ReferencedServer]()
	registerReference[ReferencedServerVariable]()
	registerReference[ReferencedChannel]()
	registerReference[ReferencedParameter]()
	registerReference[ReferencedOperation]()
	registerReference[ReferencedOperationTrait]()
	registerReference[ReferencedOperationReply]()
	registerReference[ReferencedOperationReplyAddress]()
	registerReference[ReferencedMessage]()
	registerReference[ReferencedMessageTrait]()
	registerReference[ReferencedCorrelationID]()
	registerReference[ReferencedSecurityScheme]()
	registerReference[ReferencedBindings]()

	marshaller.RegisterType(func() *[]*Tag { return &[]*Tag{} })
	marshaller.RegisterType(func() *[]*ReferenceObject { return &[]*ReferenceObject{} })
	marshaller.RegisterType(func() *[]*MessageExample { return &[]*MessageExample{} })

	marshaller.RegisterType(func() *sequencedmap.Map[string, *ExternalDocumentation] {
		return &sequencedmap.Map[string, *ExternalDocumentation]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Tag] {
		return &sequencedmap.Map[string, *Tag]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, string] {
		return &sequencedmap.Map[string, string]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, values.Value] {
		return &sequencedmap.Map[string, values.Value]{}
	})
}

// registerReference registers a Reference, a slice of References and a map of References for the provided type.
func registerReference[R any]() {
	marshaller.RegisterType(func() *R { return new(R) })
	marshaller.RegisterType(func() *[]*R { return &[]*R{} })
	marshaller.RegisterType(func() *sequencedmap.Map[string, *R] {
		return &sequencedmap.Map[string, *R]{}
	})
}
