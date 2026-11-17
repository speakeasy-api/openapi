// Package graph builds a graph of the workflows in an Arazzo document, showing how steps flow into each other and
// how workflows and steps depend on each other, and renders it as a Mermaid flowchart or a Graphviz DOT digraph.
package graph

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/internal/actions"
	"github.com/speakeasy-api/openapi/expression"
)

// Direction is the direction the graph is laid out in.
type Direction string

const (
	// DirectionTopBottom lays the graph out from top to bottom.
	DirectionTopBottom Direction = "TB"
	// DirectionLeftRight lays the graph out from left to right.
	DirectionLeftRight Direction = "LR"
)

// NodeKind is the kind of a node in the graph.
type NodeKind int

const (
	// NodeWorkflow is the start of a workflow in the document. Edges to a workflow point at it.
	NodeWorkflow NodeKind = iota
	// NodeStep is a step of a workflow in the document.
	NodeStep
	// NodeExternal is a workflow or step outside the graph, such as one in another Arazzo document referenced by a
	// sourceDescription, a workflow left out by Options.WorkflowID, or a reference that can't be resolved.
	NodeExternal
)

// Node is a node in the graph.
type Node struct {
	// ID is a unique identifier for the node that is safe to use in Mermaid and DOT.
	ID string
	// Kind is the kind of node.
	Kind NodeKind
	// Label is the workflowId, stepId, or reference of the node.
	Label string
	// Detail is the operation, operation path, workflow or channel a step targets.
	Detail string
}

// Workflow is a workflow in the document, with the node for its start and a node for each of its steps.
type Workflow struct {
	// ID is a unique identifier for the workflow's cluster that is safe to use in Mermaid and DOT.
	ID string
	// WorkflowID is the workflow's workflowId.
	WorkflowID string
	// Title is the workflow's summary, or its workflowId if it has no summary.
	Title string
	// Start is the node edges to the workflow point at, which leads to its first step.
	Start *Node
	// Steps are the nodes for the workflow's steps, in order.
	Steps []*Node
}

// EdgeKind is the kind of an edge in the graph.
type EdgeKind int

const (
	// EdgeStart goes from the start of a workflow to its first step.
	EdgeStart EdgeKind = iota
	// EdgeNext goes from a step to the next step, which runs when no success action ends the workflow or goes
	// elsewhere.
	EdgeNext
	// EdgeSuccessGoto is a success action going to a step or workflow.
	EdgeSuccessGoto
	// EdgeFailureGoto is a failure action going to a step or workflow.
	EdgeFailureGoto
	// EdgeRetry is a failure action retrying the step.
	EdgeRetry
	// EdgeDependsOn goes from a workflow or step to a workflow or step in its dependsOn.
	EdgeDependsOn
	// EdgeCall goes from a step to the workflow it runs.
	EdgeCall
)

// Edge is an edge in the graph.
type Edge struct {
	From  *Node
	To    *Node
	Kind  EdgeKind
	Label string
}

// Graph is a graph of the workflows in an Arazzo document.
type Graph struct {
	// Direction is the direction the graph is laid out in.
	Direction Direction
	// Workflows are the workflows in the graph, in document order.
	Workflows []*Workflow
	// External are the nodes for workflows and steps outside the graph, in the order they are first referenced.
	External []*Node
	// Edges are the edges in the graph.
	Edges []*Edge
}

// Options configures Build.
type Options struct {
	// WorkflowID limits the graph to a single workflow. Workflows and steps it references are shown as external nodes.
	WorkflowID string
	// Direction is the direction the graph is laid out in. Defaults to DirectionTopBottom.
	Direction Direction
}

