package openapi

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"net/url"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/validation"
)

// Info provides various information about the API and document.
type Info struct {
	marshaller.Model[core.Info]

	// The title of the API.
	Title string
	// The version of this OpenAPI document, distinct from the API version.
	Version string
	// A short summary describing the API.
	Summary *string
	// A description of the API. May contain CommonMark syntax.
	Description *string
	// A URI to the Terms of Service for the API. It MUST be in the format of a URI.
	TermsOfService *string
	// Contact information for the documented API.
	Contact *Contact
	// The license information for the API.
	License *License
	// Extensions provides a list of extensions to the Info object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Info] = (*Info)(nil)

// GetTitle returns the value of the Title field. Returns empty string if not set.
func (i *Info) GetTitle() string {
	if i == nil {
		return ""
	}
	return i.Title
}

// GetVersion returns the value of the Version field. Returns empty string if not set.
func (i *Info) GetVersion() string {
	if i == nil {
		return ""
	}
	return i.Version
}

// GetSummary returns the value of the Summary field. Returns empty string if not set.
func (i *Info) GetSummary() string {
	if i == nil || i.Summary == nil {
		return ""
	}
	return *i.Summary
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (i *Info) GetDescription() string {
	if i == nil || i.Description == nil {
		return ""
	}
	return *i.Description
}

// GetTermsOfService returns the value of the TermsOfService field. Returns empty string if not set.
func (i *Info) GetTermsOfService() string {
	if i == nil || i.TermsOfService == nil {
		return ""
	}
	return *i.TermsOfService
}

// GetContact returns the value of the Contact field. Returns nil if not set.
func (i *Info) GetContact() *Contact {
	if i == nil {
		return nil
	}
	return i.Contact
}

// GetLicense returns the value of the License field. Returns nil if not set.
func (i *Info) GetLicense() *License {
	if i == nil {
		return nil
	}
	return i.License
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (i *Info) GetExtensions() *extensions.Extensions {
	if i == nil || i.Extensions == nil {
		return extensions.New()
	}
	return i.Extensions
}

// Validate will validate the Info object against the OpenAPI Specification.
func (i *Info) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := i.GetCore()
	errs := []error{}

	if core.Title.Present && i.Title == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("info.title is required"), core, core.Title))
	}

	if core.Version.Present && i.Version == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("info.version is required"), core, core.Version))
	}

	if core.TermsOfService.Present {
		if _, err := url.Parse(*i.TermsOfService); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("info.termsOfService is not a valid uri: %w", err), core, core.TermsOfService))
		}
	}

	if core.Contact.Present {
		errs = append(errs, i.Contact.Validate(ctx, opts...)...)
	}
	if core.License.Present {
		errs = append(errs, i.License.Validate(ctx, opts...)...)
	}

	i.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// Contact information for the documented API.
type Contact struct {
	marshaller.Model[core.Contact]

	// Name is the identifying name of the contact person/organization for the API.
	Name *string
	// URL is the URL for the contact person/organization. It MUST be in the format of a URI.
	URL *string
	// Email is the email address for the contact person/organization. It MUST be in the format of an email address.
	Email *string
	// Extensions provides a list of extensions to the Contact object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Contact] = (*Contact)(nil)

// GetName returns the value of the Name field. Returns empty string if not set.
func (c *Contact) GetName() string {
	if c == nil || c.Name == nil {
		return ""
	}
	return *c.Name
}

// GetURL returns the value of the URL field. Returns empty string if not set.
func (c *Contact) GetURL() string {
	if c == nil || c.URL == nil {
		return ""
	}
	return *c.URL
}

// GetEmail returns the value of the Email field. Returns empty string if not set.
func (c *Contact) GetEmail() string {
	if c == nil || c.Email == nil {
		return ""
	}
	return *c.Email
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (c *Contact) GetExtensions() *extensions.Extensions {
	if c == nil || c.Extensions == nil {
		return extensions.New()
	}
	return c.Extensions
}

// Validate will validate the Contact object against the OpenAPI Specification.
func (c *Contact) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := c.GetCore()
	errs := []error{}

	if core.URL.Present {
		if _, err := url.Parse(*c.URL); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("contact.url is not a valid uri: %w", err), core, core.URL))
		}
	}

	if core.Email.Present {
		if _, err := mail.ParseAddress(*c.Email); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("contact.email is not a valid email address: %w", err), core, core.Email))
		}
	}

	c.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// License information for the documented API.
type License struct {
	marshaller.Model[core.License]

	// Name is the name of the license used for the API.
	Name string
	// A SPDX license identifier for the license used for the API. This.is mutually exclusive of the URL field.
	Identifier *string
	// URL is the URL to the license used for the API. It MUST be in the format of a URI. This.is mutually exclusive of the Identifier field.
	URL *string
	// Extensions provides a list of extensions to the License object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.License] = (*License)(nil)

// GetName returns the value of the Name field. Returns empty string if not set.
func (l *License) GetName() string {
	if l == nil {
		return ""
	}
	return l.Name
}

// GetIdentifier returns the value of the Identifier field. Returns empty string if not set.
func (l *License) GetIdentifier() string {
	if l == nil || l.Identifier == nil {
		return ""
	}
	return *l.Identifier
}

// GetURL returns the value of the URL field. Returns empty string if not set.
func (l *License) GetURL() string {
	if l == nil || l.URL == nil {
		return ""
	}
	return *l.URL
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (l *License) GetExtensions() *extensions.Extensions {
	if l == nil || l.Extensions == nil {
		return extensions.New()
	}
	return l.Extensions
}

// Validate will validate the License object against the OpenAPI Specification.
func (l *License) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := l.GetCore()
	errs := []error{}

	if core.Name.Present && l.Name == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("license.name is required"), core, core.Name))
	}

	if core.URL.Present {
		if _, err := url.Parse(*l.URL); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("license.url is not a valid uri: %w", err), core, core.URL))
		}
	}

	l.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
