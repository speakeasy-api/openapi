package openapi

import (
	"context"
	"fmt"
	"slices"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/version"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"go.yaml.in/yaml/v4"
)

type UpgradeOptions struct {
	upgradeSameMinorVersion bool
	targetVersion           string
}

// WithUpgradeSameMinorVersion will upgrade the same minor version of the OpenAPI document. For example 3.2.0 to 3.2.1.
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

// Upgrade upgrades any OpenAPI 3x document to OpenAPI 3.2.0 (the latest version currently supported).
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

	currentVersion, err := version.Parse(doc.OpenAPI)
	if err != nil {
		return false, err
	}

	targetVersion, err := version.Parse(options.targetVersion)
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
	if err := upgradeFrom31To32(ctx, doc, currentVersion, targetVersion); err != nil {
		return false, err
	}

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
	maxVersion, err := version.Parse("3.1.2")
	if err != nil {
		panic("failed to parse hardcoded version 3.1.2")
	}
	if targetVersion.LessThan(*maxVersion) {
		maxVersion = targetVersion
	}
	doc.OpenAPI = maxVersion.String()
}

func upgradeFrom31To32(ctx context.Context, doc *OpenAPI, currentVersion *version.Version, targetVersion *version.Version) error {
	if !targetVersion.GreaterThan(*currentVersion) {
		return nil
	}

	// Upgrade path additionalOperations for non-standard HTTP methods
	migrateAdditionalOperations31to32(ctx, doc)

	// Upgrade tags from extensions to new 3.2 fields
	if err := migrateTags31to32(ctx, doc); err != nil {
		return err
	}

	// Currently no breaking changes between 3.1.x and 3.2.x that need to be handled
	maxVersion, err := version.Parse("3.2.0")
	if err != nil {
		return err
	}
	if targetVersion.LessThan(*maxVersion) {
		maxVersion = targetVersion
	}
	doc.OpenAPI = maxVersion.String()

	return nil
}

// migrateAdditionalOperations31to32 migrates non-standard HTTP methods from the main operations map
// to the additionalOperations field in PathItem objects for OpenAPI 3.2.0+ compatibility.
func migrateAdditionalOperations31to32(_ context.Context, doc *OpenAPI) {
	if doc.Paths == nil {
		return
	}

	for _, referencedPathItem := range doc.Paths.All() {
		if referencedPathItem == nil || referencedPathItem.Object == nil {
			continue
		}

		pathItem := referencedPathItem.Object
		nonStandardMethods := sequencedmap.New[string, *Operation]()

		// Find non-standard HTTP methods in the main operations map
		for method, operation := range pathItem.All() {
			if !IsStandardMethod(string(method)) {
				nonStandardMethods.Set(string(method), operation)
			}
		}

		// If we found non-standard methods, migrate them to additionalOperations
		if nonStandardMethods.Len() > 0 {
			// Initialize additionalOperations if it doesn't exist
			if pathItem.AdditionalOperations == nil {
				pathItem.AdditionalOperations = sequencedmap.New[string, *Operation]()
			}

			// Move each non-standard operation to additionalOperations
			for method, operation := range nonStandardMethods.All() {
				pathItem.AdditionalOperations.Set(method, operation)

				// Remove from the main operations map
				pathItem.Map.Delete(HTTPMethod(method))
			}
		}
	}
}

// migrateTags31to32 migrates tag extensions to new OpenAPI 3.2 tag fields
func migrateTags31to32(_ context.Context, doc *OpenAPI) error {
	if doc == nil {
		return nil
	}

	// First, migrate x-displayName to summary for individual tags
	if doc.Tags != nil {
		for _, tag := range doc.Tags {
			if err := migrateTagDisplayName(tag); err != nil {
				return err
			}
		}
	}

	// Second, migrate x-tagGroups to parent relationships
	// This should always run to process extensions, even if no tags exist yet
	if err := migrateTagGroups(doc); err != nil {
		return err
	}

	return nil
}

