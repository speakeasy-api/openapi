// Package arazzo provides an API for working with Arazzo documents including reading, creating, mutating, walking and validating them.
//
// The Arazzo Specification is a mechanism for orchestrating API calls, defining their sequences and dependencies, to achieve specific outcomes when working with API descriptions like OpenAPI.
package arazzo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
)

// Version is the version of the Arazzo Specification that this package conforms to.
const (
	Version      = "1.0.1"
	VersionMajor = 1
	VersionMinor = 0
	VersionPatch = 1
)

// Arazzo is the root object for an Arazzo document.
type Arazzo struct {
	marshaller.Model[core.Arazzo]

	// Arazzo is the version of the Arazzo Specification that this document conforms to.
	Arazzo string
	// Info provides metadata about the Arazzo document.
	Info Info
	// SourceDescriptions provides a list of SourceDescription objects that describe the source of the data that the workflow is orchestrating.
	SourceDescriptions SourceDescriptions
	// Workflows provides a list of Workflow objects that describe the orchestration of API calls.
	Workflows Workflows
	// Components provides a list of reusable components that can be used in the workflow.
	Components *Components
	// Extensions provides a list of extensions to the Arazzo document.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Arazzo] = (*Arazzo)(nil)

type Option[T any] func(o *T)

type unmarshalOptions struct {
	skipValidation bool
}

// WithSkipValidation will skip validation of the Arazzo document during unmarshaling.
// Useful to quickly load a document that will be mutated and validated later.
func WithSkipValidation() Option[unmarshalOptions] {
	return func(o *unmarshalOptions) {
		o.skipValidation = true
	}
}

// Unmarshal will unmarshal and validate an Arazzo document from the provided io.Reader.
// Validation can be skipped by using arazzo.WithSkipValidation() as one of the options when calling this function.
func Unmarshal(ctx context.Context, doc io.Reader, opts ...Option[unmarshalOptions]) (*Arazzo, []error, error) {
	o := unmarshalOptions{}
	for _, opt := range opts {
		opt(&o)
	}

	c, err := core.Unmarshal(ctx, doc)
	if err != nil {
		return nil, nil, err
	}

	arazzo := &Arazzo{}
	if err := marshaller.PopulateModel(*c, arazzo); err != nil {
		return nil, nil, err
	}

	var validationErrs []error
	if !o.skipValidation {
		validationErrs = append(validationErrs, arazzo.Validate(ctx)...)
		slices.SortFunc(validationErrs, func(a, b error) int {
			var aValidationErr *validation.Error
			var bValidationErr *validation.Error
			aIsValidationErr := errors.As(a, &aValidationErr)
			bIsValidationErr := errors.As(b, &bValidationErr)
			if aIsValidationErr && bIsValidationErr {
				if aValidationErr.Line == bValidationErr.Line {
					return aValidationErr.Column - bValidationErr.Column
				}
				return aValidationErr.Line - bValidationErr.Line
			} else if aIsValidationErr {
				return -1
			} else if bIsValidationErr {
				return 1
			}

			return 0
		})
	}

	return arazzo, validationErrs, nil
}

// Marshal will marshal the provided Arazzo document to the provided io.Writer.
func Marshal(ctx context.Context, arazzo *Arazzo, w io.Writer) error {
	if arazzo == nil {
		return errors.New("nil *Arazzo")
	}

	if err := arazzo.Marshal(ctx, w); err != nil {
		return err
	}

	return nil
}

// Sync will sync any changes made to the Arazzo document models back to the core models.
func (a *Arazzo) Sync(ctx context.Context) error {
	if _, err := marshaller.SyncValue(ctx, a, a.GetCore(), nil, false); err != nil {
		return err
	}
	return nil
}

// Marshal will marshal the Arazzo document to the provided io.Writer.
func (a *Arazzo) Marshal(ctx context.Context, w io.Writer) error {
	ctx = yml.ContextWithConfig(ctx, a.GetCore().Config)

	if _, err := marshaller.SyncValue(ctx, a, a.GetCore(), nil, false); err != nil {
		return err
	}

	return a.GetCore().Marshal(ctx, w)
}

// Validate will validate the Arazzo document against the Arazzo Specification.
func (a *Arazzo) Validate(ctx context.Context, opts ...validation.Option) []error {
	opts = append(opts, validation.WithContextObject(a))

	core := a.GetCore()
	errs := core.GetValidationErrors()

	arazzoMajor, arazzoMinor, arazzoPatch, err := utils.ParseVersion(a.Arazzo)
	if err != nil {
		errs = append(errs, validation.NewValueError(fmt.Sprintf("invalid Arazzo version in document %s: %s", a.Arazzo, err.Error()), core, core.Arazzo))
	}

	if arazzoMajor != VersionMajor || arazzoMinor != VersionMinor || arazzoPatch > VersionPatch {
		errs = append(errs, validation.NewValueError(fmt.Sprintf("only Arazzo version %s and below is supported", Version), core, core.Arazzo))
	}

	errs = append(errs, a.Info.Validate(ctx, opts...)...)

	sourceDescriptionNames := make(map[string]bool)

	for i, sourceDescription := range a.SourceDescriptions {
		errs = append(errs, sourceDescription.Validate(ctx, opts...)...)

		if _, ok := sourceDescriptionNames[sourceDescription.Name]; ok {
			errs = append(errs, validation.NewSliceError(fmt.Sprintf("sourceDescription name %s is not unique", sourceDescription.Name), core, core.SourceDescriptions, i))
		}

		sourceDescriptionNames[sourceDescription.Name] = true
	}

	workflowIds := make(map[string]bool)

	for i, workflow := range a.Workflows {
		errs = append(errs, workflow.Validate(ctx, opts...)...)

		if _, ok := workflowIds[workflow.WorkflowID]; ok {
			errs = append(errs, validation.NewSliceError(fmt.Sprintf("workflowId %s is not unique", workflow.WorkflowID), core, core.Workflows, i))
		}

		workflowIds[workflow.WorkflowID] = true
	}

	if a.Components != nil {
		errs = append(errs, a.Components.Validate(ctx, opts...)...)
	}

	a.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
