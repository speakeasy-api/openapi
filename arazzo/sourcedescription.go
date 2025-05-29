package arazzo

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/validation"
)

// SourceDescriptions represents a list of SourceDescription objects that describe the source of the data that the workflow is orchestrating.
type SourceDescriptions []*SourceDescription

// Find will return the first SourceDescription object with the provided name.
func (s SourceDescriptions) Find(name string) *SourceDescription {
	for _, sourceDescription := range s {
		if sourceDescription.Name == name {
			return sourceDescription
		}
	}
	return nil
}

// SourceDescriptionType represents the type of the SourceDescription object.
type SourceDescriptionType string

const (
	// SourceDescriptionTypeOpenAPI represents a SourceDescription that describes an OpenAPI document.
	SourceDescriptionTypeOpenAPI = "openapi"
	// SourceDescriptionTypeArazzo represents a SourceDescription that describes an Arazzo document.
	SourceDescriptionTypeArazzo = "arazzo"
)

// SourceDescription represents an Arazzo or OpenAPI document that is referenced by this Arazzo document.
type SourceDescription struct {
	// Name is the case-sensitive name of the SourceDescription object used to reference it.
	Name string
	// URL is a URL or relative URI to the location of the referenced document.
	URL string
	// Type is the type of the referenced document.
	Type SourceDescriptionType
	// Extensions provides a list of extensions to the SourceDescription object.
	Extensions *extensions.Extensions

	// Valid indicates whether this model passed validation.
	Valid bool

	core core.SourceDescription
}

var _ interfaces.Model[core.SourceDescription] = (*SourceDescription)(nil)

// GetCore will return the low level representation of the source description object.
// Useful for accessing line and column numbers for various nodes in the backing yaml/json document.
func (s *SourceDescription) GetCore() *core.SourceDescription {
	return &s.core
}

// Validate will validate the source description object against the Arazzo specification.
func (s *SourceDescription) Validate(ctx context.Context, opts ...validation.Option) []error {
	errs := []error{}

	if s.core.Name.Present && s.Name == "" {
		errs = append(errs, &validation.Error{
			Message: "name is required",
			Line:    s.core.Name.GetValueNodeOrRoot(s.core.RootNode).Line,
			Column:  s.core.Name.GetValueNodeOrRoot(s.core.RootNode).Column,
		})
	}

	if s.core.URL.Present && s.URL == "" {
		errs = append(errs, &validation.Error{
			Message: "url is required",
			Line:    s.core.URL.GetValueNodeOrRoot(s.core.RootNode).Line,
			Column:  s.core.URL.GetValueNodeOrRoot(s.core.RootNode).Column,
		})
	} else if s.core.URL.Present {
		if _, err := url.Parse(s.URL); err != nil {
			errs = append(errs, &validation.Error{
				Message: fmt.Sprintf("url is not a valid url/uri according to RFC 3986: %s", err),
				Line:    s.core.URL.GetValueNodeOrRoot(s.core.RootNode).Line,
				Column:  s.core.URL.GetValueNodeOrRoot(s.core.RootNode).Column,
			})
		}
	}

	switch s.Type {
	case SourceDescriptionTypeOpenAPI:
	case SourceDescriptionTypeArazzo:
	default:
		errs = append(errs, &validation.Error{
			Message: fmt.Sprintf("type must be one of [%s]", strings.Join([]string{SourceDescriptionTypeOpenAPI, SourceDescriptionTypeArazzo}, ", ")),
			Line:    s.core.Type.GetValueNodeOrRoot(s.core.RootNode).Line,
			Column:  s.core.Type.GetValueNodeOrRoot(s.core.RootNode).Column,
		})
	}

	if len(errs) == 0 {
		s.Valid = true
	}

	return errs
}
