package asyncapi

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"net/url"

	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	jsonschema "github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
)

// ExternalDocumentation allows referencing an external resource for extended documentation, or is a Reference Object to one.
type ExternalDocumentation = jsonschema.ExternalDocumentation

// Info provides metadata about the API.
type Info struct {
	marshaller.Model[core.Info]

	// Title is the title of the application.
	Title string
	// Version is the version of the application API, distinct from the AsyncAPI Specification version.
	Version string
	// Description is a description of the application. May contain CommonMark syntax.
	Description *string
	// TermsOfService is a URL to the Terms of Service for the API.
	TermsOfService *string
	// Contact is the contact information for the exposed API.
	Contact *Contact
	// License is the license information for the exposed API.
	License *License
	// Tags are used for logical grouping and categorization of the application.
	Tags []*Tag
	// ExternalDocs is additional external documentation for the application.
	ExternalDocs *ExternalDocumentation
	// Extensions provides a list of extensions to the Info Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Info] = (*Info)(nil)

// GetTitle returns the title, or an empty string if not set.
func (i *Info) GetTitle() string {
	if i == nil {
		return ""
	}
	return i.Title
}

// GetVersion returns the application API version, or an empty string if not set.
func (i *Info) GetVersion() string {
	if i == nil {
		return ""
	}
	return i.Version
}

// GetDescription returns the description, or an empty string if not set.
func (i *Info) GetDescription() string {
	if i == nil || i.Description == nil {
		return ""
	}
	return *i.Description
}

// GetTermsOfService returns the Terms of Service URL, or an empty string if not set.
func (i *Info) GetTermsOfService() string {
	if i == nil || i.TermsOfService == nil {
		return ""
	}
	return *i.TermsOfService
}

// GetContact returns the contact information, or nil if not set.
func (i *Info) GetContact() *Contact {
	if i == nil {
		return nil
	}
	return i.Contact
}

// GetLicense returns the license information, or nil if not set.
func (i *Info) GetLicense() *License {
	if i == nil {
		return nil
	}
	return i.License
}

// GetTags returns the tags, or nil if not set.
func (i *Info) GetTags() []*Tag {
	if i == nil {
		return nil
	}
	return i.Tags
}

// GetExternalDocs returns the external documentation, or nil if not set.
func (i *Info) GetExternalDocs() *ExternalDocumentation {
	if i == nil {
		return nil
	}
	return i.ExternalDocs
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (i *Info) GetExtensions() *extensions.Extensions {
	if i == nil || i.Extensions == nil {
		return extensions.New()
	}
	return i.Extensions
}

// Validate will validate the Info Object against the AsyncAPI Specification.
func (i *Info) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := i.GetCore()
	errs := []error{}

	if core.Title.Present && i.Title == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`info.title` is required"), core, core.Title))
	}
	if core.Version.Present && i.Version == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`info.version` is required"), core, core.Version))
	}

	if i.TermsOfService != nil {
		if _, err := url.Parse(*i.TermsOfService); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("info.termsOfService is not a valid uri: %w", err), core, core.TermsOfService))
		}
	}

	if i.Contact != nil {
		errs = append(errs, i.Contact.Validate(ctx, opts...)...)
	}
	if i.License != nil {
		errs = append(errs, i.License.Validate(ctx, opts...)...)
	}

	tagNames := map[string]bool{}
	for idx, tag := range i.Tags {
		if tag == nil {
			continue
		}
		errs = append(errs, tag.Validate(ctx, opts...)...)
		if tag.IsReference() || tag.Name == nil {
			continue
		}
		if tagNames[*tag.Name] {
			errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationDuplicateKey, fmt.Errorf("info.tags name `%s` is not unique", *tag.Name), core, core.Tags, idx))
		}
		tagNames[*tag.Name] = true
	}

	if i.ExternalDocs != nil {
		errs = append(errs, i.ExternalDocs.Validate(ctx, opts...)...)
	}

	i.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// Contact is the contact information for the exposed API.
