package rules

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"go.yaml.in/yaml/v3"
)

const RuleSemanticUnusedWorkflowInput = "semantic-unused-workflow-input"

// inputsExpressionRegex matches `$inputs` expressions, capturing the (possibly dotted) input name if there is one.
var inputsExpressionRegex = regexp.MustCompile(`\$inputs(?:\.([A-Za-z0-9_.\-]+))?`)

type UnusedWorkflowInputRule struct{}

func (r *UnusedWorkflowInputRule) ID() string       { return RuleSemanticUnusedWorkflowInput }
func (r *UnusedWorkflowInputRule) Category() string { return CategorySemantic }
func (r *UnusedWorkflowInputRule) Description() string {
	return "Workflow inputs should be used. An input is only available to a workflow through `$inputs` expressions, so an input that is never read either isn't needed, or a step is missing the parameter or payload value that should use it."
}
func (r *UnusedWorkflowInputRule) Summary() string {
	return "Workflow inputs should be read with an $inputs expression."
}
func (r *UnusedWorkflowInputRule) HowToFix() string {
	return "Remove the input, or use it with an `$inputs.<name>` expression in a step."
}
func (r *UnusedWorkflowInputRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/arazzo/linter/README.md#semantic-unused-workflow-input"
}
func (r *UnusedWorkflowInputRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *UnusedWorkflowInputRule) Versions() []string { return nil }

func (r *UnusedWorkflowInputRule) Run(_ context.Context, docInfo *linter.DocumentInfo[*arazzo.Arazzo], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	a := docInfo.Document
	severity := config.GetSeverity(r.DefaultSeverity())

	var errs []error
	for _, w := range a.Workflows {
		if w == nil || w.Inputs == nil {
			continue
		}

		schema, schemaNode, local := workflowInputsSchema(a, w)
		if schema == nil {
			continue
		}

		used, all := usedInputs(a, w)
		if all {
			continue
		}

		propertiesNode := mapValue(schemaNode, "properties")
		for name := range schema.GetProperties().Keys() {
			if used[name] {
				continue
			}
			err := fmt.Errorf("input `%s` of workflow `%s` is never used", name, w.WorkflowID)
			node := mapKey(propertiesNode, name)
			if !local || node == nil {
				// Inputs referenced from components are reported against the workflow's inputs.
				node = schemaNode
			}
			errs = append(errs, validation.NewValidationError(severity, RuleSemanticUnusedWorkflowInput, err, node))
		}
	}
	return errs
}

// workflowInputsSchema returns the workflow's inputs schema and its node, following a reference to a components
// input. local is false when the schema comes from components.
func workflowInputsSchema(a *arazzo.Arazzo, w *arazzo.Workflow) (*oas3.Schema, *yaml.Node, bool) {
	node := w.GetCore().Inputs.ValueNode
	if !w.Inputs.IsReference() {
		return w.Inputs.GetSchema(), node, true
	}

	ref := string(w.Inputs.GetReference())
	name, ok := strings.CutPrefix(ref, "#/components/inputs/")
	if !ok || a.Components == nil {
		return nil, node, false
	}
	input, ok := a.Components.Inputs.Get(name)
	if !ok || input == nil || input.IsReference() {
		return nil, node, false
	}
	return input.GetSchema(), node, false
}

// usedInputs returns the inputs read with `$inputs` expressions within the workflow. all is true if the whole of
// `$inputs` is used, so every input counts as used. Expressions in reusable components count too, as the components
// can be referenced from the workflow.
func usedInputs(a *arazzo.Arazzo, w *arazzo.Workflow) (map[string]bool, bool) {
	used := map[string]bool{}
	all := false

	visit := func(scalar *yaml.Node) {
		for _, m := range inputsExpressionRegex.FindAllStringSubmatch(scalar.Value, -1) {
			if m[1] == "" {
				all = true
				continue
			}
			markDottedName(used, m[1])
		}
	}

	if root := w.GetRootNode(); root != nil && root.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(root.Content); i += 2 {
			if root.Content[i].Value != "inputs" {
				visitScalars(root.Content[i+1], visit)
			}
		}
	}
	for _, node := range reusableComponentNodes(a) {
		visitScalars(node, visit)
	}

	return used, all
}
