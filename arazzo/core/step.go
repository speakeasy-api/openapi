package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type Step struct {
	marshaller.CoreModel
	StepID          marshaller.Node[string]                      `key:"stepId"`
	Description     marshaller.Node[*string]                     `key:"description"`
	OperationID     marshaller.Node[*string]                     `key:"operationId"`
	OperationPath   marshaller.Node[*string]                     `key:"operationPath"`
	WorkflowID      marshaller.Node[*string]                     `key:"workflowId"`
	Parameters      marshaller.Node[[]*Reusable[*Parameter]]     `key:"parameters"`
	RequestBody     marshaller.Node[*RequestBody]                `key:"requestBody"`
	SuccessCriteria marshaller.Node[[]*Criterion]                `key:"successCriteria"`
	OnSuccess       marshaller.Node[[]*Reusable[*SuccessAction]] `key:"onSuccess"`
	OnFailure       marshaller.Node[[]*Reusable[*FailureAction]] `key:"onFailure"`
	Outputs         marshaller.Node[Outputs]                     `key:"outputs"`
	Extensions      core.Extensions                              `key:"extensions"`
}
