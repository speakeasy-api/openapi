package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

const RuleSemanticUnusedComponent = "semantic-unused-component"

type UnusedComponentRule struct{}

func (r *UnusedComponentRule) ID() string       { return RuleSemanticUnusedComponent }
func (r *UnusedComponentRule) Category() string { return CategorySemantic }
func (r *UnusedComponentRule) Description() string {
	return "Components that are declared but never referenced should be removed to keep the specification clean. Unused components create maintenance burden, increase specification size, and may confuse developers about which schemas are actually used."
}
func (r *UnusedComponentRule) Summary() string {
	return "Components should not be declared if they are never referenced."
}
func (r *UnusedComponentRule) HowToFix() string {
	return "Remove unused components or reference them where needed in the specification."
}
func (r *UnusedComponentRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#semantic-unused-component"
}
func (r *UnusedComponentRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *UnusedComponentRule) Versions() []string {
	// Applies to all OAS3 versions
	return nil
}

func (r *UnusedComponentRule) Run(_ context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	doc := docInfo.Document

	// Step 1: Collect all referenced component pointers from $ref strings
	referencedPointers := collectReferencedComponentPointers(docInfo.Index, doc, docInfo.Location)

	// Step 2: Check each component against the referenced set
	return checkUnusedComponents(doc, docInfo.Index, referencedPointers, config, r.DefaultSeverity())
}

// collectReferencedComponentPointers iterates through all reference slices in the index
// and collects the component JSON pointers (e.g., "/components/schemas/Pet").
func collectReferencedComponentPointers(idx *openapi.Index, doc *openapi.OpenAPI, docLocation string) map[string]struct{} {
	refs := make(map[string]struct{})
	self := ""
	if doc != nil {
		self = doc.GetSelf()
	}

	// Schema references
	for _, node := range idx.SchemaReferences {
		if node == nil || node.Node == nil {
			continue
		}
		if ptr := extractComponentPointer(node.Node.GetReference(), docLocation, self); ptr != "" {
			refs[ptr] = struct{}{}
		}
	}

	// Parameter references
	for _, node := range idx.ParameterReferences {
		if node == nil || node.Node == nil {
			continue
		}
		if ptr := extractComponentPointer(node.Node.GetReference(), docLocation, self); ptr != "" {
			refs[ptr] = struct{}{}
		}
	}

	// Response references
	for _, node := range idx.ResponseReferences {
		if node == nil || node.Node == nil {
			continue
		}
		if ptr := extractComponentPointer(node.Node.GetReference(), docLocation, self); ptr != "" {
			refs[ptr] = struct{}{}
		}
	}

	// RequestBody references
	for _, node := range idx.RequestBodyReferences {
		if node == nil || node.Node == nil {
			continue
		}
		if ptr := extractComponentPointer(node.Node.GetReference(), docLocation, self); ptr != "" {
			refs[ptr] = struct{}{}
		}
	}

	// Header references
	for _, node := range idx.HeaderReferences {
		if node == nil || node.Node == nil {
			continue
		}
		if ptr := extractComponentPointer(node.Node.GetReference(), docLocation, self); ptr != "" {
			refs[ptr] = struct{}{}
		}
	}

	// Example references
	for _, node := range idx.ExampleReferences {
		if node == nil || node.Node == nil {
			continue
		}
		if ptr := extractComponentPointer(node.Node.GetReference(), docLocation, self); ptr != "" {
			refs[ptr] = struct{}{}
		}
	}

	// Link references
	for _, node := range idx.LinkReferences {
		if node == nil || node.Node == nil {
			continue
		}
		if ptr := extractComponentPointer(node.Node.GetReference(), docLocation, self); ptr != "" {
			refs[ptr] = struct{}{}
		}
	}

	// Callback references
	for _, node := range idx.CallbackReferences {
		if node == nil || node.Node == nil {
			continue
		}
		if ptr := extractComponentPointer(node.Node.GetReference(), docLocation, self); ptr != "" {
			refs[ptr] = struct{}{}
		}
	}

	// PathItem references
	for _, node := range idx.PathItemReferences {
		if node == nil || node.Node == nil {
			continue
		}
		if ptr := extractComponentPointer(node.Node.GetReference(), docLocation, self); ptr != "" {
			refs[ptr] = struct{}{}
		}
	}

	// SecurityScheme references
	for _, node := range idx.SecuritySchemeReferences {
		if node == nil || node.Node == nil {
			continue
		}
		if ptr := extractComponentPointer(node.Node.GetReference(), docLocation, self); ptr != "" {
			refs[ptr] = struct{}{}
		}
	}

	// Security requirements reference security schemes by name (not $ref)
	for _, node := range idx.SecurityRequirements {
		if node == nil || node.Node == nil {
			continue
		}
		for schemeName := range node.Node.All() {
			// Security requirements reference security schemes by name
			refs["/components/securitySchemes/"+schemeName] = struct{}{}
		}
	}

	return refs
}

