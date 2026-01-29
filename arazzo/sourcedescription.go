package arazzo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
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
	marshaller.Model[core.SourceDescription]

	// Name is the case-sensitive name of the SourceDescription object used to reference it.
	Name string
	// URL is a URL or relative URI to the location of the referenced document.
	URL string
	// Type is the type of the referenced document.
	Type SourceDescriptionType
	// Extensions provides a list of extensions to the SourceDescription object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.SourceDescription] = (*SourceDescription)(nil)

// Validate will validate the source description object against the Arazzo specification.
func (s *SourceDescription) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := s.GetCore()
	errs := []error{}

	if core.Name.Present && s.Name == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("sourceDescription.name is required"), core, core.Name))
	}

	if core.URL.Present && s.URL == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("sourceDescription.url is required"), core, core.URL))
	} else if core.URL.Present {
		if _, err := url.Parse(s.URL); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("sourceDescription.url is not a valid url/uri according to RFC 3986: %w", err), core, core.URL))
		}
	}

	switch s.Type {
	case SourceDescriptionTypeOpenAPI:
	case SourceDescriptionTypeArazzo:
	default:
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("sourceDescription.type must be one of [%s]", strings.Join([]string{SourceDescriptionTypeOpenAPI, SourceDescriptionTypeArazzo}, ", ")), core, core.Type))
	}

	s.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
