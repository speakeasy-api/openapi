package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/speakeasy-api/openapi/linter"
	openapiLinter "github.com/speakeasy-api/openapi/openapi/linter"
	"github.com/spf13/cobra"
)

var listRulesCmd = &cobra.Command{
	Use:   "list-rules",
	Short: "List all available linting rules",
	Long: `List all available linting rules with their metadata.

Shows each rule's ID, category, default severity, description, and fix guidance.
Use --category to filter by category, or --ruleset to show only rules in a ruleset.

Examples:
  openapi spec list-rules
  openapi spec list-rules --category security
  openapi spec list-rules --ruleset recommended
  openapi spec list-rules --format json`,
	Run: runListRules,
}

var (
	listRulesFormat   string
	listRulesCategory string
	listRulesRuleset  string
)

func init() {
	listRulesCmd.Flags().StringVarP(&listRulesFormat, "format", "f", "text", "Output format: text or json")
	listRulesCmd.Flags().StringVar(&listRulesCategory, "category", "", "Filter by category (e.g., security, style, semantic)")
	listRulesCmd.Flags().StringVar(&listRulesRuleset, "ruleset", "", "Filter by ruleset (e.g., recommended, security, all)")
}

// howToFixer is the interface satisfied by rules that provide fix guidance.
type howToFixer interface {
	HowToFix() string
}

type ruleInfo struct {
	ID              string   `json:"id"`
	Category        string   `json:"category"`
	DefaultSeverity string   `json:"defaultSeverity"`
	Summary         string   `json:"summary"`
	Description     string   `json:"description"`
	HowToFix        string   `json:"howToFix,omitempty"`
	FixAvailable    bool     `json:"fixAvailable,omitempty"`
	Link            string   `json:"link,omitempty"`
	Rulesets        []string `json:"rulesets"`
}

func runListRules(cmd *cobra.Command, _ []string) {
	config := linter.NewConfig()
	lint, err := openapiLinter.NewLinter(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	registry := lint.Registry()
	allRules := registry.AllRules()

	var infos []ruleInfo
	for _, rule := range allRules {
		// Apply category filter
		if listRulesCategory != "" && rule.Category() != listRulesCategory {
			continue
		}

		// Apply ruleset filter
		if listRulesRuleset != "" {
			rulesets := registry.RulesetsContaining(rule.ID())
			found := false
			for _, rs := range rulesets {
				if rs == listRulesRuleset {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		info := ruleInfo{
			ID:              rule.ID(),
			Category:        rule.Category(),
			DefaultSeverity: rule.DefaultSeverity().String(),
			Summary:         rule.Summary(),
			Description:     rule.Description(),
			Link:            rule.Link(),
			Rulesets:        registry.RulesetsContaining(rule.ID()),
		}

		if fixer, ok := rule.(howToFixer); ok {
			info.HowToFix = fixer.HowToFix()
		}

		if fixable, ok := rule.(interface{ FixAvailable() bool }); ok {
			info.FixAvailable = fixable.FixAvailable()
		}

		infos = append(infos, info)
	}

	switch listRulesFormat {
	case "json":
		printRulesJSON(infos)
	default:
		printRulesText(cmd, infos, registry.AllCategories())
	}
}

func printRulesText(_ *cobra.Command, infos []ruleInfo, categories []string) {
	if len(infos) == 0 {
		fmt.Println("No rules found matching the specified filters.")
		return
	}

	// Group by category
	byCategory := make(map[string][]ruleInfo)
	for _, info := range infos {
		byCategory[info.Category] = append(byCategory[info.Category], info)
	}

	// Print in category order
	for _, cat := range categories {
		rules, ok := byCategory[cat]
		if !ok {
			continue
		}

		fmt.Printf("\n%s (%d rules)\n", strings.ToUpper(cat), len(rules))
		fmt.Println(strings.Repeat("─", 80))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, info := range rules {
			fixMarker := ""
			if info.FixAvailable {
				fixMarker = " [fixable]"
			}
			fmt.Fprintf(w, "  %s\t%s\t[%s]%s\n", info.ID, info.Summary, info.DefaultSeverity, fixMarker)
			if info.HowToFix != "" {
				fmt.Fprintf(w, "  \tFix: %s\n", info.HowToFix)
			}
			if info.Link != "" {
				fmt.Fprintf(w, "  \tDocs: %s\n", info.Link)
			}
			fmt.Fprintf(w, "  \tRulesets: %s\n", strings.Join(info.Rulesets, ", "))
		}
		w.Flush()
	}

	fmt.Printf("\n%d rules total\n", len(infos))
}

func printRulesJSON(infos []ruleInfo) {
	bytes, err := json.MarshalIndent(infos, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(bytes))
}
