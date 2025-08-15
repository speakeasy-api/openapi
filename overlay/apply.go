package overlay

import (
	"fmt"
	"strings"

	"github.com/speakeasy-api/jsonpath/pkg/jsonpath/config"
	"github.com/speakeasy-api/jsonpath/pkg/jsonpath/token"
	"gopkg.in/yaml.v3"
)

// ApplyTo will take an overlay and apply its changes to the given YAML
// document.
func (o *Overlay) ApplyTo(root *yaml.Node) error {
	for _, action := range o.Actions {
		var err error
		if action.Remove {
			err = o.applyRemoveAction(root, action, nil)
		} else {
			err = o.applyUpdateAction(root, action, &[]string{})
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Overlay) ApplyToStrict(root *yaml.Node) ([]string, error) {
	multiError := []string{}
	warnings := []string{}
	hasFilterExpression := false
	for i, action := range o.Actions {
		tokens := token.NewTokenizer(action.Target, config.WithPropertyNameExtension()).Tokenize()
		for _, tok := range tokens {
			if tok.Token == token.FILTER {
				hasFilterExpression = true
			}
		}

		actionWarnings := []string{}
		err := o.validateSelectorHasAtLeastOneTarget(root, action)
		if err != nil {
			multiError = append(multiError, err.Error())
		}
		if action.Remove {
			err = o.applyRemoveAction(root, action, &actionWarnings)
			if err != nil {
				multiError = append(multiError, err.Error())
			}
		} else {
			err = o.applyUpdateAction(root, action, &actionWarnings)
			if err != nil {
				multiError = append(multiError, err.Error())
			}
		}
		for _, warning := range actionWarnings {
			warnings = append(warnings, fmt.Sprintf("update action (%v / %v) target=%s: %s", i+1, len(o.Actions), action.Target, warning))
		}
	}

	if hasFilterExpression && !o.UsesRFC9535() {
		warnings = append(warnings, "overlay has a filter expression but lacks `x-speakeasy-jsonpath: rfc9535` extension. Deprecated jsonpath behaviour in use. See overlay.speakeasy.com for the implementation playground.")
	}

	if len(multiError) > 0 {
		return warnings, fmt.Errorf("error applying overlay (strict): %v", strings.Join(multiError, ","))
	}
	return warnings, nil
}

func (o *Overlay) validateSelectorHasAtLeastOneTarget(root *yaml.Node, action Action) error {
	if action.Target == "" {
		return nil
	}

	p, err := o.NewPath(action.Target, nil)
	if err != nil {
		return err
	}

	nodes := p.Query(root)

	if len(nodes) == 0 {
		return fmt.Errorf("selector %q did not match any targets", action.Target)
	}

	return nil
}

func (o *Overlay) applyRemoveAction(root *yaml.Node, action Action, warnings *[]string) error {
	if action.Target == "" {
		return nil
	}

	idx := newParentIndex(root)

	p, err := o.NewPath(action.Target, warnings)
	if err != nil {
		return err
	}

	nodes := p.Query(root)

	for _, node := range nodes {
		removeNode(idx, node)
	}

	return nil
}

func removeNode(idx parentIndex, node *yaml.Node) {
	parent := idx.getParent(node)
	if parent == nil {
		return
	}

	for i, child := range parent.Content {
		if child == node {
			switch parent.Kind {
			case yaml.MappingNode:
				if i%2 == 1 {
					// if we select a value, we should delete the key too
					parent.Content = append(parent.Content[:i-1], parent.Content[i+1:]...)
				} else {
					// if we select a key, we should delete the value
					parent.Content = append(parent.Content[:i], parent.Content[i+2:]...)
				}
				return
			case yaml.SequenceNode:
				parent.Content = append(parent.Content[:i], parent.Content[i+1:]...)
				return
			}
		}
	}
}

func (o *Overlay) applyUpdateAction(root *yaml.Node, action Action, warnings *[]string) error {
	if action.Target == "" {
		return nil
	}

	if action.Update.IsZero() {
		return nil
	}

	p, err := o.NewPath(action.Target, warnings)
	if err != nil {
		return err
	}

	nodes := p.Query(root)

	didMakeChange := false
	for _, node := range nodes {
		didMakeChange = updateNode(node, &action.Update) || didMakeChange
	}
	if !didMakeChange {
		*warnings = append(*warnings, "does nothing")
	}

	return nil
}

func updateNode(node *yaml.Node, updateNode *yaml.Node) bool {
	return mergeNode(node, updateNode)
}

func mergeNode(node *yaml.Node, merge *yaml.Node) bool {
	if node.Kind != merge.Kind {
		*node = *clone(merge)
		return true
	}
	switch node.Kind {
	default:
		isChanged := node.Value != merge.Value
		node.Value = merge.Value
		return isChanged
	case yaml.MappingNode:
		return mergeMappingNode(node, merge)
	case yaml.SequenceNode:
		return mergeSequenceNode(node, merge)
	}
}

// mergeMappingNode will perform a shallow merge of the merge node into the main
// node.
func mergeMappingNode(node *yaml.Node, merge *yaml.Node) bool {
	anyChange := false
NextKey:
	for i := 0; i < len(merge.Content); i += 2 {
		mergeKey := merge.Content[i].Value
		mergeValue := merge.Content[i+1]

		for j := 0; j < len(node.Content); j += 2 {
			nodeKey := node.Content[j].Value
			if nodeKey == mergeKey {
				anyChange = mergeNode(node.Content[j+1], mergeValue) || anyChange
				continue NextKey
			}
		}

		node.Content = append(node.Content, merge.Content[i], clone(mergeValue))
		anyChange = true
	}
	return anyChange
}

// mergeSequenceNode will append the merge node's content to the original node.
func mergeSequenceNode(node *yaml.Node, merge *yaml.Node) bool {
	node.Content = append(node.Content, clone(merge).Content...)
	return true
}

func clone(node *yaml.Node) *yaml.Node {
	newNode := &yaml.Node{
		Kind:        node.Kind,
		Style:       node.Style,
		Tag:         node.Tag,
		Value:       node.Value,
		Anchor:      node.Anchor,
		HeadComment: node.HeadComment,
		LineComment: node.LineComment,
		FootComment: node.FootComment,
	}
	if node.Alias != nil {
		newNode.Alias = clone(node.Alias)
	}
	if node.Content != nil {
		newNode.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			newNode.Content[i] = clone(child)
		}
	}
	return newNode
}
