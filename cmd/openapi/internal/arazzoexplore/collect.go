// Package arazzoexplore provides an interactive terminal UI for browsing the workflows of an Arazzo document.
package arazzoexplore

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"go.yaml.in/yaml/v3"
)

// ItemKind is the kind of row shown in the explorer.
type ItemKind int

const (
	// KindWorkflow is a workflow row.
	KindWorkflow ItemKind = iota
	// KindStep is a step row, shown under its workflow.
	KindStep
)

// Item is a row of the explorer: a workflow, or one of its steps.
type Item struct {
	Kind ItemKind
	// Label is the workflowId or stepId.
	Label string
	// Badge describes what a step targets, such as `OP` for an operationId. Empty for workflows.
	Badge string
	// Summary is shown after the label: a workflow's summary, or what a step targets.
	Summary string
	// Details are the lines shown when the row is expanded.
	Details []string
}

// Badges for the kinds of step targets.
const (
	BadgeOperation     = "OP"
	BadgeOperationPath = "PATH"
	BadgeWorkflow      = "FLOW"
	BadgeSend          = "SEND"
	BadgeReceive       = "RECV"
	BadgeChannel       = "CHAN"
	BadgeNone          = "?"
)

// Collect returns the rows for a document's workflows: each workflow followed by its steps.
func Collect(a *arazzo.Arazzo) []Item {
	if a == nil {
		return nil
	}

	items := []Item{}
	for _, w := range a.Workflows {
		if w == nil {
			continue
		}
		items = append(items, workflowItem(a, w))
		for _, s := range w.Steps {
			if s != nil {
				items = append(items, stepItem(a, w, s))
			}
		}
	}
	return items
}

func workflowItem(a *arazzo.Arazzo, w *arazzo.Workflow) Item {
	d := &details{}
	d.field("Summary", deref(w.Summary))
	d.field("Description", deref(w.Description))
	d.list("Depends on", expressions(w.DependsOn))
	d.list("Inputs", inputs(w.Inputs))
	d.list("Parameters", parameters(a, w.Parameters))
	d.list("Success actions", successActions(a, w.SuccessActions, ""))
	d.list("Failure actions", failureActions(a, w.FailureActions, ""))
	d.list("Outputs", outputs(w.Outputs))

	steps := 0
	for _, s := range w.Steps {
		if s != nil {
			steps++
		}
	}
	d.field("Steps", strconv.Itoa(steps))

	return Item{Kind: KindWorkflow, Label: oneLine(w.WorkflowID), Summary: oneLine(deref(w.Summary)), Details: d.lines}
}

func stepItem(a *arazzo.Arazzo, w *arazzo.Workflow, s *arazzo.Step) Item {
	badge, target, targetLabel := stepTarget(s)

	d := &details{}
	d.field("Description", deref(s.Description))
	d.field(targetLabel, target)
	d.field("Correlation ID", value(s.CorrelationID))
	if s.Timeout != nil {
		d.field("Timeout", strconv.Itoa(*s.Timeout)+"ms")
	}
	d.list("Depends on", s.DependsOn)
	d.list("Parameters", parameters(a, s.Parameters))
	d.list("Request body", requestBody(s.RequestBody))
	d.list("Success criteria", criteria(s.SuccessCriteria))

	stepSuccess := successActions(a, s.OnSuccess, "")
	d.list("On success", append(stepSuccess, successActions(a, inheritedSuccess(a, w, s), " (from workflow)")...))
	stepFailure := failureActions(a, s.OnFailure, "")
	d.list("On failure", append(stepFailure, failureActions(a, inheritedFailure(a, w, s), " (from workflow)")...))

	d.list("Outputs", outputs(s.Outputs))

	return Item{Kind: KindStep, Label: oneLine(s.StepID), Badge: badge, Summary: oneLine(target), Details: d.lines}
}

// oneLine joins the lines of a value with spaces, for values shown in a row.
func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

