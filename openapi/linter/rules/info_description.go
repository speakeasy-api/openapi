package rules

import (
	"context"
	"errors"
	"fmt"

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
func (r *InfoDescriptionRule) Summary() string {
	return "The info section should include a description field for the API."
}
func (r *InfoDescriptionRule) HowToFix() string {
	return "Add a concise info.description that explains the API's purpose."
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
		errs = append(errs, &validation.Error{
			UnderlyingError: errors.New("info section is missing a description"),
			Node:            info.GetRootNode(),
			Severity:        config.GetSeverity(r.DefaultSeverity()),
			Rule:            RuleStyleInfoDescription,
			Fix:             &addInfoDescriptionFix{},
		})
	}

	return errs
}

// addInfoDescriptionFix prompts the user for a description and sets it on the info object.
type addInfoDescriptionFix struct {
	description string
}

func (f *addInfoDescriptionFix) Description() string { return "Add a description to the info section" }
func (f *addInfoDescriptionFix) Interactive() bool   { return true }
func (f *addInfoDescriptionFix) Prompts() []validation.Prompt {
	return []validation.Prompt{
		{
			Type:    validation.PromptFreeText,
			Message: "Enter an API description",
		},
	}
}

func (f *addInfoDescriptionFix) SetInput(responses []string) error {
	if len(responses) != 1 {
		return fmt.Errorf("expected 1 response, got %d", len(responses))
	}
	f.description = responses[0]
	return nil
}

func (f *addInfoDescriptionFix) Apply(doc any) error {
	oasDoc, ok := doc.(*openapi.OpenAPI)
	if !ok {
		return fmt.Errorf("expected *openapi.OpenAPI, got %T", doc)
	}
	info := oasDoc.GetInfo()
	if info == nil {
		return errors.New("document has no info section")
	}
	info.Description = &f.description
	return nil
}
