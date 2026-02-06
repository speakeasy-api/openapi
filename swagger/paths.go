package swagger

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/swagger/core"
	"github.com/speakeasy-api/openapi/validation"
)

// Paths holds the relative paths to the individual endpoints.
type Paths struct {
	marshaller.Model[core.Paths]
	*sequencedmap.Map[string, *PathItem]

	// Extensions provides a list of extensions to the Paths object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Paths] = (*Paths)(nil)

// NewPaths creates a new Paths object with an initialized map.
func NewPaths() *Paths {
	return &Paths{
		Map: sequencedmap.New[string, *PathItem](),
	}
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (p *Paths) GetExtensions() *extensions.Extensions {
	if p == nil || p.Extensions == nil {
		return extensions.New()
	}
	return p.Extensions
}

// Validate validates the Paths object according to the Swagger Specification.
func (p *Paths) Validate(ctx context.Context, opts ...validation.Option) []error {
	c := p.GetCore()
	errs := []error{}

	// Validate that path keys start with a slash
	for path, pathItem := range p.All() {
		if !strings.HasPrefix(path, "/") {
			pathKeyNode := c.GetMapKeyNodeOrRoot(path, c.RootNode)
			errs = append(errs, validation.NewValidationError(
				validation.SeverityError,
				validation.RuleValidationInvalidSyntax,
				fmt.Errorf("path `%s` must begin with a slash '/'", path),
				pathKeyNode))
		}
		errs = append(errs, pathItem.Validate(ctx, opts...)...)
	}

	p.Valid = len(errs) == 0

	return errs
}

// HTTPMethod is an enum representing the HTTP methods available in the Swagger specification.
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
)

// PathItem describes the operations available on a single path.
type PathItem struct {
	marshaller.Model[core.PathItem]
	*sequencedmap.Map[HTTPMethod, *Operation]

	// Ref allows for an external definition of this path item.
	Ref *string
	// Parameters is a list of parameters that are applicable for all operations in this path.
	Parameters []*ReferencedParameter
	// Extensions provides a list of extensions to the PathItem object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.PathItem] = (*PathItem)(nil)

// NewPathItem creates a new PathItem object with an initialized map.
func NewPathItem() *PathItem {
	return &PathItem{
		Map: sequencedmap.New[HTTPMethod, *Operation](),
	}
}

// GetRef returns the value of the Ref field. Returns empty string if not set.
func (p *PathItem) GetRef() string {
	if p == nil || p.Ref == nil {
		return ""
	}
	return *p.Ref
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

// Validate validates the PathItem object according to the Swagger Specification.
func (p *PathItem) Validate(ctx context.Context, opts ...validation.Option) []error {
	c := p.GetCore()
	errs := []error{}

	// TODO allow validation of parameter uniqueness and body parameter count, this isn't done at the moment as we would need to resolve references
	for _, parameter := range p.Parameters {
		errs = append(errs, parameter.Validate(ctx, opts...)...)
	}

	for _, op := range p.All() {
		errs = append(errs, op.Validate(ctx, opts...)...)
	}

	p.Valid = len(errs) == 0 && c.GetValid()

	return errs
}
