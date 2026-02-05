package linter

import (
	"context"
	"fmt"
	"sync"

	baseLinter "github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/openapi/linter/rules"
	"github.com/speakeasy-api/openapi/references"
)

// CustomRuleLoaderFunc loads custom rules from configuration.
// It is called during NewLinter when custom rules are configured.
type CustomRuleLoaderFunc func(config *baseLinter.CustomRulesConfig) ([]baseLinter.RuleRunner[*openapi.OpenAPI], error)

var (
	customRuleLoaders   []CustomRuleLoaderFunc
	customRuleLoadersMu sync.Mutex
)

// RegisterCustomRuleLoader registers a custom rule loader.
// This is called by the customrules package's init() function.
// Loaders are invoked in registration order during NewLinter.
func RegisterCustomRuleLoader(loader CustomRuleLoaderFunc) {
	customRuleLoadersMu.Lock()
	defer customRuleLoadersMu.Unlock()
	customRuleLoaders = append(customRuleLoaders, loader)
}

// Linter is an OpenAPI-specific linter that automatically builds an index
// before running rules. This provides rules with efficient access to
// indexed document data via GetIndex().
type Linter struct {
	base *baseLinter.Linter[*openapi.OpenAPI]
}

// NewLinterOption is a functional option for configuring linter creation.
type NewLinterOption func(*newLinterOpts)

type newLinterOpts struct {
	skipDefaultRules bool
}

// WithoutDefaultRules creates a linter with no rules registered.
// This is useful for advanced use cases where you want to register custom
// rules or selectively register only specific rules via the Registry() method.
//
// Example:
//
//	linter := NewLinter(config, WithoutDefaultRules())
//	linter.Registry().Register(&rules.PathParamsRule{})
func WithoutDefaultRules() NewLinterOption {
	return func(o *newLinterOpts) {
		o.skipDefaultRules = true
	}
}

// NewLinter creates a new OpenAPI linter.
// By default, all built-in rules are registered. Use WithoutDefaultRules()
// to create a linter with no rules registered.
//
// Returns an error if custom rules are configured and fail to load.
//
// Example - Default behavior (all rules):
//
//	linter, err := NewLinter(config)
//
// Example - No rules registered:
//
//	linter, err := NewLinter(config, WithoutDefaultRules())
//	linter.Registry().Register(&rules.PathParamsRule{})
//
// The linter automatically builds an index before running rules.
func NewLinter(config *baseLinter.Config, opts ...NewLinterOption) (*Linter, error) {
	options := &newLinterOpts{
		skipDefaultRules: false,
	}
	for _, opt := range opts {
		opt(options)
	}

	registry := baseLinter.NewRegistry[*openapi.OpenAPI]()

	// Register all OpenAPI rules unless explicitly skipped
	if !options.skipDefaultRules {
		registerDefaultRules(registry)
	}

	// Load custom rules if configured and loaders are registered
	if config != nil && config.CustomRules != nil && len(config.CustomRules.Paths) > 0 {
		customRuleLoadersMu.Lock()
		loaders := make([]CustomRuleLoaderFunc, len(customRuleLoaders))
		copy(loaders, customRuleLoaders)
		customRuleLoadersMu.Unlock()

		for _, loader := range loaders {
			customRules, err := loader(config.CustomRules)
			if err != nil {
				return nil, fmt.Errorf("loading custom rules: %w", err)
			}
			for _, rule := range customRules {
				registry.Register(rule)
			}
		}
	}

	return &Linter{
		base: baseLinter.NewLinter(config, registry),
	}, nil
}

// Registry returns the rule registry for documentation generation
func (l *Linter) Registry() *baseLinter.Registry[*openapi.OpenAPI] {
	return l.base.Registry()
}

// FilterErrors applies rule-level overrides and match filters to arbitrary errors.
// This is useful when you collect additional validation errors after the main lint run.
func (l *Linter) FilterErrors(errs []error) []error {
	return l.base.FilterErrors(errs)
}

