package rules

import (
	"context"
	"errors"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleInfoDescription = "style-info-description"

type InfoDescriptionRule struct{}

func (r *InfoDescriptionRule) ID() string       { return RuleStyleInfoDescription }
func (r *InfoDescriptionRule) Category() string { return CategoryStyle }
func (r *InfoDescriptionRule) Description() string {
	return "The info section should include a description field that explains the purpose and capabilities of the API. A well-written description helps developers quickly understand what the API does and whether it meets their needs."
}
func (r *InfoDescriptionRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-info-description"
}
func (r *InfoDescriptionRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *InfoDescriptionRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *InfoDescriptionRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
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

	description := info.GetDescription()
	if description == "" {
		errs = append(errs, validation.NewValidationError(
			config.GetSeverity(r.DefaultSeverity()),
			RuleStyleInfoDescription,
			errors.New("info section is missing a description"),
			info.GetCore().GetRootNode(),
		))
	}

	return errs
}
