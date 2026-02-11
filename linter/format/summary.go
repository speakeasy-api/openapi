package format

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/speakeasy-api/openapi/validation"
)

// SummaryFormatter formats results as a per-rule summary table.
type SummaryFormatter struct{}

// NewSummaryFormatter creates a new SummaryFormatter.
func NewSummaryFormatter() *SummaryFormatter {
	return &SummaryFormatter{}
}

type ruleSummary struct {
	rule     string
	category string
	severity validation.Severity
	count    int
}

// Format outputs a per-rule summary table sorted by count descending.
func (f *SummaryFormatter) Format(results []error) (string, error) {
	byRule := make(map[string]*ruleSummary)

	errorCount := 0
	warningCount := 0
	hintCount := 0

	for _, err := range results {
		var vErr *validation.Error
		if errors.As(err, &vErr) {
			rs, ok := byRule[vErr.Rule]
			if !ok {
				category := "unknown"
				if idx := strings.Index(vErr.Rule, "-"); idx > 0 {
					category = vErr.Rule[:idx]
				}
				rs = &ruleSummary{
					rule:     vErr.Rule,
					category: category,
					severity: vErr.Severity,
				}
				byRule[vErr.Rule] = rs
			}
			rs.count++

			switch vErr.Severity {
			case validation.SeverityError:
				errorCount++
			case validation.SeverityWarning:
				warningCount++
			case validation.SeverityHint:
				hintCount++
			}
		} else {
			rs, ok := byRule["internal"]
			if !ok {
				rs = &ruleSummary{
					rule:     "internal",
					category: "internal",
					severity: validation.SeverityError,
				}
				byRule["internal"] = rs
			}
			rs.count++
			errorCount++
		}
	}

	// Sort by count descending, then by rule name
	sorted := make([]*ruleSummary, 0, len(byRule))
	for _, rs := range byRule {
		sorted = append(sorted, rs)
	}
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].count != sorted[j].count {
			return sorted[i].count > sorted[j].count
		}
		return sorted[i].rule < sorted[j].rule
	})

	var sb strings.Builder

	// Header
	fmt.Fprintf(&sb, "%-50s %8s %10s %8s\n", "Rule", "Severity", "Category", "Count")
	sb.WriteString(strings.Repeat("─", 80))
	sb.WriteString("\n")

	for _, rs := range sorted {
		fmt.Fprintf(&sb, "%-50s %8s %10s %8d\n", rs.rule, rs.severity, rs.category, rs.count)
	}

	sb.WriteString(strings.Repeat("─", 80))
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "✖ %d problems (%d errors, %d warnings, %d hints) across %d rules\n",
		len(results), errorCount, warningCount, hintCount, len(byRule))

	return sb.String(), nil
}
