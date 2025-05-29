package arazzo

import (
	"context"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/validation"
)

// Info represents metadata about the Arazzo document
type Info struct {
	// Title is the name of the Arazzo document
	Title string
	// Summary is a short description of the Arazzo document
	Summary *string
	// Description is a longer description of the Arazzo document. May contain CommonMark syntax.
	Description *string
	// Version is the version of the Arazzo document
	Version string
	// Extensions provides a list of extensions to the Info object.
	Extensions *extensions.Extensions

	// Valid indicates whether this model passed validation.
	Valid bool

	core core.Info
}

var _ interfaces.Model[core.Info] = (*Info)(nil)

// GetCore will return the low level representation of the info object.
// Useful for accessing line and column numbers for various nodes in the backing yaml/json document.
func (i *Info) GetCore() *core.Info {
	return &i.core
}

// Validate will validate the Info object against the Arazzo Specification.
func (i *Info) Validate(ctx context.Context, opts ...validation.Option) []error {
	errs := []error{}

	if i.core.Title.Present && i.Title == "" {
		errs = append(errs, validation.NewValueError("title is required", i.core, i.core.Title))
	}

	if i.core.Version.Present && i.Version == "" {
		errs = append(errs, validation.NewValueError("version is required", i.core, i.core.Version))
	}

	if len(errs) == 0 {
		i.Valid = true
	}

	return errs
}