// stepTarget returns the badge, target and label of the target of a step.
func stepTarget(s *arazzo.Step) (string, string, string) {
	switch {
	case s.OperationID != nil:
		return BadgeOperation, s.OperationID.String(), "Operation"
	case s.OperationPath != nil:
		return BadgeOperationPath, s.OperationPath.String(), "Operation path"
	case s.WorkflowID != nil:
		return BadgeWorkflow, s.WorkflowID.String(), "Workflow"
	case s.ChannelPath != nil:
		badge := BadgeChannel
		label := "Channel"
		if s.Action != nil {
			label += " (" + string(*s.Action) + ")"
			switch *s.Action {
			case arazzo.StepActionSend:
				badge = BadgeSend
			case arazzo.StepActionReceive:
				badge = BadgeReceive
			}
		}
		return badge, s.ChannelPath.String(), label
	default:
		return BadgeNone, "", "Target"
	}
}

// details builds the detail lines of a row. Every detail line is a single terminal line, so values with line breaks
// (such as CommonMark descriptions) continue on indented lines.
type details struct {
	lines []string
}

// field adds a `Name: value` line, skipping empty values.
func (d *details) field(name, value string) {
	if value != "" {
		d.add(name+": ", "  ", value)
	}
}

// list adds a `Name:` line followed by an indented line per entry, skipping empty lists.
func (d *details) list(name string, entries []string) {
	if len(entries) == 0 {
		return
	}
	d.lines = append(d.lines, name+":")
	for _, e := range entries {
		d.add("  - ", "    ", e)
	}
}

