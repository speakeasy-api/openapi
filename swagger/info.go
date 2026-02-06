package swagger

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"net/url"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/swagger/core"
	"github.com/speakeasy-api/openapi/validation"
)

// Info provides metadata about the API.
type Info struct {
	marshaller.Model[core.Info]

	// Title is the title of the application.
	Title string
	// Description is a short description of the application. GFM syntax can be used for rich text representation.
	Description *string
	// TermsOfService is the Terms of Service for the API.
	TermsOfService *string
	// Contact is the contact information for the exposed API.
	Contact *Contact
	// License is the license information for the exposed API.
	License *License
	// Version provides the version of the application API (not to be confused with the specification version).
	Version string
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

// GetVersion returns the value of the Version field. Returns empty string if not set.
func (i *Info) GetVersion() string {
	if i == nil {
		return ""
	}
	return i.Version
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (i *Info) GetExtensions() *extensions.Extensions {
	if i == nil || i.Extensions == nil {
		return extensions.New()
	}
	return i.Extensions
}

// Validate validates the Info object against the Swagger Specification.
func (i *Info) Validate(ctx context.Context, opts ...validation.Option) []error {
	c := i.GetCore()
	errs := []error{}

	if c.Title.Present && i.Title == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`info.title` is required"), c, c.Title))
	}

	if c.Version.Present && i.Version == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`info.version` is required"), c, c.Version))
	}

	if c.TermsOfService.Present {
		if _, err := url.Parse(*i.TermsOfService); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("info.termsOfService is not a valid uri: %w", err), c, c.TermsOfService))
		}
	}

	if c.Contact.Present {
		errs = append(errs, i.Contact.Validate(ctx, opts...)...)
	}

	if c.License.Present {
		errs = append(errs, i.License.Validate(ctx, opts...)...)
	}

	i.Valid = len(errs) == 0 && c.GetValid()

	return errs
}

// Contact information for the exposed API.
type Contact struct {
	marshaller.Model[core.Contact]

	// Name is the identifying name of the contact person/organization.
	Name *string
	// URL is the URL pointing to the contact information. MUST be in the format of a URL.
	URL *string
	// Email is the email address of the contact person/organization. MUST be in the format of an email address.
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

// Validate validates the Contact object against the Swagger Specification.
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

// License information for the exposed API.
type License struct {
	marshaller.Model[core.License]

	// Name is the license name used for the API.
	Name string
	// URL is a URL to the license used for the API. MUST be in the format of a URL.
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

// Validate validates the License object against the Swagger Specification.
func (l *License) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := l.GetCore()
	errs := []error{}

	if core.Name.Present && l.Name == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`license.name` is required"), core, core.Name))
	}

	if core.URL.Present {
		if _, err := url.Parse(*l.URL); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("license.url is not a valid uri: %w", err), core, core.URL))
		}
	}

	l.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
