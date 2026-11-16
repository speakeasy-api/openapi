package core

import (
	expression "github.com/speakeasy-api/openapi/expression/core"
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type Step struct {
	marshaller.CoreModel `model:"step"`

	StepID          marshaller.Node[string]                       `key:"stepId"`
	Description     marshaller.Node[*string]                      `key:"description"`
	OperationID     marshaller.Node[*string]                      `key:"operationId"`
	OperationPath   marshaller.Node[*string]                      `key:"operationPath"`
	WorkflowID      marshaller.Node[*string]                      `key:"workflowId"`
	ChannelPath     marshaller.Node[*string]                      `key:"channelPath"`
	Action          marshaller.Node[*string]                      `key:"action"`
	CorrelationID   marshaller.Node[expression.ValueOrExpression] `key:"correlationId"`
	Timeout         marshaller.Node[*int]                         `key:"timeout"`
	DependsOn       marshaller.Node[[]string]                     `key:"dependsOn"`
	Parameters      marshaller.Node[[]*Reusable[*Parameter]]      `key:"parameters"`
	RequestBody     marshaller.Node[*RequestBody]                 `key:"requestBody"`
	SuccessCriteria marshaller.Node[[]*Criterion]                 `key:"successCriteria"`
	OnSuccess       marshaller.Node[[]*Reusable[*SuccessAction]]  `key:"onSuccess"`
	OnFailure       marshaller.Node[[]*Reusable[*FailureAction]]  `key:"onFailure"`
	Outputs         marshaller.Node[Outputs]                      `key:"outputs"`
	Extensions      core.Extensions                               `key:"extensions"`
}
