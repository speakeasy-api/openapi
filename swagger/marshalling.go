package swagger

import (
	"context"
	"io"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/swagger/core"
	"github.com/speakeasy-api/openapi/validation"
)

type Option[T any] func(o *T)

type UnmarshalOptions struct {
	skipValidation bool
}

// WithSkipValidation will skip validation of the Swagger document during unmarshaling.
// Useful to quickly load a document that will be mutated and validated later.
func WithSkipValidation() Option[UnmarshalOptions] {
	return func(o *UnmarshalOptions) {
		o.skipValidation = true
	}
}

// Unmarshal will unmarshal and validate a Swagger 2.0 document from the provided io.Reader.
// Validation can be skipped by using swagger.WithSkipValidation() as one of the options when calling this function.
func Unmarshal(ctx context.Context, doc io.Reader, opts ...Option[UnmarshalOptions]) (*Swagger, []error, error) {
	o := UnmarshalOptions{}
	for _, opt := range opts {
		opt(&o)
	}

	var swagger Swagger

	validationErrs, err := marshaller.Unmarshal(ctx, doc, &swagger)
	if err != nil {
		return nil, nil, err
	}

	if o.skipValidation {
		return &swagger, nil, nil
	}

	if !o.skipValidation {
		validationErrs = append(validationErrs, swagger.Validate(ctx)...)
		validation.SortValidationErrors(validationErrs)
	}

	return &swagger, validationErrs, nil
}

// Marshal will marshal the provided Swagger document to the provided io.Writer.
func Marshal(ctx context.Context, swagger *Swagger, w io.Writer) error {
	return marshaller.Marshal(ctx, swagger, w)
}

// Sync will sync the high-level model to the core model.
// This is useful when creating or mutating a high-level model and wanting access to the yaml nodes that back it.
func Sync(ctx context.Context, model marshaller.Marshallable[core.Swagger]) error {
	_, err := marshaller.SyncValue(ctx, model, model.GetCore(), model.GetRootNode(), false)
	return err
}
