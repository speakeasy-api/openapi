package arazzo

import (
	"context"

	"github.com/speakeasy-api/openapi/validation"
)

type validator interface {
	Validate(context.Context, ...validation.Option) []error
}

type model[T any] interface {
	validator
	GetCore() *T
}
