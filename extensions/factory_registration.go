package extensions

import "github.com/speakeasy-api/openapi/marshaller"

// init registers all extension types with the marshaller factory system
func init() {
	// Register extension types
	marshaller.RegisterType(func() *Extensions { return &Extensions{} })
	marshaller.RegisterType(func() *Element { return &Element{} })
}
