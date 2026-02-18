package openapi

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/internal/version"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"go.yaml.in/yaml/v4"
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

	// schemaRegistry stores $id and $anchor mappings for schemas in this document.
	// Used for efficient resolution of $id and $anchor references.
	schemaRegistry oas3.SchemaRegistry
}

var _ interfaces.Model[core.OpenAPI] = (*OpenAPI)(nil)
var _ oas3.SchemaRegistryProvider = (*OpenAPI)(nil)

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

// GetSchemaRegistry returns the schema registry for this document.
// The registry stores $id and $anchor mappings for efficient schema resolution.
// If the registry has not been initialized, it creates one with the document's base URI.
func (o *OpenAPI) GetSchemaRegistry() oas3.SchemaRegistry {
	if o == nil {
		return nil
	}

	// Lazily initialize the registry if needed
	if o.schemaRegistry == nil {
		o.schemaRegistry = oas3.NewSchemaRegistry(o.GetDocumentBaseURI())
	}

	return o.schemaRegistry
}

// GetDocumentBaseURI returns the base URI for this document.
// This is used as the default base for resolving relative $id and $ref values.
// Returns the value of the $self field if set, otherwise empty string.
func (o *OpenAPI) GetDocumentBaseURI() string {
	if o == nil {
		return ""
	}
	return o.GetSelf()
}

// SetSchemaRegistry sets the schema registry for this document.
// This is primarily used during unmarshalling to set a pre-created registry.
func (o *OpenAPI) SetSchemaRegistry(registry oas3.SchemaRegistry) {
	if o == nil {
		return
	}
	o.schemaRegistry = registry
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
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationSupportedVersion, fmt.Errorf("openapi.openapi invalid OpenAPI version %s: %w", o.OpenAPI, err), core, core.OpenAPI))
	}
	if docVersion != nil {
		if docVersion.LessThan(*MinimumSupportedVersion) || docVersion.GreaterThan(*MaximumSupportedVersion) {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationSupportedVersion, fmt.Errorf("openapi.openapi only OpenAPI versions between %s and %s are supported", MinimumSupportedVersion, MaximumSupportedVersion), core, core.OpenAPI))
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
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("openapi.$self is not a valid uri reference: %w", err), core, core.Self))
		}
	}

	if core.JSONSchemaDialect.Present && o.JSONSchemaDialect != nil {
		if _, err := url.Parse(*o.JSONSchemaDialect); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("`openapi.jsonSchemaDialect` is not a valid uri: %w", err), core, core.JSONSchemaDialect))
		}
	}

	operationIdErrs := validateOperationIDUniqueness(ctx, o)
	errs = append(errs, operationIdErrs...)

	operationParameterErrs := validateOperationParameterUniqueness(ctx, o)
	errs = append(errs, operationParameterErrs...)

	o.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

func validateOperationIDUniqueness(ctx context.Context, doc *OpenAPI) []error {
	if doc == nil {
		return nil
	}

	seen := make(map[string]struct{})
	var errs []error

	for item := range Walk(ctx, doc) {
		if err := item.Match(Matcher{
			Operation: func(op *Operation) error {
				method, path := ExtractMethodAndPath(item.Location)
				if method == "" || path == "" {
					return nil
				}

				operationID := op.GetOperationID()
				if operationID == "" {
					return nil
				}

				if _, ok := seen[operationID]; ok {
					errNode := getOperationIDValueNode(op)
					if errNode == nil {
						errNode = op.GetRootNode()
					}
					err := validation.NewValidationError(
						validation.SeverityError,
						validation.RuleValidationOperationIdUnique,
						fmt.Errorf("the `%s` operation at path `%s` contains a duplicate operationId `%s`", method, path, operationID),
						errNode,
					)
					errs = append(errs, err)
					return nil
				}

				seen[operationID] = struct{}{}
				return nil
			},
		}); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func getOperationIDValueNode(op *Operation) *yaml.Node {
	if op == nil {
		return nil
	}

	core := op.GetCore()
	if core == nil || !core.OperationID.Present {
		return nil
	}

	return core.OperationID.ValueNode
}

// validateParameterUniqueness checks for duplicate parameters in a list
// methodOrLevel should be the HTTP method (GET, POST, etc.) or "TOP" for path-level
func validateParameterUniqueness(parameters []*ReferencedParameter, methodOrLevel, path string, fallbackNode *yaml.Node) []error {
	if len(parameters) == 0 {
		return nil
	}

	var errs []error
	seen := make(map[string]bool)

	for _, paramRef := range parameters {
		param := paramRef.GetObject()
		if param == nil {
			continue
		}

		paramName := param.GetName()
		paramIn := param.GetIn().String()
		if paramName == "" || paramIn == "" {
			continue
		}

		key := paramName + "::" + paramIn
		if seen[key] {
			core := param.GetCore()
			errNode := core.GetRootNode()
			if errNode == nil {
				errNode = fallbackNode
			}

			var errMsg string
			if methodOrLevel == "TOP" {
				errMsg = fmt.Sprintf("parameter %q is duplicated in path %q", paramName, path)
			} else {
				errMsg = fmt.Sprintf("parameter %q is duplicated in %s operation at path %q", paramName, methodOrLevel, path)
			}

			err := validation.NewValidationError(
				validation.SeverityError,
				validation.RuleValidationOperationParameters,
				fmt.Errorf("%s", errMsg),
				errNode,
			)
			errs = append(errs, err)
		}
		seen[key] = true
	}

	return errs
}

func validateOperationParameterUniqueness(ctx context.Context, doc *OpenAPI) []error {
	if doc == nil {
		return nil
	}

	var errs []error

	for item := range Walk(ctx, doc) {
		if err := item.Match(Matcher{
			// Check duplicate parameters at Operation level
			Operation: func(op *Operation) error {
				method, path := ExtractMethodAndPath(item.Location)
				if method == "" || path == "" {
					return nil
				}

				paramErrs := validateParameterUniqueness(
					op.GetParameters(),
					strings.ToUpper(method),
					path,
					op.GetRootNode(),
				)
				errs = append(errs, paramErrs...)

				return nil
			},
			// Check duplicate parameters at PathItem level
			ReferencedPathItem: func(refPathItem *ReferencedPathItem) error {
				pathItem := refPathItem.GetObject()
				if pathItem == nil {
					return nil
				}

				// Get the path from the location (parent key)
				path := item.Location.ParentKey()
				if path == "" {
					return nil
				}

				paramErrs := validateParameterUniqueness(
					pathItem.Parameters,
					"TOP",
					path,
					pathItem.GetRootNode(),
				)
				errs = append(errs, paramErrs...)

				return nil
			},
		}); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}
