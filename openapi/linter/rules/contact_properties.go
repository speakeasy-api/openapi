package rules

import (
	"context"
	"errors"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleContactProperties = "style-contact-properties"

type ContactPropertiesRule struct{}

func (r *ContactPropertiesRule) ID() string {
	return RuleStyleContactProperties
}

func (r *ContactPropertiesRule) Description() string {
	return "The contact object in the info section should include name, url, and email properties to provide complete contact information. Having comprehensive contact details makes it easier for API consumers to reach out for support, report issues, or ask questions about the API."
}

func (r *ContactPropertiesRule) Category() string {
	return CategoryStyle
}

func (r *ContactPropertiesRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}

func (r *ContactPropertiesRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-contact-properties"
}

func (r *ContactPropertiesRule) Versions() []string {
	return nil // applies to all versions
}

func (r *ContactPropertiesRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	doc := docInfo.Document
	info := doc.GetInfo()
	if info == nil {
		return nil
	}

	contact := info.GetContact()
	if contact == nil {
		return nil
	}

	var errs []error

	name := contact.GetName()
	url := contact.GetURL()
	email := contact.GetEmail()

	if name == "" {
		errs = append(errs, validation.NewValidationError(
			config.GetSeverity(r.DefaultSeverity()),
			RuleStyleContactProperties,
			errors.New("`contact` section must contain a `name`"),
			contact.GetRootNode(),
		))
	}

	if url == "" {
		errs = append(errs, validation.NewValidationError(
			config.GetSeverity(r.DefaultSeverity()),
			RuleStyleContactProperties,
			errors.New("`contact` section must contain a `url`"),
			contact.GetRootNode(),
		))
	}

	if email == "" {
		errs = append(errs, validation.NewValidationError(
			config.GetSeverity(r.DefaultSeverity()),
			RuleStyleContactProperties,
			errors.New("`contact` section must contain an `email`"),
			contact.GetRootNode(),
		))
	}

	return errs
}
