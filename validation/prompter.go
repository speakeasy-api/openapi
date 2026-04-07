package validation

import "errors"

// ErrSkipFix is a sentinel error returned by Prompter when the user chooses
// to skip a single fix. Use errors.Is(err, ErrSkipFix) to check.
var ErrSkipFix = errors.New("fix skipped by user")

// ErrSkipRule is a sentinel error returned by Prompter when the user chooses
// to skip all remaining fixes for the current rule in this run.
var ErrSkipRule = errors.New("remaining fixes for rule skipped by user")

// ErrExitInteractive is a sentinel error returned by Prompter when the user
// chooses to exit interactive fixing early while keeping applied fixes.
var ErrExitInteractive = errors.New("interactive fixing exited by user")

// Prompter collects user input for interactive fixes.
// Implementations can be terminal-based (stdin/stdout), GUI-based, or test stubs.
type Prompter interface {
	// PromptFix presents a fix to the user and collects input for its prompts.
	// finding provides the error being fixed so the prompter can display context.
	// fix is the fix that needs input.
	// Returns the user's responses corresponding to fix.Prompts(), or an error.
	// Return ErrSkipFix to skip this fix, ErrSkipRule to skip remaining fixes for
	// the current rule, or ErrExitInteractive to stop interactive fixing early.
	PromptFix(finding *Error, fix Fix) ([]string, error)

	// Confirm asks the user a yes/no question.
	Confirm(message string) (bool, error)
}
