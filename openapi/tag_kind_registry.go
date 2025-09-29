package openapi

// TagKind represents commonly used values for the Tag.Kind field.
// These values are registered in the OpenAPI Initiative's Tag Kind Registry
// at https://spec.openapis.org/registry/tag-kind/
type TagKind string

// Officially registered Tag Kind values from the OpenAPI Initiative registry
const (
	// TagKindNav represents tags used for navigation purposes
	TagKindNav TagKind = "nav"

	// TagKindBadge represents tags used for visible badges or labels
	TagKindBadge TagKind = "badge"

	// TagKindAudience represents tags that categorize operations by target audience
	TagKindAudience TagKind = "audience"
)

// String returns the string representation of the TagKind
func (tk TagKind) String() string {
	return string(tk)
}

// IsRegistered checks if the TagKind value is one of the officially registered values
func (tk TagKind) IsRegistered() bool {
	switch tk {
	case TagKindNav, TagKindBadge, TagKindAudience:
		return true
	default:
		return false
	}
}

// GetRegisteredTagKinds returns all officially registered tag kind values
func GetRegisteredTagKinds() []TagKind {
	return []TagKind{
		TagKindNav,
		TagKindBadge,
		TagKindAudience,
	}
}

// TagKindDescriptions provides human-readable descriptions for each registered tag kind
var TagKindDescriptions = map[TagKind]string{
	TagKindNav:      "Navigation - Used for structuring API documentation navigation",
	TagKindBadge:    "Badge - Used for visible badges or labels in documentation",
	TagKindAudience: "Audience - Used to categorize operations by target audience",
}

// GetTagKindDescription returns a human-readable description for a tag kind
func GetTagKindDescription(kind TagKind) string {
	if desc, exists := TagKindDescriptions[kind]; exists {
		return desc
	}
	return "Custom tag kind - not in the official registry (any string value is allowed)"
}
