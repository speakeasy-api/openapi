package interfaces

import (
	"context"

	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

type Validator[T any] interface {
	*T
	Validate(context.Context, ...validation.Option) []error
}

type Model[C any] interface {
	Validate(context.Context, ...validation.Option) []error
	GetCore() *C
}

type CoreModel interface {
	Unmarshal(ctx context.Context, node *yaml.Node) error
}
