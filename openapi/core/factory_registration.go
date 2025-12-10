package core

import (
	"github.com/speakeasy-api/openapi/expression"
	oascore "github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// init registers all OpenAPI core types with the marshaller factory system
func init() {
	// Register all core OpenAPI types
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

	// Register Reference types
	marshaller.RegisterType(func() *Reference[*PathItem] { return &Reference[*PathItem]{} })
	marshaller.RegisterType(func() *Reference[*Response] { return &Reference[*Response]{} })
	marshaller.RegisterType(func() *Reference[*Header] { return &Reference[*Header]{} })
	marshaller.RegisterType(func() *Reference[*Link] { return &Reference[*Link]{} })
	marshaller.RegisterType(func() *Reference[*Parameter] { return &Reference[*Parameter]{} })
	marshaller.RegisterType(func() *Reference[*Example] { return &Reference[*Example]{} })
	marshaller.RegisterType(func() *Reference[*RequestBody] { return &Reference[*RequestBody]{} })
	marshaller.RegisterType(func() *Reference[*SecurityScheme] { return &Reference[*SecurityScheme]{} })
	marshaller.RegisterType(func() *Reference[*Callback] { return &Reference[*Callback]{} })

	// Register specific sequencedmap types used in core
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Reference[*PathItem]] {
		return &sequencedmap.Map[string, *Reference[*PathItem]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Operation] {
		return &sequencedmap.Map[string, *Operation]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Reference[*Response]] {
		return &sequencedmap.Map[string, *Reference[*Response]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Reference[*Header]] {
		return &sequencedmap.Map[string, *Reference[*Header]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *MediaType] {
		return &sequencedmap.Map[string, *MediaType]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Reference[*Link]] {
		return &sequencedmap.Map[string, *Reference[*Link]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Reference[*Parameter]] {
		return &sequencedmap.Map[string, *Reference[*Parameter]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Reference[*Example]] {
		return &sequencedmap.Map[string, *Reference[*Example]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Reference[*RequestBody]] {
		return &sequencedmap.Map[string, *Reference[*RequestBody]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Reference[*SecurityScheme]] {
		return &sequencedmap.Map[string, *Reference[*SecurityScheme]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Reference[*Callback]] {
		return &sequencedmap.Map[string, *Reference[*Callback]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, oascore.JSONSchema] {
		return &sequencedmap.Map[string, oascore.JSONSchema]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *ServerVariable] {
		return &sequencedmap.Map[string, *ServerVariable]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[expression.ValueOrExpression]] {
		return &sequencedmap.Map[string, marshaller.Node[expression.ValueOrExpression]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, string] {
		return &sequencedmap.Map[string, string]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[string]] {
		return &sequencedmap.Map[string, marshaller.Node[string]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, []marshaller.Node[string]] {
		return &sequencedmap.Map[string, []marshaller.Node[string]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Encoding] {
		return &sequencedmap.Map[string, *Encoding]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, []marshaller.Node[string]] {
		return &sequencedmap.Map[string, []marshaller.Node[string]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[string]] {
		return &sequencedmap.Map[string, marshaller.Node[string]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, []marshaller.Node[string]] {
		return &sequencedmap.Map[string, []marshaller.Node[string]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[[]marshaller.Node[string]]] {
		return &sequencedmap.Map[string, marshaller.Node[[]marshaller.Node[string]]]{}
	})
	marshaller.RegisterType(func() *marshaller.Node[[]marshaller.Node[string]] {
		return &marshaller.Node[[]marshaller.Node[string]]{}
	})
	marshaller.RegisterType(func() *marshaller.Node[*Operation] {
		return &marshaller.Node[*Operation]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[*Operation]] {
		return &sequencedmap.Map[string, marshaller.Node[*Operation]]{}
	})
}
