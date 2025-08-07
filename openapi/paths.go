package openapi

import (
	"context"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

// Paths is a map of relative endpoint paths to their corresponding PathItem objects.
// Paths embeds sequencedmap.Map[string, *ReferencedPathItem] so all map operations are supported.
type Paths struct {
	marshaller.Model[core.Paths]
	sequencedmap.Map[string, *ReferencedPathItem]

	// Extensions provides a list of extensions to the Paths object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Paths] = (*Paths)(nil)

// NewPaths creates a new Paths object with the embedded map initialized.
func NewPaths() *Paths {
	return &Paths{
		Map: *sequencedmap.New[string, *ReferencedPathItem](),
	}
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
)

func (m HTTPMethod) Is(method string) bool {
	return strings.EqualFold(string(m), method)
}

// PathItem represents the available operations for a specific endpoint path.
// PathItem embeds sequencedmap.Map[HTTPMethod, *Operation] so all map operations are supported for working with HTTP methods.
type PathItem struct {
	marshaller.Model[core.PathItem]
	sequencedmap.Map[HTTPMethod, *Operation]

	// Summary is a short summary of the path and its operations.
	Summary *string
	// Description is a description of the path and its operations. May contain CommonMark syntax.
	Description *string

	// Servers are a list of servers that can be used by the operations represented by this path. Overrides servers defined at the root level.
	Servers []*Server
	// Parameters are a list of parameters that can be used by the operations represented by this path.
	Parameters []*ReferencedParameter

	// Extensions provides a list of extensions to the PathItem object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.PathItem] = (*PathItem)(nil)

// NewPathItem creates a new PathItem object with the embedded map initialized.
func NewPathItem() *PathItem {
	return &PathItem{
		Map: *sequencedmap.New[HTTPMethod, *Operation](),
	}
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

// Get returns the GET operation for this path item.
func (p *PathItem) Get() *Operation {
	return p.GetOperation(HTTPMethodGet)
}

// Put returns the PUT operation for this path item.
func (p *PathItem) Put() *Operation {
	return p.GetOperation(HTTPMethodPut)
}

// Post returns the POST operation for this path item.
func (p *PathItem) Post() *Operation {
	return p.GetOperation(HTTPMethodPost)
}

// Delete returns the DELETE operation for this path item.
func (p *PathItem) Delete() *Operation {
	return p.GetOperation(HTTPMethodDelete)
}

// Options returns the OPTIONS operation for this path item.
func (p *PathItem) Options() *Operation {
	return p.GetOperation(HTTPMethodOptions)
}

// Head returns the HEAD operation for this path item.
func (p *PathItem) Head() *Operation {
	return p.GetOperation(HTTPMethodHead)
}

// Patch returns the PATCH operation for this path item.
func (p *PathItem) Patch() *Operation {
	return p.GetOperation(HTTPMethodPatch)
}

// Trace returns the TRACE operation for this path item.
func (p *PathItem) Trace() *Operation {
	return p.GetOperation(HTTPMethodTrace)
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

	for _, op := range p.All() {
		errs = append(errs, op.Validate(ctx, opts...)...)
	}

	for _, server := range p.Servers {
		errs = append(errs, server.Validate(ctx, opts...)...)
	}

	for _, parameter := range p.Parameters {
		errs = append(errs, parameter.Validate(ctx, opts...)...)
	}

	p.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
