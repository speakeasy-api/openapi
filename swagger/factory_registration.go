package swagger

import (
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
)

// init registers all Swagger 2.0 wrapper types with the marshaller factory system
func init() {
	// Register wrapper types
	marshaller.RegisterType(func() *Swagger { return &Swagger{} })
	marshaller.RegisterType(func() *Info { return &Info{} })
	marshaller.RegisterType(func() *Contact { return &Contact{} })
	marshaller.RegisterType(func() *License { return &License{} })
	marshaller.RegisterType(func() *Paths { return &Paths{} })
	marshaller.RegisterType(func() *PathItem { return &PathItem{} })
	marshaller.RegisterType(func() *Operation { return &Operation{} })
	marshaller.RegisterType(func() *Parameter { return &Parameter{} })
	marshaller.RegisterType(func() *Items { return &Items{} })
	marshaller.RegisterType(func() *Responses { return &Responses{} })
	marshaller.RegisterType(func() *Response { return &Response{} })
	marshaller.RegisterType(func() *Header { return &Header{} })
	marshaller.RegisterType(func() *SecurityScheme { return &SecurityScheme{} })
	marshaller.RegisterType(func() *SecurityRequirement { return &SecurityRequirement{} })
	marshaller.RegisterType(func() *Tag { return &Tag{} })
	marshaller.RegisterType(func() *ExternalDocumentation { return &ExternalDocumentation{} })

	// Register Reference types
	marshaller.RegisterType(func() *ReferencedParameter { return &ReferencedParameter{} })
	marshaller.RegisterType(func() *ReferencedResponse { return &ReferencedResponse{} })

	// Register Reference types
	marshaller.RegisterType(func() *ReferencedParameter { return &ReferencedParameter{} })
	marshaller.RegisterType(func() *ReferencedResponse { return &ReferencedResponse{} })

	// Register enum types
	marshaller.RegisterType(func() *HTTPMethod { return new(HTTPMethod) })
	marshaller.RegisterType(func() *ParameterIn { return new(ParameterIn) })
	marshaller.RegisterType(func() *CollectionFormat { return new(CollectionFormat) })
	marshaller.RegisterType(func() *SecuritySchemeType { return new(SecuritySchemeType) })
	marshaller.RegisterType(func() *SecuritySchemeIn { return new(SecuritySchemeIn) })
	marshaller.RegisterType(func() *OAuth2Flow { return new(OAuth2Flow) })

	// Register sequencedmap types used in swagger package
	marshaller.RegisterType(func() *sequencedmap.Map[string, *PathItem] {
		return &sequencedmap.Map[string, *PathItem]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[HTTPMethod, *Operation] {
		return &sequencedmap.Map[HTTPMethod, *Operation]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Parameter] {
		return &sequencedmap.Map[string, *Parameter]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Response] {
		return &sequencedmap.Map[string, *Response]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *ReferencedParameter] {
		return &sequencedmap.Map[string, *ReferencedParameter]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *ReferencedResponse] {
		return &sequencedmap.Map[string, *ReferencedResponse]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Concrete]] {
		return &sequencedmap.Map[string, *oas3.JSONSchema[oas3.Concrete]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *SecurityScheme] {
		return &sequencedmap.Map[string, *SecurityScheme]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Header] {
		return &sequencedmap.Map[string, *Header]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, values.Value] {
		return &sequencedmap.Map[string, values.Value]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, string] {
		return &sequencedmap.Map[string, string]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, []string] {
		return &sequencedmap.Map[string, []string]{}
	})
}
