package core

import (
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// init registers all Swagger 2.0 core types with the marshaller factory system
func init() {
	// Register main Swagger types
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
	marshaller.RegisterType(func() *Reference[*Parameter] { return &Reference[*Parameter]{} })
	marshaller.RegisterType(func() *Reference[*Response] { return &Reference[*Response]{} })
	marshaller.RegisterType(func() *Reference[*PathItem] { return &Reference[*PathItem]{} })
	marshaller.RegisterType(func() *Reference[*SecurityScheme] { return &Reference[*SecurityScheme]{} })

	// Register Node-wrapped types
	marshaller.RegisterType(func() *marshaller.Node[*PathItem] { return &marshaller.Node[*PathItem]{} })
	marshaller.RegisterType(func() *marshaller.Node[*Operation] { return &marshaller.Node[*Operation]{} })
	marshaller.RegisterType(func() *marshaller.Node[*Parameter] { return &marshaller.Node[*Parameter]{} })
	marshaller.RegisterType(func() *marshaller.Node[*Response] { return &marshaller.Node[*Response]{} })
	marshaller.RegisterType(func() *marshaller.Node[*Reference[*Parameter]] { return &marshaller.Node[*Reference[*Parameter]]{} })
	marshaller.RegisterType(func() *marshaller.Node[*Reference[*Response]] { return &marshaller.Node[*Reference[*Response]]{} })
	marshaller.RegisterType(func() *marshaller.Node[*SecurityScheme] { return &marshaller.Node[*SecurityScheme]{} })
	marshaller.RegisterType(func() *marshaller.Node[*Header] { return &marshaller.Node[*Header]{} })
	marshaller.RegisterType(func() *marshaller.Node[[]string] { return &marshaller.Node[[]string]{} })

	// Register sequencedmap types used in swagger/core
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[*PathItem]] {
		return &sequencedmap.Map[string, marshaller.Node[*PathItem]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[*Operation]] {
		return &sequencedmap.Map[string, marshaller.Node[*Operation]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[*Parameter]] {
		return &sequencedmap.Map[string, marshaller.Node[*Parameter]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[*Response]] {
		return &sequencedmap.Map[string, marshaller.Node[*Response]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[*Reference[*Parameter]]] {
		return &sequencedmap.Map[string, marshaller.Node[*Reference[*Parameter]]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[*Reference[*Response]]] {
		return &sequencedmap.Map[string, marshaller.Node[*Reference[*Response]]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[*SecurityScheme]] {
		return &sequencedmap.Map[string, marshaller.Node[*SecurityScheme]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[*Header]] {
		return &sequencedmap.Map[string, marshaller.Node[*Header]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[string]] {
		return &sequencedmap.Map[string, marshaller.Node[string]]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[[]string]] {
		return &sequencedmap.Map[string, marshaller.Node[[]string]]{}
	})
}