// Lint runs all configured rules against the document.
// The index is automatically built and made available to rules via DocumentInfo.Index.
func (l *Linter) Lint(ctx context.Context, docInfo *baseLinter.DocumentInfo[*openapi.OpenAPI], preExistingErrors []error, opts *baseLinter.LintOptions) (*baseLinter.Output, error) {
	// Build index if not already provided
	if docInfo.Index == nil && docInfo.Document != nil {
		// Prepare resolve options for index building
		resolveOpts := references.ResolveOptions{
			RootDocument:   docInfo.Document,
			TargetDocument: docInfo.Document,
			TargetLocation: docInfo.Location,
		}

		// Override with user-provided options if available
		if opts != nil && opts.ResolveOptions != nil {
			if opts.ResolveOptions.VirtualFS != nil {
				resolveOpts.VirtualFS = opts.ResolveOptions.VirtualFS
			}
			if opts.ResolveOptions.HTTPClient != nil {
				resolveOpts.HTTPClient = opts.ResolveOptions.HTTPClient
			}
			resolveOpts.DisableExternalRefs = opts.ResolveOptions.DisableExternalRefs
			resolveOpts.SkipValidation = opts.ResolveOptions.SkipValidation
		}

		// Build the index
		idx := openapi.BuildIndex(ctx, docInfo.Document, resolveOpts)

		// Create a new DocumentInfo with the index
		docInfo = baseLinter.NewDocumentInfoWithIndex(docInfo.Document, docInfo.Location, idx)

		if idx.HasErrors() {
			preExistingErrors = append(preExistingErrors, idx.GetValidationErrors()...)
			preExistingErrors = append(preExistingErrors, idx.GetResolutionErrors()...)
			preExistingErrors = append(preExistingErrors, idx.GetCircularReferenceErrors()...)
		}
	}

	// Filter rules based on OpenAPI version
	if opts == nil {
		opts = &baseLinter.LintOptions{}
	}
	if opts.VersionFilter == nil && docInfo.Document != nil {
		version := docInfo.Document.OpenAPI
		opts.VersionFilter = &version
	}

	return l.base.Lint(ctx, docInfo, preExistingErrors, opts)
}

func registerDefaultRules(registry *baseLinter.Registry[*openapi.OpenAPI]) {
	// Register all rules
	registry.Register(&rules.PathParamsRule{})
	registry.Register(&rules.PathDeclarationsRule{})
	registry.Register(&rules.PathQueryRule{})
	registry.Register(&rules.TypedEnumRule{})
	registry.Register(&rules.DuplicatedEnumRule{})
	registry.Register(&rules.NoEvalInMarkdownRule{})
	registry.Register(&rules.OAS3APIServersRule{})
	registry.Register(&rules.NoRefSiblingsRule{})
	registry.Register(&rules.NoScriptTagsInMarkdownRule{})
	registry.Register(&rules.OAS3HostNotExampleRule{})
	registry.Register(&rules.OperationIdRule{})
	registry.Register(&rules.OperationSuccessResponseRule{})
	registry.Register(&rules.OperationErrorResponseRule{})
	registry.Register(&rules.OperationTagDefinedRule{})
	registry.Register(&rules.OperationSingularTagRule{})
	registry.Register(&rules.OperationDescriptionRule{})
	registry.Register(&rules.OperationTagsRule{})
	registry.Register(&rules.InfoDescriptionRule{})
	registry.Register(&rules.InfoContactRule{})
	registry.Register(&rules.InfoLicenseRule{})
	registry.Register(&rules.LicenseURLRule{})
	registry.Register(&rules.PathTrailingSlashRule{})
	registry.Register(&rules.PathsKebabCaseRule{})
	registry.Register(&rules.NoVerbsInPathRule{})
	registry.Register(&rules.TagDescriptionRule{})
	registry.Register(&rules.TagsAlphabeticalRule{})
	registry.Register(&rules.ContactPropertiesRule{})
	registry.Register(&rules.OpenAPITagsRule{})
	registry.Register(&rules.ComponentDescriptionRule{})
	registry.Register(&rules.UnusedComponentRule{})
	registry.Register(&rules.OperationIDValidInURLRule{})
	registry.Register(&rules.LinkOperationRule{})
	registry.Register(&rules.OAS3HostTrailingSlashRule{})
	registry.Register(&rules.OAS3ParameterDescriptionRule{})
	registry.Register(&rules.NoAmbiguousPathsRule{})
	registry.Register(&rules.DescriptionDuplicationRule{})
	registry.Register(&rules.OwaspNoHttpBasicRule{})
	registry.Register(&rules.OwaspNoAPIKeysInURLRule{})
	registry.Register(&rules.OwaspNoCredentialsInURLRule{})
	registry.Register(&rules.OwaspAuthInsecureSchemesRule{})
	registry.Register(&rules.OwaspDefineErrorResponses401Rule{})
	registry.Register(&rules.OwaspDefineErrorResponses500Rule{})
	registry.Register(&rules.OwaspDefineErrorResponses429Rule{})
	registry.Register(&rules.OwaspSecurityHostsHttpsOAS3Rule{})
	registry.Register(&rules.OwaspDefineErrorValidationRule{})
	registry.Register(&rules.OwaspProtectionGlobalUnsafeRule{})
	registry.Register(&rules.OwaspProtectionGlobalUnsafeStrictRule{})
	registry.Register(&rules.OwaspProtectionGlobalSafeRule{})
	registry.Register(&rules.OwaspRateLimitRule{})
	registry.Register(&rules.OwaspRateLimitRetryAfterRule{})
	registry.Register(&rules.OwaspNoNumericIDsRule{})
	registry.Register(&rules.OwaspJWTBestPracticesRule{})
	registry.Register(&rules.OwaspArrayLimitRule{})
	registry.Register(&rules.OwaspStringLimitRule{})
	registry.Register(&rules.OwaspStringRestrictedRule{})
	registry.Register(&rules.OwaspIntegerFormatRule{})
	registry.Register(&rules.OwaspIntegerLimitRule{})
	registry.Register(&rules.OwaspNoAdditionalPropertiesRule{})
	registry.Register(&rules.OwaspAdditionalPropertiesConstrainedRule{})
	registry.Register(&rules.OAS3NoNullableRule{})
	registry.Register(&rules.OAS3ExampleMissingRule{})
	registry.Register(&rules.OASSchemaCheckRule{})

	// Register rulesets
	registerRulesets(registry)
}