type Contact struct {
	marshaller.Model[core.Contact]

	// Name is the identifying name of the contact person or organization.
	Name *string
	// URL is the URL pointing to the contact information.
	URL *string
	// Email is the email address of the contact person or organization.
	Email *string
	// Extensions provides a list of extensions to the Contact Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Contact] = (*Contact)(nil)

// GetName returns the name, or an empty string if not set.
func (c *Contact) GetName() string {
	if c == nil || c.Name == nil {
		return ""
	}
	return *c.Name
}

// GetURL returns the URL, or an empty string if not set.
func (c *Contact) GetURL() string {
	if c == nil || c.URL == nil {
		return ""
	}
	return *c.URL
}

// GetEmail returns the email address, or an empty string if not set.
func (c *Contact) GetEmail() string {
	if c == nil || c.Email == nil {
		return ""
	}
	return *c.Email
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (c *Contact) GetExtensions() *extensions.Extensions {
	if c == nil || c.Extensions == nil {
		return extensions.New()
	}
	return c.Extensions
}

// Validate will validate the Contact Object against the AsyncAPI Specification.
func (c *Contact) Validate(_ context.Context, _ ...validation.Option) []error {
	core := c.GetCore()
	errs := []error{}

	if c.URL != nil {
		if _, err := url.Parse(*c.URL); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("contact.url is not a valid uri: %w", err), core, core.URL))
		}
	}
	if c.Email != nil {
		if _, err := mail.ParseAddress(*c.Email); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("contact.email is not a valid email address: %w", err), core, core.Email))
		}
	}

	c.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// License is the license information for the exposed API.
type License struct {
	marshaller.Model[core.License]

	// Name is the license name used for the API.
	Name string
	// URL is a URL to the license used for the API.
	URL *string
	// Extensions provides a list of extensions to the License Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.License] = (*License)(nil)

// GetName returns the license name, or an empty string if not set.
func (l *License) GetName() string {
	if l == nil {
		return ""
	}
	return l.Name
}

// GetURL returns the license URL, or an empty string if not set.
func (l *License) GetURL() string {
	if l == nil || l.URL == nil {
		return ""
	}
	return *l.URL
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (l *License) GetExtensions() *extensions.Extensions {
	if l == nil || l.Extensions == nil {
		return extensions.New()
	}
	return l.Extensions
}

// Validate will validate the License Object against the AsyncAPI Specification.
func (l *License) Validate(_ context.Context, _ ...validation.Option) []error {
	core := l.GetCore()
	errs := []error{}

	if core.Name.Present && l.Name == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`license.name` is required"), core, core.Name))
	}
	if l.URL != nil {
		if _, err := url.Parse(*l.URL); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("license.url is not a valid uri: %w", err), core, core.URL))
		}
	}

	l.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// Tag allows adding metadata to a single tag, or is a Reference Object to a Tag Object.
type Tag struct {
	marshaller.Model[core.Tag]

	// Ref is set when this is a Reference Object, in which case the other fields are ignored.
	Ref *references.Reference
	// Name is the name of the tag. Required unless this is a Reference Object.
	Name *string
	// Description is a description of the tag. May contain CommonMark syntax.
	Description *string
	// ExternalDocs is additional external documentation for the tag.
	ExternalDocs *ExternalDocumentation
	// Extensions provides a list of extensions to the Tag Object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Tag] = (*Tag)(nil)

// IsReference returns true if the tag is a Reference Object.
func (t *Tag) IsReference() bool {
	return t != nil && t.Ref != nil
}

// GetName returns the tag name, or an empty string if not set.
func (t *Tag) GetName() string {
	if t == nil || t.Name == nil {
		return ""
	}
	return *t.Name
}

// GetDescription returns the description, or an empty string if not set.
func (t *Tag) GetDescription() string {
	if t == nil || t.Description == nil {
		return ""
	}
	return *t.Description
}

// GetExtensions returns the extensions. Returns an empty extensions map if not set.
func (t *Tag) GetExtensions() *extensions.Extensions {
	if t == nil || t.Extensions == nil {
		return extensions.New()
	}
	return t.Extensions
}

// Validate will validate the Tag Object against the AsyncAPI Specification.
func (t *Tag) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := t.GetCore()
	errs := []error{}

	switch {
	case t.Ref != nil:
		if err := t.Ref.Validate(); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidReference, err, core, core.Ref))
		}
	case t.Name == nil || *t.Name == "":
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`tag.name` is required"), core, core.Name))
	default:
		if t.ExternalDocs != nil {
			errs = append(errs, t.ExternalDocs.Validate(ctx, opts...)...)
		}
	}

	t.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
