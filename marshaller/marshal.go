package marshaller

import (
	"context"
	"errors"
	"io"

	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

// Marshallable represents a high-level model that can be marshaled
type Marshallable[T any] interface {
	GetCore() *T
	GetRootNode() *yaml.Node
}

// ModelWithCore represents a high-level model that has an embedded Model[T]
type ModelWithCore interface {
	GetCore() any
	GetRootNode() *yaml.Node
}

// Marshal will marshal the provided high-level model to the provided io.Writer.
// It syncs any changes from the high-level model to the core model, then marshals the core model.
func Marshal[T any](ctx context.Context, model Marshallable[T], w io.Writer) error {
	if model == nil {
		return nil
	}

	if _, err := Sync(ctx, model); err != nil {
		return err
	}

	core, ok := any(model.GetCore()).(CoreModeler)
	if !ok {
		return errors.New("core model does not implement CoreModeler")
	}

	// Add config to context before syncing to ensure proper node styles
	ctx = yml.ContextWithConfig(ctx, core.GetConfig())

	return core.Marshal(ctx, w)
}

// Sync will sync the high-level model to the core model.
// This is useful when creating or mutating a high-level model and wanting access to the yaml nodes that back it.
func Sync[T any](ctx context.Context, model Marshallable[T]) (*yaml.Node, error) {
	if model == nil {
		return nil, errors.New("nil model")
	}

	core, ok := any(model.GetCore()).(CoreModeler)
	if !ok {
		return nil, errors.New("core model does not implement CoreModeler")
	}

	// Add config to context before syncing to ensure proper node styles
	ctx = yml.ContextWithConfig(ctx, core.GetConfig())

	// Sync changes from high-level model to core model
	// Now we pass the full high-level model (not just the embedded Model[T])
	return SyncValue(ctx, model, model.GetCore(), model.GetRootNode(), false)
}
