package rules_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/openapi/linter/rules"
	"github.com/stretchr/testify/assert"
)

// howToFixer is the interface satisfied by rules that provide fix guidance.
type howToFixer interface {
	HowToFix() string
}

// allRules returns every built-in rule instance.
func allRules() []linter.RuleRunner[*openapi.OpenAPI] {
	return []linter.RuleRunner[*openapi.OpenAPI]{
		&rules.PathParamsRule{},
		&rules.PathDeclarationsRule{},
		&rules.PathQueryRule{},
		&rules.TypedEnumRule{},
		&rules.DuplicatedEnumRule{},
		&rules.NoEvalInMarkdownRule{},
		&rules.OAS3APIServersRule{},
		&rules.NoRefSiblingsRule{},
		&rules.NoScriptTagsInMarkdownRule{},
		&rules.OAS3HostNotExampleRule{},
		&rules.OperationIdRule{},
		&rules.OperationSuccessResponseRule{},
		&rules.OperationErrorResponseRule{},
		&rules.OperationTagDefinedRule{},
		&rules.OperationSingularTagRule{},
		&rules.OperationDescriptionRule{},
		&rules.OperationTagsRule{},
		&rules.InfoDescriptionRule{},
		&rules.InfoContactRule{},
		&rules.InfoLicenseRule{},
		&rules.LicenseURLRule{},
		&rules.PathTrailingSlashRule{},
		&rules.PathsKebabCaseRule{},
		&rules.NoVerbsInPathRule{},
		&rules.TagDescriptionRule{},
		&rules.TagsAlphabeticalRule{},
		&rules.ContactPropertiesRule{},
		&rules.OpenAPITagsRule{},
		&rules.ComponentDescriptionRule{},
		&rules.UnusedComponentRule{},
		&rules.OperationIDValidInURLRule{},
		&rules.LinkOperationRule{},
		&rules.OAS3HostTrailingSlashRule{},
		&rules.OAS3ParameterDescriptionRule{},
		&rules.NoAmbiguousPathsRule{},
		&rules.DescriptionDuplicationRule{},
		&rules.OwaspNoHttpBasicRule{},
		&rules.OwaspNoAPIKeysInURLRule{},
		&rules.OwaspNoCredentialsInURLRule{},
		&rules.OwaspAuthInsecureSchemesRule{},
		&rules.OwaspDefineErrorResponses401Rule{},
		&rules.OwaspDefineErrorResponses500Rule{},
		&rules.OwaspDefineErrorResponses429Rule{},
		&rules.OwaspSecurityHostsHttpsOAS3Rule{},
		&rules.OwaspDefineErrorValidationRule{},
		&rules.OwaspProtectionGlobalUnsafeRule{},
		&rules.OwaspProtectionGlobalUnsafeStrictRule{},
		&rules.OwaspProtectionGlobalSafeRule{},
		&rules.OwaspRateLimitRule{},
		&rules.OwaspRateLimitRetryAfterRule{},
		&rules.OwaspNoNumericIDsRule{},
		&rules.OwaspJWTBestPracticesRule{},
		&rules.OwaspArrayLimitRule{},
		&rules.OwaspStringLimitRule{},
		&rules.OwaspStringRestrictedRule{},
		&rules.OwaspIntegerFormatRule{},
		&rules.OwaspIntegerLimitRule{},
		&rules.OwaspNoAdditionalPropertiesRule{},
		&rules.OwaspAdditionalPropertiesConstrainedRule{},
		&rules.OAS3NoNullableRule{},
		&rules.OAS3ExampleMissingRule{},
		&rules.OASSchemaCheckRule{},
	}
}

func TestAllRules_MetadataPopulated(t *testing.T) {
	t.Parallel()

	for _, rule := range allRules() {
		t.Run(rule.ID(), func(t *testing.T) {
			t.Parallel()

			assert.NotEmpty(t, rule.ID(), "rule ID should not be empty")
			assert.NotEmpty(t, rule.Category(), "rule category should not be empty")
			assert.NotEmpty(t, rule.Description(), "rule description should not be empty")
			assert.NotEmpty(t, rule.Summary(), "rule summary should not be empty")
			assert.NotEmpty(t, rule.DefaultSeverity(), "rule default severity should not be empty")

			if fixer, ok := rule.(howToFixer); ok {
				assert.NotEmpty(t, fixer.HowToFix(), "rule HowToFix should not be empty")
			}
		})
	}
}
