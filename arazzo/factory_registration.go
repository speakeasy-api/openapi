package arazzo

import (
	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/jsonschema/oas31"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// init registers all Arazzo types with the marshaller factory system
func init() {
	// Register all Arazzo types
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

	// Register enum types
	marshaller.RegisterType(func() *In { return new(In) })

	// Register Reusable types
	marshaller.RegisterType(func() *Reusable[Parameter, *Parameter, *core.Parameter] {
		return &Reusable[Parameter, *Parameter, *core.Parameter]{}
	})
	marshaller.RegisterType(func() *Reusable[SuccessAction, *SuccessAction, *core.SuccessAction] {
		return &Reusable[SuccessAction, *SuccessAction, *core.SuccessAction]{}
	})
	marshaller.RegisterType(func() *Reusable[FailureAction, *FailureAction, *core.FailureAction] {
		return &Reusable[FailureAction, *FailureAction, *core.FailureAction]{}
	})

	// Register expression types
	marshaller.RegisterType(func() *expression.Expression { return new(expression.Expression) })

	// Register sequencedmap types used in arazzo package
	marshaller.RegisterType(func() *sequencedmap.Map[string, oas31.JSONSchema] {
		return &sequencedmap.Map[string, oas31.JSONSchema]{}
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
}
