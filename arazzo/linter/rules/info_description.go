package rules

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleInfoDescription = "style-info-description"

type InfoDescriptionRule struct{}

func (r *InfoDescriptionRule) ID() string       { return RuleStyleInfoDescription }
func (r *InfoDescriptionRule) Category() string { return CategoryStyle }
func (r *InfoDescriptionRule) Description() string {
	return "The info section should include a description that explains what the workflows in the document do and when to use them, so readers don't have to infer it from the individual steps."
}
func (r *InfoDescriptionRule) Summary() string {
	return "The info section should include a description of the document."
}
func (r *InfoDescriptionRule) HowToFix() string {
	return "Add an info.description that explains the purpose of the workflows."
}
func (r *InfoDescriptionRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/arazzo/linter/README.md#style-info-description"
}
func (r *InfoDescriptionRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *InfoDescriptionRule) Versions() []string { return nil }
func (r *InfoDescriptionRule) FixAvailable() bool { return true }

func (r *InfoDescriptionRule) Run(_ context.Context, docInfo *linter.DocumentInfo[*arazzo.Arazzo], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	info := &docInfo.Document.Info
	if hasText(info.Description) {
		return nil
	}

	return []error{&validation.Error{
		UnderlyingError: errors.New("info section is missing a description"),
		Node:            info.GetRootNode(),
		Severity:        config.GetSeverity(r.DefaultSeverity()),
		Rule:            RuleStyleInfoDescription,
		Fix:             &addInfoDescriptionFix{},
	}}
}

// addInfoDescriptionFix prompts for a description and sets it on the info section.
type addInfoDescriptionFix struct {
	description string
}

func (f *addInfoDescriptionFix) Description() string { return "Add a description to the info section" }
func (f *addInfoDescriptionFix) Interactive() bool   { return true }
func (f *addInfoDescriptionFix) Prompts() []validation.Prompt {
	return []validation.Prompt{{Type: validation.PromptFreeText, Message: "Enter a description of the workflows"}}
}

func (f *addInfoDescriptionFix) SetInput(responses []string) error {
	if len(responses) != 1 {
		return fmt.Errorf("expected 1 response, got %d", len(responses))
	}
	f.description = strings.TrimSpace(responses[0])
	return nil
}

func (f *addInfoDescriptionFix) Apply(doc any) error {
	a, ok := doc.(*arazzo.Arazzo)
	if !ok {
		return fmt.Errorf("expected *arazzo.Arazzo, got %T", doc)
	}
	if f.description == "" {
		return errors.New("description must not be empty")
	}
	a.Info.Description = &f.description
	return nil
}
