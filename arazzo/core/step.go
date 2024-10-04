package core

import (
	"context"

	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type Step struct {
	StepID          marshaller.Node[string]                    `key:"stepId"`
	Description     marshaller.Node[*string]                   `key:"description"`
	OperationID     marshaller.Node[*Expression]               `key:"operationId"`
	OperationPath   marshaller.Node[*string]                   `key:"operationPath"`
	WorkflowID      marshaller.Node[*Expression]               `key:"workflowId"`
	Parameters      marshaller.Node[[]Reusable[Parameter]]     `key:"parameters"`
	RequestBody     marshaller.Node[*RequestBody]              `key:"requestBody"`
	SuccessCriteria marshaller.Node[[]Criterion]               `key:"successCriteria"`
	OnSuccess       marshaller.Node[[]Reusable[SuccessAction]] `key:"onSuccess"`
	OnFailure       marshaller.Node[[]Reusable[FailureAction]] `key:"onFailure"`
	Outputs         marshaller.Node[Outputs]                   `key:"outputs"`
	Extensions      Extensions                                 `key:"extensions"`

	RootNode *yaml.Node
}

var _ CoreModel = (*Step)(nil)

func (s *Step) Unmarshal(ctx context.Context, node *yaml.Node) error {
	s.RootNode = node

	return marshaller.UnmarshalStruct(ctx, node, s)
}
