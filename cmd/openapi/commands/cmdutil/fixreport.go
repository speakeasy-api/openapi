package cmdutil

import (
	"fmt"
	"io"

	"github.com/speakeasy-api/openapi/linter/fix"
)

// ReportFixResults writes the fixes applied, skipped and failed by a lint command's fix engine to w.
func ReportFixResults(w io.Writer, result *fix.Result, dryRun bool) {
	prefix := ""
	if dryRun {
		prefix = "[dry-run] "
	}

	if len(result.Applied) > 0 {
		fmt.Fprintf(w, "\n%sFixed:\n", prefix)
		for _, af := range result.Applied {
			fmt.Fprintf(w, "  [%d:%d] %s - %s\n",
				af.Error.GetLineNumber(), af.Error.GetColumnNumber(),
				af.Error.Rule, af.Fix.Description())
			if af.Before != "" || af.After != "" {
				fmt.Fprintf(w, "    %s -> %s\n", af.Before, af.After)
			}
		}
	}

	if len(result.Skipped) > 0 {
		fmt.Fprintf(w, "\n%sSkipped:\n", prefix)
		for _, sf := range result.Skipped {
			fmt.Fprintf(w, "  [%d:%d] %s - %s (%s)\n",
				sf.Error.GetLineNumber(), sf.Error.GetColumnNumber(),
				sf.Error.Rule, sf.Fix.Description(), skipReasonString(sf.Reason))
		}
	}

	if len(result.Failed) > 0 {
		fmt.Fprintf(w, "\n%sFailed:\n", prefix)
		for _, ff := range result.Failed {
			fmt.Fprintf(w, "  [%d:%d] %s - %s: %v\n",
				ff.Error.GetLineNumber(), ff.Error.GetColumnNumber(),
				ff.Error.Rule, ff.Fix.Description(), ff.FixError)
		}
	}
}

func skipReasonString(reason fix.SkipReason) string {
	switch reason {
	case fix.SkipInteractive:
		return "requires interactive input"
	case fix.SkipConflict:
		return "conflict with previous fix"
	case fix.SkipUser:
		return "skipped by user"
	default:
		return "unknown"
	}
}