// registerRulesets registers the built-in rulesets.
func registerRulesets(registry *baseLinter.Registry[*openapi.OpenAPI]) {
	// "recommended" - balanced ruleset for most APIs
	// Includes semantic rules, essential style rules, and basic security rules
	_ = registry.RegisterRuleset("recommended", []string{
		// Semantic rules (catch real bugs)
		rules.RuleSemanticPathParams,
		rules.RuleSemanticPathDeclarations,
		rules.RuleSemanticPathQuery,
		rules.RuleSemanticTypedEnum,
		rules.RuleSemanticDuplicatedEnum,
		rules.RuleSemanticNoEvalInMarkdown,
		rules.RuleSemanticNoScriptTagsInMarkdown,
		rules.RuleSemanticOperationOperationId,
		rules.RuleSemanticNoAmbiguousPaths,
		rules.RuleSemanticOperationIDValidInURL,
		rules.RuleSemanticLinkOperation,
		rules.RuleSemanticUnusedComponent,

		// Essential style rules
		rules.RuleStyleInfoDescription,
		rules.RuleStyleOperationSuccessResponse,
		rules.RuleStylePathTrailingSlash,
		rules.RuleStyleNoRefSiblings,
		rules.RuleStyleOAS3HostNotExample,
		rules.RuleStyleOAS3HostTrailingSlash,
		rules.RuleStyleOAS3APIServers,
		rules.RuleStyleDescriptionDuplication,

		// Basic security rules
		rules.RuleOwaspNoHttpBasic,
		rules.RuleOwaspNoAPIKeysInURL,
		rules.RuleOwaspNoCredentialsInURL,
		rules.RuleOwaspAuthInsecureSchemes,
		rules.RuleOwaspSecurityHostsHttpsOAS3,
	})

	// "security" - comprehensive OWASP security rules
	_ = registry.RegisterRuleset("security", []string{
		rules.RuleOwaspNoHttpBasic,
		rules.RuleOwaspNoAPIKeysInURL,
		rules.RuleOwaspNoCredentialsInURL,
		rules.RuleOwaspAuthInsecureSchemes,
		rules.RuleOwaspSecurityHostsHttpsOAS3,
		rules.RuleOwaspDefineErrorResponses401,
		rules.RuleOwaspDefineErrorResponses500,
		rules.RuleOwaspDefineErrorResponses429,
		rules.RuleOwaspDefineErrorValidation,
		rules.RuleOwaspProtectionGlobalUnsafe,
		rules.RuleOwaspProtectionGlobalUnsafeStrict,
		rules.RuleOwaspProtectionGlobalSafe,
		rules.RuleOwaspRateLimit,
		rules.RuleOwaspRateLimitRetryAfter,
		rules.RuleOwaspNoNumericIDs,
		rules.RuleOwaspJWTBestPractices,
		rules.RuleOwaspArrayLimit,
		rules.RuleOwaspStringLimit,
		rules.RuleOwaspStringRestricted,
		rules.RuleOwaspIntegerFormat,
		rules.RuleOwaspIntegerLimit,
		rules.RuleOwaspNoAdditionalProperties,
		rules.RuleOwaspAdditionalPropertiesConstrained,
	})
}
