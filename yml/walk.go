package yml

import (
	"context"

	"github.com/speakeasy-api/openapi/errors"
	"gopkg.in/yaml.v3"
)

const (
	// ErrTerminate is a sentinel error that can be returned from a Walk function to terminate the walk.
	ErrTerminate = errors.Error("terminate")
)

// VisitFunc represents a function that will be called for each node in the node structure.
// The functions receives the current node, any parent nodes, and the root node.
type VisitFunc func(ctx context.Context, node, parent *yaml.Node, root *yaml.Node) error

// Walk will walk the yaml node structure and call the provided VisitFunc for each node in the document.
// TODO should key/index be passed for nodes that are children of maps/sequences?
func Walk(ctx context.Context, node *yaml.Node, visit VisitFunc) error {
	err := walkNode(ctx, node, nil, node, visit)
	if err != nil {
		if errors.Is(err, ErrTerminate) {
			return nil
		}
		return err
	}

	return nil
}

func walkNode(ctx context.Context, node *yaml.Node, parent *yaml.Node, root *yaml.Node, visit VisitFunc) error {
	if node == nil {
		return nil
	}

	if err := visit(ctx, node, parent, root); err != nil {
		return err
	}

	switch node.Kind {
	case yaml.DocumentNode:
		return walkDocumentNode(ctx, node, root, visit)
	case yaml.MappingNode:
		return walkMappingNode(ctx, node, root, visit)
	case yaml.SequenceNode:
		return walkSequenceNode(ctx, node, root, visit)
	case yaml.AliasNode:
		return walkAliasNode(ctx, node, root, visit)
	}

	return nil
}

func walkDocumentNode(ctx context.Context, node *yaml.Node, root *yaml.Node, visit VisitFunc) error {
	for i := 0; i < len(node.Content); i++ {
		if err := walkNode(ctx, node.Content[i], node, root, visit); err != nil {
			return err
		}
	}

	return nil
}

func walkMappingNode(ctx context.Context, node *yaml.Node, root *yaml.Node, visit VisitFunc) error {
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		value := node.Content[i+1]

		if err := walkNode(ctx, key, node, root, visit); err != nil {
			return err
		}

		if err := walkNode(ctx, value, node, root, visit); err != nil {
			return err
		}
	}

	return nil
}

func walkSequenceNode(ctx context.Context, node *yaml.Node, root *yaml.Node, visit VisitFunc) error {
	for i := 0; i < len(node.Content); i++ {
		if err := walkNode(ctx, node.Content[i], node, root, visit); err != nil {
			return err
		}
	}

	return nil
}

func walkAliasNode(ctx context.Context, node *yaml.Node, root *yaml.Node, visit VisitFunc) error {
	return walkNode(ctx, node.Alias, node, root, visit)
}
