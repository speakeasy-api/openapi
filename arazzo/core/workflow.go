package core

import (
	coreExtensions "github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/jsonschema/oas31/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type Workflow struct {
	marshaller.CoreModel
	WorkflowID     marshaller.Node[string]                      `key:"workflowId"`
	Summary        marshaller.Node[*string]                     `key:"summary"`
	Description    marshaller.Node[*string]                     `key:"description"`
	Parameters     marshaller.Node[[]*Reusable[*Parameter]]     `key:"parameters"`
	Inputs         marshaller.Node[core.JSONSchema]             `key:"inputs"`
	DependsOn      marshaller.Node[[]marshaller.Node[string]]   `key:"dependsOn"`
	Steps          marshaller.Node[[]*Step]                     `key:"steps" required:"true"`
	SuccessActions marshaller.Node[[]*Reusable[*SuccessAction]] `key:"successActions"`
	FailureActions marshaller.Node[[]*Reusable[*FailureAction]] `key:"failureActions"`
	Outputs        marshaller.Node[Outputs]                     `key:"outputs"`
	Extensions     coreExtensions.Extensions                    `key:"extensions"`
}
