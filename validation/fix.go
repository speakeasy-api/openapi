package validation

import "go.yaml.in/yaml/v4"

// PromptType describes what kind of user input a fix needs.
type PromptType int

const (
	// PromptChoice indicates the fix requires selecting from a list of options.
	PromptChoice PromptType = iota
	// PromptFreeText indicates the fix requires free-form text input.
	PromptFreeText
)

// Prompt describes a single piece of input a fix needs from the user.
type Prompt struct {
	// Type is the kind of input needed.
	Type PromptType
	// Message is a human-readable description of what input is needed and why.
	Message string
	// Choices is the list of valid choices when Type is PromptChoice.
	// Ignored for other prompt types.
	Choices []string
	// Default is an optional default value.
	Default string
}

// Fix represents a suggested fix for a validation finding.
// Fixes can be non-interactive (applied automatically) or interactive
// (requiring user input before application).
type Fix interface {
	// Description returns a human-readable description of what the fix does.
	Description() string

	// Interactive returns true if the fix requires user input before being applied.
	// Non-interactive fixes can be applied directly with Apply().
	// Interactive fixes must have SetInput() called with user responses before Apply().
	Interactive() bool

	// Prompts returns the input prompts needed for this fix.
	// Returns nil for non-interactive fixes.
	Prompts() []Prompt

	// SetInput provides user responses for interactive fixes.
	// The responses slice must correspond 1:1 with the Prompts() slice.
	// Returns an error if the input is invalid.
	// Calling this on a non-interactive fix is a no-op.
	SetInput(responses []string) error

	// Apply applies the fix to the document.
	// For interactive fixes, SetInput() must be called first.
	// The doc parameter is typically *openapi.OpenAPI.
	// Returns an error if the fix cannot be applied.
	Apply(doc any) error
}

// ChangeDescriber is an optional interface that fixes can implement to provide
// human-readable before/after descriptions of what the fix changes. This enables
// richer dry-run and reporting output.
type ChangeDescriber interface {
	DescribeChange() (before, after string)
}

// NodeFix is an optional interface for fixes that operate directly on yaml.Node
// trees rather than the high-level document model. This is useful for simple
// textual changes (renaming a key, changing a value) where going through the
// model is unnecessary.
//
// The fix engine checks for this interface first; if implemented, ApplyNode is
// called instead of Apply.
type NodeFix interface {
	Fix

	// ApplyNode applies the fix directly to the YAML node tree.
	// rootNode is the document root node.
	ApplyNode(rootNode *yaml.Node) error
}
