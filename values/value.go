package values

import (
	"unsafe"

	"go.yaml.in/yaml/v3"
	gopkgyaml "gopkg.in/yaml.v3"
)

// Value represents a raw value in an OpenAPI or Arazzo document.
type Value = *yaml.Node

// FromLegacyNode converts a legacy gopkg.in/yaml.v3.Node pointer to a values.Value.
func FromLegacyNode(n *gopkgyaml.Node) Value {
	return (*yaml.Node)(unsafe.Pointer(n)) //nolint:gosec // zero-copy conversion between identical Node memory layouts
}

// ToLegacyNode converts a values.Value to a legacy gopkg.in/yaml.v3.Node pointer.
func ToLegacyNode(v Value) *gopkgyaml.Node {
	return (*gopkgyaml.Node)(unsafe.Pointer(v)) //nolint:gosec // zero-copy conversion between identical Node memory layouts
}
