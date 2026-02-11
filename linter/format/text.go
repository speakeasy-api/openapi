package format

import (
	"errors"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/validation"
)

type TextFormatter struct{}

func NewTextFormatter() *TextFormatter {
	return &TextFormatter{}
}

func (f *TextFormatter) Format(results []error) (string, error) {
	var sb strings.Builder

	type textRow struct {
		location string
		severity string
		rule     string
		message  string
	}

	rows := make([]textRow, 0, len(results))
	maxLocationWidth := 1
	maxSeverityWidth := len(validation.SeverityError.String())
	maxRuleWidth := len("internal")

	errorCount := 0
	warningCount := 0
	hintCount := 0

	for _, err := range results {
		var vErr *validation.Error
		if errors.As(err, &vErr) {
			location := fmt.Sprintf("%d:%d", vErr.GetLineNumber(), vErr.GetColumnNumber())
			severity := vErr.Severity.String()
			rule := vErr.Rule
			msg := vErr.UnderlyingError.Error()
			if vErr.DocumentLocation != "" {
				msg += " (document: " + vErr.DocumentLocation + ")"
			}

			fixable := ""
			if vErr.Fix != nil {
				fixable = " [fixable]"
			}

			rows = append(rows, textRow{
				location: location,
				severity: severity,
				rule:     rule,
				message:  msg + fixable,
			})

			if len(location) > maxLocationWidth {
				maxLocationWidth = len(location)
			}
			if len(severity) > maxSeverityWidth {
				maxSeverityWidth = len(severity)
			}
			if len(rule) > maxRuleWidth {
				maxRuleWidth = len(rule)
			}

			switch vErr.Severity {
			case validation.SeverityError:
				errorCount++
			case validation.SeverityWarning:
				warningCount++
			case validation.SeverityHint:
				hintCount++
			}
		} else {
			rows = append(rows, textRow{
				location: "-",
				severity: validation.SeverityError.String(),
				rule:     "internal",
				message:  err.Error(),
			})
			errorCount++
		}
	}

	for _, row := range rows {
		fmt.Fprintf(&sb, "%*s %-*s %-*s %s\n", maxLocationWidth, row.location, maxSeverityWidth, row.severity, maxRuleWidth, row.rule, row.message)
	}

	if len(results) > 0 {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("✖ %d problems (%d errors, %d warnings, %d hints)\n", len(results), errorCount, warningCount, hintCount))
	}

	return sb.String(), nil
}
