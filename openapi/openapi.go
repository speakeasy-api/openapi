package openapi

import (
	"context"
	"net/url"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/internal/version"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

// Version is the version of the OpenAPI Specification that this package conforms to.
const (
	Version = "3.2.0"
)

var (
	MinimumSupportedVersion = version.MustParse("3.0.0")
	MaximumSupportedVersion = version.MustParse(Version)
)

// OpenAPI represents an OpenAPI document compatible with the OpenAPI Specification 3.0.X and 3.1.X.
// Where the specification differs between versions the
type OpenAPI struct {
	marshaller.Model[core.OpenAPI]

	// OpenAPI is the version of the OpenAPI Specification that this document conforms to.
	OpenAPI string
	// Self provides the self-assigned URI of this document, which also serves as its base URI for resolving relative references.
	// It MUST be in the form of a URI reference as defined by RFC3986.
	Self *string
	// Info provides various information about the API and document.
	Info Info
	// ExternalDocs provides additional external documentation for this API.
	ExternalDocs *oas3.ExternalDocumentation
	// Tags is a list of tags used by the document.
	Tags []*Tag
	// Servers is an array of information about servers available to provide the functionality described in the API.
	Servers []*Server
	// Security is a declaration of which security mechanisms can be used for this API.
	Security []*SecurityRequirement
	// Paths is a map of relative endpoint paths to their corresponding PathItem objects.
	Paths *Paths
	// Webhooks are the incoming webhooks associated with this API.
	Webhooks *sequencedmap.Map[string, *ReferencedPathItem]

	// Components is a container for the reusable objects available to the API.
	Components *Components

	// JSONSchemaDialect is the default value for the $schema keyword within Schema objects in this document. It MUST be in the format of a URI.
	JSONSchemaDialect *string

	// Extensions provides a list of extensions to the OpenAPI document.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.OpenAPI] = (*OpenAPI)(nil)

// NewOpenAPI creates a new OpenAPI object with version set
func NewOpenAPI() *OpenAPI {
	return &OpenAPI{
		OpenAPI: Version,
	}
}

// GetOpenAPI returns the value of the OpenAPI field. Returns empty string if not set.
func (o *OpenAPI) GetOpenAPI() string {
	if o == nil {
		return ""
	}
	return o.OpenAPI
}

// GetSelf returns the value of the Self field. Returns empty string if not set.
func (o *OpenAPI) GetSelf() string {
	if o == nil || o.Self == nil {
		return ""
	}
	return *o.Self
}

// GetInfo returns the value of the Info field. Returns nil if not set.
func (o *OpenAPI) GetInfo() *Info {
	if o == nil {
		return nil
	}
	return &o.Info
}

// GetExternalDocs returns the value of the ExternalDocs field. Returns nil if not set.
func (o *OpenAPI) GetExternalDocs() *oas3.ExternalDocumentation {
	if o == nil {
		return nil
	}
	return o.ExternalDocs
}

// GetTags returns the value of the Tags field. Returns nil if not set.
func (o *OpenAPI) GetTags() []*Tag {
	if o == nil {
		return nil
	}
	return o.Tags
}

// GetServers returns the value of the Servers field. Returns a default server of "/" if not set.
func (o *OpenAPI) GetServers() []*Server {
	if o == nil || len(o.Servers) == 0 {
		return []*Server{{URL: "/"}}
	}
	return o.Servers
}

// GetSecurity returns the value of the Security field. Returns nil if not set.
func (o *OpenAPI) GetSecurity() []*SecurityRequirement {
	if o == nil {
		return nil
	}
	return o.Security
}

// GetPaths returns the value of the Paths field. Returns nil if not set.
func (o *OpenAPI) GetPaths() *Paths {
	if o == nil {
		return nil
	}
	return o.Paths
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (o *OpenAPI) GetExtensions() *extensions.Extensions {
	if o == nil || o.Extensions == nil {
		return extensions.New()
	}
	return o.Extensions
}

// GetWebhooks returns the value of the Webhooks field. Returns nil if not set.
func (o *OpenAPI) GetWebhooks() *sequencedmap.Map[string, *ReferencedPathItem] {
	if o == nil {
		return nil
	}
	return o.Webhooks
}

// GetComponents returns the value of the Components field. Returns nil if not set.
func (o *OpenAPI) GetComponents() *Components {
	if o == nil {
		return nil
	}
	return o.Components
}

// GetJSONSchemaDialect returns the value of the JSONSchemaDialect field. Returns empty string if not set.
func (o *OpenAPI) GetJSONSchemaDialect() string {
	if o == nil || o.JSONSchemaDialect == nil {
		return ""
	}
	return *o.JSONSchemaDialect
}

// Validate will validate the OpenAPI object against the OpenAPI Specification.
func (o *OpenAPI) Validate(ctx context.Context, opts ...validation.Option) []error {
	if o == nil {
		return nil
	}

	core := o.GetCore()
	errs := []error{}

	opts = append(opts, validation.WithContextObject(o))
	opts = append(opts, validation.WithContextObject(&oas3.ParentDocumentVersion{OpenAPI: pointer.From(o.OpenAPI)}))

	docVersion, err := version.Parse(o.OpenAPI)
	if err != nil {
		errs = append(errs, validation.NewValueError(validation.NewValueValidationError("openapi.openapi invalid OpenAPI version %s: %s", o.OpenAPI, err.Error()), core, core.OpenAPI))
	}
	if docVersion != nil {
		if docVersion.LessThan(*MinimumSupportedVersion) || docVersion.GreaterThan(*MaximumSupportedVersion) {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("openapi.openapi only OpenAPI versions between %s and %s are supported", MinimumSupportedVersion, MaximumSupportedVersion), core, core.OpenAPI))
		}
	}

	errs = append(errs, o.Info.Validate(ctx, opts...)...)

	if o.ExternalDocs != nil {
		errs = append(errs, o.ExternalDocs.Validate(ctx, opts...)...)
	}

	for _, tag := range o.Tags {
		errs = append(errs, tag.Validate(ctx, opts...)...)
	}

	for _, server := range o.Servers {
		errs = append(errs, server.Validate(ctx, opts...)...)
	}

	for _, securityRequirement := range o.Security {
		errs = append(errs, securityRequirement.Validate(ctx, opts...)...)
	}

	if o.Paths != nil {
		errs = append(errs, o.Paths.Validate(ctx, opts...)...)
	}

	for _, webhook := range o.Webhooks.All() {
		errs = append(errs, webhook.Validate(ctx, opts...)...)
	}

	if o.Components != nil {
		errs = append(errs, o.Components.Validate(ctx, opts...)...)
	}

	if core.Self.Present && o.Self != nil {
		if _, err := url.Parse(*o.Self); err != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("openapi.$self is not a valid uri reference: %s", err), core, core.Self))
		}
	}

	if core.JSONSchemaDialect.Present && o.JSONSchemaDialect != nil {
		if _, err := url.Parse(*o.JSONSchemaDialect); err != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("openapi.jsonSchemaDialect is not a valid uri: %s", err), core, core.JSONSchemaDialect))
		}
	}

	o.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
