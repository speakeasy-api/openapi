package arazzo

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/arazzo/internal/specversion"
	"github.com/speakeasy-api/openapi/marshaller"
)

// UpgradeOptions configures how an Arazzo document is upgraded.
type UpgradeOptions struct {
	upgradeSameMinorVersion bool
	targetVersion           string
}

// WithUpgradeSameMinorVersion will upgrade the same minor version of the Arazzo document. For example 1.0.0 to 1.0.1.
func WithUpgradeSameMinorVersion() Option[UpgradeOptions] {
	return func(uo *UpgradeOptions) {
		uo.upgradeSameMinorVersion = true
	}
}

// WithUpgradeTargetVersion will upgrade the Arazzo document to the provided version instead of the latest supported version.
func WithUpgradeTargetVersion(version string) Option[UpgradeOptions] {
	return func(uo *UpgradeOptions) {
		uo.targetVersion = version
	}
}

// Upgrade upgrades an Arazzo 1.0.x document to Arazzo 1.1.0 (the latest version currently supported), or to the
// version provided via WithUpgradeTargetVersion. It returns true if the document was modified.
//
// Upgrading from 1.0.x to 1.1.x preserves the meaning of the document: criteria that use the `jsonpath` or `xpath`
// type without an explicit version are pinned to the version implied by Arazzo 1.0 (`draft-goessner-dispatch-jsonpath-00`
// and `xpath-30`), as the defaults changed to RFC 9535 and XPath 3.1 in Arazzo 1.1.
//
// The document is not validated as part of the upgrade, runtime expressions that were accepted by Arazzo 1.0 but don't
// match the stricter Arazzo 1.1 grammar are left unchanged and will be reported when validating the upgraded document.
func Upgrade(ctx context.Context, doc *Arazzo, opts ...Option[UpgradeOptions]) (bool, error) {
	if doc == nil {
		return false, nil
	}

	options := UpgradeOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	if options.targetVersion == "" {
		options.targetVersion = Version
	}

	currentVersion, err := specversion.Parse(doc.Arazzo)
	if err != nil {
		return false, fmt.Errorf("invalid Arazzo document version `%s`: %w", doc.Arazzo, err)
	}

	targetVersion, err := specversion.Parse(options.targetVersion)
	if err != nil {
		return false, fmt.Errorf("invalid target Arazzo version `%s`: %w", options.targetVersion, err)
	}

	if !IsSupportedVersion(doc.Arazzo) {
		return false, fmt.Errorf("cannot upgrade Arazzo document version %s: only Arazzo versions `%s` are supported", currentVersion, strings.Join(supportedVersionPatterns(), "`, `"))
	}
	if !IsSupportedVersion(options.targetVersion) {
		return false, fmt.Errorf("cannot upgrade Arazzo document to version %s: only Arazzo versions `%s` are supported", targetVersion, strings.Join(supportedVersionPatterns(), "`, `"))
	}

	if targetVersion.LessThan(*currentVersion) {
		return false, fmt.Errorf("cannot downgrade Arazzo document version from %s to %s", currentVersion, targetVersion)
	}

	if targetVersion.Equal(*currentVersion) {
		return false, nil
	}

	// Skip patch-only upgrades if 'upgradeSameMinorVersion' is not set
	if targetVersion.Major == currentVersion.Major && targetVersion.Minor == currentVersion.Minor && !options.upgradeSameMinorVersion {
		return false, nil
	}

	if currentVersion.Minor == 0 && targetVersion.Minor >= 1 {
		upgradeFrom10To11(doc)
	}

	// Use the validated target string rather than the parsed version so any prerelease suffix is preserved.
	doc.Arazzo = options.targetVersion

	_, err = marshaller.Sync(ctx, doc)
	return true, err
}

// upgradeFrom10To11 pins the version of `jsonpath` and `xpath` criteria to the versions implied by Arazzo 1.0, as
// the default versions changed in Arazzo 1.1.
func upgradeFrom10To11(doc *Arazzo) {
	for _, workflow := range doc.Workflows {
		if workflow == nil {
			continue
		}

		for _, step := range workflow.Steps {
			if step == nil {
				continue
			}

			pinCriteriaVersions(step.SuccessCriteria)
			pinReusableSuccessActionCriteria(step.OnSuccess)
			pinReusableFailureActionCriteria(step.OnFailure)
		}

		pinReusableSuccessActionCriteria(workflow.SuccessActions)
		pinReusableFailureActionCriteria(workflow.FailureActions)
	}

	if doc.Components != nil {
		if doc.Components.SuccessActions != nil {
			for _, action := range doc.Components.SuccessActions.All() {
				if action != nil {
					pinCriteriaValueVersions(action.Criteria)
				}
			}
		}
		if doc.Components.FailureActions != nil {
			for _, action := range doc.Components.FailureActions.All() {
				if action != nil {
					pinCriteriaValueVersions(action.Criteria)
				}
			}
		}
	}
}

func pinReusableSuccessActionCriteria(actions []*ReusableSuccessAction) {
	for _, action := range actions {
		if action != nil && action.Object != nil {
			pinCriteriaValueVersions(action.Object.Criteria)
		}
	}
}

func pinReusableFailureActionCriteria(actions []*ReusableFailureAction) {
	for _, action := range actions {
		if action != nil && action.Object != nil {
			pinCriteriaValueVersions(action.Object.Criteria)
		}
	}
}

func pinCriteriaVersions(criteria []*criterion.Criterion) {
	for _, c := range criteria {
		if c != nil {
			pinCriterionVersion(c)
		}
	}
}

func pinCriteriaValueVersions(criteria []criterion.Criterion) {
	for i := range criteria {
		pinCriterionVersion(&criteria[i])
	}
}

// pinCriterionVersion converts a `jsonpath` or `xpath` criterion type without a version into an Expression Type Object
// with the version implied by Arazzo 1.0.
func pinCriterionVersion(c *criterion.Criterion) {
	if c.Type.Type == nil {
		return
	}

	var version criterion.CriterionTypeVersion
	switch *c.Type.Type {
	case criterion.CriterionTypeJsonPath:
		version = criterion.CriterionTypeVersionDraftGoessnerDispatchJsonPath00
	case criterion.CriterionTypeXPath:
		version = criterion.CriterionTypeVersionXPath30
	case criterion.CriterionTypeSimple, criterion.CriterionTypeRegex, criterion.CriterionTypeJsonPointer:
		return
	default:
		return
	}

	c.Type = criterion.CriterionTypeUnion{
		ExpressionType: &criterion.CriterionExpressionType{
			Type:    *c.Type.Type,
			Version: version,
		},
	}
}