// extractComponentPointer extracts the top-level component JSON pointer from a $ref.
// For example, "#/components/schemas/Pet/properties/name" becomes "/components/schemas/Pet".
// Returns empty string if the reference is not to a component or is external.
func extractComponentPointer(ref references.Reference, docLocation string, docSelf string) string {
	if ref == "" {
		return ""
	}

	uri := ref.GetURI()
	if uri != "" && uri != docLocation && uri != docSelf {
		return ""
	}

	pointer := ref.GetJSONPointer().String()
	if pointer == "" {
		return ""
	}

	// Must start with /components/
	if !strings.HasPrefix(pointer, "/components/") {
		return ""
	}

	// Extract the component type and name: /components/{type}/{name}
	// Skip "/components/" (12 chars), then find the type and name
	rest := strings.TrimPrefix(pointer, "/components/")
	parts := strings.SplitN(rest, "/", 3) // Split into at most 3 parts: type, name, rest
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}

	// Return the normalized component pointer (keep escaped form for comparison)
	return "/components/" + parts[0] + "/" + parts[1]
}

// checkUnusedComponents iterates through all component entries in the index
// and flags those not in the referenced set using ToJSONPointer.
func checkUnusedComponents(doc *openapi.OpenAPI, idx *openapi.Index, refs map[string]struct{}, config *linter.RuleConfig, severity validation.Severity) []error {
	var errs []error

	// Check component schemas
	for _, node := range idx.ComponentSchemas {
		if node == nil || node.Node == nil {
			continue
		}
		pointer := node.Location.ToJSONPointer().String()
		if _, found := refs[pointer]; !found {
			// Skip if component has a usage-marking extension
			if hasUsageMarkingExtension(node.Node.GetExtensions()) {
				continue
			}
			errNode := getComponentKeyNode(doc, node.Location)
			errs = append(errs, createUnusedComponentError(pointer, errNode, config, severity))
		}
	}

	// Check component parameters
	for _, node := range idx.ComponentParameters {
		if node == nil || node.Node == nil {
			continue
		}
		pointer := node.Location.ToJSONPointer().String()
		if _, found := refs[pointer]; !found {
			// Skip if component has a usage-marking extension
			if obj := node.Node.GetObject(); obj != nil && hasUsageMarkingExtension(obj.GetExtensions()) {
				continue
			}
			errNode := getComponentKeyNode(doc, node.Location)
			errs = append(errs, createUnusedComponentError(pointer, errNode, config, severity))
		}
	}

	// Check component responses
	for _, node := range idx.ComponentResponses {
		if node == nil || node.Node == nil {
			continue
		}
		pointer := node.Location.ToJSONPointer().String()
		if _, found := refs[pointer]; !found {
			// Skip if component has a usage-marking extension
			if obj := node.Node.GetObject(); obj != nil && hasUsageMarkingExtension(obj.GetExtensions()) {
				continue
			}
			errNode := getComponentKeyNode(doc, node.Location)
			errs = append(errs, createUnusedComponentError(pointer, errNode, config, severity))
		}
	}

	// Check component request bodies
	for _, node := range idx.ComponentRequestBodies {
		if node == nil || node.Node == nil {
			continue
		}
		pointer := node.Location.ToJSONPointer().String()
		if _, found := refs[pointer]; !found {
			// Skip if component has a usage-marking extension
			if obj := node.Node.GetObject(); obj != nil && hasUsageMarkingExtension(obj.Extensions) {
				continue
			}
			errNode := getComponentKeyNode(doc, node.Location)
			errs = append(errs, createUnusedComponentError(pointer, errNode, config, severity))
		}
	}

	// Check component headers
	for _, node := range idx.ComponentHeaders {
		if node == nil || node.Node == nil {
			continue
		}
		pointer := node.Location.ToJSONPointer().String()
		if _, found := refs[pointer]; !found {
			// Skip if component has a usage-marking extension
			if obj := node.Node.GetObject(); obj != nil && hasUsageMarkingExtension(obj.GetExtensions()) {
				continue
			}
			errNode := getComponentKeyNode(doc, node.Location)
			errs = append(errs, createUnusedComponentError(pointer, errNode, config, severity))
		}
	}

	// Check component examples
	for _, node := range idx.ComponentExamples {
		if node == nil || node.Node == nil {
			continue
		}
		pointer := node.Location.ToJSONPointer().String()
		if _, found := refs[pointer]; !found {
			// Skip if component has a usage-marking extension
			if obj := node.Node.GetObject(); obj != nil && hasUsageMarkingExtension(obj.GetExtensions()) {
				continue
			}
			errNode := getComponentKeyNode(doc, node.Location)
			errs = append(errs, createUnusedComponentError(pointer, errNode, config, severity))
		}
	}

	// Check component links
	for _, node := range idx.ComponentLinks {
		if node == nil || node.Node == nil {
			continue
		}
		pointer := node.Location.ToJSONPointer().String()
		if _, found := refs[pointer]; !found {
			// Skip if component has a usage-marking extension
			if obj := node.Node.GetObject(); obj != nil && hasUsageMarkingExtension(obj.GetExtensions()) {
				continue
			}
			errNode := getComponentKeyNode(doc, node.Location)
			errs = append(errs, createUnusedComponentError(pointer, errNode, config, severity))
		}
	}

	// Check component callbacks
	for _, node := range idx.ComponentCallbacks {
		if node == nil || node.Node == nil {
			continue
		}
		pointer := node.Location.ToJSONPointer().String()
		if _, found := refs[pointer]; !found {
			// Skip if component has a usage-marking extension
			if obj := node.Node.GetObject(); obj != nil && hasUsageMarkingExtension(obj.GetExtensions()) {
				continue
			}
			errNode := getComponentKeyNode(doc, node.Location)
			errs = append(errs, createUnusedComponentError(pointer, errNode, config, severity))
		}
	}

	// Check component path items
	for _, node := range idx.ComponentPathItems {
		if node == nil || node.Node == nil {
			continue
		}
		pointer := node.Location.ToJSONPointer().String()
		if _, found := refs[pointer]; !found {
			// Skip if component has a usage-marking extension
			if obj := node.Node.GetObject(); obj != nil && hasUsageMarkingExtension(obj.GetExtensions()) {
				continue
			}
			errNode := getComponentKeyNode(doc, node.Location)
			errs = append(errs, createUnusedComponentError(pointer, errNode, config, severity))
		}
	}

	// Check component security schemes
	for _, node := range idx.ComponentSecuritySchemes {
		if node == nil || node.Node == nil {
			continue
		}
		pointer := node.Location.ToJSONPointer().String()
		if _, found := refs[pointer]; !found {
			// Skip if component has a usage-marking extension
			if obj := node.Node.GetObject(); obj != nil && hasUsageMarkingExtension(obj.GetExtensions()) {
				continue
			}
			errNode := getComponentKeyNode(doc, node.Location)
			errs = append(errs, createUnusedComponentError(pointer, errNode, config, severity))
		}
	}

	return errs
}

