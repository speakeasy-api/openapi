package rules

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleSemanticDependencyCycle = "semantic-dependency-cycle"

var localStepReferenceRegex = regexp.MustCompile(`^\$workflows\.(.+)\.steps\.([^.]+)$`)

type DependencyCycleRule struct{}

func (r *DependencyCycleRule) ID() string       { return RuleSemanticDependencyCycle }
func (r *DependencyCycleRule) Category() string { return CategorySemantic }
func (r *DependencyCycleRule) Description() string {
	return "Workflows and steps must not depend on each other in a cycle. A workflow or step only runs once everything in its dependsOn has completed, so none of the workflows or steps in a cycle can ever run."
}
func (r *DependencyCycleRule) Summary() string {
	return "Workflow and step dependsOn entries must not form a cycle."
}
func (r *DependencyCycleRule) HowToFix() string {
	return "Remove one of the dependsOn entries in the cycle."
}
func (r *DependencyCycleRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/arazzo/linter/README.md#semantic-dependency-cycle"
}
func (r *DependencyCycleRule) DefaultSeverity() validation.Severity { return validation.SeverityError }
func (r *DependencyCycleRule) Versions() []string                   { return nil }

// dependencyEdge is a dependsOn entry of a workflow or step that references another workflow or step in the
// document. Implied edges, with no report, are orderings implied by a workflow's dependsOn, and are only reported as
// part of a cycle that also contains a dependsOn entry.
type dependencyEdge struct {
	to     string
	report func(cycle string) error
}

func (r *DependencyCycleRule) Run(_ context.Context, docInfo *linter.DocumentInfo[*arazzo.Arazzo], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	a := docInfo.Document
	severity := config.GetSeverity(r.DefaultSeverity())

	workflowNodes := []string{}
	workflowEdges := map[string][]dependencyEdge{}
	stepNodes := []string{}
	stepEdges := map[string][]dependencyEdge{}

	for _, w := range a.Workflows {
		if w == nil {
			continue
		}
		workflowNodes = append(workflowNodes, w.WorkflowID)

		wCore := w.GetCore()
		for i, dependsOn := range w.DependsOn {
			if a.Workflows.Find(string(dependsOn)) == nil {
				continue
			}
			workflowEdges[w.WorkflowID] = append(workflowEdges[w.WorkflowID], dependencyEdge{
				to: string(dependsOn),
				report: func(cycle string) error {
					return validation.NewSliceError(severity, RuleSemanticDependencyCycle, fmt.Errorf("workflow.dependsOn creates a cycle: %s", cycle), wCore, wCore.DependsOn, i)
				},
			})
		}

		for _, s := range w.Steps {
			if s == nil {
				continue
			}
			from := w.WorkflowID + ".steps." + s.StepID
			stepNodes = append(stepNodes, from)

			sCore := s.GetCore()
			for i, dependsOn := range s.DependsOn {
				to := localStepDependency(a, w, dependsOn)
				// Steps depending on themselves are reported by validation.
				if to == "" || to == from {
					continue
				}
				stepEdges[from] = append(stepEdges[from], dependencyEdge{
					to: to,
					report: func(cycle string) error {
						return validation.NewSliceError(severity, RuleSemanticDependencyCycle, fmt.Errorf("step.dependsOn creates a cycle: %s", cycle), sCore, sCore.DependsOn, i)
					},
				})
			}
		}
	}

	errs := findDependencyCycles(workflowNodes, workflowEdges)

	// The steps of a workflow only run once the workflows it depends on, and so all of their steps, have completed,
	// which can form a cycle with step dependsOn entries. Workflow cycles are reported on their own, so these
	// orderings are only added when the workflows don't already form a cycle.
	if len(errs) == 0 {
		addImpliedStepEdges(a, stepEdges)
	}

	return append(errs, findDependencyCycles(stepNodes, stepEdges)...)
}

// addImpliedStepEdges adds an edge from each step to each step of the workflows its workflow depends on.
func addImpliedStepEdges(a *arazzo.Arazzo, stepEdges map[string][]dependencyEdge) {
	for _, w := range a.Workflows {
		if w == nil {
			continue
		}
		for _, dependsOn := range w.DependsOn {
			dependency := a.Workflows.Find(string(dependsOn))
			if dependency == nil {
				continue
			}
			for _, s := range w.Steps {
				if s == nil {
					continue
				}
				from := w.WorkflowID + ".steps." + s.StepID
				for _, ds := range dependency.Steps {
					if ds != nil {
						stepEdges[from] = append(stepEdges[from], dependencyEdge{to: dependency.WorkflowID + ".steps." + ds.StepID})
					}
				}
			}
		}
	}
}

// localStepDependency returns the `<workflowId>.steps.<stepId>` key of a step dependsOn entry that references a step
// in this document, or an empty string.
func localStepDependency(a *arazzo.Arazzo, w *arazzo.Workflow, dependsOn string) string {
	workflowID, stepID := w.WorkflowID, dependsOn
	if matches := localStepReferenceRegex.FindStringSubmatch(dependsOn); matches != nil {
		workflowID, stepID = matches[1], matches[2]
	} else if strings.HasPrefix(dependsOn, "$") {
		return ""
	}

	target := a.Workflows.Find(workflowID)
	if target == nil || target.Steps.Find(stepID) == nil {
		return ""
	}
	return workflowID + ".steps." + stepID
}

// findDependencyCycles reports each cycle in the dependency graph once, against the dependsOn entry that closes it,
// or the last dependsOn entry in the cycle when it's closed by an implied edge. Cycles made only of implied edges
// aren't reported.
func findDependencyCycles(nodes []string, edges map[string][]dependencyEdge) []error {
	const (
		unvisited = iota
		visiting
		visited
	)

	var errs []error
	state := map[string]int{}
	path := []string{}
	pathEdges := []dependencyEdge{}

	var visit func(node string)
	visit = func(node string) {
		state[node] = visiting
		path = append(path, node)

		for _, edge := range edges[node] {
			switch state[edge.to] {
			case unvisited:
				pathEdges = append(pathEdges, edge)
				visit(edge.to)
				pathEdges = pathEdges[:len(pathEdges)-1]
			case visiting:
				start := 0
				for i, n := range path {
					if n == edge.to {
						start = i
						break
					}
				}
				cycle := append(append([]string{}, path[start:]...), edge.to)
				cycleEdges := append(append([]dependencyEdge{}, pathEdges[start:]...), edge)
				for i := len(cycleEdges) - 1; i >= 0; i-- {
					if cycleEdges[i].report != nil {
						errs = append(errs, cycleEdges[i].report(strings.Join(cycle, " -> ")))
						break
					}
				}
			}
		}

		path = path[:len(path)-1]
		state[node] = visited
	}

	for _, node := range nodes {
		if state[node] == unvisited {
			visit(node)
		}
	}

	return errs
}
