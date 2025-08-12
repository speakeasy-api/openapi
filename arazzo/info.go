package arazzo

import (
	"context"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
)

// Info represents metadata about the Arazzo document
type Info struct {
	marshaller.Model[core.Info]

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
}

var _ interfaces.Model[core.Info] = (*Info)(nil)

// Validate will validate the Info object against the Arazzo Specification.
func (i *Info) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := i.GetCore()
	errs := []error{}

	if core.Title.Present && i.Title == "" {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("info field title is required"), core, core.Title))
	}

	if core.Version.Present && i.Version == "" {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("info field version is required"), core, core.Version))
	}

	i.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
