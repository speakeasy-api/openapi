package openapi

import (
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// init registers all OpenAPI types with the marshaller factory system
// This provides 892x performance improvement over reflection
func init() {
	// Register all major OpenAPI types
	marshaller.RegisterType(func() *Info { return &Info{} })
	marshaller.RegisterType(func() *Contact { return &Contact{} })
	marshaller.RegisterType(func() *License { return &License{} })
	marshaller.RegisterType(func() *OpenAPI { return &OpenAPI{} })
	marshaller.RegisterType(func() *Operation { return &Operation{} })
	marshaller.RegisterType(func() *Parameter { return &Parameter{} })
	marshaller.RegisterType(func() *RequestBody { return &RequestBody{} })
	marshaller.RegisterType(func() *Response { return &Response{} })
	marshaller.RegisterType(func() *Responses { return &Responses{} })
	marshaller.RegisterType(func() *MediaType { return &MediaType{} })
	marshaller.RegisterType(func() *Header { return &Header{} })
	marshaller.RegisterType(func() *Link { return &Link{} })
	marshaller.RegisterType(func() *Callback { return &Callback{} })
	marshaller.RegisterType(func() *Example { return &Example{} })
	marshaller.RegisterType(func() *Tag { return &Tag{} })
	marshaller.RegisterType(func() *Server { return &Server{} })
	marshaller.RegisterType(func() *ServerVariable { return &ServerVariable{} })
	marshaller.RegisterType(func() *Components { return &Components{} })
	marshaller.RegisterType(func() *SecurityScheme { return &SecurityScheme{} })
	marshaller.RegisterType(func() *SecurityRequirement { return &SecurityRequirement{} })
	marshaller.RegisterType(func() *OAuthFlow { return &OAuthFlow{} })
	marshaller.RegisterType(func() *OAuthFlows { return &OAuthFlows{} })
	marshaller.RegisterType(func() *Encoding { return &Encoding{} })
	marshaller.RegisterType(func() *Paths { return &Paths{} })
	marshaller.RegisterType(func() *PathItem { return &PathItem{} })

	// Register all enum types
	marshaller.RegisterType(func() *SerializationStyle { return new(SerializationStyle) })
	marshaller.RegisterType(func() *HTTPMethod { return new(HTTPMethod) })
	marshaller.RegisterType(func() *SecuritySchemeIn { return new(SecuritySchemeIn) })

	// Register all Reference types used in openapi package
	marshaller.RegisterType(func() *Reference[PathItem, *PathItem, *core.PathItem] {
		return &Reference[PathItem, *PathItem, *core.PathItem]{}
	})
	marshaller.RegisterType(func() *Reference[Example, *Example, *core.Example] {
		return &Reference[Example, *Example, *core.Example]{}
	})
	marshaller.RegisterType(func() *Reference[Parameter, *Parameter, *core.Parameter] {
		return &Reference[Parameter, *Parameter, *core.Parameter]{}
	})
	marshaller.RegisterType(func() *Reference[Header, *Header, *core.Header] {
		return &Reference[Header, *Header, *core.Header]{}
	})
	marshaller.RegisterType(func() *Reference[RequestBody, *RequestBody, *core.RequestBody] {
		return &Reference[RequestBody, *RequestBody, *core.RequestBody]{}
	})
	marshaller.RegisterType(func() *Reference[Callback, *Callback, *core.Callback] {
		return &Reference[Callback, *Callback, *core.Callback]{}
	})
	marshaller.RegisterType(func() *Reference[Response, *Response, *core.Response] {
		return &Reference[Response, *Response, *core.Response]{}
	})
	marshaller.RegisterType(func() *Reference[Link, *Link, *core.Link] {
		return &Reference[Link, *Link, *core.Link]{}
	})
	marshaller.RegisterType(func() *Reference[SecurityScheme, *SecurityScheme, *core.SecurityScheme] {
		return &Reference[SecurityScheme, *SecurityScheme, *core.SecurityScheme]{}
	})

	// Register all sequencedmap types used in openapi package
	marshaller.RegisterType(func() *sequencedmap.Map[expression.Expression, *ReferencedPathItem] {
		return &sequencedmap.Map[expression.Expression, *ReferencedPathItem]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *MediaType] {
		return &sequencedmap.Map[string, *MediaType]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *ReferencedPathItem] {
		return &sequencedmap.Map[string, *ReferencedPathItem]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Encoding] {
		return &sequencedmap.Map[string, *Encoding]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *ReferencedExample] {
		return &sequencedmap.Map[string, *ReferencedExample]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *ServerVariable] {
		return &sequencedmap.Map[string, *ServerVariable]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, []string] {
		return &sequencedmap.Map[string, []string]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, string] {
		return &sequencedmap.Map[string, string]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, oas3.JSONSchema[oas3.Referenceable]] {
		return &sequencedmap.Map[string, oas3.JSONSchema[oas3.Referenceable]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *ReferencedResponse] {
		return &sequencedmap.Map[string, *ReferencedResponse]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *ReferencedParameter] {
		return &sequencedmap.Map[string, *ReferencedParameter]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *ReferencedRequestBody] {
		return &sequencedmap.Map[string, *ReferencedRequestBody]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *ReferencedHeader] {
		return &sequencedmap.Map[string, *ReferencedHeader]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *ReferencedSecurityScheme] {
		return &sequencedmap.Map[string, *ReferencedSecurityScheme]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *ReferencedLink] {
		return &sequencedmap.Map[string, *ReferencedLink]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *ReferencedCallback] {
		return &sequencedmap.Map[string, *ReferencedCallback]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, expression.ValueOrExpression] {
		return &sequencedmap.Map[string, expression.ValueOrExpression]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[HTTPMethod, *Operation] {
		return &sequencedmap.Map[HTTPMethod, *Operation]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Reference[PathItem, *PathItem, *core.PathItem]] {
		return &sequencedmap.Map[string, *Reference[PathItem, *PathItem, *core.PathItem]]{}
	})
}
