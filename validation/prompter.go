package validation

import "errors"

// ErrSkipFix is a sentinel error returned by Prompter when the user chooses
// to skip a fix. Use errors.Is(err, ErrSkipFix) to check.
var ErrSkipFix = errors.New("fix skipped by user")

// Prompter collects user input for interactive fixes.
// Implementations can be terminal-based (stdin/stdout), GUI-based, or test stubs.
type Prompter interface {
	// PromptFix presents a fix to the user and collects input for its prompts.
	// finding provides the error being fixed so the prompter can display context.
	// fix is the fix that needs input.
	// Returns the user's responses corresponding to fix.Prompts(), or an error.
	// Return ErrSkipFix (or wrap it) to indicate the user chose to skip this fix.
	PromptFix(finding *Error, fix Fix) ([]string, error)

	// Confirm asks the user a yes/no question.
	Confirm(message string) (bool, error)
}
