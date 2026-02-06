package openapi

import (
	"context"
	"fmt"
	"iter"
	"slices"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/internal/version"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

// Paths is a map of relative endpoint paths to their corresponding PathItem objects.
// Paths embeds sequencedmap.Map[string, *ReferencedPathItem] so all map operations are supported.
type Paths struct {
	marshaller.Model[core.Paths]
	*sequencedmap.Map[string, *ReferencedPathItem]

	// Extensions provides a list of extensions to the Paths object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Paths] = (*Paths)(nil)

// NewPaths creates a new Paths object with the embedded map initialized.
func NewPaths() *Paths {
	return &Paths{
		Map: sequencedmap.New[string, *ReferencedPathItem](),
	}
}

// Len returns the number of elements in the paths map. nil safe.
func (p *Paths) Len() int {
	if p == nil || p.Map == nil {
		return 0
	}
	return p.Map.Len()
}

// All returns an iterator over all path items in the paths map. nil safe.
func (p *Paths) All() iter.Seq2[string, *ReferencedPathItem] {
	if p == nil {
		return func(yield func(string, *ReferencedPathItem) bool) {}
	}
	return p.Map.All()
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (p *Paths) GetExtensions() *extensions.Extensions {
	if p == nil || p.Extensions == nil {
		return extensions.New()
	}
	return p.Extensions
}

// Validate validates the Paths object according to the OpenAPI specification.
func (p *Paths) Validate(ctx context.Context, opts ...validation.Option) []error {
	var errs []error

	for _, pathItem := range p.All() {
		errs = append(errs, pathItem.Validate(ctx, opts...)...)
	}

	p.Valid = len(errs) == 0

	return errs
}

// HTTPMethod is an enum representing the HTTP methods available in the OpenAPI specification.
type HTTPMethod string

const (
	// HTTPMethodGet represents the HTTP GET method.
	HTTPMethodGet HTTPMethod = "get"
	// HTTPMethodPut represents the HTTP PUT method.
	HTTPMethodPut HTTPMethod = "put"
	// HTTPMethodPost represents the HTTP POST method.
	HTTPMethodPost HTTPMethod = "post"
	// HTTPMethodDelete represents the HTTP DELETE method.
	HTTPMethodDelete HTTPMethod = "delete"
	// HTTPMethodOptions represents the HTTP OPTIONS method.
	HTTPMethodOptions HTTPMethod = "options"
	// HTTPMethodHead represents the HTTP HEAD method.
	HTTPMethodHead HTTPMethod = "head"
	// HTTPMethodPatch represents the HTTP PATCH method.
	HTTPMethodPatch HTTPMethod = "patch"
	// HTTPMethodTrace represents the HTTP TRACE method.
	HTTPMethodTrace HTTPMethod = "trace"
	// HTTPMethodQuery represents the HTTP QUERY method.
	HTTPMethodQuery HTTPMethod = "query"
)

var standardHttpMethods = []HTTPMethod{
	HTTPMethodGet,
	HTTPMethodPut,
	HTTPMethodPost,
	HTTPMethodDelete,
	HTTPMethodOptions,
	HTTPMethodHead,
	HTTPMethodPatch,
	HTTPMethodTrace,
	HTTPMethodQuery,
}

func (m HTTPMethod) Is(method string) bool {
	return strings.EqualFold(string(m), method)
}

func (m HTTPMethod) String() string {
	return string(m)
}

func IsStandardMethod(s string) bool {
	return slices.Contains(standardHttpMethods, HTTPMethod(s))
}

// PathItem represents the available operations for a specific endpoint path.
// PathItem embeds sequencedmap.Map[HTTPMethod, *Operation] so all map operations are supported for working with HTTP methods.
type PathItem struct {
	marshaller.Model[core.PathItem]
	*sequencedmap.Map[HTTPMethod, *Operation]

	// Summary is a short summary of the path and its operations.
	Summary *string
	// Description is a description of the path and its operations. May contain CommonMark syntax.
	Description *string

	// Servers are a list of servers that can be used by the operations represented by this path. Overrides servers defined at the root level.
	Servers []*Server
	// Parameters are a list of parameters that can be used by the operations represented by this path.
	Parameters []*ReferencedParameter

	// AdditionalOperations contains HTTP operations not covered by standard fixed fields (GET, POST, etc.).
	AdditionalOperations *sequencedmap.Map[string, *Operation]

	// Extensions provides a list of extensions to the PathItem object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.PathItem] = (*PathItem)(nil)

// NewPathItem creates a new PathItem object with the embedded map initialized.
func NewPathItem() *PathItem {
	return &PathItem{
		Map: sequencedmap.New[HTTPMethod, *Operation](),
	}
}

// Len returns the number of operations in the path item. nil safe.
func (p *PathItem) Len() int {
	if p == nil || p.Map == nil {
		return 0
	}
	return p.Map.Len()
}

// GetOperation returns the operation for the specified HTTP method.
func (p *PathItem) GetOperation(method HTTPMethod) *Operation {
	if p == nil || !p.IsInitialized() {
		return nil
	}

	op, ok := p.Map.Get(method)
	if !ok {
		return nil
	}

	return op
}

// Get returns the GET operation for this path item. Returns nil if not set.
func (p *PathItem) Get() *Operation {
	if p == nil {
		return nil
	}
	return p.GetOperation(HTTPMethodGet)
}

// Put returns the PUT operation for this path item. Returns nil if not set.
func (p *PathItem) Put() *Operation {
	if p == nil {
		return nil
	}
	return p.GetOperation(HTTPMethodPut)
}

// Post returns the POST operation for this path item. Returns nil if not set.
func (p *PathItem) Post() *Operation {
	if p == nil {
		return nil
	}
	return p.GetOperation(HTTPMethodPost)
}

// Delete returns the DELETE operation for this path item. Returns nil if not set.
func (p *PathItem) Delete() *Operation {
	if p == nil {
		return nil
	}
	return p.GetOperation(HTTPMethodDelete)
}

// Options returns the OPTIONS operation for this path item. Returns nil if not set.
func (p *PathItem) Options() *Operation {
	if p == nil {
		return nil
	}
	return p.GetOperation(HTTPMethodOptions)
}

// Head returns the HEAD operation for this path item. Returns nil if not set.
func (p *PathItem) Head() *Operation {
	if p == nil {
		return nil
	}
	return p.GetOperation(HTTPMethodHead)
}

// Patch returns the PATCH operation for this path item. Returns nil if not set.
func (p *PathItem) Patch() *Operation {
	if p == nil {
		return nil
	}
	return p.GetOperation(HTTPMethodPatch)
}

// Trace returns the TRACE operation for this path item. Returns nil if not set.
func (p *PathItem) Trace() *Operation {
	if p == nil {
		return nil
	}
	return p.GetOperation(HTTPMethodTrace)
}

// Query returns the QUERY operation for this path item. Returns nil if not set.
func (p *PathItem) Query() *Operation {
	if p == nil {
		return nil
	}
	return p.GetOperation(HTTPMethodQuery)
}

// GetAdditionalOperations returns the value of the AdditionalOperations field. Returns nil if not set.
func (p *PathItem) GetAdditionalOperations() *sequencedmap.Map[string, *Operation] {
	if p == nil {
		return nil
	}
	return p.AdditionalOperations
}

// GetSummary returns the value of the Summary field. Returns empty string if not set.
func (p *PathItem) GetSummary() string {
	if p == nil || p.Summary == nil {
		return ""
	}
	return *p.Summary
}

// GetServers returns the value of the Servers field. Returns nil if not set.
func (p *PathItem) GetServers() []*Server {
	if p == nil {
		return nil
	}
	return p.Servers
}

// GetParameters returns the value of the Parameters field. Returns nil if not set.
func (p *PathItem) GetParameters() []*ReferencedParameter {
	if p == nil {
		return nil
	}
	return p.Parameters
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (p *PathItem) GetExtensions() *extensions.Extensions {
	if p == nil || p.Extensions == nil {
		return extensions.New()
	}
	return p.Extensions
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (p *PathItem) GetDescription() string {
	if p == nil || p.Description == nil {
		return ""
	}
	return *p.Description
}

// Validate validates the PathItem object according to the OpenAPI specification.
func (p *PathItem) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := p.GetCore()
	errs := []error{}

	o := validation.NewOptions(opts...)

	oa := validation.GetContextObject[OpenAPI](o)
	// If OpenAPI object is not provided, assume the latest version
	openapiVersion := Version
	if oa != nil {
		openapiVersion = oa.OpenAPI
	}

	for _, op := range p.All() {
		errs = append(errs, op.Validate(ctx, opts...)...)
	}

	for _, server := range p.Servers {
		errs = append(errs, server.Validate(ctx, opts...)...)
	}

	for _, parameter := range p.Parameters {
		errs = append(errs, parameter.Validate(ctx, opts...)...)
	}

	supportsAdditionalOperations, err := version.IsGreaterOrEqual(openapiVersion, Version)
	switch {
	case err != nil:
		errs = append(errs, err)

	case supportsAdditionalOperations:
		if p.AdditionalOperations != nil {
			for methodName, op := range p.AdditionalOperations.All() {
				errs = append(errs, op.Validate(ctx, opts...)...)
				if IsStandardMethod(strings.ToLower(methodName)) {
					errs = append(errs, validation.NewMapKeyError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("pathItem.additionalOperations method [%s] is a standardized HTTP method and must be defined in its own field", methodName), core, core.AdditionalOperations, methodName))
				}
			}
		}

		for methodName := range p.Keys() {
			if !IsStandardMethod(strings.ToLower(string(methodName))) {
				errs = append(errs, validation.NewMapKeyError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("pathItem method [%s] is not a standardized HTTP method and must be listed under additionalOperations", methodName), core, core, methodName.String()))
			}
		}

	case !supportsAdditionalOperations:
		if core.AdditionalOperations.Present {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationSupportedVersion, fmt.Errorf("pathItem.additionalOperations is not supported in OpenAPI version %s", openapiVersion), core, core.AdditionalOperations))
		}
	}

	p.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
