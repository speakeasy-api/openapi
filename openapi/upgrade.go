package openapi

import (
	"context"
	"fmt"
	"slices"

	"github.com/speakeasy-api/openapi/internal/version"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"gopkg.in/yaml.v3"
)

type UpgradeOptions struct {
	upgradeSameMinorVersion bool
	targetVersion           string
}

// WithUpgradeSameMinorVersion will upgrade the same minor version of the OpenAPI document. For example 3.1.0 to 3.1.1.
func WithUpgradeSameMinorVersion() Option[UpgradeOptions] {
	return func(uo *UpgradeOptions) {
		uo.upgradeSameMinorVersion = true
	}
}

func WithUpgradeTargetVersion(version string) Option[UpgradeOptions] {
	return func(uo *UpgradeOptions) {
		uo.targetVersion = version
	}
}

// Upgrade upgrades any OpenAPI 3x document to OpenAPI 3.1.1 (the latest version currently supported).
// It currently won't resolve any external references, so only this document itself will be upgraded.
func Upgrade(ctx context.Context, doc *OpenAPI, opts ...Option[UpgradeOptions]) (bool, error) {
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

	currentVersion, err := version.ParseVersion(doc.OpenAPI)
	if err != nil {
		return false, err
	}

	targetVersion, err := version.ParseVersion(options.targetVersion)
	if err != nil {
		return false, err
	}

	invalidVersion := targetVersion.LessThan(*currentVersion)
	if invalidVersion {
		return false, fmt.Errorf("cannot downgrade OpenAPI document version from %s to %s", currentVersion, targetVersion)
	}

	if currentVersion.Major < 3 {
		return false, fmt.Errorf("cannot upgrade OpenAPI document version from %s to %s: only OpenAPI 3.x.x is supported", currentVersion, targetVersion)
	}

	if targetVersion.Equal(*currentVersion) {
		return false, nil
	}

	// Skip patch-only upgrades if 'upgradeSameMinorVersion' is not set
	if targetVersion.Major == currentVersion.Major && targetVersion.Minor == currentVersion.Minor && !options.upgradeSameMinorVersion {
		return false, nil
	}

	// We're passing current and target version to each upgrade function in case we want to
	// add logic to skip certain upgrades in certain situations in the future
	upgradeFrom30To31(ctx, doc, currentVersion, targetVersion)
	upgradeFrom310To312(ctx, doc, currentVersion, targetVersion)
	upgradeFrom31To32(ctx, doc, currentVersion, targetVersion)

	_, err = marshaller.Sync(ctx, doc)
	return true, err
}

func upgradeFrom30To31(ctx context.Context, doc *OpenAPI, _ *version.Version, _ *version.Version) {
	// Always run the upgrade logic, because 3.1 is backwards compatible, but we want to migrate if we can

	for item := range Walk(ctx, doc) {
		_ = item.Match(Matcher{
			Schema: func(js *oas3.JSONSchema[oas3.Referenceable]) error {
				upgradeSchema30to31(js)
				return nil
			},
		})
	}
	doc.OpenAPI = "3.1.0"
}

func upgradeFrom310To312(_ context.Context, doc *OpenAPI, currentVersion *version.Version, targetVersion *version.Version) {
	if !targetVersion.GreaterThan(*currentVersion) {
		return
	}

	// Currently no breaking changes between 3.1.0 and 3.1.2 that need to be handled
	maxVersion, err := version.ParseVersion("3.1.2")
	if err != nil {
		panic("failed to parse hardcoded version 3.1.2")
	}
	if targetVersion.LessThan(*maxVersion) {
		maxVersion = targetVersion
	}
	doc.OpenAPI = maxVersion.String()
}

func upgradeFrom31To32(_ context.Context, doc *OpenAPI, currentVersion *version.Version, targetVersion *version.Version) {
	if !targetVersion.GreaterThan(*currentVersion) {
		return
	}

	// TODO: Upgrade path additionalOperations for non-standard HTTP methods
	// TODO: Upgrade tags such as x-displayName to summary, and x-tagGroups with parents, etc.

	// Currently no breaking changes between 3.1.x and 3.2.x that need to be handled
	maxVersion, err := version.ParseVersion("3.2.0")
	if err != nil {
		panic("failed to parse hardcoded version 3.2.0")
	}
	if targetVersion.LessThan(*maxVersion) {
		maxVersion = targetVersion
	}
	doc.OpenAPI = maxVersion.String()
}

func upgradeSchema30to31(js *oas3.JSONSchema[oas3.Referenceable]) {
	if js == nil || js.IsReference() || js.IsRight() {
		return
	}

	schema := js.GetResolvedSchema().GetLeft()

	upgradeExample30to31(schema)
	upgradeExclusiveMinMax30to31(schema)
	upgradeNullableSchema30to31(schema)
}

func upgradeExample30to31(schema *oas3.Schema) {
	if schema == nil || schema.Example == nil {
		return
	}

	if schema.Examples == nil {
		schema.Examples = []*yaml.Node{}
	}

	schema.Examples = append(schema.Examples, schema.Example)
	schema.Example = nil
}

func upgradeExclusiveMinMax30to31(schema *oas3.Schema) {
	if schema.ExclusiveMaximum != nil && schema.ExclusiveMaximum.IsLeft() {
		if schema.Maximum == nil || !*schema.ExclusiveMaximum.GetLeft() {
			schema.ExclusiveMaximum = nil
		} else {
			schema.ExclusiveMaximum = oas3.NewExclusiveMaximumFromFloat64(*schema.Maximum)
			schema.Maximum = nil
		}
	}

	if schema.ExclusiveMinimum != nil && schema.ExclusiveMinimum.IsLeft() {
		if schema.Minimum == nil || !*schema.ExclusiveMinimum.GetLeft() {
			schema.ExclusiveMinimum = nil
		} else {
			schema.ExclusiveMinimum = oas3.NewExclusiveMinimumFromFloat64(*schema.Minimum)
			schema.Minimum = nil
		}
	}
}

func upgradeNullableSchema30to31(schema *oas3.Schema) {
	if schema == nil {
		return
	}

	if schema.Nullable == nil || !*schema.Nullable {
		schema.Nullable = nil // clear it out if it was set to false
		return
	}

	schema.Nullable = nil

	switch {
	case len(schema.GetType()) > 0:
		if !slices.Contains(schema.GetType(), "null") {
			schema.Type = oas3.NewTypeFromArray(append(schema.GetType(), "null"))
		}
	case len(schema.AnyOf) > 0:
		nullSchema := createNullSchema()
		schema.AnyOf = append(schema.AnyOf, nullSchema)
	case len(schema.OneOf) > 0:
		nullSchema := createNullSchema()
		schema.OneOf = append(schema.OneOf, nullSchema)
	default:
		nullSchema := createNullSchema()
		clone := *schema
		newSchema := oas3.Schema{}
		newSchema.OneOf = []*oas3.JSONSchema[oas3.Referenceable]{oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&clone), nullSchema}
		*schema = newSchema
	}
}

func createNullSchema() *oas3.JSONSchema[oas3.Referenceable] {
	return oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
		Type: oas3.NewTypeFromArray([]oas3.SchemaType{oas3.SchemaTypeNull}),
	})
}
