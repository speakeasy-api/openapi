package rules

import (
	"context"
	"errors"
	"slices"
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

// hasText returns true if s is set to something other than whitespace.
func hasText(s *string) bool {
	return s != nil && strings.TrimSpace(*s) != ""
}

// withFix attaches a fix to a validation error.
func withFix(err error, fix validation.Fix) error {
	var vErr *validation.Error
	if errors.As(err, &vErr) {
		vErr.Fix = fix
	}
	return err
}

// proseKeys are the keys of documentation fields, which can mention runtime expressions without using them.
var proseKeys = map[string]bool{"description": true, "summary": true}

// visitScalars calls visit for every scalar node within node, following aliases and skipping documentation fields.
// Each node is visited once.
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
		case yaml.MappingNode:
			for i := 0; i+1 < len(node.Content); i += 2 {
				// Documentation can mention expressions without using them.
				if proseKeys[node.Content[i].Value] {
					continue
				}
				walk(node.Content[i])
				walk(node.Content[i+1])
			}
		default:
			for _, child := range node.Content {
				walk(child)
			}
		}
	}
	walk(node)
}

// reusableComponentNodes returns the parameter and action sections of the document's components, which can contain
// runtime expressions evaluated in the context of the workflow and step that reference them.
func reusableComponentNodes(a *arazzo.Arazzo) []*yaml.Node {
	componentsNode := mapValue(a.GetRootNode(), "components")
	nodes := []*yaml.Node{}
	for _, section := range []string{"parameters", "successActions", "failureActions"} {
		if node := mapValue(componentsNode, section); node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
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
	if node != nil && node.Kind == yaml.AliasNode {
		return mapKey(node.Alias, key)
	}
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
// contain dots, so `a.b.c` may refer to `a.b` followed by a nested property `c`. Names in except are not marked.
func markDottedName(used map[string]bool, name string, except ...string) {
	for {
		if !slices.Contains(except, name) {
			used[name] = true
		}
		i := strings.LastIndex(name, ".")
		if i < 0 {
			return
		}
		name = name[:i]
	}
}
