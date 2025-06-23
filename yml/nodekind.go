package yml

import "gopkg.in/yaml.v3"

// NodeKindToString returns a human-readable string representation of a yaml.Kind.
// This helper function is useful for creating more user-friendly error messages
// when dealing with YAML node kinds in error reporting.
func NodeKindToString(kind yaml.Kind) string {
	switch kind {
	case yaml.DocumentNode:
		return "document"
	case yaml.SequenceNode:
		return "sequence"
	case yaml.MappingNode:
		return "mapping"
	case yaml.ScalarNode:
		return "scalar"
	case yaml.AliasNode:
		return "alias"
	default:
		return "unknown"
	}
}
