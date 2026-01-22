package rules

import (
	"context"
	"errors"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleLicenseURL = "style-license-url"

type LicenseURLRule struct{}

func (r *LicenseURLRule) ID() string       { return RuleStyleLicenseURL }
func (r *LicenseURLRule) Category() string { return CategoryStyle }
func (r *LicenseURLRule) Description() string {
	return "The license object should include a URL that points to the full license text. Providing a license URL allows API consumers to review the complete terms and conditions governing API usage."
}
func (r *LicenseURLRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-license-url"
}
func (r *LicenseURLRule) DefaultSeverity() validation.Severity {
	return validation.SeverityHint
}
func (r *LicenseURLRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *LicenseURLRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	var errs []error
	doc := docInfo.Document

	info := doc.GetInfo()
	if info == nil {
		return nil
	}

	license := info.GetLicense()
	if license == nil {
		// No license - covered by info-license rule
		return nil
	}

	url := license.GetURL()
	if url == "" {
		errs = append(errs, validation.NewValidationError(
			config.GetSeverity(r.DefaultSeverity()),
			RuleStyleLicenseURL,
			errors.New("license should contain a URL"),
			license.GetCore().GetRootNode(),
		))
	}

	return errs
}
