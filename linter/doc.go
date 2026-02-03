package linter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// DocGenerator generates documentation from registered rules
type DocGenerator[T any] struct {
	registry *Registry[T]
}

// NewDocGenerator creates a new documentation generator
func NewDocGenerator[T any](registry *Registry[T]) *DocGenerator[T] {
	return &DocGenerator[T]{registry: registry}
}

// RuleDoc represents documentation for a single rule
type RuleDoc struct {
	ID              string         `json:"id" yaml:"id"`
	Category        string         `json:"category" yaml:"category"`
	Summary         string         `json:"summary" yaml:"summary"`
	Description     string         `json:"description" yaml:"description"`
	Rationale       string         `json:"rationale,omitempty" yaml:"rationale,omitempty"`
	Link            string         `json:"link,omitempty" yaml:"link,omitempty"`
	DefaultSeverity string         `json:"default_severity" yaml:"default_severity"`
	Versions        []string       `json:"versions,omitempty" yaml:"versions,omitempty"`
	GoodExample     string         `json:"good_example,omitempty" yaml:"good_example,omitempty"`
	BadExample      string         `json:"bad_example,omitempty" yaml:"bad_example,omitempty"`
	FixAvailable    bool           `json:"fix_available" yaml:"fix_available"`
	ConfigSchema    map[string]any `json:"config_schema,omitempty" yaml:"config_schema,omitempty"`
	ConfigDefaults  map[string]any `json:"config_defaults,omitempty" yaml:"config_defaults,omitempty"`
	Rulesets        []string       `json:"rulesets" yaml:"rulesets"`
}

// GenerateRuleDoc generates documentation for a single rule
func (g *DocGenerator[T]) GenerateRuleDoc(rule RuleRunner[T]) *RuleDoc {
	doc := &RuleDoc{
		ID:              rule.ID(),
		Category:        rule.Category(),
		Summary:         rule.Summary(),
		Description:     rule.Description(),
		Link:            rule.Link(),
		DefaultSeverity: rule.DefaultSeverity().String(),
		Versions:        rule.Versions(),
		Rulesets:        g.registry.RulesetsContaining(rule.ID()),
	}

	// Check for optional documentation interface
	if documented, ok := any(rule).(DocumentedRule); ok {
		doc.GoodExample = documented.GoodExample()
		doc.BadExample = documented.BadExample()
		doc.Rationale = documented.Rationale()
		doc.FixAvailable = documented.FixAvailable()
	}

	// Check for configuration interface
	if configurable, ok := any(rule).(ConfigurableRule); ok {
		doc.ConfigSchema = configurable.ConfigSchema()
		doc.ConfigDefaults = configurable.ConfigDefaults()
	}

	return doc
}

// GenerateAllRuleDocs generates documentation for all registered rules
func (g *DocGenerator[T]) GenerateAllRuleDocs() []*RuleDoc {
	var docs []*RuleDoc
	for _, rule := range g.registry.AllRules() {
		docs = append(docs, g.GenerateRuleDoc(rule))
	}
	return docs
}

// GenerateCategoryDocs groups rules by category
func (g *DocGenerator[T]) GenerateCategoryDocs() map[string][]*RuleDoc {
	categories := make(map[string][]*RuleDoc)
	for _, rule := range g.registry.AllRules() {
		doc := g.GenerateRuleDoc(rule)
		categories[doc.Category] = append(categories[doc.Category], doc)
	}
	return categories
}

// WriteJSON writes rule documentation as JSON
func (g *DocGenerator[T]) WriteJSON(w io.Writer) error {
	docs := g.GenerateAllRuleDocs()
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{
		"rules":      docs,
		"categories": g.registry.AllCategories(),
		"rulesets":   g.registry.AllRulesets(),
	})
}

