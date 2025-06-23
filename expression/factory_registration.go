package expression

import (
	"github.com/speakeasy-api/openapi/marshaller"
)

// init registers all Arazzo types with the marshaller factory system
func init() {
	// Register expression types
	marshaller.RegisterType(func() *Expression { return new(Expression) })
}
