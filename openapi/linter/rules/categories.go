package rules

// Rule categories for OpenAPI linting

const (
	// CategorySemantic represents rules that check for semantic correctness
	// These are logical issues that would make the API spec broken or unusable
	// Examples: missing path parameters, duplicate operation IDs, invalid references
	CategorySemantic = "semantic"

	// CategoryStyle represents rules that check for formatting and naming conventions
	// These are cosmetic preferences that don't affect functionality
	// Examples: kebab-case paths, camelCase properties, trailing slashes
	CategoryStyle = "style"

	// CategorySecurity represents rules that check for security concerns
	// Examples: OWASP rules, API keys in URLs, HTTPS enforcement
	CategorySecurity = "security"

	// CategorySchemas represents rules that check for schema-related issues
	// Examples: nullable keyword usage, schema validation, type constraints
	CategorySchemas = "schemas"
)
