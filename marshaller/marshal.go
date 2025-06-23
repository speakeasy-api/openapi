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
		return errors.New("nil model")
	}

	core, ok := any(model.GetCore()).(CoreModeler)
	if !ok {
		return errors.New("core model does not implement CoreModeler")
	}

	// Add config to context before syncing to ensure proper node styles
	ctx = yml.ContextWithConfig(ctx, core.GetConfig())

	// Sync changes from high-level model to core model
	// Now we pass the full high-level model (not just the embedded Model[T])
	if _, err := SyncValue(ctx, model, model.GetCore(), model.GetRootNode(), false); err != nil {
		return err
	}

	return core.Marshal(ctx, w)
}