// WriteMarkdown writes rule documentation as Markdown
func (g *DocGenerator[T]) WriteMarkdown(w io.Writer) error {
	docs := g.GenerateCategoryDocs()

	if err := writeLine(w, "# Lint Rules Reference"); err != nil {
		return err
	}
	if err := writeEmptyLine(w); err != nil {
		return err
	}

	// Table of contents
	if err := writeLine(w, "## Categories"); err != nil {
		return err
	}
	if err := writeEmptyLine(w); err != nil {
		return err
	}
	for category := range docs {
		if err := writeF(w, "- [%s](#%s)\n", category, category); err != nil {
			return err
		}
	}
	if err := writeEmptyLine(w); err != nil {
		return err
	}

	// Rules by category
	for category, rules := range docs {
		if err := writeF(w, "## %s\n\n", category); err != nil {
			return err
		}

		for _, rule := range rules {
			if err := g.writeRuleMarkdown(w, rule); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *DocGenerator[T]) writeRuleMarkdown(w io.Writer, rule *RuleDoc) error {
	if err := writeF(w, "### %s\n\n", rule.ID); err != nil {
		return err
	}
	if err := writeF(w, "**Severity:** %s  \n", rule.DefaultSeverity); err != nil {
		return err
	}
	if err := writeF(w, "**Category:** %s  \n", rule.Category); err != nil {
		return err
	}
	if rule.Summary != "" {
		if err := writeF(w, "**Summary:** %s  \n", rule.Summary); err != nil {
			return err
		}
	}

	if len(rule.Versions) > 0 {
		if err := writeF(w, "**Applies to:** %s  \n", strings.Join(rule.Versions, ", ")); err != nil {
			return err
		}
	}

	if rule.FixAvailable {
		if err := writeLine(w, "**Auto-fix available:** Yes  "); err != nil {
			return err
		}
	}
	if err := writeEmptyLine(w); err != nil {
		return err
	}

	if err := writeF(w, "%s\n\n", rule.Description); err != nil {
		return err
	}

	if rule.Rationale != "" {
		if err := writeF(w, "#### Rationale\n\n%s\n\n", rule.Rationale); err != nil {
			return err
		}
	}

	if rule.BadExample != "" {
		if err := writeLine(w, "#### ❌ Incorrect"); err != nil {
			return err
		}
		if err := writeLine(w, "```yaml"); err != nil {
			return err
		}
		if err := writeLine(w, rule.BadExample); err != nil {
			return err
		}
		if err := writeLine(w, "```"); err != nil {
			return err
		}
		if err := writeEmptyLine(w); err != nil {
			return err
		}
	}

	if rule.GoodExample != "" {
		if err := writeLine(w, "#### ✅ Correct"); err != nil {
			return err
		}
		if err := writeLine(w, "```yaml"); err != nil {
			return err
		}
		if err := writeLine(w, rule.GoodExample); err != nil {
			return err
		}
		if err := writeLine(w, "```"); err != nil {
			return err
		}
		if err := writeEmptyLine(w); err != nil {
			return err
		}
	}

	if len(rule.ConfigSchema) > 0 {
		if err := writeLine(w, "#### Configuration"); err != nil {
			return err
		}
		if err := writeEmptyLine(w); err != nil {
			return err
		}
		if err := writeLine(w, "| Option | Type | Default | Description |"); err != nil {
			return err
		}
		if err := writeLine(w, "|--------|------|---------|-------------|"); err != nil {
			return err
		}
		// Write config options table
		if err := writeEmptyLine(w); err != nil {
			return err
		}
	}

	if rule.Link != "" {
		if err := writeF(w, "[Documentation →](%s)\n\n", rule.Link); err != nil {
			return err
		}
	}

	if err := writeLine(w, "---"); err != nil {
		return err
	}
	if err := writeEmptyLine(w); err != nil {
		return err
	}

	return nil
}

func writeLine(w io.Writer, text string) error {
	_, err := fmt.Fprintln(w, text)
	return err
}

func writeEmptyLine(w io.Writer) error {
	_, err := fmt.Fprintln(w)
	return err
}

func writeF(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format, args...)
	return err
}
