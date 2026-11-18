package asyncapi

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v3"
)

var (
	// operationTraitForbidden are Operation Object fields an Operation Trait must not supply.
	operationTraitForbidden = []string{"action", "channel", "messages", "traits"}
	// messageTraitForbidden are Message Object fields a Message Trait must not supply.
	messageTraitForbidden = []string{"payload", "traits"}
)

// effectiveView is an object with its traits applied, and where each of its top-level fields came from.
type effectiveView[T any] struct {
	object  *T
	sources map[string]int
}

func (v *effectiveView[T]) get() *T {
	if v == nil {
		return nil
	}
	return v.object
}

func (v *effectiveView[T]) source(field string) (int, bool) {
	if v == nil {
		return 0, false
	}
	index, ok := v.sources[field]
	return index, ok
}

// authoredSource reports a field written in an object without traits as set by the object itself.
func authoredSource(root *yaml.Node, field string) (int, bool) {
	root = yml.ResolveAlias(root)
	if root == nil || root.Kind != yaml.MappingNode {
		return 0, false
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == field && !isNull(root.Content[i+1]) {
			return -1, true
		}
	}
	return 0, false
}

// traitPatch is a trait's node and the document it came from, empty for the root document.
type traitPatch struct {
	node        *yaml.Node
	docLocation string
}

// mergeTraits applies traits to a target object using JSON Merge Patch (RFC 7386), with the target's own values
// winning:
//
//	effective = {}
//	for each trait in order: effective = MergePatch(effective, trait)
//	effective = MergePatch(effective, target without traits)
//
// Objects merge recursively, later traits win over earlier ones, the target wins at leaves, arrays are replaced and
// null deletes a value, including a target null deleting an inherited value. Fields a trait must not supply are
// ignored. The authored nodes are copied, keeping their positions, so the result shares nothing with the document
// and diagnostics on it point at where each value was written.
func mergeTraits(target *yaml.Node, targetDoc string, traits []traitPatch, forbidden []string) (*yaml.Node, map[string]int, map[*yaml.Node]string) {
	m := merger{nodeDocs: map[*yaml.Node]string{}}
	sources := map[string]int{}

	target = yml.ResolveAlias(target)
	effective := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if target != nil {
		effective.Line, effective.Column = target.Line, target.Column
	}

	for i, trait := range traits {
		patch := withoutKeys(trait.node, forbidden)
		if patch == nil {
			continue
		}
		recordSources(sources, patch, i)
		effective = m.patch(effective, patch, trait.docLocation)
	}

	if patch := withoutKeys(target, []string{"traits"}); patch != nil {
		recordSources(sources, patch, -1)
		effective = m.patch(effective, patch, targetDoc)
	}

	return effective, sources, m.nodeDocs
}

func recordSources(sources map[string]int, patch *yaml.Node, index int) {
	for i := 0; i+1 < len(patch.Content); i += 2 {
		key := patch.Content[i].Value
		if isNull(patch.Content[i+1]) {
			delete(sources, key)
		} else {
			sources[key] = index
		}
	}
}

// withoutKeys returns a shallow copy of a mapping without the provided keys, or nil if the node isn't a mapping.
func withoutKeys(node *yaml.Node, keys []string) *yaml.Node {
	node = yml.ResolveAlias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	out := *node
	out.Content = make([]*yaml.Node, 0, len(node.Content))
	for i := 0; i+1 < len(node.Content); i += 2 {
		if !slices.Contains(keys, node.Content[i].Value) {
			out.Content = append(out.Content, node.Content[i], node.Content[i+1])
		}
	}
	return &out
}

func isNull(node *yaml.Node) bool {
	node = yml.ResolveAlias(node)
	return node == nil || (node.Kind == yaml.ScalarNode && node.Tag == "!!null")
}

type merger struct {
	// nodeDocs records the document each copied node came from, empty for the root document.
	nodeDocs map[*yaml.Node]string
}

// patch applies a JSON Merge Patch to target, which is always a node owned by the merger.
func (m *merger) patch(target, patch *yaml.Node, doc string) *yaml.Node {
	patch = yml.ResolveAlias(patch)
	if patch == nil || patch.Kind != yaml.MappingNode {
		return m.copy(patch, doc)
	}

	// A merged mapping takes the position of the patch, the latest contributor, whose values win. The target is
	// always merged last, so diagnostics on a mapping both it and a trait supply point at the target's mapping.
	result := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Style: patch.Style, Line: patch.Line, Column: patch.Column}
	if target != nil && target.Kind == yaml.MappingNode {
		result.Content = append(result.Content, target.Content...)
	}
	m.nodeDocs[result] = doc

	for i := 0; i+1 < len(patch.Content); i += 2 {
		key, value := patch.Content[i], patch.Content[i+1]
		index := -1
		for j := 0; j+1 < len(result.Content); j += 2 {
			if result.Content[j].Value == key.Value {
				index = j
				break
			}
		}

		if isNull(value) {
			if index >= 0 {
				result.Content = slices.Delete(result.Content, index, index+2)
			}
			continue
		}

		if index >= 0 {
			result.Content[index+1] = m.patch(result.Content[index+1], value, doc)
		} else {
			result.Content = append(result.Content, m.copy(key, doc), m.patch(nil, value, doc))
		}
	}

	return result
}

// copy deep copies a node, resolving aliases and keeping positions.
func (m *merger) copy(node *yaml.Node, doc string) *yaml.Node {
	node = yml.ResolveAlias(node)
	if node == nil {
		return nil
	}
	c := *node
	c.Anchor = ""
	c.Content = make([]*yaml.Node, len(node.Content))
	for i, child := range node.Content {
		c.Content[i] = m.copy(child, doc)
	}
	m.nodeDocs[&c] = doc
	return &c
}

