package asyncapi

import (
	"github.com/speakeasy-api/openapi/marshaller"
)

// init registers all AsyncAPI types with the marshaller factory system
func init() {
	marshaller.RegisterType(func() *AsyncAPI { return &AsyncAPI{} })
	marshaller.RegisterType(func() *Info { return &Info{} })
	marshaller.RegisterType(func() *Contact { return &Contact{} })
	marshaller.RegisterType(func() *License { return &License{} })
	marshaller.RegisterType(func() *Tag { return &Tag{} })
	marshaller.RegisterType(func() *[]*Tag { return &[]*Tag{} })
}
