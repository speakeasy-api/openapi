// Package asyncapi provides an API for working with AsyncAPI 3.x documents including reading, creating,
// mutating and validating them.
//
// The package currently supports the AsyncAPI 3.0 and 3.1 minor lines. It is under active development: the root
// document and Info Object are modelled, while servers, channels, operations and components are preserved as raw
// values that will be replaced by typed models in a later release.
package asyncapi

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"strings"

	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/asyncapiversion"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	jsonschema "github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/values"
	"go.yaml.in/yaml/v3"
)

// Version is the latest version of the AsyncAPI Specification that this package supports, used for new documents.
const Version = asyncapiversion.Latest

// SupportedMinorVersions returns a copy of the `major.minor` AsyncAPI Specification versions supported by this package.
// Any patch release (and permitted suffix) of a supported minor version is accepted.
func SupportedMinorVersions() []string {
	return asyncapiversion.SupportedMinorVersions()
}

// IsSupportedVersion returns true if the provided AsyncAPI version string is valid and belongs to a supported minor version.
func IsSupportedVersion(v string) bool {
	return asyncapiversion.IsSupported(v)
}

// AsyncAPI is the root object of an AsyncAPI document.
type AsyncAPI struct {
	marshaller.Model[core.AsyncAPI]

	// AsyncAPI is the version of the AsyncAPI Specification the document conforms to.
	AsyncAPI string
	// ID is the identifier of the application the document is defining. It must be a URI.
	ID *string
	// Info provides metadata about the API.
	Info Info
	// Servers are the connection details of servers. Preserved as a raw value until servers are modelled.
	Servers values.Value
	// DefaultContentType is the default content type used to encode and decode message payloads.
	DefaultContentType *string
	// Channels are the channels used by the application. Preserved as a raw value until channels are modelled.
	Channels values.Value
	// Operations are the operations the application implements. Preserved as a raw value until operations are modelled.
	Operations values.Value
	// Components hold reusable objects. Preserved as a raw value until components are modelled.
	Components values.Value
	// Extensions provides a list of extensions to the AsyncAPI Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.AsyncAPI] = (*AsyncAPI)(nil)

// NewAsyncAPI creates a new AsyncAPI document for the latest supported version.
func NewAsyncAPI() *AsyncAPI {
	return &AsyncAPI{AsyncAPI: Version}
}

// GetAsyncAPI returns the AsyncAPI Specification version of the document.
func (a *AsyncAPI) GetAsyncAPI() string {
	if a == nil {
		return ""
	}
	return a.AsyncAPI
}

// GetID returns the application identifier, or an empty string if not set.
func (a *AsyncAPI) GetID() string {
	if a == nil || a.ID == nil {
		return ""
	}
	return *a.ID
}

// GetInfo returns the Info Object, or nil if the document is nil.
func (a *AsyncAPI) GetInfo() *Info {
	if a == nil {
		return nil
	}
	return &a.Info
}

// GetDefaultContentType returns the default content type, or an empty string if not set.
func (a *AsyncAPI) GetDefaultContentType() string {
	if a == nil || a.DefaultContentType == nil {
		return ""
	}
	return *a.DefaultContentType
}

// GetExtensions returns the extensions of the document. Returns an empty extensions map if not set.
func (a *AsyncAPI) GetExtensions() *extensions.Extensions {
	if a == nil || a.Extensions == nil {
		return extensions.New()
	}
	return a.Extensions
}

// Option is an option for working with AsyncAPI documents.
type Option[T any] func(o *T)

// UnmarshalOptions are the options available when unmarshalling an AsyncAPI document.
type UnmarshalOptions struct {
	skipValidation bool
}

// WithSkipValidation will skip validation of the AsyncAPI document during unmarshalling.
// Useful to quickly load a document that will be mutated and validated later.
func WithSkipValidation() Option[UnmarshalOptions] {
	return func(o *UnmarshalOptions) {
		o.skipValidation = true
	}
}

// Unmarshal will unmarshal and validate an AsyncAPI document from the provided io.Reader.
// The returned error is reserved for failures to read or parse the document, while problems with its content
// are returned as validation errors. Validation can be skipped with WithSkipValidation.
//
// Validation does not read any files or make any network requests.
func Unmarshal(ctx context.Context, doc io.Reader, opts ...Option[UnmarshalOptions]) (*AsyncAPI, []error, error) {
	o := UnmarshalOptions{}
	for _, opt := range opts {
		opt(&o)
	}

	var asyncAPI AsyncAPI
	validationErrs, err := marshaller.Unmarshal(ctx, doc, &asyncAPI)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal AsyncAPI document: %w", err)
	}

	if o.skipValidation {
		return &asyncAPI, nil, nil
	}

	validationErrs = append(validationErrs, asyncAPI.Validate(ctx)...)
	validation.SortValidationErrors(validationErrs)

	return &asyncAPI, validationErrs, nil
}

// Marshal will marshal the provided AsyncAPI document to the provided io.Writer, syncing any changes first.
func Marshal(ctx context.Context, asyncAPI *AsyncAPI, w io.Writer) error {
	return marshaller.Marshal(ctx, asyncAPI, w)
}

// Sync will sync any changes made to the AsyncAPI document models back to the core models.
func (a *AsyncAPI) Sync(ctx context.Context) error {
	_, err := marshaller.Sync(ctx, a)
	return err
}

// Validate will validate the AsyncAPI document against the AsyncAPI Specification.
func (a *AsyncAPI) Validate(ctx context.Context, opts ...validation.Option) []error {
	if a == nil {
		return nil
	}

	core := a.GetCore()
	errs := []error{}

	if v, err := asyncapiversion.Parse(a.AsyncAPI); err != nil {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("asyncapi version is invalid: %w", err), core, core.AsyncAPI))
	} else {
		if !v.IsSupported() {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationSupportedVersion, fmt.Errorf("asyncapi version `%s` is not supported, only AsyncAPI versions `%s` are supported", a.AsyncAPI, strings.Join(asyncapiversion.SupportedVersionPatterns(), "`, `")), core, core.AsyncAPI))
		}
		opts = append(opts, jsonschema.WithSchemaFormat(asyncapiversion.DefaultSchemaFormat(a.AsyncAPI)))
	}

	if a.ID != nil {
		if u, err := url.Parse(*a.ID); err != nil || u.Scheme == "" {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("asyncapi.id `%s` must be a URI", *a.ID), core, core.ID))
		}
	}

	if a.DefaultContentType != nil {
		if _, _, err := mime.ParseMediaType(*a.DefaultContentType); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("asyncapi.defaultContentType `%s` is not a valid media type", *a.DefaultContentType), core, core.DefaultContentType))
		}
	}

	// A missing `info` is reported as a required field when unmarshalling.
	errs = append(errs, a.Info.Validate(ctx, opts...)...)

	a.Valid = len(errs) == 0 && coreValid(core)

	return errs
}

// coreValid reports whether a core model is free of unmarshalling errors. Models built in code rather than
// parsed have no root node and no unmarshalling errors, so they are treated as valid.
func coreValid(c interface {
	GetRootNode() *yaml.Node
	GetValid() bool
},
) bool {
	return c.GetRootNode() == nil || c.GetValid()
}