// documentOf returns the document a resolved reference points into, or empty for the root document.
func (rc *resolveCall) documentOf(resolvedRef string, base location) string {
	if resolvedRef == "" {
		if rc.isRoot(base) {
			return ""
		}
		return base.uri
	}
	uri, _, _ := strings.Cut(resolvedRef, "#")
	if uri == rc.root.uri || uri == "" {
		return ""
	}
	return uri
}

func (rc *resolveCall) applyOperationTraits(ctx context.Context, op *Operation, base location) {
	if len(op.Traits) == 0 {
		return
	}

	targetDoc := rc.documentOf("", base)
	patches := make([]traitPatch, 0, len(op.Traits))
	authored := diagnosticSet{}
	authored.add(op.Validate(ctx, rc.validationOpts...), targetDoc)
	sameDocument := true
	for _, trait := range op.Traits {
		obj := trait.GetObject()
		if obj == nil {
			patches = append(patches, traitPatch{})
			continue
		}
		doc := rc.documentOf(string(trait.GetResolvedReference()), base)
		sameDocument = sameDocument && doc == targetDoc
		patches = append(patches, traitPatch{node: obj.GetCore().GetRootNode(), docLocation: doc})
		authored.add(obj.Validate(ctx, rc.validationOpts...), doc)
	}

	node, sources, nodeDocs := mergeTraits(op.GetCore().GetRootNode(), targetDoc, patches, operationTraitForbidden)

	var effective Operation
	if _, err := marshaller.UnmarshalNode(ctx, "operation", node, &effective); err != nil {
		rc.addErrors([]error{validation.NewValidationError(validation.SeverityError, validation.RuleValidationInvalidSyntax, err, op.GetCore().GetRootNode())}, base)
		return
	}
	rc.inheritedErrors(effective.Validate(ctx, rc.validationOpts...), authored, nodeDocs, targetDoc, base)

	op.setEffective(&effectiveView[Operation]{object: &effective, sources: sources})
	rc.track(func() { op.setEffective(nil) })
	rc.resolveEffective(ctx, &effective, sameDocument, base)
}

func (rc *resolveCall) applyMessageTraits(ctx context.Context, message *Message, base location) {
	if len(message.Traits) == 0 {
		return
	}

	targetDoc := rc.documentOf("", base)
	patches := make([]traitPatch, 0, len(message.Traits))
	authored := diagnosticSet{}
	authored.add(message.Validate(ctx, rc.validationOpts...), targetDoc)
	sameDocument := true
	for _, trait := range message.Traits {
		obj := trait.GetObject()
		if obj == nil {
			patches = append(patches, traitPatch{})
			continue
		}
		doc := rc.documentOf(string(trait.GetResolvedReference()), base)
		sameDocument = sameDocument && doc == targetDoc
		patches = append(patches, traitPatch{node: obj.GetCore().GetRootNode(), docLocation: doc})
		authored.add(obj.Validate(ctx, rc.validationOpts...), doc)
	}

	node, sources, nodeDocs := mergeTraits(message.GetCore().GetRootNode(), targetDoc, patches, messageTraitForbidden)

	var effective Message
	if _, err := marshaller.UnmarshalNode(ctx, "message", node, &effective); err != nil {
		rc.addErrors([]error{validation.NewValidationError(validation.SeverityError, validation.RuleValidationInvalidSyntax, err, message.GetCore().GetRootNode())}, base)
		return
	}
	rc.inheritedErrors(effective.Validate(ctx, rc.validationOpts...), authored, nodeDocs, targetDoc, base)

	message.setEffective(&effectiveView[Message]{object: &effective, sources: sources})
	rc.track(func() { message.setEffective(nil) })
	rc.resolveEffective(ctx, &effective, sameDocument, base)
}

// resolveEffective resolves the references in an effective view. Only views built from a single document can be
// resolved, because the merged references no longer record which document each was written in.
func (rc *resolveCall) resolveEffective(ctx context.Context, effective any, sameDocument bool, base location) {
	if !sameDocument {
		return
	}
	rc.targets = append(rc.targets, walkTarget{items: walkObject(ctx, rc.doc, effective), base: base, effective: true})
}

// inheritedErrors records diagnostics that only appear once traits are applied, such as an invalid combination of
// a trait's and the target's values. Diagnostics already reported for the authored target or traits are skipped.
func (rc *resolveCall) inheritedErrors(errs []error, authored diagnosticSet, nodeDocs map[*yaml.Node]string, targetDoc string, base location) {
	for _, err := range errs {
		var vErr *validation.Error
		if !errors.As(err, &vErr) {
			continue
		}
		doc, ok := nodeDocs[vErr.Node]
		if !ok {
			doc = targetDoc
		}
		if authored.has(vErr, doc) {
			continue
		}
		if doc != "" && vErr.DocumentLocation == "" {
			vErr.DocumentLocation = doc
		}
		rc.addErrors([]error{err}, base)
	}
}

// diagnosticKey identifies a diagnostic by where it was found, including the document, and its rule.
type diagnosticKey struct {
	doc          string
	line, column int
	rule         string
}

type diagnosticSet map[diagnosticKey]bool

// add records diagnostics found in the provided document, empty for the root document.
func (s diagnosticSet) add(errs []error, doc string) {
	for _, err := range errs {
		var vErr *validation.Error
		if errors.As(err, &vErr) {
			s[diagnosticKey{doc: doc, line: vErr.GetLineNumber(), column: vErr.GetColumnNumber(), rule: vErr.Rule}] = true
		}
	}
}

func (s diagnosticSet) has(vErr *validation.Error, doc string) bool {
	return s[diagnosticKey{doc: doc, line: vErr.GetLineNumber(), column: vErr.GetColumnNumber(), rule: vErr.Rule}]
}
