package core

import (
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/jsonschema/oas31/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// init registers all Arazzo core types with the marshaller factory system
func init() {
	// Register all Arazzo core types
	marshaller.RegisterType(func() *Arazzo { return &Arazzo{} })
	marshaller.RegisterType(func() *Info { return &Info{} })
	marshaller.RegisterType(func() *SourceDescription { return &SourceDescription{} })
	marshaller.RegisterType(func() *Workflow { return &Workflow{} })
	marshaller.RegisterType(func() *Step { return &Step{} })
	marshaller.RegisterType(func() *Parameter { return &Parameter{} })
	marshaller.RegisterType(func() *RequestBody { return &RequestBody{} })
	marshaller.RegisterType(func() *PayloadReplacement { return &PayloadReplacement{} })
	marshaller.RegisterType(func() *SuccessAction { return &SuccessAction{} })
	marshaller.RegisterType(func() *FailureAction { return &FailureAction{} })
	marshaller.RegisterType(func() *Components { return &Components{} })
	marshaller.RegisterType(func() *Criterion { return &Criterion{} })
	marshaller.RegisterType(func() *CriterionExpressionType { return &CriterionExpressionType{} })
	marshaller.RegisterType(func() *CriterionTypeUnion { return &CriterionTypeUnion{} })

	// Register Reusable types for core
	marshaller.RegisterType(func() *Reusable[*Parameter] {
		return &Reusable[*Parameter]{}
	})
	marshaller.RegisterType(func() *Reusable[*SuccessAction] {
		return &Reusable[*SuccessAction]{}
	})
	marshaller.RegisterType(func() *Reusable[*FailureAction] {
		return &Reusable[*FailureAction]{}
	})

	// Register expression types
	marshaller.RegisterType(func() *expression.Expression { return new(expression.Expression) })

	// Register sequencedmap types used in arazzo core package
	marshaller.RegisterType(func() *sequencedmap.Map[string, core.JSONSchema] {
		return &sequencedmap.Map[string, core.JSONSchema]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *Parameter] {
		return &sequencedmap.Map[string, *Parameter]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *SuccessAction] {
		return &sequencedmap.Map[string, *SuccessAction]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, *FailureAction] {
		return &sequencedmap.Map[string, *FailureAction]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, expression.Expression] {
		return &sequencedmap.Map[string, expression.Expression]{}
	})
	marshaller.RegisterType(func() *sequencedmap.Map[string, marshaller.Node[string]] {
		return &sequencedmap.Map[string, marshaller.Node[string]]{}
	})
}
