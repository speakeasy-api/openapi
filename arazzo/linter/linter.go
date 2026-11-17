// Package linter lints Arazzo documents for semantic issues, style and security concerns beyond what
// arazzo.Validate reports, using the shared linter engine, configuration and output formats.
package linter

import (
	"context"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/linter/rules"
	baseLinter "github.com/speakeasy-api/openapi/linter"
)

// Linter lints Arazzo documents.
type Linter struct {
	base *baseLinter.Linter[*arazzo.Arazzo]
}

// NewLinterOption configures NewLinter.
type NewLinterOption func(*newLinterOpts)

type newLinterOpts struct {
	skipDefaultRules bool
}

// WithoutDefaultRules creates a linter with no rules registered, so specific rules can be registered with
// Registry().Register().
func WithoutDefaultRules() NewLinterOption {
	return func(o *newLinterOpts) {
		o.skipDefaultRules = true
	}
}

// NewLinter creates a new Arazzo linter. By default all built-in rules and rulesets are registered.
func NewLinter(config *baseLinter.Config, opts ...NewLinterOption) *Linter {
	options := &newLinterOpts{}
	for _, opt := range opts {
		opt(options)
	}

	if config == nil {
		config = baseLinter.NewConfig()
	}

	registry := baseLinter.NewRegistry[*arazzo.Arazzo]()
	if !options.skipDefaultRules {
		registerDefaultRules(registry)
	}

	return &Linter{base: baseLinter.NewLinter(config, registry)}
}

// Registry returns the rule registry, for registering rules and generating documentation.
func (l *Linter) Registry() *baseLinter.Registry[*arazzo.Arazzo] {
	return l.base.Registry()
}

// FilterErrors applies rule-level overrides and match filters to arbitrary errors, such as validation errors
// collected separately from Lint.
func (l *Linter) FilterErrors(errs []error) []error {
	return l.base.FilterErrors(errs)
}

// Lint runs the configured rules against the document. preExistingErrors, such as the errors returned by
// arazzo.Unmarshal and (*arazzo.Arazzo).ValidateSources, are included in the output after severity overrides and
// filters are applied.
//
// Rules can make use of the documents loaded for the document's sourceDescriptions when they are provided with
// WithSourceDocuments.
func (l *Linter) Lint(ctx context.Context, docInfo *baseLinter.DocumentInfo[*arazzo.Arazzo], preExistingErrors []error, opts *baseLinter.LintOptions) (*baseLinter.Output, error) {
	if opts == nil {
		opts = &baseLinter.LintOptions{}
	}
	if opts.VersionFilter == nil && docInfo != nil && docInfo.Document != nil {
		version := docInfo.Document.Arazzo
		opts.VersionFilter = &version
	}
	return l.base.Lint(ctx, docInfo, preExistingErrors, opts)
}

// WithSourceDocuments returns a context carrying the documents loaded for the Arazzo document's sourceDescriptions
// (see arazzo.LoadSourceDescriptions), which rules use to improve their findings and fixes.
func WithSourceDocuments(ctx context.Context, sources *arazzo.SourceDocuments) context.Context {
	return rules.ContextWithSourceDocuments(ctx, sources)
}

func registerDefaultRules(registry *baseLinter.Registry[*arazzo.Arazzo]) {
	registry.Register(&rules.UnreachableStepRule{})
	registry.Register(&rules.DependencyCycleRule{})
	registry.Register(&rules.UnusedComponentRule{})
	registry.Register(&rules.UnusedWorkflowInputRule{})
	registry.Register(&rules.UnusedStepOutputRule{})
	registry.Register(&rules.StepSuccessCriteriaRule{})
	registry.Register(&rules.SourceDescriptionTypeRule{})
	registry.Register(&rules.InfoDescriptionRule{})
	registry.Register(&rules.WorkflowDescriptionRule{})
	registry.Register(&rules.RetryAfterRule{})
	registry.Register(&rules.SourceDescriptionHTTPSRule{})

	// "recommended" leaves out the hint level rules, which are useful when authoring but noisier.
	_ = registry.RegisterRuleset("recommended", []string{
		rules.RuleSemanticUnreachableStep,
		rules.RuleSemanticDependencyCycle,
		rules.RuleSemanticUnusedComponent,
		rules.RuleSemanticUnusedWorkflowInput,
		rules.RuleStyleStepSuccessCriteria,
		rules.RuleStyleSourceDescriptionType,
		rules.RuleStyleInfoDescription,
		rules.RuleSecuritySourceDescriptionHTTPS,
	})
}
