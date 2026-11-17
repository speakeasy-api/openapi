package rules

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"go.yaml.in/yaml/v3"
)

const RuleSemanticUnusedComponent = "semantic-unused-component"

// componentSections are the component types that can be referenced.
var componentSections = []string{"inputs", "parameters", "successActions", "failureActions"}

var (
	componentExpressionRegex = regexp.MustCompile(`\$components\.(inputs|parameters|successActions|failureActions)\.([A-Za-z0-9_.\-]+)`)
	componentPointerRegex    = regexp.MustCompile(`#/components/(inputs|parameters|successActions|failureActions)/([^/\s"'#]+)`)
)

type UnusedComponentRule struct{}

func (r *UnusedComponentRule) ID() string       { return RuleSemanticUnusedComponent }
func (r *UnusedComponentRule) Category() string { return CategorySemantic }
func (r *UnusedComponentRule) Description() string {
	return "Components that are never referenced should be removed. Inputs are referenced with `$ref: '#/components/inputs/<name>'`, and parameters and actions with `reference: $components.<type>.<name>`. A component only referenced by itself is unused."
}
func (r *UnusedComponentRule) Summary() string {
	return "Components should be referenced somewhere in the document."
}
func (r *UnusedComponentRule) HowToFix() string {
	return "Remove the component, or reference it where it was meant to be used."
}
func (r *UnusedComponentRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/arazzo/linter/README.md#semantic-unused-component"
}
func (r *UnusedComponentRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *UnusedComponentRule) Versions() []string { return nil }
func (r *UnusedComponentRule) FixAvailable() bool { return true }

func (r *UnusedComponentRule) Run(_ context.Context, docInfo *linter.DocumentInfo[*arazzo.Arazzo], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Document.Components == nil {
		return nil
	}

	root := docInfo.Document.GetRootNode()
	used := map[string]map[string]bool{}
	for _, section := range componentSections {
		used[section] = map[string]bool{}
	}

	collect := func(node *yaml.Node, ownerSection, ownerName string) {
		visitScalars(node, func(scalar *yaml.Node) {
			for _, m := range componentExpressionRegex.FindAllStringSubmatch(scalar.Value, -1) {
				markComponentUse(used, m[1], m[2], ownerSection, ownerName)
			}
			for _, m := range componentPointerRegex.FindAllStringSubmatch(scalar.Value, -1) {
				name := strings.NewReplacer("~1", "/", "~0", "~").Replace(m[2])
				if used[m[1]] != nil && (m[1] != ownerSection || name != ownerName) {
					used[m[1]][name] = true
				}
			}
		})
	}

	// References from outside the components section, then references between components, ignoring a component's
	// references to itself.
	if root != nil && root.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(root.Content); i += 2 {
			if root.Content[i].Value != "components" {
				collect(root.Content[i+1], "", "")
			}
		}
	}
	componentsNode := mapValue(root, "components")
	for _, section := range componentSections {
		sectionNode := mapValue(componentsNode, section)
		if sectionNode == nil || sectionNode.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(sectionNode.Content); i += 2 {
			collect(sectionNode.Content[i+1], section, sectionNode.Content[i].Value)
		}
	}

	c := docInfo.Document.Components
	core := c.GetCore()
	severity := config.GetSeverity(r.DefaultSeverity())

	var errs []error
	report := func(section, name string, node validation.MapKeyNodeGetter) {
		if used[section][name] {
			return
		}
		err := validation.NewMapKeyError(severity, RuleSemanticUnusedComponent, fmt.Errorf("component `%s` in `components.%s` is never referenced", name, section), core, node, name)
		errs = append(errs, withFix(err, &removeComponentFix{section: section, name: name}))
	}

	for name := range c.Inputs.Keys() {
		report("inputs", name, core.Inputs)
	}
	for name := range c.Parameters.Keys() {
		report("parameters", name, core.Parameters)
	}
	for name := range c.SuccessActions.Keys() {
		report("successActions", name, core.SuccessActions)
	}
	for name := range c.FailureActions.Keys() {
		report("failureActions", name, core.FailureActions)
	}

	return errs
}

// markComponentUse marks a `$components.<section>.<name>` reference as used. Component names may contain dots, so
// the name and each of its dotted prefixes are marked.
func markComponentUse(used map[string]map[string]bool, section, name, ownerSection, ownerName string) {
	if section == ownerSection {
		// A reference to the component itself (or a dotted path within it) doesn't count as a use.
		if name == ownerName || strings.HasPrefix(name, ownerName+".") {
			return
		}
	}
	if used[section] == nil {
		return
	}
	markDottedName(used[section], name)
}

// removeComponentFix removes an unused component.
type removeComponentFix struct {
	section string
	name    string
}

func (f *removeComponentFix) Description() string {
	return fmt.Sprintf("Remove unused component `%s` from `components.%s`", f.name, f.section)
}
func (f *removeComponentFix) Interactive() bool            { return false }
func (f *removeComponentFix) Prompts() []validation.Prompt { return nil }
func (f *removeComponentFix) SetInput([]string) error      { return nil }

func (f *removeComponentFix) Apply(doc any) error {
	a, ok := doc.(*arazzo.Arazzo)
	if !ok {
		return fmt.Errorf("expected *arazzo.Arazzo, got %T", doc)
	}
	c := a.Components
	if c == nil {
		return fmt.Errorf("component `%s` not found in `components.%s`", f.name, f.section)
	}

	var err error
	switch f.section {
	case "inputs":
		err = deleteComponent(c.Inputs, f.section, f.name)
	case "parameters":
		err = deleteComponent(c.Parameters, f.section, f.name)
	case "successActions":
		err = deleteComponent(c.SuccessActions, f.section, f.name)
	default:
		err = deleteComponent(c.FailureActions, f.section, f.name)
	}
	if err != nil {
		return err
	}

	// Remove sections, and the components object, left empty so the document doesn't keep empty maps.
	if c.Inputs.Len() == 0 {
		c.Inputs = nil
	}
	if c.Parameters.Len() == 0 {
		c.Parameters = nil
	}
	if c.SuccessActions.Len() == 0 {
		c.SuccessActions = nil
	}
	if c.FailureActions.Len() == 0 {
		c.FailureActions = nil
	}
	if c.Inputs == nil && c.Parameters == nil && c.SuccessActions == nil && c.FailureActions == nil && c.Extensions.Len() == 0 {
		a.Components = nil
	}
	return nil
}

func (f *removeComponentFix) DescribeChange() (before, after string) {
	return fmt.Sprintf("components.%s.%s", f.section, f.name), ""
}

type componentMap interface {
	Has(key string) bool
	Delete(key string)
}

func deleteComponent(m componentMap, section, name string) error {
	if m == nil || !m.Has(name) {
		return fmt.Errorf("component `%s` not found in `components.%s`", name, section)
	}
	m.Delete(name)
	return nil
}
