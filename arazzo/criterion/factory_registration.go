package criterion

import "github.com/speakeasy-api/openapi/marshaller"

// init registers all Arazzo criterion types with the marshaller factory system
func init() {
	// Register all Arazzo criterion types
	marshaller.RegisterType(func() *Criterion { return &Criterion{} })
	marshaller.RegisterType(func() *CriterionExpressionType { return &CriterionExpressionType{} })
	marshaller.RegisterType(func() *CriterionTypeUnion { return &CriterionTypeUnion{} })
}