func getComponentKeyNode(doc *openapi.OpenAPI, location openapi.Locations) *yaml.Node {
	if doc == nil || len(location) == 0 {
		return nil
	}
	last := location[len(location)-1]
	if last.ParentKey == nil {
		return nil
	}
	componentName := *last.ParentKey
	componentType := last.ParentField

	core := doc.GetCore()
	if core == nil {
		return nil
	}
	rootNode := core.GetRootNode()
	if !core.Components.Present || core.Components.Value == nil {
		return rootNode
	}
	componentsCore := core.Components.Value
	componentsRoot := componentsCore.GetRootNode()
	if componentsRoot == nil {
		componentsRoot = rootNode
	}

	switch componentType {
	case "schemas":
		return componentsCore.Schemas.GetMapKeyNodeOrRoot(componentName, componentsRoot)
	case "responses":
		return componentsCore.Responses.GetMapKeyNodeOrRoot(componentName, componentsRoot)
	case "parameters":
		return componentsCore.Parameters.GetMapKeyNodeOrRoot(componentName, componentsRoot)
	case "examples":
		return componentsCore.Examples.GetMapKeyNodeOrRoot(componentName, componentsRoot)
	case "requestBodies":
		return componentsCore.RequestBodies.GetMapKeyNodeOrRoot(componentName, componentsRoot)
	case "headers":
		return componentsCore.Headers.GetMapKeyNodeOrRoot(componentName, componentsRoot)
	case "securitySchemes":
		return componentsCore.SecuritySchemes.GetMapKeyNodeOrRoot(componentName, componentsRoot)
	case "links":
		return componentsCore.Links.GetMapKeyNodeOrRoot(componentName, componentsRoot)
	case "callbacks":
		return componentsCore.Callbacks.GetMapKeyNodeOrRoot(componentName, componentsRoot)
	case "pathItems":
		return componentsCore.PathItems.GetMapKeyNodeOrRoot(componentName, componentsRoot)
	default:
		return componentsRoot
	}
}

// usageMarkingExtensions is the list of extensions that mark a component as being used
// even when not directly referenced in the specification.
var usageMarkingExtensions = []string{
	"x-speakeasy-include",
	"x-include",
	"x-used",
}

// hasUsageMarkingExtension checks if the extensions contain any of the usage-marking
// extensions (x-speakeasy-include, x-include, x-used) set to true.
func hasUsageMarkingExtension(exts *extensions.Extensions) bool {
	if exts == nil {
		return false
	}

	for _, ext := range usageMarkingExtensions {
		val, err := extensions.GetExtensionValue[bool](exts, ext)
		if err == nil && val != nil && *val {
			return true
		}
	}

	return false
}

// createUnusedComponentError creates a validation error for an unused component.
func createUnusedComponentError(pointer string, errNode *yaml.Node, config *linter.RuleConfig, severity validation.Severity) error {
	componentRef := "#" + pointer
	return validation.NewValidationError(
		config.GetSeverity(severity),
		RuleSemanticUnusedComponent,
		fmt.Errorf("`%s` is potentially unused or has been orphaned", componentRef),
		errNode,
	)
}
