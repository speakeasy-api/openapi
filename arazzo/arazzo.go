// Package arazzo provides an API for working with Arazzo documents including reading, creating, mutating, walking and validating them.
//
// The Arazzo Specification is a mechanism for orchestrating API calls, defining their sequences and dependencies, to achieve specific outcomes when working with API descriptions like OpenAPI.
package arazzo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/internal/specversion"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/internal/version"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
)

// Version is the latest version of the Arazzo Specification that this package conforms to.
const (
	Version = "1.1.0"
)

var (
	// MinimumSupportedVersion is the earliest supported Arazzo Specification version.
	MinimumSupportedVersion = version.MustParse("1.0.0")
	// MaximumSupportedVersion is the latest Arazzo Specification version this package was built against.
	// As the Arazzo Specification states the patch version should not be considered by tooling, any
	// patch release of a supported minor version (see SupportedMinorVersions) is accepted.
	MaximumSupportedVersion = version.MustParse(Version)
	// SupportedMinorVersions are the `major.minor` Arazzo Specification versions supported by this package.
	SupportedMinorVersions = []string{"1.0", "1.1"}
)

// IsSupportedVersion returns true if the provided Arazzo version string is a supported version.
// Any patch release (and prerelease suffix) of a supported minor version is accepted.
func IsSupportedVersion(v string) bool {
	parsed, err := specversion.Parse(v)
	if err != nil {
		return false
	}

	for _, supported := range SupportedMinorVersions {
		if fmt.Sprintf("%d.%d", parsed.Major, parsed.Minor) == supported {
			return true
		}
	}

	return false
}

// Arazzo is the root object for an Arazzo document.
type Arazzo struct {
	marshaller.Model[core.Arazzo]

	// Arazzo is the version of the Arazzo Specification that this document conforms to.
	Arazzo string
	// Self is the canonical URI of this Arazzo document, used for identity-based referencing and as the base URI
	// for resolving relative references. Must not contain a fragment. Arazzo 1.1+ only.
	Self *string
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
	validation.SortValidationErrors(validationErrs)

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
	opts = append(opts, validation.WithContextObject(&oas3.ParentDocumentVersion{Arazzo: pointer.From(a.Arazzo)}))

	core := a.GetCore()
	errs := []error{}

	if _, err := specversion.Parse(a.Arazzo); err != nil {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("arazzo.version is invalid `%s`: %w", a.Arazzo, err), core, core.Arazzo))
	} else if !IsSupportedVersion(a.Arazzo) {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationSupportedVersion, fmt.Errorf("arazzo.version only Arazzo versions `%s` are supported", strings.Join(supportedVersionPatterns(), "`, `")), core, core.Arazzo))
	}

	if a.Self != nil {
		vo := validation.NewOptions(opts...)
		switch {
		case !specversion.IsAtLeast11(vo):
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationSupportedVersion, specversion.Requires11Error("`$self`"), core, core.Self))
		case strings.Contains(*a.Self, "#"):
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, errors.New("`$self` must not contain a fragment identifier"), core, core.Self))
		default:
			if _, err := url.Parse(*a.Self); err != nil {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("`$self` must be a valid URI reference: %w", err), core, core.Self))
			}
		}
	}

	errs = append(errs, a.Info.Validate(ctx, opts...)...)

	sourceDescriptionNames := make(map[string]bool)

	for i, sourceDescription := range a.SourceDescriptions {
		if sourceDescription == nil {
			// null entries are reported as type mismatches when unmarshalling
			continue
		}
		errs = append(errs, sourceDescription.Validate(ctx, opts...)...)

		if _, ok := sourceDescriptionNames[sourceDescription.Name]; ok {
			errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationDuplicateKey, fmt.Errorf("sourceDescription.name `%s` is not unique", sourceDescription.Name), core, core.SourceDescriptions, i))
		}

		sourceDescriptionNames[sourceDescription.Name] = true
	}

	workflowIds := make(map[string]bool)

	for i, workflow := range a.Workflows {
		if workflow == nil {
			// null entries are reported as type mismatches when unmarshalling
			continue
		}
		errs = append(errs, workflow.Validate(ctx, opts...)...)

		if _, ok := workflowIds[workflow.WorkflowID]; ok {
			errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationDuplicateKey, fmt.Errorf("workflow.workflowId `%s` is not unique", workflow.WorkflowID), core, core.Workflows, i))
		}

		workflowIds[workflow.WorkflowID] = true
	}

	if a.Components != nil {
		errs = append(errs, a.Components.Validate(ctx, opts...)...)
	}

	a.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

func supportedVersionPatterns() []string {
	patterns := make([]string, 0, len(SupportedMinorVersions))
	for _, v := range SupportedMinorVersions {
		patterns = append(patterns, v+".x")
	}
	return patterns
}