// migrateTagDisplayName migrates x-displayName extension to summary field
func migrateTagDisplayName(tag *Tag) error {
	if tag == nil || tag.Extensions == nil {
		return nil
	}

	// Check if x-displayName extension exists and summary is not already set
	if displayNameExt, exists := tag.Extensions.Get("x-displayName"); exists {
		if tag.Summary != nil {
			// Error out if we can't migrate as summary is already set
			return fmt.Errorf("cannot migrate x-displayName to summary for tag %q as summary is already set", tag.Name)
		}
		// The extension value is stored as a string
		if displayNameExt.Value != "" {
			displayName := displayNameExt.Value
			tag.Summary = &displayName
			// Remove the extension after migration
			tag.Extensions.Delete("x-displayName")
		}
	}
	return nil
}

// TagGroup represents a single tag group from x-tagGroups extension
type TagGroup struct {
	Name string   `yaml:"name"`
	Tags []string `yaml:"tags"`
}

// migrateTagGroups migrates x-tagGroups extension to parent field relationships
func migrateTagGroups(doc *OpenAPI) error {
	if doc.Extensions == nil {
		return nil
	}

	// Check if x-tagGroups extension exists first
	_, exists := doc.Extensions.Get("x-tagGroups")
	if !exists {
		return nil // No x-tagGroups extension found
	}

	// Parse x-tagGroups extension
	tagGroups, err := extensions.GetExtensionValue[[]TagGroup](doc.Extensions, "x-tagGroups")
	if err != nil {
		return fmt.Errorf("failed to parse x-tagGroups extension: %w", err)
	}

	// Always remove the extension, even if empty or invalid
	defer doc.Extensions.Delete("x-tagGroups")

	if tagGroups == nil || len(*tagGroups) == 0 {
		return nil // Nothing to migrate
	}

	// Initialize tags slice if it doesn't exist
	if doc.Tags == nil {
		doc.Tags = []*Tag{}
	}

	// Create a map for quick tag lookup
	tagMap := make(map[string]*Tag)
	for _, tag := range doc.Tags {
		if tag != nil {
			tagMap[tag.Name] = tag
		}
	}

	// Process each tag group
	for _, group := range *tagGroups {
		if group.Name == "" {
			continue // Skip groups without names
		}

		// Ensure parent tag exists for this group
		parentTag := ensureParentTagExists(doc, tagMap, group.Name)
		if parentTag == nil {
			return fmt.Errorf("failed to create parent tag for group: %s", group.Name)
		}

		// Set parent relationships for all child tags in this group
		for _, childTagName := range group.Tags {
			if childTagName == "" {
				continue // Skip empty tag names
			}

			if err := setTagParent(doc, tagMap, childTagName, group.Name); err != nil {
				return fmt.Errorf("failed to set parent for tag %s in group %s: %w", childTagName, group.Name, err)
			}
		}
	}

	return nil
}

// ensureParentTagExists creates a parent tag if it doesn't already exist
func ensureParentTagExists(doc *OpenAPI, tagMap map[string]*Tag, groupName string) *Tag {
	// Check if parent tag already exists
	if existingTag, exists := tagMap[groupName]; exists {
		// Set kind to "nav" if not already set (common pattern for navigation groups)
		if existingTag.Kind == nil {
			kind := "nav"
			existingTag.Kind = &kind
		}
		return existingTag
	}

	// Create new parent tag
	kind := "nav"
	parentTag := &Tag{
		Name:    groupName,
		Summary: &groupName, // Use group name as summary for display
		Kind:    &kind,
	}

	// Add to document and map
	doc.Tags = append(doc.Tags, parentTag)
	tagMap[groupName] = parentTag

	return parentTag
}

// setTagParent sets the parent field for a child tag, creating the child tag if it doesn't exist
func setTagParent(doc *OpenAPI, tagMap map[string]*Tag, childTagName, parentTagName string) error {
	// Prevent self-referencing (tag can't be its own parent)
	if childTagName == parentTagName {
		return fmt.Errorf("tag cannot be its own parent: %s", childTagName)
	}

	// Check if child tag exists
	childTag, exists := tagMap[childTagName]
	if !exists {
		// Create child tag if it doesn't exist
		childTag = &Tag{
			Name: childTagName,
		}
		doc.Tags = append(doc.Tags, childTag)
		tagMap[childTagName] = childTag
	}

	// Check if child tag already has a different parent
	if childTag.Parent != nil && *childTag.Parent != parentTagName {
		return fmt.Errorf("tag %s already has parent %s, cannot assign new parent %s", childTagName, *childTag.Parent, parentTagName)
	}

	// Set the parent relationship
	childTag.Parent = &parentTagName

	return nil
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
