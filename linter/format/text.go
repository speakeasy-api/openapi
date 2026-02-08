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

	errorCount := 0
	warningCount := 0
	hintCount := 0

	for _, err := range results {
		var vErr *validation.Error
		if errors.As(err, &vErr) {
			line := vErr.GetLineNumber()
			col := vErr.GetColumnNumber()
			severity := vErr.Severity
			rule := vErr.Rule
			msg := vErr.UnderlyingError.Error()
			if vErr.DocumentLocation != "" {
				msg = fmt.Sprintf("%s (document: %s)", msg, vErr.DocumentLocation)
			}

			fixable := ""
			if vErr.Fix != nil {
				fixable = " [fixable]"
			}

			fmt.Fprintf(&sb, "%d:%d\t%s\t%s\t%s%s\n", line, col, severity, rule, msg, fixable)

			switch severity {
			case validation.SeverityError:
				errorCount++
			case validation.SeverityWarning:
				warningCount++
			case validation.SeverityHint:
				hintCount++
			}
		} else {
			// Non-validation error
			sb.WriteString(fmt.Sprintf("-\t-\terror\tinternal\t%s\n", err.Error()))
			errorCount++
		}
	}

	if len(results) > 0 {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("✖ %d problems (%d errors, %d warnings, %d hints)\n", len(results), errorCount, warningCount, hintCount))
	}

	return sb.String(), nil
}
