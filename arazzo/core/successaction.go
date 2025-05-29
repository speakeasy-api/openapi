package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type SuccessAction struct {
	Name       marshaller.Node[string]       `key:"name"`
	Type       marshaller.Node[string]       `key:"type"`
	WorkflowID marshaller.Node[*Expression]  `key:"workflowId"`
	StepID     marshaller.Node[*string]      `key:"stepId"`
	Criteria   marshaller.Node[[]*Criterion] `key:"criteria"`
	Extensions core.Extensions               `key:"extensions"`

	RootNode *yaml.Node
}
