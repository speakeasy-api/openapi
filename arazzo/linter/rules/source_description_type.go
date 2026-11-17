package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleStyleSourceDescriptionType = "style-source-description-type"

type SourceDescriptionTypeRule struct{}

func (r *SourceDescriptionTypeRule) ID() string       { return RuleStyleSourceDescriptionType }
func (r *SourceDescriptionTypeRule) Category() string { return CategoryStyle }
func (r *SourceDescriptionTypeRule) Description() string {
	return "Source descriptions should declare their type. Without it, tools have to load the document to know whether it is an OpenAPI, AsyncAPI or Arazzo document, and some checks (such as which steps may use channelPath or action) can't be made from the Arazzo document alone."
}
func (r *SourceDescriptionTypeRule) Summary() string {
	return "Source descriptions should declare their type."
}
func (r *SourceDescriptionTypeRule) HowToFix() string {
	return "Set type to `openapi`, `arazzo` or (Arazzo 1.1+) `asyncapi`."
}
func (r *SourceDescriptionTypeRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/arazzo/linter/README.md#style-source-description-type"
}
func (r *SourceDescriptionTypeRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *SourceDescriptionTypeRule) Versions() []string { return nil }
func (r *SourceDescriptionTypeRule) FixAvailable() bool { return true }

func (r *SourceDescriptionTypeRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*arazzo.Arazzo], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil {
		return nil
	}

	sources := sourceDocumentsFromContext(ctx)

	// AsyncAPI sources are only allowed from Arazzo 1.1.
	choices := []string{string(arazzo.SourceDescriptionTypeOpenAPI), string(arazzo.SourceDescriptionTypeArazzo)}
	if isAtLeast11(docInfo.Document) {
		choices = append(choices, string(arazzo.SourceDescriptionTypeAsyncAPI))
	}

	var errs []error
	for i, sd := range docInfo.Document.SourceDescriptions {
		if sd == nil || sd.Type != "" {
			continue
		}

		// When the source was loaded its type is known, so the fix can be applied automatically.
		fix := &setSourceDescriptionTypeFix{index: i, name: sd.Name, choices: choices}
		if doc := sources.Get(sd.Name); doc != nil {
			fix.sourceType = doc.Type
		}

		core := sd.GetCore()
		err := validation.NewValueError(config.GetSeverity(r.DefaultSeverity()), RuleStyleSourceDescriptionType, fmt.Errorf("sourceDescription `%s` has no type", sd.Name), core, core.Name)
		errs = append(errs, withFix(err, fix))
	}
	return errs
}

// setSourceDescriptionTypeFix sets the type of a source description, either from the loaded document or by asking
// the user.
type setSourceDescriptionTypeFix struct {
	index      int
	name       string
	choices    []string
	sourceType arazzo.SourceDescriptionType
}

func (f *setSourceDescriptionTypeFix) Description() string {
	if f.sourceType != "" {
		return fmt.Sprintf("Set the type of sourceDescription `%s` to `%s`", f.name, f.sourceType)
	}
	return fmt.Sprintf("Set the type of sourceDescription `%s`", f.name)
}
func (f *setSourceDescriptionTypeFix) Interactive() bool { return f.sourceType == "" }
func (f *setSourceDescriptionTypeFix) Prompts() []validation.Prompt {
	if !f.Interactive() {
		return nil
	}
	return []validation.Prompt{{
		Type:    validation.PromptChoice,
		Message: fmt.Sprintf("Choose the type of sourceDescription `%s`", f.name),
		Choices: f.choices,
	}}
}

func (f *setSourceDescriptionTypeFix) SetInput(responses []string) error {
	if len(responses) != 1 {
		return fmt.Errorf("expected 1 response, got %d", len(responses))
	}
	f.sourceType = arazzo.SourceDescriptionType(responses[0])
	return nil
}

func (f *setSourceDescriptionTypeFix) Apply(doc any) error {
	a, ok := doc.(*arazzo.Arazzo)
	if !ok {
		return fmt.Errorf("expected *arazzo.Arazzo, got %T", doc)
	}
	if f.index >= len(a.SourceDescriptions) || a.SourceDescriptions[f.index] == nil || a.SourceDescriptions[f.index].Name != f.name {
		return fmt.Errorf("sourceDescription `%s` not found", f.name)
	}
	a.SourceDescriptions[f.index].Type = f.sourceType
	return nil
}

func (f *setSourceDescriptionTypeFix) DescribeChange() (before, after string) {
	return "", "type: " + string(f.sourceType)
}
