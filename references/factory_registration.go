package references

import (
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
)

func init() {
	marshaller.RegisterType(func() *Reference { return pointer.From(Reference("")) })
}
