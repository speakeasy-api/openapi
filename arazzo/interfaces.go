package arazzo

import (
	"context"

	"github.com/speakeasy-api/openapi/validation"
)

type validator[T any] interface {
	*T
	Validate(context.Context, ...validation.Option) []error
}

type model[C any] interface {
	Validate(context.Context, ...validation.Option) []error
	GetCore() *C
}
