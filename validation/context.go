package validation

import "context"

type contextKey string

func (c contextKey) String() string {
	return "validation-context-key-" + string(c)
}

const errorsContextKey = contextKey("errors")

type validationContext struct {
	Errors []error
}

func ContextWithValidationContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, errorsContextKey, &validationContext{})
}

func AddValidationError(ctx context.Context, err error) {
	validationContext, ok := ctx.Value(errorsContextKey).(*validationContext)
	if !ok {
		return
	}

	validationContext.Errors = append(validationContext.Errors, err)
}

func GetValidationErrors(ctx context.Context) []error {
	validationContext, ok := ctx.Value(errorsContextKey).(*validationContext)
	if !ok || len(validationContext.Errors) == 0 {
		return nil
	}

	return validationContext.Errors
}
