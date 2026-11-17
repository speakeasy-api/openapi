// Package actions resolves the success and failure actions that apply to a step.
package actions

import "github.com/speakeasy-api/openapi/arazzo"

// Success returns the success actions that apply to the step, resolving references to components. Step actions are
// followed by the workflow's actions that the step doesn't override by name. Actions that can't be resolved are
// left out.
func Success(a *arazzo.Arazzo, w *arazzo.Workflow, s *arazzo.Step) []*arazzo.SuccessAction {
	resolved := []*arazzo.SuccessAction{}
	if a == nil || w == nil || s == nil {
		return resolved
	}
	names := map[string]bool{}
	for _, action := range s.OnSuccess {
		if r := resolveSuccess(a, action); r != nil {
			resolved = append(resolved, r)
			names[r.Name] = true
		}
	}
	for _, action := range w.SuccessActions {
		if r := resolveSuccess(a, action); r != nil && !names[r.Name] {
			resolved = append(resolved, r)
		}
	}
	return resolved
}

// Failure returns the failure actions that apply to the step, resolving references to components. Step actions are
// followed by the workflow's actions that the step doesn't override by name. Actions that can't be resolved are
// left out.
func Failure(a *arazzo.Arazzo, w *arazzo.Workflow, s *arazzo.Step) []*arazzo.FailureAction {
	resolved := []*arazzo.FailureAction{}
	if a == nil || w == nil || s == nil {
		return resolved
	}
	names := map[string]bool{}
	for _, action := range s.OnFailure {
		if r := resolveFailure(a, action); r != nil {
			resolved = append(resolved, r)
			names[r.Name] = true
		}
	}
	for _, action := range w.FailureActions {
		if r := resolveFailure(a, action); r != nil && !names[r.Name] {
			resolved = append(resolved, r)
		}
	}
	return resolved
}

// resolveSuccess returns the inline or referenced action, or nil if it can't be resolved.
func resolveSuccess(a *arazzo.Arazzo, action *arazzo.ReusableSuccessAction) *arazzo.SuccessAction {
	if action == nil || (action.IsReference() && a.Components == nil) {
		return nil
	}
	return action.Get(a.Components)
}

// resolveFailure returns the inline or referenced action, or nil if it can't be resolved.
func resolveFailure(a *arazzo.Arazzo, action *arazzo.ReusableFailureAction) *arazzo.FailureAction {
	if action == nil || (action.IsReference() && a.Components == nil) {
		return nil
	}
	return action.Get(a.Components)
}
