package rules

import (
	"context"
	"errors"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleInfoLicense = "style-info-license"

type InfoLicenseRule struct{}

func (r *InfoLicenseRule) ID() string       { return RuleStyleInfoLicense }
func (r *InfoLicenseRule) Category() string { return CategoryStyle }
func (r *InfoLicenseRule) Description() string {
	return "The info section should include a license object that specifies the terms under which the API can be used. Clearly stating the license helps API consumers understand their rights and obligations when integrating with your API."
}
func (r *InfoLicenseRule) Summary() string {
	return "The info section should include a license object."
}
func (r *InfoLicenseRule) HowToFix() string {
	return "Add an info.license object describing the API license."
}
func (r *InfoLicenseRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-info-license"
}
func (r *InfoLicenseRule) DefaultSeverity() validation.Severity {
	return validation.SeverityHint
}
func (r *InfoLicenseRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *InfoLicenseRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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

	license := info.GetLicense()
	if license == nil {
		errs = append(errs, validation.NewValidationError(
			config.GetSeverity(r.DefaultSeverity()),
			RuleStyleInfoLicense,
			errors.New("info section should contain a license"),
			info.GetRootNode(),
		))
	}

	return errs
}
