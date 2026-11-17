package arazzo

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	arazzoLinter "github.com/speakeasy-api/openapi/arazzo/linter"
	"github.com/speakeasy-api/openapi/cmd/openapi/commands/cmdutil"
	"github.com/spf13/cobra"
)

var listRulesCmd = &cobra.Command{
	Use:   "list-rules",
	Short: "List all available Arazzo linting rules",
	Long: `List all available Arazzo linting rules with their metadata.

Shows each rule's ID, category, default severity, summary, and fix guidance. Use --format json to include the
full description. Use --category to filter by category, or --ruleset to show only rules in a ruleset.

Examples:
  openapi arazzo list-rules
  openapi arazzo list-rules --category semantic
  openapi arazzo list-rules --ruleset recommended
  openapi arazzo list-rules --format json`,
	Args: cobra.NoArgs,
	Run:  runListRules,
}

// listRulesFlags are the flags of the list-rules command.
type listRulesFlags struct {
	format   string
	category string
	ruleset  string
}

var arazzoListRulesFlags listRulesFlags

func init() {
	listRulesCmd.Flags().StringVarP(&arazzoListRulesFlags.format, "format", "f", "text", "Output format: text or json")
	listRulesCmd.Flags().StringVar(&arazzoListRulesFlags.category, "category", "", "Filter by category (semantic, style or security)")
	listRulesCmd.Flags().StringVar(&arazzoListRulesFlags.ruleset, "ruleset", "", "Filter by ruleset (all or recommended)")
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

func runListRules(_ *cobra.Command, _ []string) {
	if err := listRules(os.Stdout, arazzoListRulesFlags); err != nil {
		cmdutil.Die(err)
	}
}

func listRules(w io.Writer, flags listRulesFlags) error {
	registry := arazzoLinter.NewLinter(nil).Registry()
	if flags.ruleset != "" && !slices.Contains(registry.AllRulesets(), flags.ruleset) {
		return fmt.Errorf("unknown ruleset `%s`, available rulesets: %s", flags.ruleset, strings.Join(registry.AllRulesets(), ", "))
	}

	infos := []ruleInfo{}
	for _, rule := range registry.AllRules() {
		rulesets := registry.RulesetsContaining(rule.ID())
		if flags.category != "" && rule.Category() != flags.category {
			continue
		}
		if flags.ruleset != "" && !slices.Contains(rulesets, flags.ruleset) {
			continue
		}

		info := ruleInfo{
			ID:              rule.ID(),
			Category:        rule.Category(),
			DefaultSeverity: rule.DefaultSeverity().String(),
			Summary:         rule.Summary(),
			Description:     rule.Description(),
			Link:            rule.Link(),
			Rulesets:        rulesets,
		}
		if fixer, ok := rule.(interface{ HowToFix() string }); ok {
			info.HowToFix = fixer.HowToFix()
		}
		if fixable, ok := rule.(interface{ FixAvailable() bool }); ok {
			info.FixAvailable = fixable.FixAvailable()
		}
		infos = append(infos, info)
	}

	if flags.format == "json" {
		data, err := json.MarshalIndent(infos, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal rules: %w", err)
		}
		fmt.Fprintln(w, string(data))
		return nil
	}

	printRulesText(w, infos, registry.AllCategories())
	return nil
}

func printRulesText(w io.Writer, infos []ruleInfo, categories []string) {
	if len(infos) == 0 {
		fmt.Fprintln(w, "No rules found matching the specified filters.")
		return
	}

	byCategory := map[string][]ruleInfo{}
	for _, info := range infos {
		byCategory[info.Category] = append(byCategory[info.Category], info)
	}

	for _, category := range categories {
		rules, ok := byCategory[category]
		if !ok {
			continue
		}

		fmt.Fprintf(w, "\n%s (%s)\n", strings.ToUpper(category), pluralRules(len(rules)))
		fmt.Fprintln(w, strings.Repeat("─", 80))

		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, info := range rules {
			fixMarker := ""
			if info.FixAvailable {
				fixMarker = " [fixable]"
			}
			fmt.Fprintf(tw, "  %s\t%s\t[%s]%s\n", info.ID, info.Summary, info.DefaultSeverity, fixMarker)
			fmt.Fprintf(tw, "  \tFix: %s\n", info.HowToFix)
			fmt.Fprintf(tw, "  \tDocs: %s\n", info.Link)
			fmt.Fprintf(tw, "  \tRulesets: %s\n", strings.Join(info.Rulesets, ", "))
		}
		_ = tw.Flush()
	}

	fmt.Fprintf(w, "\n%s total\n", pluralRules(len(infos)))
}

// pluralRules returns "1 rule" or "<n> rules".
func pluralRules(n int) string {
	if n == 1 {
		return "1 rule"
	}
	return strconv.Itoa(n) + " rules"
}

// GetListRulesCommand returns the list-rules command for external use
func GetListRulesCommand() *cobra.Command {
	return listRulesCmd
}
