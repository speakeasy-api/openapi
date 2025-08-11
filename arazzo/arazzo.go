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

	var arazzo Arazzo
	validationErrs, err := marshaller.Unmarshal(ctx, doc, &arazzo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal Arazzo document: %w", err)
	}

	if o.skipValidation {
		return &arazzo, nil, nil
	}

	validationErrs = append(validationErrs, arazzo.Validate(ctx)...)
	slices.SortFunc(validationErrs, func(a, b error) int {
		var aValidationErr *validation.Error
		var bValidationErr *validation.Error
		aIsValidationErr := errors.As(a, &aValidationErr)
		bIsValidationErr := errors.As(b, &bValidationErr)
		if aIsValidationErr && bIsValidationErr {
			if aValidationErr.GetLineNumber() == bValidationErr.GetLineNumber() {
				return aValidationErr.GetColumnNumber() - bValidationErr.GetColumnNumber()
			}
			return aValidationErr.GetLineNumber() - bValidationErr.GetLineNumber()
		} else if aIsValidationErr {
			return -1
		} else if bIsValidationErr {
			return 1
		}

		return 0
	})

	return &arazzo, validationErrs, nil
}

// Marshal will marshal the provided Arazzo document to the provided io.Writer.
func Marshal(ctx context.Context, arazzo *Arazzo, w io.Writer) error {
	return marshaller.Marshal(ctx, arazzo, w)
}

// Sync will sync any changes made to the Arazzo document models back to the core models.
func (a *Arazzo) Sync(ctx context.Context) error {
	if _, err := marshaller.SyncValue(ctx, a, a.GetCore(), a.GetRootNode(), false); err != nil {
		return err
	}
	return nil
}

// Validate will validate the Arazzo document against the Arazzo Specification.
func (a *Arazzo) Validate(ctx context.Context, opts ...validation.Option) []error {
	opts = append(opts, validation.WithContextObject(a))

	core := a.GetCore()
	errs := []error{}

	arazzoMajor, arazzoMinor, arazzoPatch, err := utils.ParseVersion(a.Arazzo)
	if err != nil {
		errs = append(errs, validation.NewValueError(validation.NewValueValidationError("invalid Arazzo version in document %s: %s", a.Arazzo, err.Error()), core, core.Arazzo))
	}

	if arazzoMajor != VersionMajor || arazzoMinor != VersionMinor || arazzoPatch > VersionPatch {
		errs = append(errs, validation.NewValueError(validation.NewValueValidationError("only Arazzo version %s and below is supported", Version), core, core.Arazzo))
	}

	errs = append(errs, a.Info.Validate(ctx, opts...)...)

	sourceDescriptionNames := make(map[string]bool)

	for i, sourceDescription := range a.SourceDescriptions {
		errs = append(errs, sourceDescription.Validate(ctx, opts...)...)

		if _, ok := sourceDescriptionNames[sourceDescription.Name]; ok {
			errs = append(errs, validation.NewSliceError(validation.NewValueValidationError("sourceDescription name %s is not unique", sourceDescription.Name), core, core.SourceDescriptions, i))
		}

		sourceDescriptionNames[sourceDescription.Name] = true
	}

	workflowIds := make(map[string]bool)

	for i, workflow := range a.Workflows {
		errs = append(errs, workflow.Validate(ctx, opts...)...)

		if _, ok := workflowIds[workflow.WorkflowID]; ok {
			errs = append(errs, validation.NewSliceError(validation.NewValueValidationError("workflowId %s is not unique", workflow.WorkflowID), core, core.Workflows, i))
		}

		workflowIds[workflow.WorkflowID] = true
	}

	if a.Components != nil {
		errs = append(errs, a.Components.Validate(ctx, opts...)...)
	}

	a.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
