package core

import (
	"context"

	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type SuccessAction struct {
	Name       marshaller.Node[string]      `key:"name"`
	Type       marshaller.Node[string]      `key:"type"`
	WorkflowID marshaller.Node[*Expression] `key:"workflowId"`
	StepID     marshaller.Node[*string]     `key:"stepId"`
	Criteria   marshaller.Node[[]Criterion] `key:"criteria"`
	Extensions Extensions                   `key:"extensions"`

	RootNode *yaml.Node
}

var _ CoreModel = (*SuccessAction)(nil)

func (s *SuccessAction) Unmarshal(ctx context.Context, node *yaml.Node) error {
	s.RootNode = node

	return marshaller.UnmarshalStruct(ctx, node, s)
}
