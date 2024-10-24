package core

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type FailureAction struct {
	Name       marshaller.Node[string]      `key:"name"`
	Type       marshaller.Node[string]      `key:"type"`
	WorkflowID marshaller.Node[*Expression] `key:"workflowId"`
	StepID     marshaller.Node[*string]     `key:"stepId"`
	RetryAfter marshaller.Node[*float64]    `key:"retryAfter"`
	RetryLimit marshaller.Node[*int]        `key:"retryLimit"`
	Criteria   marshaller.Node[[]Criterion] `key:"criteria"`
	Extensions core.Extensions              `key:"extensions"`

	RootNode *yaml.Node
}

var _ CoreModel = (*FailureAction)(nil)

func (f *FailureAction) Unmarshal(ctx context.Context, node *yaml.Node) error {
	f.RootNode = node

	return marshaller.UnmarshalStruct(ctx, node, f)
}
