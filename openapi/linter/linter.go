package linter

import (
	"context"

	baseLinter "github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/openapi/linter/rules"
	"github.com/speakeasy-api/openapi/references"
)

// Linter is an OpenAPI-specific linter that automatically builds an index
// before running rules. This provides rules with efficient access to
// indexed document data via GetIndex().
type Linter struct {
	base *baseLinter.Linter[*openapi.OpenAPI]
}

// NewLinter creates a new OpenAPI linter with all default rules registered.
// The linter automatically builds an index before running rules.
func NewLinter(config *baseLinter.Config) *Linter {
	registry := baseLinter.NewRegistry[*openapi.OpenAPI]()

	// Register all OpenAPI rules
	registerDefaultRules(registry)

	return &Linter{
		base: baseLinter.NewLinter(config, registry),
	}
}

// Registry returns the rule registry for documentation generation
func (l *Linter) Registry() *baseLinter.Registry[*openapi.OpenAPI] {
	return l.base.Registry()
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
	// Future rules will be registered here
}
