package rules

// Rule categories for Arazzo linting

const (
	// CategorySemantic represents rules that check for logical issues that make a workflow broken, unreachable or
	// harder to maintain. Examples: unreachable steps, dependency cycles, unused components.
	CategorySemantic = "semantic"

	// CategoryStyle represents rules that check for documentation and authoring conventions.
	// Examples: missing descriptions, steps without success criteria.
	CategoryStyle = "style"

	// CategorySecurity represents rules that check for security concerns.
	// Examples: source descriptions loaded over plain HTTP.
	CategorySecurity = "security"
)