// add adds a value prefixed by prefix, with any further lines of the value prefixed by indent.
func (d *details) add(prefix, indent, value string) {
	for i, line := range strings.Split(strings.TrimRight(strings.ReplaceAll(value, "\r\n", "\n"), "\n"), "\n") {
		if i == 0 {
			d.lines = append(d.lines, prefix+line)
		} else {
			d.lines = append(d.lines, indent+line)
		}
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func expressions(exprs []expression.Expression) []string {
	out := make([]string, 0, len(exprs))
	for _, e := range exprs {
		out = append(out, e.String())
	}
	return out
}

// inputs describes the top level properties of a workflow's inputs schema.
func inputs(schema *oas3.JSONSchema[oas3.Referenceable]) []string {
	if schema == nil {
		return nil
	}
	if schema.IsReference() {
		return []string{string(schema.GetReference())}
	}
	s := schema.GetSchema()
	if s == nil {
		return nil
	}

	required := map[string]bool{}
	for _, name := range s.Required {
		required[name] = true
	}

	out := []string{}
	for name, property := range s.GetProperties().All() {
		var attrs []string
		var description string
		switch {
		case property.IsReference():
			attrs = append(attrs, string(property.GetReference()))
		case property.GetSchema() != nil:
			ps := property.GetSchema()
			for _, t := range ps.GetType() {
				attrs = append(attrs, string(t))
			}
			description = ps.GetDescription()
		}
		if required[name] {
			attrs = append(attrs, "required")
		}

		entry := name
		if len(attrs) > 0 {
			entry += " (" + strings.Join(attrs, ", ") + ")"
		}
		if description != "" {
			entry += ": " + description
		}
		out = append(out, entry)
	}
	return out
}

// parameters describes parameters, resolving references to components.
func parameters(a *arazzo.Arazzo, params []*arazzo.ReusableParameter) []string {
	out := []string{}
	for _, rp := range params {
		if rp == nil {
			continue
		}

		p := resolve[arazzo.Parameter](a, rp)
		if p == nil {
			out = append(out, unresolved(rp.Reference))
			continue
		}

		entry := p.Name
		if p.In != nil {
			entry += " (" + string(*p.In) + ")"
		}
		v := p.Value
		if rp.Value != nil {
			// A value alongside a reference overrides the component's value.
			v = rp.Value
		}
		entry += " = " + value(v)
		if rp.IsReference() {
			entry += " (from " + rp.Reference.String() + ")"
		}
		out = append(out, entry)
	}
	return out
}

// unresolved describes a reusable parameter or action whose object isn't available.
func unresolved(reference *expression.Expression) string {
	if reference == nil {
		return "(empty)"
	}
	return reference.String() + " (unresolved)"
}

// reusable is a parameter or action that is either declared inline or references a component.
type reusable[T any] interface {
	IsReference() bool
	Get(components *arazzo.Components) *T
}

// resolve returns the object of a reusable parameter or action, or nil if it's a reference that can't be resolved.
func resolve[T any](a *arazzo.Arazzo, r reusable[T]) *T {
	if r.IsReference() && a.Components == nil {
		return nil
	}
	return r.Get(a.Components)
}

func requestBody(rb *arazzo.RequestBody) []string {
	if rb == nil {
		return nil
	}
	out := []string{}
	if rb.ContentType != nil {
		out = append(out, "content type: "+*rb.ContentType)
	}
	if rb.Payload != nil {
		out = append(out, "payload: "+value(rb.Payload))
	}
	for _, r := range rb.Replacements {
		if r != nil {
			out = append(out, "replace "+r.Target.String()+" = "+value(r.Value))
		}
	}
	return out
}

// criteria describes criteria, such as `$statusCode == 200` or `$[0] (jsonpath rfc9535, context: $response.body)`.
func criteria[C criterion.Criterion | *criterion.Criterion](cs []C) []string {
	out := []string{}
	for _, c := range cs {
		var cr *criterion.Criterion
		switch v := any(c).(type) {
		case criterion.Criterion:
			cr = &v
		case *criterion.Criterion:
			cr = v
		}
		if cr == nil {
			continue
		}
		out = append(out, describeCriterion(cr))
	}
	return out
}

func describeCriterion(c *criterion.Criterion) string {
	var attrs []string
	if typ := c.Type.GetType(); typ != criterion.CriterionTypeSimple || c.Type.IsTypeProvided() {
		t := string(typ)
		if v := c.Type.GetVersion(); v != criterion.CriterionTypeVersionNone {
			t += " " + string(v)
		}
		attrs = append(attrs, t)
	}
	if c.Context != nil {
		attrs = append(attrs, "context: "+c.Context.String())
	}

	if len(attrs) == 0 {
		return c.Condition
	}
	return c.Condition + " (" + strings.Join(attrs, ", ") + ")"
}

func successActions(a *arazzo.Arazzo, actions []*arazzo.ReusableSuccessAction, suffix string) []string {
	out := []string{}
	for _, ra := range actions {
		if ra == nil {
			continue
		}
		action := resolve[arazzo.SuccessAction](a, ra)
		if action == nil {
			out = append(out, unresolved(ra.Reference)+suffix)
			continue
		}

		var what string
		switch action.Type {
		case arazzo.SuccessActionTypeEnd:
			what = "end the workflow"
		case arazzo.SuccessActionTypeGoto:
			what = "go to " + gotoTarget(action.StepID, action.WorkflowID)
		default:
			what = string(action.Type)
		}
		out = append(out, describeAction(action.Name, what, action.Criteria, ra.Reference)+suffix)
	}
	return out
}

func failureActions(a *arazzo.Arazzo, actions []*arazzo.ReusableFailureAction, suffix string) []string {
	out := []string{}
	for _, ra := range actions {
		if ra == nil {
			continue
		}
		action := resolve[arazzo.FailureAction](a, ra)
		if action == nil {
			out = append(out, unresolved(ra.Reference)+suffix)
			continue
		}

		var what string
		switch action.Type {
		case arazzo.FailureActionTypeEnd:
			what = "end the workflow"
		case arazzo.FailureActionTypeGoto:
			what = "go to " + gotoTarget(action.StepID, action.WorkflowID)
		case arazzo.FailureActionTypeRetry:
			what = "retry"
			if action.RetryAfter != nil {
				what += " after " + strconv.FormatFloat(*action.RetryAfter, 'f', -1, 64) + "s"
			}
			if action.RetryLimit != nil {
				what += " up to " + strconv.Itoa(*action.RetryLimit) + " times"
			}
			if action.StepID != nil || action.WorkflowID != nil {
				what += " via " + gotoTarget(action.StepID, action.WorkflowID)
			}
		default:
			what = string(action.Type)
		}
		out = append(out, describeAction(action.Name, what, action.Criteria, ra.Reference)+suffix)
	}
	return out
}

func gotoTarget(stepID *string, workflowID *expression.Expression) string {
	switch {
	case stepID != nil:
		return "step " + *stepID
	case workflowID != nil:
		return "workflow " + workflowID.String()
	default:
		return "?"
	}
}

// describeAction describes an action, such as `retryBusy: retry after 1s when $statusCode == 503`.
func describeAction(name, what string, cs []criterion.Criterion, reference *expression.Expression) string {
	out := name + ": " + what
	if conditions := criteria(cs); len(conditions) > 0 {
		out += " when " + strings.Join(conditions, " and ")
	}
	if reference != nil {
		out += " (from " + reference.String() + ")"
	}
	return out
}

// inheritedSuccess returns the workflow's success actions that apply to the step, which are those the step doesn't
// override with an action of the same name.
func inheritedSuccess(a *arazzo.Arazzo, w *arazzo.Workflow, s *arazzo.Step) []*arazzo.ReusableSuccessAction {
	names := map[string]bool{}
	for _, ra := range s.OnSuccess {
		if ra == nil {
			continue
		}
		if action := resolve[arazzo.SuccessAction](a, ra); action != nil {
			names[action.Name] = true
		}
	}
	out := []*arazzo.ReusableSuccessAction{}
	for _, ra := range w.SuccessActions {
		if ra == nil {
			continue
		}
		if action := resolve[arazzo.SuccessAction](a, ra); action == nil || !names[action.Name] {
			out = append(out, ra)
		}
	}
	return out
}

// inheritedFailure returns the workflow's failure actions that apply to the step, which are those the step doesn't
// override with an action of the same name.
func inheritedFailure(a *arazzo.Arazzo, w *arazzo.Workflow, s *arazzo.Step) []*arazzo.ReusableFailureAction {
	names := map[string]bool{}
	for _, ra := range s.OnFailure {
		if ra == nil {
			continue
		}
		if action := resolve[arazzo.FailureAction](a, ra); action != nil {
			names[action.Name] = true
		}
	}
	out := []*arazzo.ReusableFailureAction{}
	for _, ra := range w.FailureActions {
		if ra == nil {
			continue
		}
		if action := resolve[arazzo.FailureAction](a, ra); action == nil || !names[action.Name] {
			out = append(out, ra)
		}
	}
	return out
}

func outputs(o arazzo.Outputs) []string {
	out := []string{}
	for name, v := range o.All() {
		switch {
		case v == nil:
			out = append(out, name)
		case v.GetSelector() != nil:
			sel := v.GetSelector()
			out = append(out, fmt.Sprintf("%s: %s from %s (%s)", name, sel.Selector, sel.Context, sel.Type.GetType()))
		case v.GetExpression() != nil:
			out = append(out, name+": "+v.GetExpression().String())
		default:
			// An output without a value, which validation reports.
			out = append(out, name)
		}
	}
	return out
}

// value renders a YAML value on a single line, using flow style for mappings and sequences.
func value(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	if node.Kind == yaml.ScalarNode {
		return node.Value
	}

	flow := *node
	flow.Style |= yaml.FlowStyle
	data, err := yaml.Marshal(&flow)
	if err != nil {
		return "<" + err.Error() + ">"
	}
	return strings.TrimSpace(string(data))
}
