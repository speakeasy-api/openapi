package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
)

type FailureAction struct {
	marshaller.CoreModel
	Name       marshaller.Node[string]       `key:"name"`
	Type       marshaller.Node[string]       `key:"type"`
	WorkflowID marshaller.Node[*string]      `key:"workflowId"`
	StepID     marshaller.Node[*string]      `key:"stepId"`
	RetryAfter marshaller.Node[*float64]     `key:"retryAfter"`
	RetryLimit marshaller.Node[*int]         `key:"retryLimit"`
	Criteria   marshaller.Node[[]*Criterion] `key:"criteria"`
	Extensions core.Extensions               `key:"extensions"`
}