// Build builds a graph of the workflows in the document.
//
// Steps run in order unless a success action ends the workflow or goes elsewhere. As actions are taken in order and
// the first that applies wins, an action without criteria always applies, so no later actions are shown, and the
// step only leads on to the next step if none of its success actions apply unconditionally. Workflow level actions
// apply to every step, unless the step has an action with the same name.
func Build(a *arazzo.Arazzo, opts Options) (*Graph, error) {
	if a == nil {
		return nil, errors.New("document is nil")
	}

	direction := opts.Direction
	switch direction {
	case "":
		direction = DirectionTopBottom
	case DirectionTopBottom, DirectionLeftRight:
	default:
		return nil, fmt.Errorf("unknown direction `%s`, expected %s or %s", direction, DirectionTopBottom, DirectionLeftRight)
	}

	b := &builder{
		a:         a,
		g:         &Graph{Direction: direction},
		workflows: map[string]*Workflow{},
		steps:     map[string]*Node{},
		external:  map[string]*Node{},
	}

	selected := []*arazzo.Workflow{}
	for _, w := range a.Workflows {
		if w != nil && (opts.WorkflowID == "" || w.WorkflowID == opts.WorkflowID) {
			selected = append(selected, w)
		}
	}
	if opts.WorkflowID != "" && len(selected) == 0 {
		return nil, fmt.Errorf("workflow `%s` not found", opts.WorkflowID)
	}

	clusters := make([]*Workflow, 0, len(selected))
	for i, w := range selected {
		clusters = append(clusters, b.addWorkflow(i, w))
	}
	for i, w := range selected {
		b.addEdges(w, clusters[i])
	}

	return b.g, nil
}

type builder struct {
	a *arazzo.Arazzo
	g *Graph
	// workflows are the workflows in the graph by workflowId.
	workflows map[string]*Workflow
	// steps are the step nodes in the graph by `<workflowId>.steps.<stepId>`.
	steps map[string]*Node
	// external are the external nodes by reference.
	external map[string]*Node
}

// addWorkflow adds the workflow and its steps to the graph. References to a workflowId or stepId that is used more
// than once resolve to the first.
func (b *builder) addWorkflow(index int, w *arazzo.Workflow) *Workflow {
	id := "w" + strconv.Itoa(index)
	title := w.WorkflowID
	if w.Summary != nil && strings.TrimSpace(*w.Summary) != "" {
		title = *w.Summary
	}

	gw := &Workflow{
		ID:         "wf" + strconv.Itoa(index),
		WorkflowID: w.WorkflowID,
		Title:      title,
		Start:      &Node{ID: id, Kind: NodeWorkflow, Label: w.WorkflowID},
	}
	for j, s := range w.Steps {
		if s == nil {
			continue
		}
		node := &Node{ID: id + "s" + strconv.Itoa(j), Kind: NodeStep, Label: s.StepID}
		// Steps are often named after their operation, which doesn't need repeating.
		if target := stepTarget(s); target != s.StepID {
			node.Detail = target
		}
		gw.Steps = append(gw.Steps, node)
		if key := stepKey(w.WorkflowID, s.StepID); b.steps[key] == nil {
			b.steps[key] = node
		}
	}

	b.g.Workflows = append(b.g.Workflows, gw)
	if b.workflows[w.WorkflowID] == nil {
		b.workflows[w.WorkflowID] = gw
	}
	return gw
}

