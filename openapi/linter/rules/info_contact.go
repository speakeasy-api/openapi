package rules

import (
	"context"
	"errors"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleInfoContact = "style-info-contact"

type InfoContactRule struct{}

func (r *InfoContactRule) ID() string       { return RuleStyleInfoContact }
func (r *InfoContactRule) Category() string { return CategoryStyle }
func (r *InfoContactRule) Description() string {
	return "The info section should include a contact object with details for reaching the API team. Providing contact information helps API consumers get support, report issues, and connect with maintainers when needed."
}
func (r *InfoContactRule) Summary() string {
	return "The info section should include a contact object for API support."
}
func (r *InfoContactRule) HowToFix() string {
	return "Add an info.contact object with support details."
}
func (r *InfoContactRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-info-contact"
}
func (r *InfoContactRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *InfoContactRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *InfoContactRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	var errs []error
	doc := docInfo.Document

	info := doc.GetInfo()
	if info == nil {
		// No info object - should be caught by validation
		return nil
	}

	contact := info.GetContact()
	if contact == nil {
		errs = append(errs, validation.NewValidationError(
			config.GetSeverity(r.DefaultSeverity()),
			RuleStyleInfoContact,
			errors.New("info section is missing contact details"),
			info.GetRootNode(),
		))
	}

	return errs
}
