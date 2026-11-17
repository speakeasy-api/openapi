package rules

import (
	"context"
	"errors"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/internal/specversion"
	"github.com/speakeasy-api/openapi/validation"
	"go.yaml.in/yaml/v3"
)

type sourceDocumentsKey struct{}

// ContextWithSourceDocuments returns a context carrying the documents loaded for the Arazzo document's
// sourceDescriptions (see arazzo.LoadSourceDescriptions). Rules use them to improve their findings and fixes, for
// example to suggest a sourceDescription's type from the document it references.
func ContextWithSourceDocuments(ctx context.Context, sources *arazzo.SourceDocuments) context.Context {
	return context.WithValue(ctx, sourceDocumentsKey{}, sources)
}

// sourceDocumentsFromContext returns the source documents carried by the context, or nil.
func sourceDocumentsFromContext(ctx context.Context) *arazzo.SourceDocuments {
	sources, _ := ctx.Value(sourceDocumentsKey{}).(*arazzo.SourceDocuments)
	return sources
}

// isAtLeast11 returns true if the document declares Arazzo 1.1 or later.
func isAtLeast11(a *arazzo.Arazzo) bool {
	v, err := specversion.Parse(a.Arazzo)
	return err == nil && v != nil && (v.Major > 1 || (v.Major == 1 && v.Minor >= 1))
}

// withFix attaches a fix to a validation error.
func withFix(err error, fix validation.Fix) error {
	var vErr *validation.Error
	if errors.As(err, &vErr) {
		vErr.Fix = fix
	}
	return err
}

// visitScalars calls visit for every scalar node within node, following aliases. Each node is visited once.
func visitScalars(node *yaml.Node, visit func(node *yaml.Node)) {
	seen := map[*yaml.Node]bool{}
	var walk func(node *yaml.Node)
	walk = func(node *yaml.Node) {
		if node == nil || seen[node] {
			return
		}
		seen[node] = true

		switch node.Kind {
		case yaml.ScalarNode:
			visit(node)
		case yaml.AliasNode:
			walk(node.Alias)
		default:
			for _, child := range node.Content {
				walk(child)
			}
		}
	}
	walk(node)
}

// mapValue returns the value node for key within a mapping node, or nil.
func mapValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode {
		return mapValue(node.Alias, key)
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// mapKey returns the key node for key within a mapping node, or nil.
func mapKey(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i]
		}
	}
	return nil
}

// markDottedName marks name, and each of its dotted prefixes, as used. Names of inputs, outputs and components may
// contain dots, so `a.b.c` may refer to `a.b` followed by a nested property `c`.
func markDottedName(used map[string]bool, name string) {
	for {
		used[name] = true
		i := strings.LastIndex(name, ".")
		if i < 0 {
			return
		}
		name = name[:i]
	}
}

// successActions returns the success actions that apply to the step, resolving references to components. Step
// actions are followed by the workflow's actions that the step doesn't override by name.
func successActions(a *arazzo.Arazzo, w *arazzo.Workflow, s *arazzo.Step) []*arazzo.SuccessAction {
	actions := []*arazzo.SuccessAction{}
	if s == nil {
		return actions
	}
	names := map[string]bool{}
	for _, action := range s.OnSuccess {
		if resolved := resolveSuccessAction(a, action); resolved != nil {
			actions = append(actions, resolved)
			names[resolved.Name] = true
		}
	}
	for _, action := range w.SuccessActions {
		if resolved := resolveSuccessAction(a, action); resolved != nil && !names[resolved.Name] {
			actions = append(actions, resolved)
		}
	}
	return actions
}

func resolveSuccessAction(a *arazzo.Arazzo, action *arazzo.ReusableSuccessAction) *arazzo.SuccessAction {
	if action == nil {
		return nil
	}
	return action.Get(a.Components)
}

func resolveFailureAction(a *arazzo.Arazzo, action *arazzo.ReusableFailureAction) *arazzo.FailureAction {
	if action == nil {
		return nil
	}
	return action.Get(a.Components)
}

// failureActions returns the failure actions that apply to the step, resolving references to components. Step
// actions are followed by the workflow's actions that the step doesn't override by name.
func failureActions(a *arazzo.Arazzo, w *arazzo.Workflow, s *arazzo.Step) []*arazzo.FailureAction {
	actions := []*arazzo.FailureAction{}
	if s == nil {
		return actions
	}
	names := map[string]bool{}
	for _, action := range s.OnFailure {
		if resolved := resolveFailureAction(a, action); resolved != nil {
			actions = append(actions, resolved)
			names[resolved.Name] = true
		}
	}
	for _, action := range w.FailureActions {
		if resolved := resolveFailureAction(a, action); resolved != nil && !names[resolved.Name] {
			actions = append(actions, resolved)
		}
	}
	return actions
}