func (b *builder) addEdges(w *arazzo.Workflow, gw *Workflow) {
	for _, dependsOn := range w.DependsOn {
		b.addEdge(gw.Start, b.workflowRef(string(dependsOn)), EdgeDependsOn, "dependsOn")
	}
	if len(gw.Steps) > 0 {
		b.addEdge(gw.Start, gw.Steps[0], EdgeStart, "")
	}

	steps := []*arazzo.Step{}
	for _, s := range w.Steps {
		if s != nil {
			steps = append(steps, s)
		}
	}

	for i, s := range steps {
		node := gw.Steps[i]

		if s.WorkflowID != nil {
			b.addEdge(node, b.workflowRef(string(*s.WorkflowID)), EdgeCall, "runs")
		}
		for _, dependsOn := range s.DependsOn {
			b.addEdge(node, b.stepRef(w, dependsOn), EdgeDependsOn, "dependsOn")
		}

		continues := true
		for _, action := range actions.Success(b.a, w, s) {
			if action.Type == arazzo.SuccessActionTypeGoto {
				b.addEdge(node, b.actionTarget(w, action.StepID, action.WorkflowID), EdgeSuccessGoto, action.Name)
			}
			if len(action.Criteria) == 0 {
				continues = false
				break
			}
		}
		for _, action := range actions.Failure(b.a, w, s) {
			switch action.Type {
			case arazzo.FailureActionTypeGoto:
				b.addEdge(node, b.actionTarget(w, action.StepID, action.WorkflowID), EdgeFailureGoto, action.Name)
			case arazzo.FailureActionTypeRetry:
				b.addEdge(node, node, EdgeRetry, action.Name)
			case arazzo.FailureActionTypeEnd:
			}
			if len(action.Criteria) == 0 {
				break
			}
		}

		if continues && i+1 < len(gw.Steps) {
			b.addEdge(node, gw.Steps[i+1], EdgeNext, "")
		}
	}
}

func (b *builder) addEdge(from, to *Node, kind EdgeKind, label string) {
	b.g.Edges = append(b.g.Edges, &Edge{From: from, To: to, Kind: kind, Label: label})
}

// actionTarget returns the node a goto action goes to.
func (b *builder) actionTarget(w *arazzo.Workflow, stepID *string, workflowID *expression.Expression) *Node {
	if stepID != nil {
		return b.stepRef(w, *stepID)
	}
	if workflowID != nil {
		return b.workflowRef(string(*workflowID))
	}
	return b.externalNode("?", "unknown target")
}

// workflowRef returns the node for a workflowId or `$sourceDescriptions.<name>.<workflowId>` reference.
func (b *builder) workflowRef(ref string) *Node {
	if gw, ok := b.workflows[ref]; ok {
		return gw.Start
	}
	return b.externalNode(ref, strings.TrimPrefix(ref, "$sourceDescriptions."))
}

// stepRef returns the node for a stepId in the workflow, or a `$workflows.<workflowId>.steps.<stepId>` or
// `$sourceDescriptions.<name>.<workflowId>.steps.<stepId>` reference.
func (b *builder) stepRef(w *arazzo.Workflow, ref string) *Node {
	key := ""
	switch {
	case strings.HasPrefix(ref, "$workflows."):
		key = strings.TrimPrefix(ref, "$workflows.")
	case !strings.HasPrefix(ref, "$"):
		key = stepKey(w.WorkflowID, ref)
	}
	if node, ok := b.steps[key]; ok && key != "" {
		return node
	}

	label := strings.TrimPrefix(strings.TrimPrefix(ref, "$sourceDescriptions."), "$workflows.")
	if key != "" {
		label = key
	}
	return b.externalNode(ref, label)
}

// externalNode returns the external node for the reference and label, creating it the first time they are seen. The
// label is part of the key as a bare stepId resolves to a different step in each workflow.
func (b *builder) externalNode(ref, label string) *Node {
	key := ref + "\x00" + label
	if node, ok := b.external[key]; ok {
		return node
	}
	node := &Node{ID: "x" + strconv.Itoa(len(b.g.External)), Kind: NodeExternal, Label: label}
	b.external[key] = node
	b.g.External = append(b.g.External, node)
	return node
}

func stepKey(workflowID, stepID string) string {
	return workflowID + ".steps." + stepID
}

// stepTarget returns the operation, operation path, workflow or channel the step targets.
func stepTarget(s *arazzo.Step) string {
	switch {
	case s.OperationID != nil:
		return string(*s.OperationID)
	case s.OperationPath != nil:
		return string(*s.OperationPath)
	case s.WorkflowID != nil:
		return "workflow " + string(*s.WorkflowID)
	case s.ChannelPath != nil:
		return string(*s.ChannelPath)
	default:
		return ""
	}
}
